package service

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"testing"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/samber/lo"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/meta"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/usagebased"
	featuremeterservice "github.com/openmeterio/openmeter/openmeter/billing/featuremeter/service"
	"github.com/openmeterio/openmeter/openmeter/billing/rating"
	billingratingservice "github.com/openmeterio/openmeter/openmeter/billing/rating/service"
	"github.com/openmeterio/openmeter/openmeter/meter"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/feature"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/framework/transaction"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/pagination"
	"github.com/openmeterio/openmeter/pkg/timeutil"
)

const usageBasedBillabilityInvoiceID = "invoice-id"

func TestAreLinesBillableAsOfResolvesChargeFeatureMeters(t *testing.T) {
	periods := []timeutil.ClosedPeriod{
		{From: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC), To: time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)},
		{From: time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC), To: time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)},
	}
	lines := []billing.GatheringLine{
		newUsageBasedBillabilityLine("namespace", "line-1", "charge-1", periods[0]),
		newUsageBasedBillabilityLine("namespace", "line-2", "charge-2", periods[1]),
	}
	charges := []usagebased.Charge{
		newUsageBasedBillabilityCharge("namespace", "charge-1", "feature-1"),
		newUsageBasedBillabilityCharge("namespace", "charge-2", "feature-2"),
	}
	featureEntities := []feature.Feature{
		{Namespace: "namespace", ID: "feature-1", Key: "charge-feature-1", MeterID: lo.ToPtr("meter-1")},
		{Namespace: "namespace", ID: "feature-2", Key: "charge-feature-2", MeterID: lo.ToPtr("meter-2")},
	}
	meterEntities := []meter.Meter{
		newUsageBasedBillabilityMeter("namespace", "meter-1"),
		newUsageBasedBillabilityMeter("namespace", "meter-2"),
	}

	// Given usage-based gathering lines whose copied feature keys differ from
	// the persisted feature snapshots owned by their charges.
	adapter := &usageBasedBillabilityAdapter{charges: charges}
	ratingService := &usageBasedBillabilityRatingService{}
	featureMeterResolver, err := featuremeterservice.New(featuremeterservice.Config{
		FeatureService: usageBasedBillabilityFeatureService{features: featureEntities},
		MeterService:   usageBasedBillabilityMeterService{meters: meterEntities},
		Logger:         slog.Default(),
	})
	require.NoError(t, err)
	engine := &LineEngine{service: &service{
		adapter:              adapter,
		featureMeterResolver: featureMeterResolver,
		ratingService:        ratingService,
	}}
	ctx, err := transaction.SetDriverOnContext(t.Context(), usageBasedBillabilityTransaction{})
	require.NoError(t, err)

	invoice := billing.GatheringInvoice{GatheringInvoiceBase: billing.GatheringInvoiceBase{
		ManagedResource: models.ManagedResource{
			NamespacedModel: models.NamespacedModel{Namespace: "namespace"},
			ID:              usageBasedBillabilityInvoiceID,
		},
	}}

	// When billability is resolved as an ordered batch.
	results, err := engine.AreLinesBillableAsOf(ctx, billing.AreLinesBillableAsOfInput{
		Invoice:            invoice,
		AsOf:               lines[len(lines)-1].InvoiceAt,
		ProgressiveBilling: true,
		Lines:              lines,
	})

	// Then rating receives the charge-owned feature and meter for each line,
	// while results retain the original input order.
	require.NoError(t, err)
	require.Equal(t, []billing.IsLineBillableAsOfResult{
		{Billable: true, BillablePeriod: periods[0]},
		{Billable: true, BillablePeriod: periods[1]},
	}, results)
	require.Equal(t, []usagebased.GetByIDsInput{{
		Namespace: "namespace",
		IDs:       []string{"charge-1", "charge-2"},
	}}, adapter.inputs)

	ratingInputsByLineID := lo.SliceToMap(ratingService.inputs, func(input rating.ResolveBillablePeriodInput) (string, rating.ResolveBillablePeriodInput) {
		return input.Line.GetID(), input
	})
	require.Equal(t, "feature-1", ratingInputsByLineID["line-1"].Feature.ID)
	require.Equal(t, "meter-1", ratingInputsByLineID["line-1"].Meter.ID)
	require.Equal(t, "feature-2", ratingInputsByLineID["line-2"].Feature.ID)
	require.Equal(t, "meter-2", ratingInputsByLineID["line-2"].Meter.ID)
}

