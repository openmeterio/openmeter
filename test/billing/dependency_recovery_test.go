package billing

import (
	"fmt"
	"testing"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/oklog/ulid/v2"
	"github.com/samber/lo"
	"github.com/stretchr/testify/suite"

	appsandbox "github.com/openmeterio/openmeter/openmeter/app/sandbox"
	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/openmeter/meter"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/feature"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/datetime"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/timeutil"
)

type LegacyDependencyRecoverySuite struct {
	BaseSuite
}

func TestLegacyDependencyRecovery(t *testing.T) {
	suite.Run(t, new(LegacyDependencyRecoverySuite))
}

type legacyDependencyFixture struct {
	customer customer.CustomerID
	feature  feature.Feature
	meter    meter.Meter
	period   timeutil.ClosedPeriod
}

func (s *LegacyDependencyRecoverySuite) newDependencyFixture(opts ...BillingProfileProvisionOption) legacyDependencyFixture {
	s.T().Helper()
	ctx := s.T().Context()
	ns := s.GetUniqueNamespace("legacy-dependency-recovery")
	app := s.InstallSandboxApp(s.T(), ns)
	opts = append([]BillingProfileProvisionOption{
		WithManualApproval(),
		WithCollectionInterval(datetime.MustParseDuration(s.T(), "PT1H")),
	}, opts...)
	s.ProvisionBillingProfile(ctx, ns, app.GetID(), opts...)
	cust := s.CreateTestCustomer(ns, "legacy-subject")
	s.Require().NotNil(cust)
	testFeature := s.SetupApiRequestsTotalFeature(ctx, ns)
	s.T().Cleanup(testFeature.Cleanup)
	meters, err := s.MeterAdapter.ListMeters(ctx, meter.ListMetersParams{Namespace: ns})
	s.Require().NoError(err)
	s.Require().Len(meters.Items, 1)

	return legacyDependencyFixture{
		customer: cust.GetID(),
		feature:  testFeature.Feature,
		meter:    meters.Items[0],
		period: timeutil.ClosedPeriod{
			From: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
			To:   time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC),
		},
	}
}

func (f legacyDependencyFixture) usageLine(name string) billing.GatheringLine {
	return billing.GatheringLine{GatheringLineBase: billing.GatheringLineBase{
		ManagedResource: models.NewManagedResource(models.ManagedResourceInput{Name: name}),
		Engine:          billing.LineEngineTypeInvoice,
		ManagedBy:       billing.ManuallyManagedLine,
		ServicePeriod:   f.period,
		InvoiceAt:       f.period.To,
		FeatureKey:      f.feature.Key,
		Price: *productcatalog.NewPriceFrom(productcatalog.UnitPrice{
			Amount: alpacadecimal.NewFromInt(2),
		}),
	}}
}

func (s *LegacyDependencyRecoverySuite) createDependencyLines(f legacyDependencyFixture, lines ...billing.GatheringLine) billing.CreatePendingInvoiceLinesResult {
	s.T().Helper()
	created, err := s.BillingService.CreatePendingInvoiceLines(s.T().Context(), billing.CreatePendingInvoiceLinesInput{
		Customer: f.customer,
		Currency: currencyx.FiatCode("USD"),
		Lines:    lines,
	})
	s.Require().NoError(err)
	s.Require().Len(created.Lines, len(lines))
	for _, line := range created.Lines {
		s.Equal(billing.LineEngineTypeInvoice, line.Engine)
		s.Nil(line.ChargeID)
	}

	return *created
}

func (s *LegacyDependencyRecoverySuite) collectDependencyInvoice(f legacyDependencyFixture, asOf time.Time) billing.StandardInvoice {
	s.T().Helper()
	var invoices []billing.StandardInvoice
	var err error
	s.Require().NotPanics(func() {
		invoices, err = s.BillingService.InvoicePendingLines(s.T().Context(), billing.InvoicePendingLinesInput{
			Customer: f.customer,
			AsOf:     &asOf,
		})
	}, "dependency failures must materialize an invoice instead of panicking")
	s.Require().NoError(err)
	s.Require().Len(invoices, 1)

	return invoices[0]
}

