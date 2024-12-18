package appstripe

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/invopop/gobl/currency"
	"github.com/samber/lo"
	"github.com/stretchr/testify/suite"

	"github.com/openmeterio/openmeter/openmeter/billing"
	customerentity "github.com/openmeterio/openmeter/openmeter/customer/entity"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/feature"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/models"
	billingtest "github.com/openmeterio/openmeter/test/billing"
)

type StripeInvoiceTestSuite struct {
	billingtest.BaseSuite
}

func TestStripeInvoicing(t *testing.T) {
	suite.Run(t, &StripeInvoiceTestSuite{})
}

func (s *StripeInvoiceTestSuite) SetupSuite() {
	s.BaseSuite.SetupSuite()
	// TODO: add any additional initialization required here
}

type ubpFeatures struct {
	flatPerUnit     feature.Feature
	flatPerUsage    feature.Feature
	tieredGraduated feature.Feature
	tieredVolume    feature.Feature
	aiFlatPerUnit   feature.Feature
}

func (s *StripeInvoiceTestSuite) TestComplexInvoice() {
	namespace := "ns-ubp-invoicing"
	ctx := context.Background()

	periodStart := lo.Must(time.Parse(time.RFC3339, "2024-09-02T12:13:14Z"))
	periodEnd := lo.Must(time.Parse(time.RFC3339, "2024-09-03T12:13:14Z"))

	_ = s.InstallSandboxApp(s.T(), namespace)

	s.MeterRepo.ReplaceMeters(ctx, []models.Meter{
		{
			Namespace:   namespace,
			Slug:        "flat-per-unit",
			WindowSize:  models.WindowSizeMinute,
			Aggregation: models.MeterAggregationSum,
		},
		{
			Namespace:   namespace,
			Slug:        "flat-per-usage",
			WindowSize:  models.WindowSizeMinute,
			Aggregation: models.MeterAggregationSum,
		},
		{
			Namespace:   namespace,
			Slug:        "tiered-graduated",
			WindowSize:  models.WindowSizeMinute,
			Aggregation: models.MeterAggregationSum,
		},
		{
			Namespace:   namespace,
			Slug:        "tiered-volume",
			WindowSize:  models.WindowSizeMinute,
			Aggregation: models.MeterAggregationSum,
		},
		{
			Namespace:   namespace,
			Slug:        "ai-flat-per-unit",
			WindowSize:  models.WindowSizeMinute,
			Aggregation: models.MeterAggregationSum,
		},
	})
	defer s.MeterRepo.ReplaceMeters(ctx, []models.Meter{})

	// Let's initialize the mock streaming connector with data that is out of the period so that we
	// can start with empty values
	for _, slug := range []string{"flat-per-unit", "flat-per-usage", "tiered-graduated", "tiered-volume"} {
		s.MockStreamingConnector.AddSimpleEvent(slug, 0, periodStart.Add(-time.Minute))
	}

	defer s.MockStreamingConnector.Reset()

	// Let's create the features
	// TODO[later]: we need to handle archived features, do we want to issue a warning? Can features be archived when used
	// by a draft invoice?
	features := ubpFeatures{
		flatPerUnit: lo.Must(s.FeatureService.CreateFeature(ctx, feature.CreateFeatureInputs{
			Namespace: namespace,
			Name:      "flat-per-unit",
			Key:       "flat-per-unit",
			MeterSlug: lo.ToPtr("flat-per-unit"),
		})),
		flatPerUsage: lo.Must(s.FeatureService.CreateFeature(ctx, feature.CreateFeatureInputs{
			Namespace: namespace,
			Name:      "flat-per-usage",
			Key:       "flat-per-usage",
			MeterSlug: lo.ToPtr("flat-per-usage"),
		})),
		tieredGraduated: lo.Must(s.FeatureService.CreateFeature(ctx, feature.CreateFeatureInputs{
			Namespace: namespace,
			Name:      "tiered-graduated",
			Key:       "tiered-graduated",
			MeterSlug: lo.ToPtr("tiered-graduated"),
		})),
		tieredVolume: lo.Must(s.FeatureService.CreateFeature(ctx, feature.CreateFeatureInputs{
			Namespace: namespace,
			Name:      "tiered-volume",
			Key:       "tiered-volume",
			MeterSlug: lo.ToPtr("tiered-volume"),
		})),
		aiFlatPerUnit: lo.Must(s.FeatureService.CreateFeature(ctx, feature.CreateFeatureInputs{
			Namespace: namespace,
			Name:      "ai-flat-per-unit",
			Key:       "ai-flat-per-unit",
			MeterSlug: lo.ToPtr("ai-flat-per-unit"),
		})),
	}

	// Given we have a test customer

	customerEntity, err := s.CustomerService.CreateCustomer(ctx, customerentity.CreateCustomerInput{
		Namespace: namespace,

		CustomerMutate: customerentity.CustomerMutate{
			Name:         "Test Customer",
			PrimaryEmail: lo.ToPtr("test@test.com"),
			Currency:     lo.ToPtr(currencyx.Code(currency.USD)),
			BillingAddress: &models.Address{
				Country: lo.ToPtr(models.CountryCode("US")),
			},
			UsageAttribution: customerentity.CustomerUsageAttribution{
				SubjectKeys: []string{"test"},
			},
		},
	})
	s.NoError(err)
	s.NotNil(customerEntity)
	s.NotEmpty(customerEntity.ID)

	// Given we have a default profile for the namespace
	minimalCreateProfileInput := billingtest.MinimalCreateProfileInputTemplate
	minimalCreateProfileInput.Namespace = namespace

	profile, err := s.BillingService.CreateProfile(ctx, minimalCreateProfileInput)

	s.NoError(err)
	s.NotNil(profile)

	s.Run("create pending invoice items", func() {
		// When we create pending invoice items
		pendingLines, err := s.BillingService.CreatePendingInvoiceLines(ctx,
			billing.CreateInvoiceLinesInput{
				Namespace: namespace,
				Lines: []billing.LineWithCustomer{
					{
						// Covered case: standalone flat line
						Line: billing.Line{
							LineBase: billing.LineBase{
								Period:    billing.Period{Start: periodStart, End: periodEnd},
								InvoiceAt: periodEnd,
								Currency:  currencyx.Code(currency.USD),
								Type:      billing.InvoiceLineTypeFee,
								Name:      "Fee",
							},
							FlatFee: &billing.FlatFeeLine{
								PerUnitAmount: alpacadecimal.NewFromFloat(100),
								PaymentTerm:   productcatalog.InArrearsPaymentTerm,
								Quantity:      alpacadecimal.NewFromFloat(1),
							},
						},
						CustomerID: customerEntity.ID,
					},
					{
						// Covered case: Discount caused by maximum amount
						Line: billing.Line{
							LineBase: billing.LineBase{
								Period:    billing.Period{Start: periodStart, End: periodEnd},
								InvoiceAt: periodEnd,
								Currency:  currencyx.Code(currency.USD),
								Type:      billing.InvoiceLineTypeUsageBased,
								Name:      "UBP - FLAT per unit",
							},
							UsageBased: &billing.UsageBasedLine{
								FeatureKey: features.flatPerUnit.Key,
								Price: productcatalog.NewPriceFrom(productcatalog.UnitPrice{
									Amount:        alpacadecimal.NewFromFloat(100),
									MaximumAmount: lo.ToPtr(alpacadecimal.NewFromFloat(2000)),
								}),
							},
						},
						CustomerID: customerEntity.ID,
					},
					{
						// Covered case: Very small per unit amount, high quantity, rounding to two decimal places
						Line: billing.Line{
							LineBase: billing.LineBase{
								Period:    billing.Period{Start: periodStart, End: periodEnd},
								InvoiceAt: periodEnd,
								Currency:  currencyx.Code(currency.USD),
								Type:      billing.InvoiceLineTypeUsageBased,
								Name:      "UBP - AI Usecase",
							},
							UsageBased: &billing.UsageBasedLine{
								FeatureKey: features.aiFlatPerUnit.Key,
								Price: productcatalog.NewPriceFrom(productcatalog.UnitPrice{
									Amount: alpacadecimal.NewFromFloat(0.00000075),
								}),
							},
						},
						CustomerID: customerEntity.ID,
					},
					{
						// Covered case: Flat line represented as UBP item
						Line: billing.Line{
							LineBase: billing.LineBase{
								Period:    billing.Period{Start: periodStart, End: periodEnd},
								InvoiceAt: periodEnd,
								Currency:  currencyx.Code(currency.USD),
								Type:      billing.InvoiceLineTypeUsageBased,
								Name:      "UBP - FLAT per any usage",
							},
							UsageBased: &billing.UsageBasedLine{
								FeatureKey: features.flatPerUsage.Key,
								Price: productcatalog.NewPriceFrom(productcatalog.FlatPrice{
									Amount:      alpacadecimal.NewFromFloat(100),
									PaymentTerm: productcatalog.InArrearsPaymentTerm,
								}),
							},
						},
						CustomerID: customerEntity.ID,
					},
					{
						// Covered case: Multiple lines per item, tier boundary is fractional
						Line: billing.Line{
							LineBase: billing.LineBase{
								Period:    billing.Period{Start: periodStart, End: periodEnd},
								InvoiceAt: periodEnd,
								Currency:  currencyx.Code(currency.USD),
								Type:      billing.InvoiceLineTypeUsageBased,
								Name:      "UBP - Tiered graduated",
							},
							UsageBased: &billing.UsageBasedLine{
								FeatureKey: features.tieredGraduated.Key,
								Price: productcatalog.NewPriceFrom(productcatalog.TieredPrice{
									Mode: productcatalog.GraduatedTieredPrice,
									Tiers: []productcatalog.PriceTier{
										{
											UpToAmount: lo.ToPtr(alpacadecimal.NewFromFloat(9.5)),
											UnitPrice: &productcatalog.PriceTierUnitPrice{
												Amount: alpacadecimal.NewFromFloat(100),
											},
										},
										{
											UpToAmount: lo.ToPtr(alpacadecimal.NewFromFloat(20)),
											UnitPrice: &productcatalog.PriceTierUnitPrice{
												Amount: alpacadecimal.NewFromFloat(90),
											},
										},
										{
											UnitPrice: &productcatalog.PriceTierUnitPrice{
												Amount: alpacadecimal.NewFromFloat(80),
											},
										},
									},
								}),
							},
						},
						CustomerID: customerEntity.ID,
					},
					{
						// Covered case: minimum amount charges
						Line: billing.Line{
							LineBase: billing.LineBase{
								Period:    billing.Period{Start: periodStart, End: periodEnd},
								InvoiceAt: periodEnd,
								Currency:  currencyx.Code(currency.USD),
								Type:      billing.InvoiceLineTypeUsageBased,
								Name:      "UBP - Tiered volume",
							},
							UsageBased: &billing.UsageBasedLine{
								FeatureKey: features.tieredVolume.Key,
								Price: productcatalog.NewPriceFrom(productcatalog.TieredPrice{
									Mode: productcatalog.VolumeTieredPrice,
									Tiers: []productcatalog.PriceTier{
										{
											UpToAmount: lo.ToPtr(alpacadecimal.NewFromFloat(10)),
											UnitPrice: &productcatalog.PriceTierUnitPrice{
												Amount: alpacadecimal.NewFromFloat(100),
											},
										},
										{
											UpToAmount: lo.ToPtr(alpacadecimal.NewFromFloat(20)),
											UnitPrice: &productcatalog.PriceTierUnitPrice{
												Amount: alpacadecimal.NewFromFloat(90),
											},
										},
										{
											UnitPrice: &productcatalog.PriceTierUnitPrice{
												Amount: alpacadecimal.NewFromFloat(80),
											},
										},
									},
									MinimumAmount: lo.ToPtr(alpacadecimal.NewFromFloat(3000)),
								}),
							},
						},
						CustomerID: customerEntity.ID,
					},
				},
			},
		)
		s.NoError(err)
		s.Len(pendingLines, 6)
	})

	s.Run("create invoice", func() {
		// Covered case: most measurements are fractional
		s.MockStreamingConnector.AddSimpleEvent("flat-per-unit", 32.2, periodStart.Add(time.Minute))
		s.MockStreamingConnector.AddSimpleEvent("flat-per-usage", 2, periodStart.Add(time.Minute))
		s.MockStreamingConnector.AddSimpleEvent("tiered-graduated", 35.3, periodStart.Add(time.Minute))
		s.MockStreamingConnector.AddSimpleEvent("tiered-volume", 15.3, periodStart.Add(time.Minute))
		s.MockStreamingConnector.AddSimpleEvent("ai-flat-per-unit", 103000025, periodStart.Add(time.Minute))

		// When we create an invoice
		invoices, err := s.BillingService.InvoicePendingLines(ctx, billing.InvoicePendingLinesInput{
			Customer: customerentity.CustomerID{
				ID:        customerEntity.ID,
				Namespace: namespace,
			},
			AsOf: &periodEnd,
		})
		s.NoError(err)
		s.Len(invoices, 1)

		invoice := invoices[0].RemoveCircularReferences()

		dump, err := json.Marshal(invoice)
		s.NoError(err)
		s.NotEmpty(dump)
		// TODO: temporary until the stripe invoice is implemented
		// fmt.Println(string(dump))
	})
}