func TestAreLinesBillableAsOfRequiresChargeID(t *testing.T) {
	period := timeutil.ClosedPeriod{
		From: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		To:   time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC),
	}
	line := newUsageBasedBillabilityLine("namespace", "line", "charge", period)
	line.ChargeID = nil

	// Given a usage-based gathering line without its owning charge identity.
	engine := &LineEngine{}

	// When billability is requested.
	_, err := engine.AreLinesBillableAsOf(t.Context(), billing.AreLinesBillableAsOfInput{
		Invoice: billing.GatheringInvoice{GatheringInvoiceBase: billing.GatheringInvoiceBase{
			ManagedResource: models.ManagedResource{
				NamespacedModel: models.NamespacedModel{Namespace: line.Namespace},
				ID:              line.InvoiceID,
			},
		}},
		AsOf:  line.InvoiceAt,
		Lines: billing.GatheringLines{line},
	})

	// Then the engine rejects the line before resolving charge dependencies.
	require.ErrorContains(t, err, "charge id is required")
}

func TestAreLinesBillableAsOfFallsBackWhenChargeFeatureIsMissing(t *testing.T) {
	period := timeutil.ClosedPeriod{
		From: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		To:   time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC),
	}
	line := newUsageBasedBillabilityLine("namespace", "line", "charge", period)
	charge := newUsageBasedBillabilityCharge("namespace", "charge", "missing-feature")
	featureMeterResolver, err := featuremeterservice.New(featuremeterservice.Config{
		FeatureService: usageBasedBillabilityFeatureService{},
		MeterService:   usageBasedBillabilityMeterService{},
		Logger:         slog.Default(),
	})
	require.NoError(t, err)
	engine := &LineEngine{service: &service{
		adapter:              &usageBasedBillabilityAdapter{charges: []usagebased.Charge{charge}},
		featureMeterResolver: featureMeterResolver,
		ratingService:        billingratingservice.New(billingratingservice.Config{}),
	}}
	ctx, err := transaction.SetDriverOnContext(t.Context(), usageBasedBillabilityTransaction{})
	require.NoError(t, err)
	invoice := billing.GatheringInvoice{GatheringInvoiceBase: billing.GatheringInvoiceBase{
		ManagedResource: models.ManagedResource{
			NamespacedModel: models.NamespacedModel{Namespace: line.Namespace},
			ID:              line.InvoiceID,
		},
	}}

	// Given a usage-based charge whose persisted feature no longer exists.
	// When billability is checked with progressive billing enabled.
	results, err := engine.AreLinesBillableAsOf(ctx, billing.AreLinesBillableAsOfInput{
		Invoice:            invoice,
		AsOf:               period.From.Add(24 * time.Hour),
		ProgressiveBilling: true,
		Lines:              billing.GatheringLines{line},
	})

	// Then the engine returns a usable non-progressive result and the missing-feature validation issue.
	issues, systemErr := billing.ToValidationIssues(err)
	require.NoError(t, systemErr)
	require.Equal(t, billing.ValidationIssues{{
		Severity:   billing.ValidationIssueSeverityCritical,
		Code:       billing.ErrInvoiceLineFeatureNotFound.Code,
		Message:    "invoice line: feature not found",
		Path:       "/charges/charge",
		Attributes: models.Annotations{"feature_id": "missing-feature"},
	}}, issues)
	require.Equal(t, []billing.IsLineBillableAsOfResult{{}}, results)
}

