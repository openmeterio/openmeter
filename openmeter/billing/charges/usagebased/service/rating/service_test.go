package rating

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/samber/lo"
	"github.com/samber/mo"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/billing"
	chargesmeta "github.com/openmeterio/openmeter/openmeter/billing/charges/meta"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/usagebased"
	"github.com/openmeterio/openmeter/openmeter/billing/models/stddetailedline"
	"github.com/openmeterio/openmeter/openmeter/billing/models/totals"
	billingrating "github.com/openmeterio/openmeter/openmeter/billing/rating"
	billingratingservice "github.com/openmeterio/openmeter/openmeter/billing/rating/service"
	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/openmeter/meter"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/feature"
	streamingtestutils "github.com/openmeterio/openmeter/openmeter/streaming/testutils"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/timeutil"
)

type getDetailedRatingForUsageFixture struct {
	config Config
	input  GetDetailedRatingForUsageInput
	rater  *stubRatingService
}

func TestFormatDetailedLineChildUniqueReferenceID(t *testing.T) {
	t.Parallel()

	servicePeriod := timeutil.ClosedPeriod{
		From: time.Date(2025, 1, 1, 12, 30, 45, 0, time.FixedZone("CET", 3600)),
		To:   time.Date(2025, 2, 1, 1, 2, 3, 0, time.FixedZone("PST", -8*3600)),
	}

	require.Equal(
		t,
		"unit-price-usage@[2025-01-01T11:30:45Z..2025-02-01T09:02:03Z]",
		formatDetailedLineChildUniqueReferenceID("unit-price-usage", servicePeriod),
	)
}

func TestGetDetailedRatingForUsageAddsServicePeriodToDetailedLineChildUniqueReferenceIDs(t *testing.T) {
	t.Parallel()

	fixture := newGetDetailedRatingForUsageFixture(t, billingrating.GenerateDetailedLinesResult{
		DetailedLines: billingrating.DetailedLines{
			{
				Name:                   "Usage",
				Quantity:               alpacadecimal.NewFromInt(12),
				PerUnitAmount:          alpacadecimal.NewFromInt(3),
				ChildUniqueReferenceID: "unit-price-usage",
			},
			{
				Name:                   "Discount",
				Quantity:               alpacadecimal.NewFromInt(1),
				PerUnitAmount:          alpacadecimal.NewFromInt(1),
				ChildUniqueReferenceID: "rateCardDiscount/correlationID=discount-50pct",
			},
		},
	})

	svc, err := New(fixture.config)
	require.NoError(t, err)

	out, err := svc.GetDetailedRatingForUsage(t.Context(), fixture.input)
	require.NoError(t, err)

	require.Equal(t, "unit-price-usage@[2025-01-01T00:00:00Z..2025-02-01T00:00:00Z]", out.DetailedLines[0].ChildUniqueReferenceID)
	require.Equal(t, "rateCardDiscount/correlationID=discount-50pct@[2025-01-01T00:00:00Z..2025-02-01T00:00:00Z]", out.DetailedLines[1].ChildUniqueReferenceID)
	require.Equal(t, float64(12), out.Quantity.InexactFloat64())
	require.False(t, fixture.rater.lastOpts.IgnoreMinimumCommitment)
}