func (s *LegacyDependencyRecoverySuite) requireDependencyIssues(invoice billing.StandardInvoice, codesByLine map[string]string) {
	s.T().Helper()
	persisted, err := s.BillingService.GetStandardInvoiceById(s.T().Context(), billing.GetStandardInvoiceByIdInput{
		Invoice: invoice.GetInvoiceID(),
		Expand:  billing.StandardInvoiceExpandAll,
	})
	s.Require().NoError(err)
	for _, actual := range []billing.StandardInvoice{invoice, persisted} {
		s.Equal(billing.StandardInvoiceStatusDraftInvalidCreated, actual.Status)
		s.NotNil(actual.StatusDetails.AvailableActions.Retry)
		s.NotNil(actual.StatusDetails.AvailableActions.Delete)
		s.Nil(actual.IssuedAt)
		s.Require().Len(actual.ValidationIssues, len(codesByLine))
		actualCodes := make(map[string]string, len(actual.ValidationIssues))
		for _, issue := range actual.ValidationIssues {
			s.Equal(billing.ValidationIssueSeverityCritical, issue.Severity)
			s.Equal(billing.LineEngineValidationComponent(billing.LineEngineTypeInvoice), issue.Component)
			s.NotEmpty(issue.Message)
			actualCodes[issue.Path] = issue.Code
		}
		for lineID, code := range codesByLine {
			s.Equal(code, actualCodes[fmt.Sprintf("/lines/%s", lineID)])
		}
	}
}

func (s *LegacyDependencyRecoverySuite) restoreDependencyFeature(f legacyDependencyFixture) {
	s.T().Helper()
	// Legacy lines resolve by key; archival cannot simulate loss because historical resolution includes archived features.
	_, err := s.FeatureService.CreateFeature(s.T().Context(), feature.CreateFeatureInputs{
		Namespace: f.customer.Namespace,
		Name:      f.feature.Name,
		Key:       f.feature.Key,
		MeterID:   f.feature.MeterID,
	})
	s.Require().NoError(err)
}

func (s *LegacyDependencyRecoverySuite) requireInvoiceReadyForApproval(invoice billing.StandardInvoice, expectedTotal float64) {
	s.T().Helper()
	s.Require().Equal(billing.StandardInvoiceStatusDraftManualApprovalNeeded, invoice.Status)
	s.Empty(invoice.ValidationIssues)
	s.Equal(expectedTotal, invoice.Totals.Total.InexactFloat64())
}

func (s *LegacyDependencyRecoverySuite) expectInvoiceIssuance(expectedTotal float64) *appsandbox.MockApp {
	s.T().Helper()
	mockApp := s.SandboxApp.EnableMock(s.T())
	s.T().Cleanup(s.SandboxApp.DisableMock)
	mockApp.OnFinalizeStandardInvoice(nil)
	mockApp.OnUpsertStandardInvoice(func(actual billing.StandardInvoice) (*billing.UpsertStandardInvoiceResult, error) {
		s.Equal(expectedTotal, actual.Totals.Total.InexactFloat64())
		s.Empty(actual.ValidationIssues)
		return billing.NewUpsertStandardInvoiceResult(), nil
	})
	return mockApp
}

func (s *LegacyDependencyRecoverySuite) requirePersistedIssuedInvoice(issued, draft billing.StandardInvoice, expectedTotal float64) {
	s.T().Helper()
	persisted, err := s.BillingService.GetStandardInvoiceById(s.T().Context(), billing.GetStandardInvoiceByIdInput{
		Invoice: draft.GetInvoiceID(),
		Expand:  billing.StandardInvoiceExpandAll,
	})
	s.Require().NoError(err)
	lineIDs := lo.Map(draft.Lines.OrEmpty(), func(line *billing.StandardLine, _ int) string { return line.ID })
	for _, actual := range []billing.StandardInvoice{issued, persisted} {
		s.NotNil(actual.IssuedAt)
		s.Equal(draft.GetInvoiceID(), actual.GetInvoiceID())
		s.ElementsMatch(lineIDs, lo.Map(actual.Lines.OrEmpty(), func(line *billing.StandardLine, _ int) string { return line.ID }))
		s.Equal(expectedTotal, actual.Totals.Total.InexactFloat64())
	}
}