func TestGateInvoiceAssignmentReconcilesFeatureMeterReadiness(t *testing.T) {
	period := timeutil.ClosedPeriod{
		From: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		To:   time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC),
	}
	blockedLines := billing.GatheringLines{
		newUsageBasedBillabilityLine("namespace", "blocked-line-1", "blocked-charge", period),
		newUsageBasedBillabilityLine("namespace", "blocked-line-2", "blocked-charge", period),
	}
	readyLine := newUsageBasedBillabilityLine("namespace", "ready-line", "ready-charge", period)
	unrelatedIssue := billing.ValidationIssue{Code: "unrelated"}
	staleFeatureIssue := billing.ValidationIssue{
		Code:      billing.ErrInvoiceLineFeatureNotFound.Code,
		Component: billing.ValidationComponentProductCatalog,
	}
	staleLineEngineIssue := billing.ValidationIssue{
		Code:      usagebased.ValidationIssueCodeInvoiceAssignmentBlockedActiveRun,
		Component: usagebased.ValidationIssueComponentLineEngine,
	}
	currentRunIssue := billing.ValidationIssue{
		Severity:  billing.ValidationIssueSeverityCritical,
		Code:      usagebased.ValidationIssueCodeInvoiceAssignmentBlockedActiveRun,
		Message:   activeRunInvoiceAssignmentIssueMessage,
		Component: usagebased.ValidationIssueComponentLineEngine,
		Attributes: models.Annotations{
			"invoice_id": "current-invoice",
			"line_id":    "current-line",
		},
	}
	blockedCharge := newUsageBasedBillabilityCharge("namespace", "blocked-charge", "blocked-feature")
	blockedCharge.ValidationIssues = billing.ValidationIssues{unrelatedIssue}
	blockedCharge.State.CurrentRealizationRunID = lo.ToPtr("current-run")
	blockedCharge.Realizations = usagebased.RealizationRuns{{
		RealizationRunBase: usagebased.RealizationRunBase{
			ID: usagebased.RealizationRunID{
				Namespace: "namespace",
				ID:        "current-run",
			},
			InvoiceID: lo.ToPtr("current-invoice"),
			LineID:    lo.ToPtr("current-line"),
		},
	}}
	readyCharge := newUsageBasedBillabilityCharge("namespace", "ready-charge", "ready-feature")
	readyCharge.ValidationIssues = billing.ValidationIssues{unrelatedIssue, staleFeatureIssue, staleLineEngineIssue}
	adapter := &usageBasedBillabilityAdapter{charges: []usagebased.Charge{blockedCharge, readyCharge}}

	newEngine := func(meters []meter.Meter) *LineEngine {
		featureMeterResolver, err := featuremeterservice.New(featuremeterservice.Config{
			FeatureService: usageBasedBillabilityFeatureService{features: []feature.Feature{
				{Namespace: "namespace", ID: "blocked-feature", Key: "blocked-feature-key", MeterID: lo.ToPtr("blocked-meter")},
				{Namespace: "namespace", ID: "ready-feature", Key: "ready-feature-key", MeterID: lo.ToPtr("ready-meter")},
			}},
			MeterService: usageBasedBillabilityMeterService{meters: meters},
			Logger:       slog.Default(),
		})
		require.NoError(t, err)

		return &LineEngine{service: &service{
			adapter:              adapter,
			featureMeterResolver: featureMeterResolver,
		}}
	}
	input := billing.GateInvoiceAssignmentInput{
		Lines: append(blockedLines, readyLine),
	}
	ctx, err := transaction.SetDriverOnContext(t.Context(), usageBasedBillabilityTransaction{})
	require.NoError(t, err)

	// Given one charge without its required meter and with an active run,
	// and another charge whose dependency recovered and has no current run.
	engine := newEngine([]meter.Meter{newUsageBasedBillabilityMeter("namespace", "ready-meter")})

	// When their gathering lines are considered for invoice assignment.
	result, err := engine.GateInvoiceAssignment(ctx, input)

	// Then only the blocked charge's lines are excluded, its readiness issue is recorded once,
	// and the recovered charge keeps only its unrelated issue.
	require.NoError(t, err)
	require.Equal(t, billing.GateInvoiceAssignmentResult{
		blockedLines[0].GetLineID(): {ExcludeFromInvoice: true},
		blockedLines[1].GetLineID(): {ExcludeFromInvoice: true},
	}, result)
	require.Equal(t, billing.ValidationIssues{
		unrelatedIssue,
		{
			Severity:  billing.ValidationIssueSeverityCritical,
			Code:      billing.ErrInvoiceLineFeatureHasNoMeters.Code,
			Message:   "usage based invoice line: feature has no meters",
			Component: billing.ValidationComponentProductCatalog,
			Path:      "/charges/blocked-charge",
			Attributes: models.Annotations{
				"feature_id":  "blocked-feature",
				"feature_key": "blocked-feature-key",
				"meter_id":    "blocked-meter",
			},
		},
		currentRunIssue,
	}, adapter.charge("blocked-charge").ValidationIssues)
	require.Equal(t, billing.ValidationIssues{unrelatedIssue}, adapter.charge("ready-charge").ValidationIssues)

	// When the same blocked lines are evaluated again.
	_, err = engine.GateInvoiceAssignment(ctx, input)

	// Then both blocking issues are replaced rather than duplicated.
	require.NoError(t, err)
	require.Len(t, adapter.charge("blocked-charge").ValidationIssues, 3)

	// When the missing meter is restored and assignment is retried.
	engine = newEngine([]meter.Meter{
		newUsageBasedBillabilityMeter("namespace", "blocked-meter"),
		newUsageBasedBillabilityMeter("namespace", "ready-meter"),
	})
	result, err = engine.GateInvoiceAssignment(ctx, input)

	// Then the recovered charge remains blocked only by its active run.
	require.NoError(t, err)
	require.Equal(t, billing.GateInvoiceAssignmentResult{
		blockedLines[0].GetLineID(): {ExcludeFromInvoice: true},
		blockedLines[1].GetLineID(): {ExcludeFromInvoice: true},
	}, result)
	require.Equal(t, billing.ValidationIssues{unrelatedIssue, currentRunIssue}, adapter.charge("blocked-charge").ValidationIssues)

	// When the current run is completed and assignment is retried.
	adapter.charges[0].State.CurrentRealizationRunID = nil
	result, err = engine.GateInvoiceAssignment(ctx, input)

	// Then both charges are eligible and only their unrelated issues remain.
	require.NoError(t, err)
	require.Empty(t, result)
	require.Equal(t, billing.ValidationIssues{unrelatedIssue}, adapter.charge("blocked-charge").ValidationIssues)
	require.Equal(t, billing.ValidationIssues{unrelatedIssue}, adapter.charge("ready-charge").ValidationIssues)
}