func TestMapBillingRatingDetailedLinesToUsageBasedDetailedLines(t *testing.T) {
	t.Parallel()

	defaultServicePeriod := timeutil.ClosedPeriod{
		From: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
		To:   time.Date(2025, 2, 1, 0, 0, 0, 0, time.UTC),
	}
	explicitServicePeriod := timeutil.ClosedPeriod{
		From: time.Date(2025, 1, 15, 0, 0, 0, 0, time.UTC),
		To:   time.Date(2025, 2, 1, 0, 0, 0, 0, time.UTC),
	}
	intent := newDetailedRatingTestCharge(defaultServicePeriod, nil).Intent
	intent.TaxConfig = &productcatalog.TaxCodeConfig{
		Behavior: lo.ToPtr(productcatalog.ExclusiveTaxBehavior),
	}

	out := mapBillingRatingDetailedLinesToUsageBasedDetailedLines(
		intent,
		defaultServicePeriod,
		billingrating.DetailedLines{
			{
				Name:                   "Usage",
				Quantity:               alpacadecimal.NewFromInt(12),
				PerUnitAmount:          alpacadecimal.NewFromInt(3),
				ChildUniqueReferenceID: "unit-price-usage",
				Totals: totals.Totals{
					Amount: alpacadecimal.NewFromInt(36),
					Total:  alpacadecimal.NewFromInt(36),
				},
			},
			{
				Name:                   "Commitment",
				Quantity:               alpacadecimal.NewFromInt(1),
				PerUnitAmount:          alpacadecimal.NewFromInt(100),
				ChildUniqueReferenceID: "minimum-spend",
				Period:                 lo.ToPtr(explicitServicePeriod),
				PaymentTerm:            productcatalog.InAdvancePaymentTerm,
				Category:               stddetailedline.CategoryCommitment,
			},
		},
	)

	require.Len(t, out, 2)
	require.Empty(t, out[0].Namespace)
	require.Equal(t, "Usage", out[0].Name)
	require.Equal(t, defaultServicePeriod, out[0].ServicePeriod)
	require.Equal(t, currencyx.Code("USD"), out[0].Currency)
	require.Equal(t, productcatalog.InArrearsPaymentTerm, out[0].PaymentTerm)
	require.Equal(t, stddetailedline.CategoryRegular, out[0].Category)
	require.Equal(t, float64(12), out[0].Quantity.InexactFloat64())
	require.Equal(t, float64(3), out[0].PerUnitAmount.InexactFloat64())
	require.Equal(t, float64(36), out[0].Totals.Total.InexactFloat64())
	require.NotNil(t, out[0].TaxConfig)
	require.NotSame(t, intent.TaxConfig, out[0].TaxConfig)

	require.Equal(t, explicitServicePeriod, out[1].ServicePeriod)
	require.Equal(t, productcatalog.InAdvancePaymentTerm, out[1].PaymentTerm)
	require.Equal(t, stddetailedline.CategoryCommitment, out[1].Category)
}

func TestGetDetailedRatingForUsageIgnoresMinimumCommitmentForPartialRun(t *testing.T) {
	t.Parallel()

	fixture := newGetDetailedRatingForUsageFixture(t, billingrating.GenerateDetailedLinesResult{})
	partialServicePeriodTo := fixture.input.Charge.Intent.ServicePeriod.From.Add(24 * time.Hour)
	fixture.input.ServicePeriodTo = partialServicePeriodTo

	svc, err := New(fixture.config)
	require.NoError(t, err)

	_, err = svc.GetDetailedRatingForUsage(t.Context(), fixture.input)
	require.NoError(t, err)
	require.True(t, fixture.rater.lastOpts.IgnoreMinimumCommitment)
}

func TestGetDetailedRatingForUsageIgnoresCurrentRunOnCharge(t *testing.T) {
	t.Parallel()

	fixture := newGetDetailedRatingForUsageFixture(t, billingrating.GenerateDetailedLinesResult{})
	currentRun := newDetailedRatingTestRun("current", fixture.input.ServicePeriodTo, 0)
	fixture.input.Charge.Realizations = usagebased.RealizationRuns{currentRun}

	svc, err := New(fixture.config)
	require.NoError(t, err)

	_, err = svc.GetDetailedRatingForUsage(t.Context(), fixture.input)
	require.NoError(t, err)
}

func TestGetDetailedRatingForUsageLoadsPriorDetailedLines(t *testing.T) {
	t.Parallel()

	fixture := newGetDetailedRatingForUsageFixture(t, billingrating.GenerateDetailedLinesResult{})
	priorRun := newDetailedRatingTestRun("prior", fixture.input.Charge.Intent.ServicePeriod.From.Add(24*time.Hour), 0)
	fixture.input.Charge.Realizations = usagebased.RealizationRuns{priorRun}

	var called int
	fixture.config.DetailedLinesFetcher = detailedLinesFetcherFunc(func(_ context.Context, charge usagebased.Charge) (usagebased.Charge, error) {
		called++
		charge.Realizations[0].DetailedLines = mo.Some(usagebased.DetailedLines{})

		return charge, nil
	})

	svc, err := New(fixture.config)
	require.NoError(t, err)

	_, err = svc.GetDetailedRatingForUsage(t.Context(), fixture.input)
	require.NoError(t, err)
	require.Equal(t, 1, called)
}