func (s *LegacyDependencyRecoverySuite) TestMissingFeatureRecoveryThroughIssuance() {
	// given: a persisted legacy usage line loses its referenced feature after usage was recorded.
	f := s.newDependencyFixture()
	clock.FreezeTime(f.period.From)
	defer clock.UnFreeze()
	created := s.createDependencyLines(f, f.usageLine("usage"))
	s.MockStreamingConnector.AddSimpleEvent(f.meter.Key, 7, f.period.From.Add(time.Hour))
	s.Require().NoError(s.DBClient.Feature.DeleteOneID(f.feature.ID).Exec(s.T().Context()))

	// when: the customer collects the due usage, the invoice must retain the missing-feature issue.
	clock.FreezeTime(f.period.To.Add(2 * time.Hour))
	defer clock.UnFreeze()
	invoice := s.collectDependencyInvoice(f, f.period.To)
	issues := map[string]string{created.Lines[0].ID: billing.ErrInvoiceLineFeatureNotFound.Code}
	s.requireDependencyIssues(invoice, issues)
	s.Nil(invoice.Lines.OrEmpty()[0].UsageBased.MeteredQuantity)

	// when: the customer retries without repairing the feature, the same issue must remain.
	invoice, err := s.BillingService.RetryInvoice(s.T().Context(), invoice.GetInvoiceID())
	s.Require().NoError(err)
	s.requireDependencyIssues(invoice, issues)

	// when: the customer restores the feature and retries the invoice.
	s.restoreDependencyFeature(f)
	invoice, err = s.BillingService.RetryInvoice(s.T().Context(), invoice.GetInvoiceID())
	s.Require().NoError(err)

	// then: the same invoice and line are repaired and issued with actual metered usage.
	s.Equal(created.Lines[0].ID, invoice.Lines.OrEmpty()[0].ID)
	s.Require().NotNil(invoice.Lines.OrEmpty()[0].UsageBased.MeteredQuantity)
	s.Equal(float64(7), invoice.Lines.OrEmpty()[0].UsageBased.MeteredQuantity.InexactFloat64())
	s.requireInvoiceReadyForApproval(invoice, 14)
	invoiceApp := s.expectInvoiceIssuance(14)
	issuedInvoice, err := s.BillingService.ApproveInvoice(s.T().Context(), invoice.GetInvoiceID())
	s.Require().NoError(err)
	s.requirePersistedIssuedInvoice(issuedInvoice, invoice, 14)
	s.requireInvoiceFinalizedOnce(invoiceApp)
	s.SandboxApp.DisableMock()
}

func (s *LegacyDependencyRecoverySuite) TestMultipleDependenciesPartialRepair() {
	for _, repairFirst := range []string{"feature", "meter"} {
		s.Run(repairFirst+" first", func() {
			// given: one missing feature, one missing meter, and a healthy flat fee share an invoice.
			f := s.newDependencyFixture()
			clock.FreezeTime(f.period.From)
			defer clock.UnFreeze()
			secondMeter := meter.Meter{
				ManagedResource: models.NewManagedResource(models.ManagedResourceInput{
					Namespace: f.customer.Namespace, ID: ulid.Make().String(), Name: "Second usage meter",
					CreatedAt: f.period.From, UpdatedAt: f.period.From,
				}),
				Key: "second-meter", Aggregation: meter.MeterAggregationSum,
				EventType: "test", ValueProperty: lo.ToPtr("$.value"),
			}
			s.Require().NoError(s.MeterAdapter.ReplaceMeters(s.T().Context(), []meter.Meter{f.meter, secondMeter}))
			second, err := s.FeatureService.CreateFeature(s.T().Context(), feature.CreateFeatureInputs{
				Namespace: f.customer.Namespace, Name: "second", Key: "second", MeterID: &secondMeter.ID,
			})
			s.Require().NoError(err)
			missingMeter := f.usageLine("missing meter")
			missingMeter.FeatureKey = second.Key
			healthy := f.usageLine("healthy flat fee")
			healthy.FeatureKey = ""
			healthy.Price = *productcatalog.NewPriceFrom(productcatalog.FlatPrice{Amount: alpacadecimal.NewFromInt(10), PaymentTerm: productcatalog.InArrearsPaymentTerm})
			created := s.createDependencyLines(f, f.usageLine("missing feature"), missingMeter, healthy)
			s.MockStreamingConnector.AddSimpleEvent(f.meter.Key, 7, f.period.From.Add(time.Hour))
			s.MockStreamingConnector.AddSimpleEvent(secondMeter.Key, 7, f.period.From.Add(time.Hour))
			s.Require().NoError(s.DBClient.Feature.DeleteOneID(f.feature.ID).Exec(s.T().Context()))
			s.Require().NoError(s.MeterAdapter.ReplaceMeters(s.T().Context(), []meter.Meter{f.meter}))
			clock.FreezeTime(f.period.To.Add(2 * time.Hour))
			defer clock.UnFreeze()

			// when: the customer collects the three lines, each broken line must expose its own dependency issue.
			invoice := s.collectDependencyInvoice(f, f.period.To)
			invoiceID := invoice.GetInvoiceID()
			s.requireDependencyIssues(invoice, map[string]string{
				created.Lines[0].ID: billing.ErrInvoiceLineFeatureNotFound.Code,
				created.Lines[1].ID: billing.ErrInvoiceLineFeatureHasNoMeters.Code,
			})

			// when: the customer repairs just one dependency and retries, only the unresolved issue must remain.
			remaining := map[string]string{created.Lines[0].ID: billing.ErrInvoiceLineFeatureNotFound.Code}
			if repairFirst == "meter" {
				s.Require().NoError(s.MeterAdapter.ReplaceMeters(s.T().Context(), []meter.Meter{f.meter, secondMeter}))
			} else {
				s.restoreDependencyFeature(f)
				remaining = map[string]string{
					created.Lines[1].ID: billing.ErrInvoiceLineFeatureHasNoMeters.Code,
				}
			}
			s.Require().NotPanics(func() {
				invoice, err = s.BillingService.RetryInvoice(s.T().Context(), invoiceID)
			}, "partial repair must retain an invalid invoice instead of panicking")
			s.Require().NoError(err)
			s.requireDependencyIssues(invoice, remaining)

			// when: the customer repairs the remaining dependency and retries the same invoice.
			if repairFirst == "meter" {
				s.restoreDependencyFeature(f)
			} else {
				s.Require().NoError(s.MeterAdapter.ReplaceMeters(s.T().Context(), []meter.Meter{f.meter, secondMeter}))
			}
			invoice, err = s.BillingService.RetryInvoice(s.T().Context(), invoiceID)
			s.Require().NoError(err)

			// then: the original three lines can be issued for two $14 usage charges and the $10 flat fee.
			s.Equal(invoiceID, invoice.GetInvoiceID())
			s.ElementsMatch(lo.Map(created.Lines, func(line billing.GatheringLine, _ int) string { return line.ID }),
				lo.Map(invoice.Lines.OrEmpty(), func(line *billing.StandardLine, _ int) string { return line.ID }))
			s.requireInvoiceReadyForApproval(invoice, 38)
			invoiceApp := s.expectInvoiceIssuance(38)
			issuedInvoice, err := s.BillingService.ApproveInvoice(s.T().Context(), invoice.GetInvoiceID())
			s.Require().NoError(err)
			s.requirePersistedIssuedInvoice(issuedInvoice, invoice, 38)
			s.requireInvoiceFinalizedOnce(invoiceApp)
			s.SandboxApp.DisableMock()
		})
	}
}