func TestGateInvoiceAssignmentRecordsMissingFeature(t *testing.T) {
	period := timeutil.ClosedPeriod{
		From: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		To:   time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC),
	}
	line := newUsageBasedBillabilityLine("namespace", "line", "charge", period)
	adapter := &usageBasedBillabilityAdapter{charges: []usagebased.Charge{
		newUsageBasedBillabilityCharge("namespace", "charge", "missing-feature"),
	}}
	featureMeterResolver, err := featuremeterservice.New(featuremeterservice.Config{
		FeatureService: usageBasedBillabilityFeatureService{},
		MeterService:   usageBasedBillabilityMeterService{},
		Logger:         slog.Default(),
	})
	require.NoError(t, err)
	engine := &LineEngine{service: &service{
		adapter:              adapter,
		featureMeterResolver: featureMeterResolver,
	}}
	ctx, err := transaction.SetDriverOnContext(t.Context(), usageBasedBillabilityTransaction{})
	require.NoError(t, err)

	// Given a charge whose pinned feature is unavailable when assignment is evaluated.
	result, err := engine.GateInvoiceAssignment(ctx, billing.GateInvoiceAssignmentInput{
		Lines: billing.GatheringLines{line},
	})

	// Then the line is excluded and the distinct missing-feature issue is persisted on the charge.
	require.NoError(t, err)
	require.Equal(t, billing.GateInvoiceAssignmentResult{
		line.GetLineID(): {ExcludeFromInvoice: true},
	}, result)
	require.Equal(t, billing.ErrInvoiceLineFeatureNotFound.Code, adapter.charge("charge").ValidationIssues[0].Code)
	require.Equal(t, billing.ValidationComponentProductCatalog, adapter.charge("charge").ValidationIssues[0].Component)
	require.Equal(t, models.Annotations{"feature_id": "missing-feature"}, adapter.charge("charge").ValidationIssues[0].Attributes)
}

func TestGateInvoiceAssignmentPropagatesFeatureResolverErrors(t *testing.T) {
	period := timeutil.ClosedPeriod{
		From: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		To:   time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC),
	}
	line := newUsageBasedBillabilityLine("namespace", "line", "charge", period)
	adapter := &usageBasedBillabilityAdapter{charges: []usagebased.Charge{
		newUsageBasedBillabilityCharge("namespace", "charge", "feature"),
	}}
	resolverErr := errors.New("feature service unavailable")
	featureMeterResolver, err := featuremeterservice.New(featuremeterservice.Config{
		FeatureService: usageBasedBillabilityFeatureService{err: resolverErr},
		MeterService:   usageBasedBillabilityMeterService{},
		Logger:         slog.Default(),
	})
	require.NoError(t, err)
	engine := &LineEngine{service: &service{
		adapter:              adapter,
		featureMeterResolver: featureMeterResolver,
	}}
	ctx, err := transaction.SetDriverOnContext(t.Context(), usageBasedBillabilityTransaction{})
	require.NoError(t, err)

	_, err = engine.GateInvoiceAssignment(ctx, billing.GateInvoiceAssignmentInput{
		Lines: billing.GatheringLines{line},
	})

	require.ErrorIs(t, err, resolverErr)
	require.Empty(t, adapter.charge("charge").ValidationIssues)
}