func TestGetDetailedRatingForUsageDoesNotLoadCurrentRunDetailedLines(t *testing.T) {
	t.Parallel()

	fixture := newGetDetailedRatingForUsageFixture(t, billingrating.GenerateDetailedLinesResult{})
	currentRun := newDetailedRatingTestRun("current", fixture.input.ServicePeriodTo, 0)
	fixture.input.Charge.Realizations = usagebased.RealizationRuns{currentRun}

	var called int
	fixture.config.DetailedLinesFetcher = detailedLinesFetcherFunc(func(_ context.Context, charge usagebased.Charge) (usagebased.Charge, error) {
		called++

		return charge, nil
	})

	svc, err := New(fixture.config)
	require.NoError(t, err)

	_, err = svc.GetDetailedRatingForUsage(t.Context(), fixture.input)
	require.NoError(t, err)
	require.Zero(t, called)
}

func TestGetDetailedRatingForUsageWrapsDetailedLinesLoadError(t *testing.T) {
	t.Parallel()

	fixture := newGetDetailedRatingForUsageFixture(t, billingrating.GenerateDetailedLinesResult{})
	priorRun := newDetailedRatingTestRun("prior", fixture.input.Charge.Intent.ServicePeriod.From.Add(24*time.Hour), 0)
	fixture.input.Charge.Realizations = usagebased.RealizationRuns{priorRun}
	fixture.config.DetailedLinesFetcher = detailedLinesFetcherFunc(func(_ context.Context, charge usagebased.Charge) (usagebased.Charge, error) {
		return charge, errors.New("boom")
	})

	svc, err := New(fixture.config)
	require.NoError(t, err)

	_, err = svc.GetDetailedRatingForUsage(t.Context(), fixture.input)
	require.ErrorContains(t, err, "fetch detailed lines")
	require.ErrorContains(t, err, "boom")
}

func TestGetDetailedRatingForUsageFiltersQuantityByServicePeriodToAndStoredAtLT(t *testing.T) {
	t.Parallel()

	servicePeriod := timeutil.ClosedPeriod{
		From: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
		To:   time.Date(2025, 2, 1, 0, 0, 0, 0, time.UTC),
	}
	servicePeriodTo := servicePeriod.From.Add(24 * time.Hour)
	storedAtLT := servicePeriod.From.Add(48 * time.Hour)

	streamingConnector := streamingtestutils.NewMockStreamingConnector(t)
	streamingConnector.AddSimpleEvent("meter-1", 2, servicePeriod.From.Add(time.Hour), streamingtestutils.WithStoredAt(storedAtLT.Add(-time.Second)))
	streamingConnector.AddSimpleEvent("meter-1", 3, servicePeriod.From.Add(2*time.Hour), streamingtestutils.WithStoredAt(storedAtLT))
	streamingConnector.AddSimpleEvent("meter-1", 5, servicePeriodTo, streamingtestutils.WithStoredAt(storedAtLT.Add(-time.Second)))
	streamingConnector.AddSimpleEvent("meter-1", 7, servicePeriodTo.Add(time.Hour), streamingtestutils.WithStoredAt(storedAtLT.Add(-time.Second)))

	svc, err := New(Config{
		StreamingConnector:   streamingConnector,
		RatingService:        &stubRatingService{},
		DetailedLinesFetcher: passthroughDetailedLinesFetcher,
	})
	require.NoError(t, err)

	out, err := svc.GetDetailedRatingForUsage(t.Context(), GetDetailedRatingForUsageInput{
		Charge:          newDetailedRatingTestCharge(servicePeriod, usagebased.RealizationRuns{}),
		ServicePeriodTo: servicePeriodTo,
		StoredAtLT:      storedAtLT,
		Customer:        newDetailedRatingTestCustomer(),
		FeatureMeter:    newDetailedRatingTestFeatureMeter(),
	})
	require.NoError(t, err)

	require.Equal(t, float64(2), out.Quantity.InexactFloat64())
}

