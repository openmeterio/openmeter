package service

import (
	"context"
	"fmt"
	"log/slog"
	"testing"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/samber/lo"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/flatfee"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/meta"
	featuremeterservice "github.com/openmeterio/openmeter/openmeter/billing/featuremeter/service"
	"github.com/openmeterio/openmeter/openmeter/billing/rating"
	"github.com/openmeterio/openmeter/openmeter/meter"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/feature"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/framework/transaction"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/pagination"
	"github.com/openmeterio/openmeter/pkg/timeutil"
)

func TestAreLinesBillableAsOfUsesBillingRating(t *testing.T) {
	period := timeutil.ClosedPeriod{
		From: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		To:   time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC),
	}
	ratingService := &billabilityRatingService{
		result: billing.IsLineBillableAsOfResult{
			Billable:       true,
			BillablePeriod: period,
		},
	}
	featureMeterResolver := newFlatFeeBillabilityFeatureMeterResolver(t)
	const chargeID = "charge-id"
	engine := &LineEngine{service: &service{
		adapter:              &flatFeeBillabilityAdapter{charges: []flatfee.Charge{newFlatFeeBillabilityCharge("namespace", chargeID, nil)}},
		featureMeterResolver: featureMeterResolver,
		ratingService:        ratingService,
	}}
	line := billing.GatheringLine{
		GatheringLineBase: billing.GatheringLineBase{
			ManagedResource: models.ManagedResource{
				NamespacedModel: models.NamespacedModel{Namespace: "namespace"},
				ID:              "line-id",
				Name:            "flat fee",
			},
			ManagedBy:     billing.SystemManagedLine,
			Engine:        billing.LineEngineTypeChargeFlatFee,
			InvoiceID:     "invoice-id",
			Currency:      currencyx.FiatCode("USD"),
			ServicePeriod: period,
			InvoiceAt:     period.From,
			Price: *productcatalog.NewPriceFrom(productcatalog.FlatPrice{
				Amount: alpacadecimal.NewFromInt(10),
			}),
			ChargeID: lo.ToPtr(chargeID),
		},
	}

	ctx, err := transaction.SetDriverOnContext(t.Context(), flatFeeBillabilityTransaction{})
	require.NoError(t, err)
	results, err := engine.AreLinesBillableAsOf(ctx, billing.AreLinesBillableAsOfInput{
		Invoice: billing.GatheringInvoice{GatheringInvoiceBase: billing.GatheringInvoiceBase{
			ManagedResource: models.ManagedResource{
				NamespacedModel: models.NamespacedModel{Namespace: line.Namespace},
				ID:              line.InvoiceID,
			},
		}},
		AsOf:               period.From,
		ProgressiveBilling: true,
		Lines:              billing.GatheringLines{line},
	})
	require.NoError(t, err)
	require.Equal(t, []billing.IsLineBillableAsOfResult{ratingService.result}, results)
	require.Len(t, ratingService.inputs, 1)
	require.Equal(t, line.ID, ratingService.inputs[0].Line.GetID())
	require.True(t, ratingService.inputs[0].ProgressiveBilling)
	require.Nil(t, ratingService.inputs[0].Feature)
	require.Nil(t, ratingService.inputs[0].Meter)
}

func TestAreLinesBillableAsOfValidatesChargeFeature(t *testing.T) {
	period := timeutil.ClosedPeriod{
		From: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		To:   time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC),
	}
	const chargeID = "charge-id"
	line := billing.GatheringLine{
		GatheringLineBase: billing.GatheringLineBase{
			ManagedResource: models.ManagedResource{
				NamespacedModel: models.NamespacedModel{Namespace: "namespace"},
				ID:              "line-id",
				Name:            "flat fee",
			},
			ManagedBy:     billing.SystemManagedLine,
			Engine:        billing.LineEngineTypeChargeFlatFee,
			InvoiceID:     "invoice-id",
			Currency:      currencyx.FiatCode("USD"),
			ServicePeriod: period,
			InvoiceAt:     period.From,
			Price: *productcatalog.NewPriceFrom(productcatalog.FlatPrice{
				Amount: alpacadecimal.NewFromInt(10),
			}),
			ChargeID: lo.ToPtr(chargeID),
		},
	}
	ratingService := &billabilityRatingService{result: billing.IsLineBillableAsOfResult{
		Billable:       true,
		BillablePeriod: period,
	}}
	engine := &LineEngine{service: &service{
		adapter: &flatFeeBillabilityAdapter{charges: []flatfee.Charge{
			newFlatFeeBillabilityCharge("namespace", chargeID, lo.ToPtr("missing-feature")),
		}},
		featureMeterResolver: newFlatFeeBillabilityFeatureMeterResolver(t),
		ratingService:        ratingService,
	}}
	ctx, err := transaction.SetDriverOnContext(t.Context(), flatFeeBillabilityTransaction{})
	require.NoError(t, err)

	results, err := engine.AreLinesBillableAsOf(ctx, billing.AreLinesBillableAsOfInput{
		Invoice: billing.GatheringInvoice{GatheringInvoiceBase: billing.GatheringInvoiceBase{
			ManagedResource: models.ManagedResource{
				NamespacedModel: models.NamespacedModel{Namespace: line.Namespace},
				ID:              line.InvoiceID,
			},
		}},
		AsOf:  period.From,
		Lines: billing.GatheringLines{line},
	})
	issues, systemErr := billing.ToValidationIssues(err)
	require.NoError(t, systemErr)
	require.Equal(t, billing.ValidationIssues{{
		Severity: billing.ValidationIssueSeverityCritical,
		Code:     billing.ErrInvoiceLineFeatureNotFound.Code,
		Message:  "feature[missing-feature]: invoice line: feature not found",
		Path:     "/charges/charge-id",
	}}, issues)
	require.Equal(t, []billing.IsLineBillableAsOfResult{{
		Billable:       true,
		BillablePeriod: period,
	}}, results)
	require.Len(t, ratingService.inputs, 1)
}