type usageBasedBillabilityAdapter struct {
	usagebased.Adapter

	charges []usagebased.Charge
	inputs  []usagebased.GetByIDsInput
}

func (a *usageBasedBillabilityAdapter) GetByIDs(_ context.Context, input usagebased.GetByIDsInput) ([]usagebased.Charge, error) {
	a.inputs = append(a.inputs, input)

	return lo.Filter(a.charges, func(charge usagebased.Charge, _ int) bool {
		return charge.Namespace == input.Namespace && lo.Contains(input.IDs, charge.ID)
	}), nil
}

func (a *usageBasedBillabilityAdapter) UpdateChargeValidationIssues(_ context.Context, input usagebased.UpdateChargeValidationIssuesInput) error {
	for idx := range a.charges {
		if a.charges[idx].GetChargeID() == input.ChargeID {
			a.charges[idx].ValidationIssues = input.ValidationIssues
			return nil
		}
	}

	return fmt.Errorf("charge[%s] not found", input.ChargeID.ID)
}

func (a *usageBasedBillabilityAdapter) charge(id string) usagebased.Charge {
	charge, _ := lo.Find(a.charges, func(charge usagebased.Charge) bool {
		return charge.ID == id
	})

	return charge
}

type usageBasedBillabilityFeatureService struct {
	features []feature.Feature
	err      error
}

func (s usageBasedBillabilityFeatureService) ListFeatures(_ context.Context, params feature.ListFeaturesParams) (pagination.Result[feature.Feature], error) {
	if s.err != nil {
		return pagination.Result[feature.Feature]{}, s.err
	}

	return pagination.Result[feature.Feature]{Items: lo.Filter(s.features, func(featureEntity feature.Feature, _ int) bool {
		return featureEntity.Namespace == params.Namespace && (lo.Contains(params.IDsOrKeys, featureEntity.ID) || lo.Contains(params.IDsOrKeys, featureEntity.Key))
	})}, nil
}

type usageBasedBillabilityMeterService struct {
	meters []meter.Meter
}

func (s usageBasedBillabilityMeterService) ListMeters(_ context.Context, params meter.ListMetersParams) (pagination.Result[meter.Meter], error) {
	return pagination.Result[meter.Meter]{Items: lo.Filter(s.meters, func(meterEntity meter.Meter, _ int) bool {
		return meterEntity.Namespace == params.Namespace && params.IDFilter != nil && lo.Contains(*params.IDFilter, meterEntity.ID)
	})}, nil
}

type usageBasedBillabilityRatingService struct {
	inputs []rating.ResolveBillablePeriodInput
}

func (s *usageBasedBillabilityRatingService) ResolveBillablePeriod(input rating.ResolveBillablePeriodInput) (billing.IsLineBillableAsOfResult, error) {
	s.inputs = append(s.inputs, input)

	return billing.IsLineBillableAsOfResult{
		Billable:       true,
		BillablePeriod: input.Line.GetServicePeriod(),
	}, nil
}

func (*usageBasedBillabilityRatingService) GenerateDetailedLines(rating.StandardLineAccessor, ...rating.GenerateDetailedLinesOption) (rating.GenerateDetailedLinesResult, error) {
	return rating.GenerateDetailedLinesResult{}, nil
}

type usageBasedBillabilityTransaction struct{}

func (usageBasedBillabilityTransaction) Commit() error    { return nil }
func (usageBasedBillabilityTransaction) Rollback() error  { return nil }
func (usageBasedBillabilityTransaction) SavePoint() error { return nil }

func newUsageBasedBillabilityCharge(namespace, chargeID, featureID string) usagebased.Charge {
	return usagebased.Charge{ChargeBase: usagebased.ChargeBase{
		ManagedResource: meta.ManagedResource{
			NamespacedModel: models.NamespacedModel{Namespace: namespace},
			ID:              chargeID,
		},
		Status: usagebased.StatusActive,
		State:  usagebased.State{FeatureID: featureID},
	}}
}