func TestGetTotalsForUsageMinimumCommitment(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name                    string
		ignoreMinimumCommitment bool
		expectTotal             float64
	}{
		{
			name:                    "ignored",
			ignoreMinimumCommitment: true,
			expectTotal:             36,
		},
		{
			name:                    "included",
			ignoreMinimumCommitment: false,
			expectTotal:             100,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			servicePeriod := timeutil.ClosedPeriod{
				From: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
				To:   time.Date(2025, 2, 1, 0, 0, 0, 0, time.UTC),
			}

			streamingConnector := streamingtestutils.NewMockStreamingConnector(t)
			streamingConnector.AddSimpleEvent("meter-1", 12, servicePeriod.From.Add(30*time.Minute))

			svc, err := New(Config{
				StreamingConnector:   streamingConnector,
				RatingService:        billingratingservice.New(),
				DetailedLinesFetcher: passthroughDetailedLinesFetcher,
			})
			require.NoError(t, err)

			charge := newDetailedRatingTestCharge(servicePeriod, usagebased.RealizationRuns{})
			charge.Intent.Price = *productcatalog.NewPriceFrom(productcatalog.UnitPrice{
				Amount: alpacadecimal.NewFromInt(3),
				Commitments: productcatalog.Commitments{
					MinimumAmount: lo.ToPtr(alpacadecimal.NewFromInt(100)),
				},
			})

			out, err := svc.GetTotalsForUsage(t.Context(), GetTotalsForUsageInput{
				Charge:                  charge,
				Customer:                newDetailedRatingTestCustomer(),
				FeatureMeter:            newDetailedRatingTestFeatureMeter(),
				StoredAtLT:              servicePeriod.To,
				IgnoreMinimumCommitment: tt.ignoreMinimumCommitment,
			})
			require.NoError(t, err)

			require.Equal(t, tt.expectTotal, out.Total.InexactFloat64())
		})
	}
}

func newGetDetailedRatingForUsageFixture(t *testing.T, result billingrating.GenerateDetailedLinesResult) getDetailedRatingForUsageFixture {
	t.Helper()

	servicePeriod := timeutil.ClosedPeriod{
		From: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
		To:   time.Date(2025, 2, 1, 0, 0, 0, 0, time.UTC),
	}

	streamingConnector := streamingtestutils.NewMockStreamingConnector(t)
	streamingConnector.AddSimpleEvent("meter-1", 12, servicePeriod.From.Add(30*time.Minute))

	ratingService := &stubRatingService{result: result}
	currentRun := newDetailedRatingTestRun("current", servicePeriod.To, 0)

	return getDetailedRatingForUsageFixture{
		config: Config{
			StreamingConnector:   streamingConnector,
			RatingService:        ratingService,
			DetailedLinesFetcher: passthroughDetailedLinesFetcher,
		},
		input: GetDetailedRatingForUsageInput{
			Charge:          newDetailedRatingTestCharge(servicePeriod, usagebased.RealizationRuns{}),
			ServicePeriodTo: servicePeriod.To,
			StoredAtLT:      currentRun.StoredAtLT,
			Customer:        newDetailedRatingTestCustomer(),
			FeatureMeter:    newDetailedRatingTestFeatureMeter(),
		},
		rater: ratingService,
	}
}

func newDetailedRatingTestRun(id string, servicePeriodTo time.Time, meteredQuantity int64) usagebased.RealizationRun {
	now := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)

	return usagebased.RealizationRun{
		RealizationRunBase: usagebased.RealizationRunBase{
			ID: usagebased.RealizationRunID{
				Namespace: "ns",
				ID:        id,
			},
			ManagedModel:    models.ManagedModel{CreatedAt: now, UpdatedAt: now},
			FeatureID:       "feature-1",
			Type:            usagebased.RealizationRunTypeFinalRealization,
			StoredAtLT:      time.Date(2025, 2, 2, 0, 0, 0, 0, time.UTC),
			ServicePeriodTo: servicePeriodTo,
			MeteredQuantity: alpacadecimal.NewFromInt(meteredQuantity),
			Totals: totals.Totals{
				Amount: alpacadecimal.NewFromInt(meteredQuantity),
				Total:  alpacadecimal.NewFromInt(meteredQuantity),
			},
		},
	}
}