func (s *LegacyDependencyRecoverySuite) TestSharedMissingDependencyKeepsLineIssues() {
	for _, dependency := range []string{"feature", "meter"} {
		s.Run(dependency, func() {
			// given: two different lines reference the same broken dependency.
			f := s.newDependencyFixture()
			clock.FreezeTime(f.period.From)
			defer clock.UnFreeze()
			created := s.createDependencyLines(f, f.usageLine("first"), f.usageLine("second"))
			code := billing.ErrInvoiceLineFeatureHasNoMeters.Code
			if dependency == "feature" {
				s.Require().NoError(s.DBClient.Feature.DeleteOneID(f.feature.ID).Exec(s.T().Context()))
				code = billing.ErrInvoiceLineFeatureNotFound.Code
			} else {
				s.Require().NoError(s.MeterAdapter.ReplaceMeters(s.T().Context(), nil))
			}
			clock.FreezeTime(f.period.To.Add(2 * time.Hour))
			defer clock.UnFreeze()

			// when: collection and multiple unrepaired retries run.
			invoice := s.collectDependencyInvoice(f, f.period.To)
			for range 3 {
				// then: both line identities retain one issue each without accumulation.
				s.requireDependencyIssues(invoice, map[string]string{created.Lines[0].ID: code, created.Lines[1].ID: code})
				var err error
				invoice, err = s.BillingService.RetryInvoice(s.T().Context(), invoice.GetInvoiceID())
				s.Require().NoError(err)
			}
			s.requireDependencyIssues(invoice, map[string]string{created.Lines[0].ID: code, created.Lines[1].ID: code})
		})
	}
}

