package rating

import (
	"testing"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/samber/lo"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/billing"
	chargesmeta "github.com/openmeterio/openmeter/openmeter/billing/charges/meta"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/usagebased"
	billingrating "github.com/openmeterio/openmeter/openmeter/billing/rating"
	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/openmeter/meter"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/feature"
	streamingtestutils "github.com/openmeterio/openmeter/openmeter/streaming/testutils"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/timeutil"
)

type getDetailedLinesForUsageFixture struct {
	config Config
	input  GetDetailedLinesForUsageInput
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

func TestGetRatingForUsageAddsServicePeriodToDetailedLineChildUniqueReferenceIDs(t *testing.T) {
	t.Parallel()

	ctx := t.Context()

	fixture := newGetDetailedLinesForUsageFixture(t, billingrating.GenerateDetailedLinesResult{
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

	out, err := svc.GetDetailedLinesForUsage(ctx, fixture.input)
	require.NoError(t, err)

	require.Equal(t, "unit-price-usage@[2025-01-01T00:00:00Z..2025-02-01T00:00:00Z]", out.DetailedLines[0].ChildUniqueReferenceID)
	require.Equal(t, "rateCardDiscount/correlationID=discount-50pct@[2025-01-01T00:00:00Z..2025-02-01T00:00:00Z]", out.DetailedLines[1].ChildUniqueReferenceID)
	require.False(t, fixture.rater.lastOpts.IgnoreMinimumCommitment)
}

func TestGetRatingForUsageCanIgnoreMinimumCommitment(t *testing.T) {
	t.Parallel()

	ctx := t.Context()
	fixture := newGetDetailedLinesForUsageFixture(t, billingrating.GenerateDetailedLinesResult{})
	fixture.input.IgnoreMinimumCommitment = true

	svc, err := New(fixture.config)
	require.NoError(t, err)

	_, err = svc.GetDetailedLinesForUsage(ctx, fixture.input)
	require.NoError(t, err)
	require.True(t, fixture.rater.lastOpts.IgnoreMinimumCommitment)
}

func newGetDetailedLinesForUsageFixture(t *testing.T, result billingrating.GenerateDetailedLinesResult) getDetailedLinesForUsageFixture {
	t.Helper()

	servicePeriod := timeutil.ClosedPeriod{
		From: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
		To:   time.Date(2025, 2, 1, 0, 0, 0, 0, time.UTC),
	}

	streamingConnector := streamingtestutils.NewMockStreamingConnector(t)
	streamingConnector.AddSimpleEvent("meter-1", 12, servicePeriod.From.Add(30*time.Minute))

	ratingService := &stubRatingService{result: result}

	return getDetailedLinesForUsageFixture{
		config: Config{
			StreamingConnector: streamingConnector,
			RatingService:      ratingService,
		},
		input: GetDetailedLinesForUsageInput{
			Charge: usagebased.Charge{
				ChargeBase: usagebased.ChargeBase{
					ManagedResource: chargesmeta.ManagedResource{
						NamespacedModel: models.NamespacedModel{Namespace: "ns"},
						ManagedModel: models.ManagedModel{
							CreatedAt: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
							UpdatedAt: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
						},
						ID: "charge-1",
					},
					Intent: usagebased.Intent{
						Intent: chargesmeta.Intent{
							Name:              "usage-charge",
							ManagedBy:         billing.SubscriptionManagedLine,
							CustomerID:        "customer-1",
							Currency:          currencyx.Code("USD"),
							ServicePeriod:     servicePeriod,
							FullServicePeriod: servicePeriod,
							BillingPeriod:     servicePeriod,
						},
						InvoiceAt:      time.Date(2025, 2, 1, 0, 0, 0, 0, time.UTC),
						SettlementMode: productcatalog.InvoiceOnlySettlementMode,
						FeatureKey:     "feature-1",
						Price: *productcatalog.NewPriceFrom(productcatalog.UnitPrice{
							Amount: alpacadecimal.NewFromInt(3),
						}),
					},
					Status: usagebased.StatusCreated,
					State: usagebased.State{
						FeatureID: "feature-1",
					},
				},
			},
			Customer: billing.CustomerOverrideWithDetails{
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
			},
			FeatureMeter: feature.FeatureMeter{
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
			},
			StoredAtOffset: time.Date(2025, 2, 2, 0, 0, 0, 0, time.UTC),
		},
		rater: ratingService,
	}
}

type stubRatingService struct {
	result   billingrating.GenerateDetailedLinesResult
	lastOpts billingrating.GenerateDetailedLinesOptions
}

func (s *stubRatingService) ResolveBillablePeriod(in billingrating.ResolveBillablePeriodInput) (*timeutil.ClosedPeriod, error) {
	return nil, nil
}

func (s *stubRatingService) GenerateDetailedLines(in billingrating.StandardLineAccessor, opts ...billingrating.GenerateDetailedLinesOption) (billingrating.GenerateDetailedLinesResult, error) {
	s.lastOpts = billingrating.NewGenerateDetailedLinesOptions(opts...)
	return s.result, nil
}
