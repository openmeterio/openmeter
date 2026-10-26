package billing

import (
	"encoding/json"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/lineengine"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/timeutil"
)

func (s *BillingAdapterTestSuite) TestLegacyCalculationInvariants() {
	// Given a monthly minimum of 100, persisted earlier billing of 40, and current usage of 20.
	ctx := s.T().Context()
	namespace := s.GetUniqueNamespace("legacy-calculation-context")
	invoice := s.setupInvoice(ctx, namespace)
	invoice.Status = billing.StandardInvoiceStatusDraftCreated
	_, err := s.BillingAdapter.UpdateStandardInvoice(ctx, *invoice)
	s.Require().NoError(err)
	testFeature := s.SetupApiRequestsTotalFeature(ctx, namespace)
	defer testFeature.Cleanup()
	fullPeriod := timeutil.ClosedPeriod{
		From: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		To:   time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC),
	}
	boundary := fullPeriod.From.Add(fullPeriod.Duration() / 2)
	clock.FreezeTime(fullPeriod.To)
	defer clock.UnFreeze()
	price := productcatalog.NewPriceFrom(productcatalog.UnitPrice{
		Amount: alpacadecimal.NewFromInt(1),
		Commitments: productcatalog.Commitments{
			MinimumAmount: lo.ToPtr(alpacadecimal.NewFromInt(100)),
		},
	})
	group, err := s.BillingAdapter.CreateSplitLineGroup(ctx, billing.CreateSplitLineGroupAdapterInput{
		Namespace: namespace,
		SplitLineGroupMutableFields: billing.SplitLineGroupMutableFields{
			Name:          "monthly usage",
			ServicePeriod: fullPeriod,
		},
		Currency:   invoice.Currency,
		Price:      price,
		FeatureKey: &testFeature.Feature.Key,
	})
	s.Require().NoError(err)
	prior := newLine(newLineInput{
		Namespace: namespace,
		Invoice:   invoice,
		Name:      "earlier billing",
		Period:    timeutil.ClosedPeriod{From: fullPeriod.From, To: boundary},
	})
	prior.SplitLineGroupID = &group.ID
	prior.UsageBased.FeatureKey = testFeature.Feature.Key
	prior.Totals.Amount = alpacadecimal.NewFromInt(40)
	historyQueries := &splitLineHistoryQueryCounter{}
	s.DBClient.BillingInvoiceSplitLineGroup.Intercept(historyQueries)
	persistedLines, err := s.BillingAdapter.UpsertInvoiceLines(ctx, billing.UpsertInvoiceLinesAdapterInput{
		Namespace: namespace,
		InvoiceID: invoice.ID,
		Lines:     billing.StandardLines{prior},
	})
	s.Require().NoError(err)
	s.Require().Len(persistedLines, 1)
	prior = persistedLines[0]

	// Ordinary line persistence and reads do not depend on complete group history.
	loadedInvoice, err := s.BillingAdapter.GetStandardInvoiceById(ctx, billing.GetStandardInvoiceByIdInput{
		Invoice: invoice.GetInvoiceID(),
		Expand:  billing.StandardInvoiceExpandAll,
	})
	s.Require().NoError(err)
	s.Require().Len(loadedInvoice.Lines.OrEmpty(), 1)
	s.Equal(group.ID, *loadedInvoice.Lines.OrEmpty()[0].SplitLineGroupID)
	listedLines, err := s.BillingAdapter.ListInvoiceLines(ctx, billing.ListInvoiceLinesAdapterInput{
		Namespace:  namespace,
		CustomerID: invoice.Customer.CustomerID,
	})
	s.Require().NoError(err)
	s.Require().Len(listedLines, 1)
	s.Equal(group.ID, *listedLines[0].SplitLineGroupID)
	s.Zero(historyQueries.count)

	s.MockStreamingConnector.AddSimpleEvent(testFeature.Feature.Key, 40, fullPeriod.From.Add(time.Hour))
	s.MockStreamingConnector.AddSimpleEvent(testFeature.Feature.Key, 20, boundary.Add(time.Hour))
	current := newLine(newLineInput{
		Namespace: namespace,
		Invoice:   invoice,
		Name:      "pending edited name",
		Period:    timeutil.ClosedPeriod{From: boundary, To: fullPeriod.To},
	})
	current.ID = "unpersisted-current-line"
	current.SplitLineGroupID = &group.ID
	current.UsageBased.Price = price
	current.UsageBased.FeatureKey = testFeature.Feature.Key
	current.UsageBased.MeteredQuantity = lo.ToPtr(alpacadecimal.NewFromInt(20))
	current.UsageBased.MeteredPreLinePeriodQuantity = lo.ToPtr(alpacadecimal.NewFromInt(40))
	before, err := json.Marshal(current)
	s.Require().NoError(err)

	// When mutable recalculation resolves and rates the persisted billing history.
	ratedSnapshot, err := s.LegacyBillingLineEngine.RecalculateStandardLine(ctx, lineengine.RecalculateStandardLineInput{
		Invoice: invoice,
		Line:    current,
	})
	s.Require().NoError(err)
	s.Equal(1, historyQueries.count)
	expectedTotals := ratedSnapshot.Totals

	// Then prior billing plus the final slice meets the period minimum exactly once.
	s.Equal(40.0, expectedTotals.ChargesTotal.InexactFloat64())
	s.Equal(60.0, expectedTotals.Total.InexactFloat64())
	s.Equal(100.0, prior.Totals.Amount.Add(expectedTotals.Total).InexactFloat64())

	s.Run("simulation resolves history while retaining supplied snapshots", func() {
		// Given a hypothetical price and quantity different from the persisted group and metered usage.
		simulatedLine, err := current.Clone()
		s.Require().NoError(err)
		simulatedLine.Namespace = "caller-namespace"
		simulatedLine.ID = ""
		simulatedLine.UsageBased.Price = productcatalog.NewPriceFrom(productcatalog.UnitPrice{
			Amount:      alpacadecimal.NewFromInt(2),
			Commitments: productcatalog.Commitments{MinimumAmount: lo.ToPtr(alpacadecimal.NewFromInt(100))},
		})
		simulatedLine.UsageBased.MeteredQuantity = lo.ToPtr(alpacadecimal.NewFromInt(15))
		before, err := json.Marshal(simulatedLine)
		s.Require().NoError(err)
		queriesBefore := historyQueries.count

		// When simulation loads history using the group ID after normalizing the supplied line.
		simulatedInvoice, err := s.BillingService.SimulateInvoice(ctx, billing.SimulateInvoiceInput{
			Namespace:  namespace,
			CustomerID: &invoice.Customer.CustomerID,
			Currency:   invoice.Currency,
			Lines:      billing.NewStandardInvoiceLines(billing.StandardLines{simulatedLine}),
		})
		s.Require().NoError(err)

		// Then prior billing of 40 leaves 60 due, using the supplied 30 of usage and 30 adjustment.
		s.Require().Len(simulatedInvoice.Lines.OrEmpty(), 1)
		ratedLine := simulatedInvoice.Lines.OrEmpty()[0]
		s.NotEmpty(ratedLine.ID)
		s.Equal(namespace, ratedLine.Namespace)
		s.Equal(current.Name, ratedLine.Name)
		s.Equal(group.ID, *ratedLine.SplitLineGroupID)
		s.Equal(15.0, ratedLine.UsageBased.MeteredQuantity.InexactFloat64())
		s.Equal(40.0, ratedLine.UsageBased.MeteredPreLinePeriodQuantity.InexactFloat64())
		s.Equal(30.0, ratedLine.Totals.Amount.InexactFloat64())
		s.Equal(30.0, ratedLine.Totals.ChargesTotal.InexactFloat64())
		s.Equal(60.0, simulatedInvoice.Totals.Total.InexactFloat64())
		s.Equal(queriesBefore+1, historyQueries.count)
		s.Empty(simulatedInvoice.ValidationIssues)
		after, err := json.Marshal(simulatedLine)
		s.Require().NoError(err)
		s.JSONEq(string(before), string(after))
		_, err = s.BillingAdapter.GetStandardInvoiceById(ctx, billing.GetStandardInvoiceByIdInput{
			Invoice: simulatedInvoice.GetInvoiceID(),
		})
		s.ErrorIs(err, billing.ErrInvoiceNotFound)
	})

	s.Run("supplied snapshot rating cannot load another namespace", func() {
		// Given a supplied line referencing a group belonging to a different namespace.
		foreignLine, err := current.Clone()
		s.Require().NoError(err)
		foreignLine.Namespace = s.GetUniqueNamespace("foreign-simulation")
		before, err := json.Marshal(foreignLine)
		s.Require().NoError(err)

		// When rating resolves history in the supplied namespace.
		lines, err := s.LegacyBillingLineEngine.RateStandardLines(ctx, lineengine.RateStandardLinesInput{
			Namespace: foreignLine.Namespace,
			Lines:     billing.StandardLines{foreignLine},
		})

		// Then the group remains missing and the failed calculation leaves its input intact.
		s.ErrorContains(err, "split line group["+group.ID+"] for line["+foreignLine.ID+"] not found")
		s.Nil(lines)
		after, err := json.Marshal(foreignLine)
		s.Require().NoError(err)
		s.JSONEq(string(before), string(after))
	})

	s.Run("materialization and preview agree with mutable recalculation", func() {
		// When the same final slice is materialized and previewed.
		gatheringBase, err := current.ToGatheringLineBase()
		s.Require().NoError(err)
		input := billing.BuildStandardInvoiceLinesInput{
			Invoice:        *invoice,
			GatheringLines: billing.GatheringLines{{GatheringLineBase: gatheringBase}},
		}
		built, err := s.LegacyBillingLineEngine.BuildStandardInvoiceLines(ctx, input)
		s.Require().NoError(err)
		preview, err := s.LegacyBillingLineEngine.BuildStandardLinesForGatheringPreview(ctx, input)
		s.Require().NoError(err)

		// Then the entrypoint does not change the rated totals for the same usage and history.
		s.Require().Len(built, 1)
		s.Require().Len(preview, 1)
		s.Equal(expectedTotals, built[0].Totals)
		s.Equal(expectedTotals, preview[0].Totals)
	})

	s.Run("persisted snapshot resolves context without rating history", func() {
		// When an immutable comparison snapshots a pending period edit without expansion.
		edited, err := current.Clone()
		s.Require().NoError(err)
		edited.Period.To = boundary.Add(time.Minute)
		snapshot, err := s.LegacyBillingLineEngine.SnapshotLineQuantity(ctx, lineengine.SnapshotLineQuantityInput{
			Invoice: invoice,
			Line:    edited,
		})
		s.Require().NoError(err)

		// Then the supplied period is authoritative and previously metered usage is retained.
		s.Zero(snapshot.UsageBased.MeteredQuantity.InexactFloat64())
		s.Equal(40.0, snapshot.UsageBased.MeteredPreLinePeriodQuantity.InexactFloat64())
		s.Empty(snapshot.DetailedLines)
	})

	s.Run("collection batches children sharing a group", func() {
		// When two current children share one group's history, both retain their own periods.
		collectionInvoice, err := invoice.Clone()
		s.Require().NoError(err)
		lines, err := s.LegacyBillingLineEngine.OnCollectionCompleted(ctx, billing.OnCollectionCompletedInput{
			Invoice: collectionInvoice,
			Lines:   billing.StandardLines{prior, current},
		})
		s.Require().NoError(err)

		// Then batching preserves the earlier amount and the final slice's independently rated totals.
		s.Require().Len(lines, 2)
		s.Equal(prior.ID, lines[0].ID)
		s.Equal(prior.Totals.Amount.InexactFloat64(), lines[0].Totals.Total.InexactFloat64())
		s.Equal(current.ID, lines[1].ID)
		s.Equal(expectedTotals, lines[1].Totals)
	})

	for _, tc := range []struct {
		name  string
		setup func(*billing.StandardInvoice)
	}{
		{name: "fresh snapshot", setup: func(*billing.StandardInvoice) {}},
		{name: "completed snapshot", setup: func(invoice *billing.StandardInvoice) {
			invoice.QuantitySnapshotedAt = lo.ToPtr(invoice.DefaultCollectionAtForStandardInvoice())
		}},
		{name: "early collection", setup: func(invoice *billing.StandardInvoice) {
			invoice.CollectionAt = lo.ToPtr(fullPeriod.To.Add(time.Hour))
		}},
	} {
		s.Run(tc.name, func() {
			// When collection rates either fresh or reusable quantities without embedded history.
			collectionInvoice, err := invoice.Clone()
			s.Require().NoError(err)
			collectionInvoice.Lines = billing.NewStandardInvoiceLines(billing.StandardLines{current})
			tc.setup(&collectionInvoice)
			lines, err := s.LegacyBillingLineEngine.OnCollectionCompleted(ctx, billing.OnCollectionCompletedInput{
				Invoice: collectionInvoice,
				Lines:   collectionInvoice.Lines.OrEmpty(),
			})
			s.Require().NoError(err)

			// Then snapshot reuse does not change rated totals or replace the pending line.
			s.Require().Len(lines, 1)
			s.Equal(current.ID, lines[0].ID)
			s.Equal(current.Name, lines[0].Name)
			s.Equal(expectedTotals, lines[0].Totals)
		})
	}

	s.Run("mutable recalculation preserves pending edits", func() {
		// When an edited line is recalculated, its identity need not appear in persisted history.
		rated, err := s.LegacyBillingLineEngine.RecalculateStandardLine(ctx, lineengine.RecalculateStandardLineInput{
			Invoice: invoice,
			Line:    current,
		})
		s.Require().NoError(err)

		// Then the current fields survive and recalculation agrees with the other rating entrypoints.
		s.Equal(current.ID, rated.ID)
		s.Equal(current.Name, rated.Name)
		s.Equal(expectedTotals, rated.Totals)
	})

	s.Run("missing persisted context fails before snapshotting", func() {
		// When a line references a group outside its namespace, it cannot obtain that group's period.
		foreignInvoice, err := invoice.Clone()
		s.Require().NoError(err)
		foreignInvoice.Namespace = s.GetUniqueNamespace("foreign-snapshot")
		foreignLine, err := current.Clone()
		s.Require().NoError(err)
		foreignLine.Namespace = foreignInvoice.Namespace
		result, err := s.LegacyBillingLineEngine.SnapshotLineQuantity(ctx, lineengine.SnapshotLineQuantityInput{
			Invoice: &foreignInvoice,
			Line:    foreignLine,
		})

		// Then missing context remains a resolver error instead of silently using ordinary usage.
		s.ErrorContains(err, "split line group["+group.ID+"] for line["+foreignLine.ID+"] not found")
		s.Nil(result)
		s.Equal(20.0, foreignLine.UsageBased.MeteredQuantity.InexactFloat64())
	})

	// Calculation leaves caller state and persisted history untouched, including previews.
	after, err := json.Marshal(current)
	s.Require().NoError(err)
	s.JSONEq(string(before), string(after))
	histories, err := s.BillingAdapter.GetSplitLineGroups(ctx, billing.GetSplitLineGroupsInput{
		Namespace:         namespace,
		SplitLineGroupIDs: []string{group.ID, group.ID},
	})
	s.Require().NoError(err)
	s.Require().Len(histories, 1)
	s.Len(histories[0].Lines, 1)
	foreignHistories, err := s.BillingAdapter.GetSplitLineGroups(ctx, billing.GetSplitLineGroupsInput{
		Namespace:         s.GetUniqueNamespace("foreign-context"),
		SplitLineGroupIDs: []string{group.ID},
	})
	s.Require().NoError(err)
	s.Empty(foreignHistories)
}