func (s *LegacyDependencyRecoverySuite) TestFlatPriceDependencyRecovery() {
	for _, dependency := range []string{"missing feature", "meterless feature", "no feature"} {
		s.Run(dependency, func() {
			// given: flat prices may omit features or use meterless features, but supplied references must resolve.
			f := s.newDependencyFixture()
			clock.FreezeTime(f.period.From)
			defer clock.UnFreeze()
			line := f.usageLine("flat fee")
			line.Price = *productcatalog.NewPriceFrom(productcatalog.FlatPrice{Amount: alpacadecimal.NewFromInt(10), PaymentTerm: productcatalog.InArrearsPaymentTerm})
			switch dependency {
			case "no feature":
				line.FeatureKey = ""
			case "meterless feature":
				meterless, err := s.FeatureService.CreateFeature(s.T().Context(), feature.CreateFeatureInputs{
					Namespace: f.customer.Namespace, Name: "meterless", Key: "meterless",
				})
				s.Require().NoError(err)
				line.FeatureKey = meterless.Key
			}
			created := s.createDependencyLines(f, line)
			if dependency == "missing feature" {
				s.Require().NoError(s.DBClient.Feature.DeleteOneID(f.feature.ID).Exec(s.T().Context()))
			}
			clock.FreezeTime(f.period.To.Add(2 * time.Hour))
			defer clock.UnFreeze()

			// when: collection runs, a missing reference must block approval until repaired.
			invoice := s.collectDependencyInvoice(f, f.period.To)
			if dependency == "missing feature" {
				s.requireDependencyIssues(invoice, map[string]string{created.Lines[0].ID: billing.ErrInvoiceLineFeatureNotFound.Code})
				s.restoreDependencyFeature(f)
				var err error
				invoice, err = s.BillingService.RetryInvoice(s.T().Context(), invoice.GetInvoiceID())
				s.Require().NoError(err)
			}

			// then: valid references and repaired references permit the original fixed amount to be issued.
			s.requireInvoiceReadyForApproval(invoice, 10)
			invoiceApp := s.expectInvoiceIssuance(10)
			issuedInvoice, err := s.BillingService.ApproveInvoice(s.T().Context(), invoice.GetInvoiceID())
			s.Require().NoError(err)
			s.requirePersistedIssuedInvoice(issuedInvoice, invoice, 10)
			s.requireInvoiceFinalizedOnce(invoiceApp)
			s.SandboxApp.DisableMock()
		})
	}
}

func (s *LegacyDependencyRecoverySuite) TestDeleteWithBrokenDependency() {
	for _, dependency := range []string{"feature", "meter", "flat feature"} {
		s.Run(dependency, func() {
			// given: an invalid invoice retains a dependency that is still unavailable.
			f := s.newDependencyFixture()
			clock.FreezeTime(f.period.From)
			defer clock.UnFreeze()
			line := f.usageLine("broken")
			if dependency == "flat feature" {
				line.Price = *productcatalog.NewPriceFrom(productcatalog.FlatPrice{Amount: alpacadecimal.NewFromInt(10), PaymentTerm: productcatalog.InArrearsPaymentTerm})
			}
			created := s.createDependencyLines(f, line)
			code := billing.ErrInvoiceLineFeatureNotFound.Code
			if dependency == "meter" {
				s.Require().NoError(s.MeterAdapter.ReplaceMeters(s.T().Context(), nil))
				code = billing.ErrInvoiceLineFeatureHasNoMeters.Code
			} else {
				s.Require().NoError(s.DBClient.Feature.DeleteOneID(f.feature.ID).Exec(s.T().Context()))
			}
			clock.FreezeTime(f.period.To.Add(2 * time.Hour))
			defer clock.UnFreeze()
			invoice := s.collectDependencyInvoice(f, f.period.To)
			s.requireDependencyIssues(invoice, map[string]string{created.Lines[0].ID: code})

			// when: the customer deletes without repairing the dependency.
			deleted, err := s.BillingService.DeleteInvoice(s.T().Context(), billing.DeleteInvoiceInput{
				Invoice: invoice.GetInvoiceID(), DeletionSource: billing.ChangeSourceAPIRequest,
			})
			s.Require().NoError(err)

			// then: deletion persists, issuance is unavailable, and collection cannot recreate consumed gathering lines.
			persisted, err := s.BillingService.GetStandardInvoiceById(s.T().Context(), billing.GetStandardInvoiceByIdInput{
				Invoice: invoice.GetInvoiceID(), Expand: billing.StandardInvoiceExpandAll,
			})
			s.Require().NoError(err)
			for _, actual := range []billing.StandardInvoice{deleted, persisted} {
				s.Equal(billing.StandardInvoiceStatusDeleted, actual.Status)
				s.NotNil(actual.DeletedAt)
				s.Nil(actual.IssuedAt)
				s.Nil(actual.StatusDetails.AvailableActions.Advance)
				s.Nil(actual.StatusDetails.AvailableActions.Approve)
			}
			invoices, err := s.BillingService.InvoicePendingLines(s.T().Context(), billing.InvoicePendingLinesInput{Customer: f.customer})
			s.ErrorIs(err, billing.ErrInvoiceCreateNoLines)
			s.Empty(invoices)
		})
	}
}

