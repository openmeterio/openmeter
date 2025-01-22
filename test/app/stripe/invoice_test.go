package appstripe

import (
	"context"
	"fmt"
	"regexp"
	"testing"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/invopop/gobl/currency"
	"github.com/samber/lo"
	"github.com/stretchr/testify/suite"
	"github.com/stripe/stripe-go/v80"

	appstripe "github.com/openmeterio/openmeter/openmeter/app/stripe"
	appstripeadapter "github.com/openmeterio/openmeter/openmeter/app/stripe/adapter"
	stripeclient "github.com/openmeterio/openmeter/openmeter/app/stripe/client"
	appstripeservice "github.com/openmeterio/openmeter/openmeter/app/stripe/service"
	"github.com/openmeterio/openmeter/openmeter/billing"
	customerentity "github.com/openmeterio/openmeter/openmeter/customer/entity"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/feature"
	"github.com/openmeterio/openmeter/openmeter/secret"
	secretadapter "github.com/openmeterio/openmeter/openmeter/secret/adapter"
	secretservice "github.com/openmeterio/openmeter/openmeter/secret/service"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/models"
	billingtest "github.com/openmeterio/openmeter/test/billing"
)

type StripeInvoiceTestSuite struct {
	billingtest.BaseSuite

	AppStripeService appstripe.Service
	Fixture          *Fixture
	SecretService    secret.Service
	StripeAppClient  *StripeAppClientMock
}

func TestStripeInvoicing(t *testing.T) {
	suite.Run(t, &StripeInvoiceTestSuite{})
}