func newUsageBasedBillabilityMeter(namespace, meterID string) meter.Meter {
	return meter.Meter{
		ManagedResource: models.ManagedResource{
			NamespacedModel: models.NamespacedModel{Namespace: namespace},
			ID:              meterID,
		},
		Aggregation: meter.MeterAggregationSum,
	}
}

func newUsageBasedBillabilityLine(namespace, lineID, chargeID string, period timeutil.ClosedPeriod) billing.GatheringLine {
	return billing.GatheringLine{GatheringLineBase: billing.GatheringLineBase{
		ManagedResource: models.ManagedResource{
			NamespacedModel: models.NamespacedModel{Namespace: namespace},
			ID:              lineID,
			Name:            "usage",
		},
		ManagedBy:     billing.SystemManagedLine,
		Engine:        billing.LineEngineTypeChargeUsageBased,
		InvoiceID:     usageBasedBillabilityInvoiceID,
		Currency:      currencyx.FiatCode("USD"),
		ServicePeriod: period,
		InvoiceAt:     period.To,
		Price: *productcatalog.NewPriceFrom(productcatalog.UnitPrice{
			Amount: alpacadecimal.NewFromInt(1),
		}),
		FeatureKey: "line-feature-must-not-be-resolved",
		ChargeID:   lo.ToPtr(chargeID),
	}}
}

func TestLineEngineSplitGatheringLineKeepsChargeGroupingWithoutChildReferences(t *testing.T) {
	engine := &LineEngine{}

	servicePeriod := timeutil.ClosedPeriod{
		From: time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC),
		To:   time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC),
	}
	splitAt := time.Date(2026, 4, 16, 0, 0, 0, 0, time.UTC)

	price := productcatalog.NewPriceFrom(productcatalog.UnitPrice{
		Amount: alpacadecimal.NewFromInt(2),
	})

	line := billing.GatheringLine{
		GatheringLineBase: billing.GatheringLineBase{
			ManagedResource: models.NewManagedResource(models.ManagedResourceInput{
				Namespace: "test",
				ID:        "01K00000000000000000000001",
				Name:      "usage",
			}),
			ManagedBy:              billing.SystemManagedLine,
			Engine:                 billing.LineEngineTypeChargeUsageBased,
			Currency:               currencyx.FiatCode("USD"),
			ServicePeriod:          servicePeriod,
			InvoiceAt:              servicePeriod.To,
			Price:                  lo.FromPtr(price),
			FeatureKey:             "api-calls",
			ChargeID:               lo.ToPtr("01K00000000000000000000002"),
			ChildUniqueReferenceID: lo.ToPtr("original"),
		},
	}

	result, err := engine.SplitGatheringLine(t.Context(), billing.SplitGatheringLineInput{
		Line:    line,
		SplitAt: splitAt,
	})
	require.NoError(t, err)

	require.Nil(t, result.PreSplitAtLine.SplitLineGroupID)
	require.Nil(t, result.PreSplitAtLine.ChildUniqueReferenceID)
	require.Equal(t, splitAt, result.PreSplitAtLine.ServicePeriod.To)

	require.NotNil(t, result.PostSplitAtLine)
	require.Nil(t, result.PostSplitAtLine.SplitLineGroupID)
	require.Nil(t, result.PostSplitAtLine.ChildUniqueReferenceID)
	require.Equal(t, splitAt, result.PostSplitAtLine.ServicePeriod.From)
}

func TestValidateCustomCurrencyInvoiceLineDeleteAllowsDraftInvoice(t *testing.T) {
	servicePeriod := timeutil.ClosedPeriod{
		From: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		To:   time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC),
	}
	line := newUsageBasedStandardLineForTest(servicePeriod)
	run := usagebased.RealizationRun{
		RealizationRunBase: usagebased.RealizationRunBase{
			ID: usagebased.RealizationRunID{
				Namespace: line.Namespace,
				ID:        "run-id",
			},
			LineID:    lo.ToPtr(line.ID),
			InvoiceID: lo.ToPtr(line.InvoiceID),
		},
	}
	invoice := &billing.StandardInvoice{
		StandardInvoiceBase: billing.StandardInvoiceBase{
			Namespace: line.Namespace,
			ID:        line.InvoiceID,
			Status:    billing.StandardInvoiceStatusDraftCreated,
		},
	}

	require.NoError(t, validateCustomCurrencyInvoiceLineDelete(invoice, line.AsGenericLine(), run))
}