func (s *LegacyDependencyRecoverySuite) TestDeleteRetryWithBrokenMeter() {
	// given: an invalid invoice has an unavailable meter and an invoicing app whose deletion fails once.
	f := s.newDependencyFixture()
	clock.FreezeTime(f.period.From)
	defer clock.UnFreeze()
	created := s.createDependencyLines(f, f.usageLine("usage"))
	s.Require().NoError(s.MeterAdapter.ReplaceMeters(s.T().Context(), nil))
	clock.FreezeTime(f.period.To.Add(2 * time.Hour))
	defer clock.UnFreeze()
	invoice := s.collectDependencyInvoice(f, f.period.To)
	s.requireDependencyIssues(invoice, map[string]string{created.Lines[0].ID: billing.ErrInvoiceLineFeatureHasNoMeters.Code})
	mockApp := s.SandboxApp.EnableMock(s.T())
	defer s.SandboxApp.DisableMock()
	mockApp.OnDeleteStandardInvoice(billing.NewValidationError("delete_failed", "provider unavailable"))

	// when: the provider rejects deletion, the invoice must expose the provider failure.
	input := billing.DeleteInvoiceInput{Invoice: invoice.GetInvoiceID(), DeletionSource: billing.ChangeSourceAPIRequest}
	invoice, err := s.BillingService.DeleteInvoice(s.T().Context(), input)
	s.Require().NoError(err)
	s.Equal(billing.StandardInvoiceStatusDeleteFailed, invoice.Status)
	s.Require().NotNil(invoice.DeletedAt)
	deletedAt := *invoice.DeletedAt
	s.Require().Len(invoice.ValidationIssues, 1)
	s.Equal("delete_failed", invoice.ValidationIssues[0].Code)

	// when: the provider recovers, the customer repeats deletion without restoring the meter.
	mockApp.OnDeleteStandardInvoice(nil)
	invoice, err = s.BillingService.DeleteInvoice(s.T().Context(), input)
	s.Require().NoError(err)

	// then: the invoice is deleted without meter repair or replaying its local deletion timestamp.
	s.Equal(billing.StandardInvoiceStatusDeleted, invoice.Status)
	s.Equal(deletedAt, *invoice.DeletedAt)
	s.Empty(invoice.ValidationIssues)
	s.Equal(2, mockApp.DeleteInvoiceCallCount())
	s.Zero(mockApp.FinalizeInvoiceCallCount())
	mockApp.AssertExpectations(s.T())
}

func (s *LegacyDependencyRecoverySuite) TestDependencyLostBeforeCollectionCompletes() {
	for _, dependency := range []string{"feature", "meter"} {
		s.Run(dependency, func() {
			// given: a healthy draft has preliminary usage but is waiting for its final collection snapshot.
			f := s.newDependencyFixture()
			clock.FreezeTime(f.period.From)
			defer clock.UnFreeze()
			created := s.createDependencyLines(f, f.usageLine("usage"))
			s.MockStreamingConnector.AddSimpleEvent(f.meter.Key, 7, f.period.From.Add(time.Hour))
			clock.FreezeTime(f.period.To)
			defer clock.UnFreeze()
			invoice := s.collectDependencyInvoice(f, f.period.To)
			s.Require().Equal(billing.StandardInvoiceStatusDraftWaitingForCollection, invoice.Status)
			s.Require().Len(invoice.Lines.OrEmpty(), 1)
			s.Require().NotNil(invoice.Lines.OrEmpty()[0].UsageBased.MeteredQuantity)
			s.Equal(float64(7), invoice.Lines.OrEmpty()[0].UsageBased.MeteredQuantity.InexactFloat64())
			code := billing.ErrInvoiceLineFeatureNotFound.Code
			if dependency == "meter" {
				s.Require().NoError(s.MeterAdapter.ReplaceMeters(s.T().Context(), nil))
				code = billing.ErrInvoiceLineFeatureHasNoMeters.Code
			} else {
				s.Require().NoError(s.DBClient.Feature.DeleteOneID(f.feature.ID).Exec(s.T().Context()))
			}

			// when: collection completes with a broken dependency, the preliminary snapshot must not become final.
			clock.FreezeTime(invoice.DefaultCollectionAtForStandardInvoice())
			defer clock.UnFreeze()
			invoice, err := s.BillingService.AdvanceInvoice(s.T().Context(), invoice.GetInvoiceID())
			s.Require().NoError(err)
			s.requireDependencyIssues(invoice, map[string]string{created.Lines[0].ID: code})
			s.Nil(invoice.QuantitySnapshotedAt)

			// when: the customer repairs the dependency and retries after more usage arrives.
			if dependency == "meter" {
				s.Require().NoError(s.MeterAdapter.ReplaceMeters(s.T().Context(), []meter.Meter{f.meter}))
			} else {
				s.restoreDependencyFeature(f)
			}
			s.MockStreamingConnector.AddSimpleEvent(f.meter.Key, 3, f.period.From.Add(2*time.Hour))
			invoice, err = s.BillingService.RetryInvoice(s.T().Context(), invoice.GetInvoiceID())
			s.Require().NoError(err)

			// then: refreshed usage, rather than the preliminary quantity, determines the issued amount.
			s.Require().NotNil(invoice.Lines.OrEmpty()[0].UsageBased.MeteredQuantity)
			s.Equal(float64(10), invoice.Lines.OrEmpty()[0].UsageBased.MeteredQuantity.InexactFloat64())
			s.requireInvoiceReadyForApproval(invoice, 20)
			invoiceApp := s.expectInvoiceIssuance(20)
			issuedInvoice, err := s.BillingService.ApproveInvoice(s.T().Context(), invoice.GetInvoiceID())
			s.Require().NoError(err)
			s.requirePersistedIssuedInvoice(issuedInvoice, invoice, 20)
			s.requireInvoiceFinalizedOnce(invoiceApp)
			s.SandboxApp.DisableMock()
		})
	}
}