func newDetailedRatingTestCharge(period timeutil.ClosedPeriod, runs usagebased.RealizationRuns) usagebased.Charge {
	return usagebased.Charge{
		ChargeBase: usagebased.ChargeBase{
			ManagedResource: chargesmeta.ManagedResource{
				NamespacedModel: models.NamespacedModel{Namespace: "ns"},
				ManagedModel: models.ManagedModel{
					CreatedAt: period.From,
					UpdatedAt: period.From,
				},
				ID: "charge-1",
			},
			Intent: usagebased.Intent{
				Intent: chargesmeta.Intent{
					Name:              "usage-charge",
					ManagedBy:         billing.SubscriptionManagedLine,
					CustomerID:        "customer-1",
					Currency:          currencyx.Code("USD"),
					ServicePeriod:     period,
					FullServicePeriod: period,
					BillingPeriod:     period,
				},
				InvoiceAt:      period.To,
				SettlementMode: productcatalog.InvoiceOnlySettlementMode,
				FeatureKey:     "feature-1",
				Price: *productcatalog.NewPriceFrom(productcatalog.UnitPrice{
					Amount: alpacadecimal.NewFromInt(3),
				}),
			},
			Status: usagebased.StatusCreated,
			State: usagebased.State{
				FeatureID:    "feature-1",
				RatingEngine: usagebased.RatingEngineDelta,
			},
		},
		Realizations: runs,
	}
}

func newDetailedRatingTestCustomer() billing.CustomerOverrideWithDetails {
	return billing.CustomerOverrideWithDetails{
		Customer: &customer.Customer{
			ManagedResource: models.NewManagedResource(models.ManagedResourceInput{
				Namespace: "ns",
				ID:        "customer-1",
				Name:      "Customer 1",
				CreatedAt: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
				UpdatedAt: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
			}),
			Key: lo.ToPtr("cust-1"),
		},
	}
}

func newDetailedRatingTestFeatureMeter() feature.FeatureMeter {
	return feature.FeatureMeter{
		Feature: feature.Feature{
			Namespace: "ns",
			ID:        "feature-1",
			Name:      "Feature 1",
			Key:       "feature-1",
			MeterID:   lo.ToPtr("meter-1"),
			CreatedAt: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
			UpdatedAt: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
		},
		Meter: &meter.Meter{
			ManagedResource: models.NewManagedResource(models.ManagedResourceInput{
				Namespace: "ns",
				ID:        "meter-1",
				Name:      "Meter 1",
				CreatedAt: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
				UpdatedAt: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
			}),
			Key:           "meter-1",
			Aggregation:   meter.MeterAggregationSum,
			EventType:     "event.type",
			ValueProperty: lo.ToPtr("value"),
		},
	}
}

type stubRatingService struct {
	result   billingrating.GenerateDetailedLinesResult
	lastOpts billingrating.GenerateDetailedLinesOptions
}

// detailedLinesFetcherFunc keeps rating tests focused on the FetchDetailedLines behavior under test
// without forcing each case to define a one-off struct for the single-method dependency.
type detailedLinesFetcherFunc func(ctx context.Context, charge usagebased.Charge) (usagebased.Charge, error)

func (f detailedLinesFetcherFunc) FetchDetailedLines(ctx context.Context, charge usagebased.Charge) (usagebased.Charge, error) {
	return f(ctx, charge)
}

var passthroughDetailedLinesFetcher = detailedLinesFetcherFunc(func(_ context.Context, charge usagebased.Charge) (usagebased.Charge, error) {
	return charge, nil
})

func (s *stubRatingService) ResolveBillablePeriod(in billingrating.ResolveBillablePeriodInput) (*timeutil.ClosedPeriod, error) {
	return nil, nil
}

func (s *stubRatingService) GenerateDetailedLines(in billingrating.StandardLineAccessor, opts ...billingrating.GenerateDetailedLinesOption) (billingrating.GenerateDetailedLinesResult, error) {
	s.lastOpts = billingrating.NewGenerateDetailedLinesOptions(opts...)
	return s.result, nil
}