func (s *StripeInvoiceTestSuite) SetupSuite() {
	s.BaseSuite.SetupSuite()

	// Secret
	secretAdapter := secretadapter.New()

	secretService, err := secretservice.New(secretservice.Config{
		Adapter: secretAdapter,
	})
	s.Require().NoError(err, "failed to create secret service")

	s.SecretService = secretService

	// Stripe Client
	stripeClient := &StripeClientMock{}
	stripeAppClient := &StripeAppClientMock{}

	s.StripeAppClient = stripeAppClient

	// App Stripe
	appStripeAdapter, err := appstripeadapter.New(appstripeadapter.Config{
		Client:          s.DBClient,
		AppService:      s.AppService,
		CustomerService: s.CustomerService,
		SecretService:   secretService,
		StripeClientFactory: func(config stripeclient.StripeClientConfig) (stripeclient.StripeClient, error) {
			return stripeClient, nil
		},
		StripeAppClientFactory: func(config stripeclient.StripeAppClientConfig) (stripeclient.StripeAppClient, error) {
			return stripeAppClient, nil
		},
	})
	s.Require().NoError(err, "failed to create app stripe adapter")

	appStripeService, err := appstripeservice.New(appstripeservice.Config{
		Adapter:       appStripeAdapter,
		AppService:    s.AppService,
		SecretService: secretService,
	})
	s.Require().NoError(err, "failed to create app stripe service")

	s.AppStripeService = appStripeService

	// Fixture
	s.Fixture = NewFixture(s.AppService, s.CustomerService, stripeClient)
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
								Category:      billing.FlatFeeCategoryRegular,
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

	s.Run("upsert invoice", func() {
		// Setup the app with the customer
		app, err := s.Fixture.setupApp(ctx, namespace)
		s.NoError(err)

		customerData, err := s.Fixture.setupAppCustomerData(ctx, app, customerEntity)
		s.NoError(err)

		// Covered case: most measurements are fractional
		s.MockStreamingConnector.AddSimpleEvent("flat-per-unit", 32.2, periodStart.Add(time.Minute))
		s.MockStreamingConnector.AddSimpleEvent("flat-per-usage", 2, periodStart.Add(time.Minute))
		s.MockStreamingConnector.AddSimpleEvent("tiered-graduated", 35.3, periodStart.Add(time.Minute))
		s.MockStreamingConnector.AddSimpleEvent("tiered-volume", 15.3, periodStart.Add(time.Minute))
		s.MockStreamingConnector.AddSimpleEvent("ai-flat-per-unit", 103000025, periodStart.Add(time.Minute))

		// When we create an invoice
		invoices, err := s.BillingService.InvoicePendingLines(ctx, billing.InvoicePendingLinesInput{
			Customer: customerEntity.GetID(),
			AsOf:     &periodEnd,
		})
		s.NoError(err)
		s.Len(invoices, 1)

		invoice := invoices[0].RemoveCircularReferences()

		// Create a new invoice for the customer.
		invoicingApp, err := billing.GetApp(app)
		s.NoError(err)

		// Mock the stripe client to return the created invoice.
		s.StripeAppClient.
			On("CreateInvoice", stripeclient.CreateInvoiceInput{
				AutomaticTaxEnabled: true,
				StripeCustomerID:    customerData.StripeCustomerID,
				Currency:            "USD",
			}).
			Return(&stripe.Invoice{
				ID: "stripe-invoice-id",
				Customer: &stripe.Customer{
					ID: customerData.StripeCustomerID,
				},
				Currency: "USD",
				Lines: &stripe.InvoiceLineItemList{
					Data: []*stripe.InvoiceLineItem{},
				},
			}, nil)

		// TODO: do not share env between tests
		defer s.StripeAppClient.Restore()

		expectedPeriodStart := time.Unix(int64(1725279180), 0)
		expectedPeriodEnd := time.Unix(int64(1725365580), 0)

		getLine := func(description string) *billing.Line {
			for _, line := range invoice.GetLeafLines() {
				name := line.Name
				if line.Description != nil {
					name = fmt.Sprintf("%s (%s)", name, *line.Description)
				}

				if name == description {
					return line
				}
			}

			return nil
		}

		getLineID := func(description string) string {
			line := getLine(description)
			return line.ID
		}

		getDiscountID := func(description string) string {
			r, err := regexp.Compile(`^(.+) \((.+)\)$`)
			s.NoError(err)

			group := r.FindAllStringSubmatch(description, -1)

			id := ""

			line := getLine(group[0][1])
			if line == nil {
				return id
			}

			line.Discounts.ForEach(func(discounts []billing.LineDiscount) {
				for _, discount := range discounts {
					if discount.Description != nil && group[0][2] == *discount.Description {
						id = discount.ID
						return
					}
				}
			})

			return id
		}

		expectedInvoiceAddLines := []*stripe.InvoiceAddLinesLineParams{
			{
				Amount:      lo.ToPtr(int64(10000)),
				Description: lo.ToPtr("Fee"),
				Period: &stripe.InvoiceAddLinesLinePeriodParams{
					Start: lo.ToPtr(periodStart.Unix()),
					// TODO: check time shift
					End: lo.ToPtr(periodStart.Add(time.Hour * 24).Unix()),
				},
				Metadata: map[string]string{
					"om_line_id":   getLineID("Fee"),
					"om_line_type": "line",
				},
			},
			{
				Amount:      lo.ToPtr(int64(7725)),
				Description: lo.ToPtr("UBP - AI Usecase: usage in period (103,000,025 x $0.000001)"),
				Period: &stripe.InvoiceAddLinesLinePeriodParams{
					Start: lo.ToPtr(expectedPeriodStart.Unix()),
					End:   lo.ToPtr(expectedPeriodEnd.Unix()),
				},
				Metadata: map[string]string{
					"om_line_id":   getLineID("UBP - AI Usecase: usage in period"),
					"om_line_type": "line",
				},
			},
			{
				Amount:      lo.ToPtr(int64(10000)),
				Description: lo.ToPtr("UBP - FLAT per any usage"),
				Period: &stripe.InvoiceAddLinesLinePeriodParams{
					Start: lo.ToPtr(expectedPeriodStart.Unix()),
					End:   lo.ToPtr(expectedPeriodEnd.Unix()),
				},
				Metadata: map[string]string{
					"om_line_id":   getLineID("UBP - FLAT per any usage"),
					"om_line_type": "line",
				},
			},
			{
				Amount:      lo.ToPtr(int64(322000)),
				Description: lo.ToPtr("UBP - FLAT per unit: usage in period (32.20 x $100)"),
				Period: &stripe.InvoiceAddLinesLinePeriodParams{
					Start: lo.ToPtr(expectedPeriodStart.Unix()),
					End:   lo.ToPtr(expectedPeriodEnd.Unix()),
				},
				Metadata: map[string]string{
					"om_line_id":   getLineID("UBP - FLAT per unit: usage in period"),
					"om_line_type": "line",
				},
			},
			{
				Amount:      lo.ToPtr(int64(-122000)),
				Description: lo.ToPtr("UBP - FLAT per unit: usage in period (Maximum spend discount for charges over 2000)"),
				Period: &stripe.InvoiceAddLinesLinePeriodParams{
					Start: lo.ToPtr(expectedPeriodStart.Unix()),
					End:   lo.ToPtr(expectedPeriodEnd.Unix()),
				},
				Metadata: map[string]string{
					"om_line_id":   getDiscountID("UBP - FLAT per unit: usage in period (Maximum spend discount for charges over 2000)"),
					"om_line_type": "discount",
				},
			},
			{
				Amount:      lo.ToPtr(int64(95000)),
				Description: lo.ToPtr("UBP - Tiered graduated: usage price for tier 1 (9.50 x $100)"),
				Period: &stripe.InvoiceAddLinesLinePeriodParams{
					Start: lo.ToPtr(expectedPeriodStart.Unix()),
					End:   lo.ToPtr(expectedPeriodEnd.Unix()),
				},
				Metadata: map[string]string{
					"om_line_id":   getLineID("UBP - Tiered graduated: usage price for tier 1"),
					"om_line_type": "line",
				},
			},
			{
				Amount:      lo.ToPtr(int64(94500)),
				Description: lo.ToPtr("UBP - Tiered graduated: usage price for tier 2 (10.50 x $90)"),
				Period: &stripe.InvoiceAddLinesLinePeriodParams{
					Start: lo.ToPtr(expectedPeriodStart.Unix()),
					End:   lo.ToPtr(expectedPeriodEnd.Unix()),
				},
				Metadata: map[string]string{
					"om_line_id":   getLineID("UBP - Tiered graduated: usage price for tier 2"),
					"om_line_type": "line",
				},
			},
			{
				Amount:      lo.ToPtr(int64(122400)),
				Description: lo.ToPtr("UBP - Tiered graduated: usage price for tier 3 (15.30 x $80)"),
				Period: &stripe.InvoiceAddLinesLinePeriodParams{
					Start: lo.ToPtr(expectedPeriodStart.Unix()),
					End:   lo.ToPtr(expectedPeriodEnd.Unix()),
				},
				Metadata: map[string]string{
					"om_line_id":   getLineID("UBP - Tiered graduated: usage price for tier 3"),
					"om_line_type": "line",
				},
			},
			{
				Amount:      lo.ToPtr(int64(162300)),
				Description: lo.ToPtr("UBP - Tiered volume: minimum spend"),
				Period: &stripe.InvoiceAddLinesLinePeriodParams{
					// TODO: check rounding
					Start: lo.ToPtr(periodStart.Truncate(time.Minute).Unix()),
					End:   lo.ToPtr(periodEnd.Truncate(time.Minute).Unix()),
				},
				Metadata: map[string]string{
					"om_line_id":   getLineID("UBP - Tiered volume: minimum spend"),
					"om_line_type": "line",
				},
			},
			{
				Amount:      lo.ToPtr(int64(137700)),
				Description: lo.ToPtr("UBP - Tiered volume: unit price for tier 2 (15.30 x $90)"),
				Period: &stripe.InvoiceAddLinesLinePeriodParams{
					// TODO: check rounding
					Start: lo.ToPtr(periodStart.Truncate(time.Minute).Unix()),
					End:   lo.ToPtr(expectedPeriodEnd.Unix()),
				},
				Metadata: map[string]string{
					"om_line_id":   getLineID("UBP - Tiered volume: unit price for tier 2"),
					"om_line_type": "line",
				},
			},
		}

		stripeInvoice := &stripe.Invoice{
			ID: "stripe-invoice-id",
			Customer: &stripe.Customer{
				ID: customerData.StripeCustomerID,
			},
			Currency: "USD",
			Lines: &stripe.InvoiceLineItemList{
				Data: lo.Map(expectedInvoiceAddLines, func(line *stripe.InvoiceAddLinesLineParams, idx int) *stripe.InvoiceLineItem {
					return &stripe.InvoiceLineItem{
						ID:          fmt.Sprintf("il_%d", idx),
						Amount:      *line.Amount,
						Description: *line.Description,
						Period: &stripe.Period{
							Start: *line.Period.Start,
							End:   *line.Period.End,
						},
						Metadata: line.Metadata,
					}
				}),
			},
			StatementDescriptor: invoice.Supplier.Name,
		}

		s.StripeAppClient.
			On("AddInvoiceLines", stripeclient.AddInvoiceLinesInput{
				StripeInvoiceID: "stripe-invoice-id",
				Lines:           expectedInvoiceAddLines,
			}).
			Return(stripeInvoice, nil)

		// Create the invoice.
		results, err := invoicingApp.UpsertInvoice(ctx, invoice)
		s.NoError(err, "failed to upsert invoice")

		// Assert external ID is set.
		externalId, ok := results.GetExternalID()
		s.True(ok, "external ID is not set")
		s.Equal("stripe-invoice-id", externalId)

		// Assert results.
		// TODO: discount line items are not in the results
		s.Len(results.GetLineExternalIDs(), len(expectedInvoiceAddLines)-1)

		expectedResult := map[string]string{}

		for _, stripeLine := range stripeInvoice.Lines.Data {
			// TODO: currently we don't have a way to match Stripe discount line items
			if stripeLine.Metadata["om_line_type"] == "discount" {
				continue
			}

			expectedResult[stripeLine.Metadata["om_line_id"]] = stripeLine.ID
		}

		s.Equal(expectedResult, results.GetLineExternalIDs())

		// Update the invoice.

		updateInvoice := invoice.Clone()

		// We merge external IDs into the invoice manually to simulate the update.
		// Normally this is done by the state machine.
		err = billing.MergeUpsertInvoiceResult(&updateInvoice, results)
		s.NoError(err)

		// Remove a line item.
		lineToRemove := getLine("Fee")
		s.NotNil(lineToRemove, "line ID to remove is not found")

		// Find the stripe line ID to remove.
		var stripeLineIDToRemove string

		for lineID, stripeLineID := range expectedResult {
			if lineID == lineToRemove.ID {
				stripeLineIDToRemove = stripeLineID
			}
		}

		delete(expectedResult, lineToRemove.ID)

		s.NotEmpty(stripeLineIDToRemove, "stripe line ID to remove is empty")

		ok = updateInvoice.Lines.RemoveByID(lineToRemove.ID)
		s.True(ok, "failed to remove line item")

		// To simulate the update, we will update the external ID of the invoice.
		// Which will go into update path of the upsert invoice.
		updateInvoice.ExternalIDs.Invoicing = "stripe-invoice-id"

		stripeInvoiceUpdated := &stripe.Invoice{
			ID:       stripeInvoice.ID,
			Customer: stripeInvoice.Customer,
			Currency: stripeInvoice.Currency,
			Lines: &stripe.InvoiceLineItemList{
				Data: lo.Filter(stripeInvoice.Lines.Data, func(line *stripe.InvoiceLineItem, _ int) bool {
					return line.ID != lineToRemove.ID
				}),
			},
			StatementDescriptor: invoice.Supplier.Name,
		}

		s.StripeAppClient.
			On("UpdateInvoice", stripeclient.UpdateInvoiceInput{
				StripeInvoiceID: updateInvoice.ExternalIDs.Invoicing,
			}).
			// We return the updated invoice.
			Return(stripeInvoiceUpdated, nil)

		// Mocks to fulfill add, update and remove invoice lines:
		// From existing lines, one is removed and the rest are updated.

		s.StripeAppClient.
			On("UpdateInvoiceLines", stripeclient.UpdateInvoiceLinesInput{
				StripeInvoiceID: updateInvoice.ExternalIDs.Invoicing,
				Lines: lo.FilterMap(stripeInvoice.Lines.Data, func(line *stripe.InvoiceLineItem, idx int) (*stripe.InvoiceUpdateLinesLineParams, bool) {
					// No changes to the line items.
					return &stripe.InvoiceUpdateLinesLineParams{
						ID:          &line.ID,
						Amount:      &line.Amount,
						Description: &line.Description,
						Period: &stripe.InvoiceUpdateLinesLinePeriodParams{
							Start: &line.Period.Start,
							End:   &line.Period.End,
						},
						Metadata: line.Metadata,
					}, line.ID != stripeLineIDToRemove
				}),
			}).
			Return(stripeInvoice, nil)

		s.StripeAppClient.
			On("RemoveInvoiceLines", stripeclient.RemoveInvoiceLinesInput{
				StripeInvoiceID: updateInvoice.ExternalIDs.Invoicing,
				Lines: []*stripe.InvoiceRemoveLinesLineParams{
					{
						ID:       lo.ToPtr(stripeLineIDToRemove),
						Behavior: lo.ToPtr("delete"),
					},
				},
			}).
			Return(stripeInvoice, nil)

		// TODO: do not share env between tests
		defer s.StripeAppClient.Restore()

		// Update the invoice.
		results, err = invoicingApp.UpsertInvoice(ctx, updateInvoice)
		s.NoError(err, "failed to upsert invoice")

		// Assert results.
		s.Equal(expectedResult, results.GetLineExternalIDs())

		// Assert invoice is created in stripe.
		s.StripeAppClient.AssertExpectations(s.T())
	})
}