func (s *LegacyDependencyRecoverySuite) TestProgressiveRetryPreservesFinalBilling() {
	// given: a progressively billed line has 10 units before the split and 20 after, both discounted by 10%.
	f := s.newDependencyFixture(WithProgressiveBilling())
	midpoint := f.period.From.Add(15 * 24 * time.Hour)
	s.createProgressiveDependencyLine(f, midpoint)

	// when: the meter disappears between partial invoice creation and its final usage snapshot.
	clock.FreezeTime(midpoint)
	defer clock.UnFreeze()
	partial := s.collectDependencyInvoice(f, midpoint)
	s.Require().Len(partial.Lines.OrEmpty(), 1)
	partialLine := partial.Lines.OrEmpty()[0]
	s.Require().NotNil(partialLine.SplitLineGroupID)
	groupID := *partialLine.SplitLineGroupID
	s.Require().NoError(s.MeterAdapter.ReplaceMeters(s.T().Context(), nil))
	clock.FreezeTime(partial.DefaultCollectionAtForStandardInvoice())
	defer clock.UnFreeze()
	partial, err := s.BillingService.AdvanceInvoice(s.T().Context(), partial.GetInvoiceID())
	s.Require().NoError(err)
	s.requireDependencyIssues(partial, map[string]string{partialLine.ID: billing.ErrInvoiceLineFeatureHasNoMeters.Code})

	// when: the customer repairs the meter and retries the partial invoice.
	s.Require().NoError(s.MeterAdapter.ReplaceMeters(s.T().Context(), []meter.Meter{f.meter}))
	partial, err = s.BillingService.RetryInvoice(s.T().Context(), partial.GetInvoiceID())
	s.Require().NoError(err)

	// then: the partial invoice can be approved for 10 units at $2 less 10%.
	s.requireInvoiceReadyForApproval(partial, 18)
	partialApp := s.expectInvoiceIssuance(18)
	issuedPartial, err := s.BillingService.ApproveInvoice(s.T().Context(), partial.GetInvoiceID())
	s.Require().NoError(err)
	s.requirePersistedIssuedInvoice(issuedPartial, partial, 18)
	s.requireInvoiceFinalizedOnce(partialApp)
	s.SandboxApp.DisableMock()

	// when: the remaining service period becomes due.
	clock.FreezeTime(f.period.To.Add(2 * time.Hour))
	defer clock.UnFreeze()
	final := s.collectDependencyInvoice(f, f.period.To)
	s.Require().Len(final.Lines.OrEmpty(), 1)
	finalLine := final.Lines.OrEmpty()[0]
	s.Equal(groupID, lo.FromPtr(finalLine.SplitLineGroupID))
	s.True(finalLine.Period.From.Equal(midpoint))
	s.True(finalLine.Period.To.Equal(f.period.To))
	s.Require().NotNil(finalLine.UsageBased.MeteredQuantity)
	s.Equal(float64(20), finalLine.UsageBased.MeteredQuantity.InexactFloat64())

	// then: final billing retains the issued partial amount and charges the remaining 20 units once.
	s.Require().NotNil(finalLine.SplitLineHierarchy)
	prior, err := finalLine.SplitLineHierarchy.SumNetAmount(billing.SumNetAmountInput{PeriodEndLTE: midpoint})
	s.Require().NoError(err)
	s.Equal(issuedPartial.Totals.Amount.InexactFloat64(), prior.InexactFloat64())
	s.Equal(float64(54), issuedPartial.Totals.Total.Add(final.Totals.Total).InexactFloat64())
	s.requireInvoiceReadyForApproval(final, 36)
	finalApp := s.expectInvoiceIssuance(36)
	issuedFinal, err := s.BillingService.ApproveInvoice(s.T().Context(), final.GetInvoiceID())
	s.Require().NoError(err)
	s.requirePersistedIssuedInvoice(issuedFinal, final, 36)
	s.requireInvoiceFinalizedOnce(finalApp)
	s.SandboxApp.DisableMock()
}

