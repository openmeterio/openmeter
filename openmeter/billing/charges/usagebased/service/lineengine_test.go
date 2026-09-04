package service

import (
	"context"
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
		Severity: billing.ValidationIssueSeverityCritical,
		Code:     billing.ErrInvoiceLineFeatureNotFound.Code,
		Message:  "feature[missing-feature]: invoice line: feature not found",
		Path:     "/charges/charge",
	}}, issues)
	require.Equal(t, []billing.IsLineBillableAsOfResult{{}}, results)
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

type usageBasedBillabilityFeatureService struct {
	features []feature.Feature
}

func (s usageBasedBillabilityFeatureService) ListFeatures(_ context.Context, params feature.ListFeaturesParams) (pagination.Result[feature.Feature], error) {
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