type flatFeeBillabilityAdapter struct {
	flatfee.Adapter
	charges []flatfee.Charge
}

func (a *flatFeeBillabilityAdapter) GetByIDs(_ context.Context, input flatfee.GetByIDsInput) ([]flatfee.Charge, error) {
	return lo.MapErr(input.IDs, func(chargeID string, _ int) (flatfee.Charge, error) {
		charge, ok := lo.Find(a.charges, func(charge flatfee.Charge) bool {
			return charge.Namespace == input.Namespace && charge.ID == chargeID
		})
		if !ok {
			return flatfee.Charge{}, fmt.Errorf("charge[%s] not found", chargeID)
		}

		return charge, nil
	})
}

type flatFeeBillabilityFeatureService struct{}

func (flatFeeBillabilityFeatureService) ListFeatures(context.Context, feature.ListFeaturesParams) (pagination.Result[feature.Feature], error) {
	return pagination.Result[feature.Feature]{}, nil
}

type flatFeeBillabilityMeterService struct{}

func (flatFeeBillabilityMeterService) ListMeters(context.Context, meter.ListMetersParams) (pagination.Result[meter.Meter], error) {
	return pagination.Result[meter.Meter]{}, nil
}

func newFlatFeeBillabilityFeatureMeterResolver(t *testing.T) *featuremeterservice.Resolver {
	t.Helper()

	resolver, err := featuremeterservice.New(featuremeterservice.Config{
		FeatureService: flatFeeBillabilityFeatureService{},
		MeterService:   flatFeeBillabilityMeterService{},
		Logger:         slog.Default(),
	})
	require.NoError(t, err)

	return resolver
}

func newFlatFeeBillabilityCharge(namespace, chargeID string, featureID *string) flatfee.Charge {
	return flatfee.Charge{ChargeBase: flatfee.ChargeBase{
		ManagedResource: meta.ManagedResource{
			NamespacedModel: models.NamespacedModel{Namespace: namespace},
			ID:              chargeID,
		},
		State: flatfee.State{FeatureID: featureID},
	}}
}

type flatFeeBillabilityTransaction struct{}

func (flatFeeBillabilityTransaction) Commit() error    { return nil }
func (flatFeeBillabilityTransaction) Rollback() error  { return nil }
func (flatFeeBillabilityTransaction) SavePoint() error { return nil }

type billabilityRatingService struct {
	result billing.IsLineBillableAsOfResult
	inputs []rating.ResolveBillablePeriodInput
}

func (s *billabilityRatingService) ResolveBillablePeriod(input rating.ResolveBillablePeriodInput) (billing.IsLineBillableAsOfResult, error) {
	s.inputs = append(s.inputs, input)

	return s.result, nil
}

func (*billabilityRatingService) GenerateDetailedLines(rating.StandardLineAccessor, ...rating.GenerateDetailedLinesOption) (rating.GenerateDetailedLinesResult, error) {
	return rating.GenerateDetailedLinesResult{}, nil
}

func TestValidateCustomCurrencyInvoiceLineDeleteAllowsDraftInvoice(t *testing.T) {
	servicePeriod := timeutil.ClosedPeriod{
		From: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		To:   time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC),
	}
	line := newFlatFeeStandardLineForTest(servicePeriod)
	charge := newFlatFeeCustomCurrencyCreditThenInvoiceChargeForTest(t, servicePeriod)
	run := newFlatFeeCustomCurrencyRunForTest(servicePeriod, line.Totals, false)
	run.LineID = lo.ToPtr(line.ID)
	run.InvoiceID = lo.ToPtr(line.InvoiceID)
	charge.Realizations.CurrentRun = &run

	invoice := &billing.StandardInvoice{
		StandardInvoiceBase: billing.StandardInvoiceBase{
			Namespace: line.Namespace,
			ID:        line.InvoiceID,
			Status:    billing.StandardInvoiceStatusDraftCreated,
		},
	}

	require.NoError(t, validateCustomCurrencyInvoiceLineDelete(invoice, line.AsGenericLine(), charge))
}