func (s *LegacyDependencyRecoverySuite) TestProgressiveDeletePreservesFinalBilling() {
	// given: a progressively billed line has 10 units before the split and 20 after, both discounted by 10%.
	f := s.newDependencyFixture(WithProgressiveBilling())
	midpoint := f.period.From.Add(15 * 24 * time.Hour)
	s.createProgressiveDependencyLine(f, midpoint)

	// when: the meter disappears between partial invoice creation and its final usage snapshot.
	clock.FreezeTime(midpoint)
	defer clock.UnFreeze()
	partial := s.collectDependencyInvoice(f, midpoint)
	s.Require().Len(partial.Lines.OrEmpty(), 1)
	partialLine := partial.Lines.OrEmpty()[0]
	s.Require().NotNil(partialLine.SplitLineGroupID)
	groupID := *partialLine.SplitLineGroupID
	s.Require().NoError(s.MeterAdapter.ReplaceMeters(s.T().Context(), nil))
	clock.FreezeTime(partial.DefaultCollectionAtForStandardInvoice())
	defer clock.UnFreeze()
	partial, err := s.BillingService.AdvanceInvoice(s.T().Context(), partial.GetInvoiceID())
	s.Require().NoError(err)
	s.requireDependencyIssues(partial, map[string]string{partialLine.ID: billing.ErrInvoiceLineFeatureHasNoMeters.Code})

	// when: the customer deletes the partial invoice while the meter is still unavailable.
	partial, err = s.BillingService.DeleteInvoice(s.T().Context(), billing.DeleteInvoiceInput{
		Invoice: partial.GetInvoiceID(), DeletionSource: billing.ChangeSourceAPIRequest,
	})
	s.Require().NoError(err)
	s.Equal(billing.StandardInvoiceStatusDeleted, partial.Status)

	// when: the meter is restored and the remaining service period becomes due.
	s.Require().NoError(s.MeterAdapter.ReplaceMeters(s.T().Context(), []meter.Meter{f.meter}))
	clock.FreezeTime(f.period.To.Add(2 * time.Hour))
	defer clock.UnFreeze()
	final := s.collectDependencyInvoice(f, f.period.To)
	s.Require().Len(final.Lines.OrEmpty(), 1)
	finalLine := final.Lines.OrEmpty()[0]
	s.Equal(groupID, lo.FromPtr(finalLine.SplitLineGroupID))
	s.True(finalLine.Period.From.Equal(midpoint))
	s.True(finalLine.Period.To.Equal(f.period.To))
	s.Require().NotNil(finalLine.UsageBased.MeteredQuantity)
	s.Equal(float64(20), finalLine.UsageBased.MeteredQuantity.InexactFloat64())

	// then: final billing excludes the deleted partial amount and charges only the surviving period.
	s.Require().NotNil(finalLine.SplitLineHierarchy)
	prior, err := finalLine.SplitLineHierarchy.SumNetAmount(billing.SumNetAmountInput{PeriodEndLTE: midpoint})
	s.Require().NoError(err)
	s.Zero(prior.InexactFloat64())
	s.requireInvoiceReadyForApproval(final, 36)
	finalApp := s.expectInvoiceIssuance(36)
	issuedFinal, err := s.BillingService.ApproveInvoice(s.T().Context(), final.GetInvoiceID())
	s.Require().NoError(err)
	s.requirePersistedIssuedInvoice(issuedFinal, final, 36)
	s.requireInvoiceFinalizedOnce(finalApp)
	s.SandboxApp.DisableMock()
}

func (s *LegacyDependencyRecoverySuite) createProgressiveDependencyLine(f legacyDependencyFixture, midpoint time.Time) {
	s.T().Helper()
	clock.FreezeTime(f.period.From)
	defer clock.UnFreeze()
	line := f.usageLine("progressive usage")
	line.RateCardDiscounts = billing.Discounts{Percentage: &billing.PercentageDiscount{
		PercentageDiscount: productcatalog.PercentageDiscount{Percentage: models.NewPercentage(10)},
	}}
	s.createDependencyLines(f, line)
	s.MockStreamingConnector.AddSimpleEvent(f.meter.Key, 10, f.period.From.Add(time.Hour))
	s.MockStreamingConnector.AddSimpleEvent(f.meter.Key, 20, midpoint.Add(24*time.Hour))
}

func (s *LegacyDependencyRecoverySuite) requireInvoiceFinalizedOnce(mockApp *appsandbox.MockApp) {
	s.T().Helper()
	s.Equal(1, mockApp.FinalizeInvoiceCallCount())
	s.Zero(mockApp.DeleteInvoiceCallCount())
	mockApp.AssertExpectations(s.T())
}
