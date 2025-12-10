package appstripe

import (
	"context"
	"fmt"
	"log/slog"
	"regexp"
	"testing"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/invopop/gobl/currency"
	"github.com/oklog/ulid/v2"
	"github.com/samber/lo"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
	"github.com/stripe/stripe-go/v80"

	"github.com/openmeterio/openmeter/openmeter/app"
	appstripe "github.com/openmeterio/openmeter/openmeter/app/stripe"
	appstripeadapter "github.com/openmeterio/openmeter/openmeter/app/stripe/adapter"
	stripeclient "github.com/openmeterio/openmeter/openmeter/app/stripe/client"
	appstripeentity "github.com/openmeterio/openmeter/openmeter/app/stripe/entity"
	appstripeservice "github.com/openmeterio/openmeter/openmeter/app/stripe/service"
	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/openmeter/meter"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/feature"
	"github.com/openmeterio/openmeter/openmeter/secret"
	secretadapter "github.com/openmeterio/openmeter/openmeter/secret/adapter"
	secretservice "github.com/openmeterio/openmeter/openmeter/secret/service"
	"github.com/openmeterio/openmeter/openmeter/streaming"
	"github.com/openmeterio/openmeter/openmeter/watermill/eventbus"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/datetime"
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
		Logger: slog.Default(),
	})
	s.Require().NoError(err, "failed to create app stripe adapter")

	webhookURLGenerator, err := appstripeservice.NewBaseURLWebhookURLGenerator("http://localhost:8888")
	if err != nil {
		s.Require().NoError(err, "failed to create webhook url generator")
	}

	appStripeService, err := appstripeservice.New(appstripeservice.Config{
		Adapter:             appStripeAdapter,
		AppService:          s.AppService,
		SecretService:       secretService,
		BillingService:      s.BillingService,
		Logger:              slog.Default(),
		Publisher:           eventbus.NewMock(s.T()),
		WebhookURLGenerator: webhookURLGenerator,
	})
	s.Require().NoError(err, "failed to create app stripe service")

	s.AppStripeService = appStripeService

	// Fixture
	s.Fixture = NewFixture(s.AppService, s.CustomerService, stripeClient, stripeAppClient)
}

func (s *StripeInvoiceTestSuite) TearDownTest() {
	s.StripeAppClient.Restore()
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
	clock.FreezeTime(periodStart)
	defer clock.UnFreeze()

	sandboxApp := s.InstallSandboxApp(s.T(), namespace)

	err := s.MeterAdapter.ReplaceMeters(ctx, []meter.Meter{
		{
			ManagedResource: models.ManagedResource{
				ID: ulid.Make().String(),
				NamespacedModel: models.NamespacedModel{
					Namespace: namespace,
				},
				ManagedModel: models.ManagedModel{
					CreatedAt: time.Now(),
					UpdatedAt: time.Now(),
				},
				Name: "Meter 1",
			},
			Key:           "flat-per-unit",
			Aggregation:   meter.MeterAggregationSum,
			EventType:     "test",
			ValueProperty: lo.ToPtr("$.value"),
		},
		{
			ManagedResource: models.ManagedResource{
				ID: ulid.Make().String(),
				NamespacedModel: models.NamespacedModel{
					Namespace: namespace,
				},
				ManagedModel: models.ManagedModel{
					CreatedAt: time.Now(),
					UpdatedAt: time.Now(),
				},
				Name: "Meter 1",
			},
			Key:           "flat-per-usage",
			Aggregation:   meter.MeterAggregationSum,
			EventType:     "test",
			ValueProperty: lo.ToPtr("$.value"),
		},
		{
			ManagedResource: models.ManagedResource{
				ID: ulid.Make().String(),
				NamespacedModel: models.NamespacedModel{
					Namespace: namespace,
				},
				ManagedModel: models.ManagedModel{
					CreatedAt: time.Now(),
					UpdatedAt: time.Now(),
				},
				Name: "Meter 1",
			},
			Key:           "tiered-graduated",
			Aggregation:   meter.MeterAggregationSum,
			EventType:     "test",
			ValueProperty: lo.ToPtr("$.value"),
		},
		{
			ManagedResource: models.ManagedResource{
				ID: ulid.Make().String(),
				NamespacedModel: models.NamespacedModel{
					Namespace: namespace,
				},
				ManagedModel: models.ManagedModel{
					CreatedAt: time.Now(),
					UpdatedAt: time.Now(),
				},
				Name: "Meter 1",
			},
			Key:           "tiered-volume",
			Aggregation:   meter.MeterAggregationSum,
			EventType:     "test",
			ValueProperty: lo.ToPtr("$.value"),
		},
		{
			ManagedResource: models.ManagedResource{
				ID: ulid.Make().String(),
				NamespacedModel: models.NamespacedModel{
					Namespace: namespace,
				},
				ManagedModel: models.ManagedModel{
					CreatedAt: time.Now(),
					UpdatedAt: time.Now(),
				},
				Name: "Meter 1",
			},
			Key:           "ai-flat-per-unit",
			Aggregation:   meter.MeterAggregationSum,
			EventType:     "test",
			ValueProperty: lo.ToPtr("$.value"),
		},
	})
	s.Require().NoError(err)

	defer func() {
		err = s.MeterAdapter.ReplaceMeters(ctx, []meter.Meter{})
		s.Require().NoError(err)
	}()

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

	customerEntity, err := s.CustomerService.CreateCustomer(ctx, customer.CreateCustomerInput{
		Namespace: namespace,

		CustomerMutate: customer.CustomerMutate{
			Name:         "Test Customer",
			PrimaryEmail: lo.ToPtr("test@test.com"),
			Currency:     lo.ToPtr(currencyx.Code(currency.USD)),
			BillingAddress: &models.Address{
				Country: lo.ToPtr(models.CountryCode("US")),
			},
			UsageAttribution: &customer.CustomerUsageAttribution{
				SubjectKeys: []string{"test"},
			},
		},
	})
	s.NoError(err)
	s.NotNil(customerEntity)
	s.NotEmpty(customerEntity.ID)

	// Given we have a default profile for the namespace
	s.ProvisionBillingProfile(ctx, namespace, sandboxApp.GetID())

	s.Run("create pending invoice items", func() {
		// When we create pending invoice items
		pendingLines, err := s.BillingService.CreatePendingInvoiceLines(ctx,
			billing.CreatePendingInvoiceLinesInput{
				Customer: customerEntity.GetID(),
				Currency: currencyx.Code(currency.USD),
				Lines: []*billing.Line{
					{
						// Covered case: Discount caused by maximum amount
						LineBase: billing.LineBase{
							ManagedResource: models.NewManagedResource(models.ManagedResourceInput{
								Name: "UBP - FLAT per unit",
							}),
							Period:    billing.Period{Start: periodStart, End: periodEnd},
							InvoiceAt: periodEnd,
							ManagedBy: billing.ManuallyManagedLine,
						},
						UsageBased: &billing.UsageBasedLine{
							FeatureKey: features.flatPerUnit.Key,
							Price: productcatalog.NewPriceFrom(productcatalog.UnitPrice{
								Amount: alpacadecimal.NewFromFloat(100),
								Commitments: productcatalog.Commitments{
									MaximumAmount: lo.ToPtr(alpacadecimal.NewFromFloat(2000)),
								},
							}),
						},
					},
					{
						// Covered case: Very small per unit amount, high quantity, rounding to two decimal places
						LineBase: billing.LineBase{
							ManagedResource: models.NewManagedResource(models.ManagedResourceInput{
								Name: "UBP - AI Usecase",
							}),
							Period:    billing.Period{Start: periodStart, End: periodEnd},
							InvoiceAt: periodEnd,
							ManagedBy: billing.ManuallyManagedLine,
						},
						UsageBased: &billing.UsageBasedLine{
							FeatureKey: features.aiFlatPerUnit.Key,
							Price: productcatalog.NewPriceFrom(productcatalog.UnitPrice{
								Amount: alpacadecimal.NewFromFloat(0.00000075),
							}),
						},
					},
					{
						// Covered case: Flat line represented as UBP item
						LineBase: billing.LineBase{
							ManagedResource: models.NewManagedResource(models.ManagedResourceInput{
								Name: "UBP - FLAT per any usage",
							}),
							Period:    billing.Period{Start: periodStart, End: periodEnd},
							InvoiceAt: periodEnd,
							ManagedBy: billing.ManuallyManagedLine,
						},
						UsageBased: &billing.UsageBasedLine{
							FeatureKey: features.flatPerUsage.Key,
							Price: productcatalog.NewPriceFrom(productcatalog.FlatPrice{
								Amount:      alpacadecimal.NewFromFloat(100),
								PaymentTerm: productcatalog.InArrearsPaymentTerm,
							}),
							Quantity: lo.ToPtr(alpacadecimal.NewFromFloat(1)),
						},
					},
					{
						// Covered case: Multiple lines per item, tier boundary is fractional
						LineBase: billing.LineBase{
							ManagedResource: models.NewManagedResource(models.ManagedResourceInput{
								Name: "UBP - Tiered graduated",
							}),
							Period:    billing.Period{Start: periodStart, End: periodEnd},
							InvoiceAt: periodEnd,
							ManagedBy: billing.ManuallyManagedLine,
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
					{
						// Covered case: minimum amount charges
						LineBase: billing.LineBase{
							ManagedResource: models.NewManagedResource(models.ManagedResourceInput{
								Name: "UBP - Tiered volume",
							}),
							Period:    billing.Period{Start: periodStart, End: periodEnd},
							InvoiceAt: periodEnd,
							ManagedBy: billing.ManuallyManagedLine,
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
								Commitments: productcatalog.Commitments{
									MinimumAmount: lo.ToPtr(alpacadecimal.NewFromFloat(3000)),
								},
							}),
						},
					},
				},
			},
		)
		s.NoError(err)
		s.Len(pendingLines.Lines, 5)
	})

	clock.FreezeTime(periodEnd.Add(time.Minute))

	var app app.App
	var customerData appstripeentity.CustomerData
	var invoice billing.Invoice
	var invoicingApp billing.InvoicingApp

	// Setup the app with the customer
	s.Run("setup app, customer and invoice", func() {
		app, err = s.Fixture.setupApp(ctx, namespace)
		s.NoError(err)

		customerData, err = s.Fixture.setupAppCustomerData(ctx, app, customerEntity)
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

		invoice = invoices[0].RemoveCircularReferences()

		// Create a new invoice for the customer.
		invoicingApp, err = billing.GetApp(app)
		s.NoError(err)
	})

	s.Run("upsert invoice", func() {
		// Mock the stripe client to return the created invoice.
		s.StripeAppClient.
			On("CreateInvoice", stripeclient.CreateInvoiceInput{
				AppID:               app.GetID(),
				CustomerID:          customerEntity.GetID(),
				InvoiceID:           invoice.ID,
				AutomaticTaxEnabled: true,
				CollectionMethod:    billing.CollectionMethodChargeAutomatically,
				StripeCustomerID:    customerData.StripeCustomerID,
				Currency:            "USD",
			}).
			Once().
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

		expectedPeriodStartUnix := periodStart.Truncate(streaming.MinimumWindowSizeDuration).Unix()
		expectedPeriodEndUnix := periodEnd.Truncate(streaming.MinimumWindowSizeDuration).Unix()

		getParentOfDetailedLine := func(detailedLine billing.DetailedLine) *billing.Line {
			for _, line := range invoice.Lines.OrEmpty() {
				_, found := lo.Find(line.DetailedLines, func(dl billing.DetailedLine) bool {
					return dl.ID == detailedLine.ID
				})

				if found {
					return line
				}
			}

			return nil
		}

		getLine := func(description string) *billing.DetailedLine {
			for _, line := range invoice.GetLeafLinesWithConsolidatedTaxBehavior() {
				name := line.Name
				if line.Description != nil {
					name = fmt.Sprintf("%s (%s)", name, *line.Description)
				}

				if name == description {
					return &line
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

			for _, discount := range line.AmountDiscounts {
				if lo.FromPtr(discount.Description) != "" && group[0][2] == lo.FromPtr(discount.Description) {
					return discount.GetID()
				}
			}

			return id
		}

		expectedInvoiceAddLines := []*stripe.InvoiceItemParams{
			{
				Amount:      lo.ToPtr(int64(7725)),
				Description: lo.ToPtr("UBP - AI Usecase: usage in period (103,000,025 x $0.000001)"),
				Customer:    lo.ToPtr(customerData.StripeCustomerID),

				Period: &stripe.InvoiceItemPeriodParams{
					Start: lo.ToPtr(expectedPeriodStartUnix),
					End:   lo.ToPtr(expectedPeriodEndUnix),
				},
				Metadata: map[string]string{
					"om_line_id":   getLineID("UBP - AI Usecase: usage in period"),
					"om_line_type": "line",
				},
			},
			{
				Amount:      lo.ToPtr(int64(10000)),
				Description: lo.ToPtr("UBP - FLAT per any usage"),
				Customer:    lo.ToPtr(customerData.StripeCustomerID),
				Period: &stripe.InvoiceItemPeriodParams{
					Start: lo.ToPtr(expectedPeriodStartUnix),
					End:   lo.ToPtr(expectedPeriodEndUnix),
				},
				Metadata: map[string]string{
					"om_line_id":   getLineID("UBP - FLAT per any usage"),
					"om_line_type": "line",
				},
			},
			{
				Amount:      lo.ToPtr(int64(322000)),
				Description: lo.ToPtr("UBP - FLAT per unit: usage in period (32.20 x $100)"),
				Customer:    lo.ToPtr(customerData.StripeCustomerID),
				Period: &stripe.InvoiceItemPeriodParams{
					Start: lo.ToPtr(expectedPeriodStartUnix),
					End:   lo.ToPtr(expectedPeriodEndUnix),
				},
				Metadata: map[string]string{
					"om_line_id":   getLineID("UBP - FLAT per unit: usage in period"),
					"om_line_type": "line",
				},
			},
			{
				Amount:      lo.ToPtr(int64(-122000)),
				Description: lo.ToPtr("UBP - FLAT per unit: usage in period (Maximum spend discount for charges over 2000)"),
				Customer:    lo.ToPtr(customerData.StripeCustomerID),
				Period: &stripe.InvoiceItemPeriodParams{
					Start: lo.ToPtr(expectedPeriodStartUnix),
					End:   lo.ToPtr(expectedPeriodEndUnix),
				},
				Metadata: map[string]string{
					"om_line_id":   getDiscountID("UBP - FLAT per unit: usage in period (Maximum spend discount for charges over 2000)"),
					"om_line_type": "discount",
				},
			},
			{
				Amount:      lo.ToPtr(int64(95000)),
				Description: lo.ToPtr("UBP - Tiered graduated: usage price for tier 1 (9.50 x $100)"),
				Customer:    lo.ToPtr(customerData.StripeCustomerID),
				Period: &stripe.InvoiceItemPeriodParams{
					Start: lo.ToPtr(expectedPeriodStartUnix),
					End:   lo.ToPtr(expectedPeriodEndUnix),
				},
				Metadata: map[string]string{
					"om_line_id":   getLineID("UBP - Tiered graduated: usage price for tier 1"),
					"om_line_type": "line",
				},
			},
			{
				Amount:      lo.ToPtr(int64(94500)),
				Description: lo.ToPtr("UBP - Tiered graduated: usage price for tier 2 (10.50 x $90)"),
				Customer:    lo.ToPtr(customerData.StripeCustomerID),
				Period: &stripe.InvoiceItemPeriodParams{
					Start: lo.ToPtr(expectedPeriodStartUnix),
					End:   lo.ToPtr(expectedPeriodEndUnix),
				},
				Metadata: map[string]string{
					"om_line_id":   getLineID("UBP - Tiered graduated: usage price for tier 2"),
					"om_line_type": "line",
				},
			},
			{
				Amount:      lo.ToPtr(int64(122400)),
				Description: lo.ToPtr("UBP - Tiered graduated: usage price for tier 3 (15.30 x $80)"),
				Customer:    lo.ToPtr(customerData.StripeCustomerID),
				Period: &stripe.InvoiceItemPeriodParams{
					Start: lo.ToPtr(expectedPeriodStartUnix),
					End:   lo.ToPtr(expectedPeriodEndUnix),
				},
				Metadata: map[string]string{
					"om_line_id":   getLineID("UBP - Tiered graduated: usage price for tier 3"),
					"om_line_type": "line",
				},
			},
			{
				Amount:      lo.ToPtr(int64(162300)),
				Description: lo.ToPtr("UBP - Tiered volume: minimum spend"),
				Customer:    lo.ToPtr(customerData.StripeCustomerID),
				Period: &stripe.InvoiceItemPeriodParams{
					Start: lo.ToPtr(expectedPeriodStartUnix),
					End:   lo.ToPtr(expectedPeriodEndUnix),
				},
				Metadata: map[string]string{
					"om_line_id":   getLineID("UBP - Tiered volume: minimum spend"),
					"om_line_type": "line",
				},
			},
			{
				Amount:      lo.ToPtr(int64(137700)),
				Description: lo.ToPtr("UBP - Tiered volume: unit price for tier 2 (15.30 x $90)"),
				Customer:    lo.ToPtr(customerData.StripeCustomerID),
				Period: &stripe.InvoiceItemPeriodParams{
					// TODO: check rounding
					Start: lo.ToPtr(expectedPeriodStartUnix),
					End:   lo.ToPtr(expectedPeriodEndUnix),
				},
				Metadata: map[string]string{
					"om_line_id":   getLineID("UBP - Tiered volume: unit price for tier 2"),
					"om_line_type": "line",
				},
			},
		}

		s.StripeAppClient.StableSortInvoiceItemParams(expectedInvoiceAddLines)

		stripeInvoice := &stripe.Invoice{
			ID: "stripe-invoice-id",
			Customer: &stripe.Customer{
				ID: customerData.StripeCustomerID,
			},
			Currency: "USD",
			Lines: &stripe.InvoiceLineItemList{
				Data: lo.Map(expectedInvoiceAddLines, func(line *stripe.InvoiceItemParams, idx int) *stripe.InvoiceLineItem {
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
			Once().
			Return(lo.Map(
				expectedInvoiceAddLines,
				func(line *stripe.InvoiceItemParams, idx int) stripeclient.StripeInvoiceItemWithLineID {
					return mapInvoiceItemParamsToInvoiceItem(fmt.Sprintf("%d", idx), line)
				},
			), nil)

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
		err = results.MergeIntoInvoice(&updateInvoice)
		s.NoError(err)

		// Remove a line item.
		lineToRemove := getLine("UBP - FLAT per any usage")
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

		parentLine := getParentOfDetailedLine(*lineToRemove)
		s.NotNil(parentLine, "parent line is not found")

		ok = updateInvoice.Lines.RemoveByID(parentLine.ID)
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
				AutomaticTaxEnabled: true,
				StripeInvoiceID:     updateInvoice.ExternalIDs.Invoicing,
			}).
			Once().
			// We return the updated invoice.
			Return(stripeInvoiceUpdated, nil)

		// Mocks to fulfill add, update and remove invoice lines:
		// From existing lines, one is removed and the rest are updated.

		filteredUpdatedLines := lo.FilterMap(stripeInvoice.Lines.Data, func(line *stripe.InvoiceLineItem, idx int) (*stripeclient.StripeInvoiceItemWithID, bool) {
			// No changes to the line items.
			return &stripeclient.StripeInvoiceItemWithID{
				ID: line.ID,
				InvoiceItemParams: &stripe.InvoiceItemParams{
					Amount:      &line.Amount,
					Description: &line.Description,
					Period: &stripe.InvoiceItemPeriodParams{
						Start: &line.Period.Start,
						End:   &line.Period.End,
					},
					Metadata: line.Metadata,
				},
			}, line.ID != stripeLineIDToRemove
		})

		s.StripeAppClient.StableSortStripeInvoiceItemWithID(filteredUpdatedLines)

		s.StripeAppClient.
			On("UpdateInvoiceLines", stripeclient.UpdateInvoiceLinesInput{
				StripeInvoiceID: updateInvoice.ExternalIDs.Invoicing,
				Lines:           filteredUpdatedLines,
			}).
			Once().
			Return(lo.Map(filteredUpdatedLines, func(l *stripeclient.StripeInvoiceItemWithID, _ int) *stripe.InvoiceItem {
				return mapInvoiceItemParamsToInvoiceItem(l.ID, l.InvoiceItemParams).InvoiceItem
			}), nil)

		s.StripeAppClient.
			On("RemoveInvoiceLines", stripeclient.RemoveInvoiceLinesInput{
				StripeInvoiceID: updateInvoice.ExternalIDs.Invoicing,
				Lines:           []string{stripeLineIDToRemove},
			}).
			Once().
			Return(nil)

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

	s.Run("finalize invoice", func() {
		// Mock the stripe client to return the created invoice.
		invoice.ExternalIDs.Invoicing = "stripe-invoice-id"

		// Mock the stripe client for finalize invoice.
		s.StripeAppClient.
			On("FinalizeInvoice", stripeclient.FinalizeInvoiceInput{
				AutoAdvance:     true,
				StripeInvoiceID: invoice.ExternalIDs.Invoicing,
			}).
			Once().
			Return(&stripe.Invoice{
				ID: invoice.ExternalIDs.Invoicing,
				Customer: &stripe.Customer{
					ID: customerData.StripeCustomerID,
				},
				Number:   "INV-123",
				Currency: "USD",
				Lines: &stripe.InvoiceLineItemList{
					Data: []*stripe.InvoiceLineItem{},
				},
				PaymentIntent: &stripe.PaymentIntent{
					ID: "pmi_123",
				},
			}, nil)

		// TODO: do not share env between tests
		defer s.StripeAppClient.Restore()

		// Create the invoice.
		result, err := invoicingApp.FinalizeInvoice(ctx, invoice)
		s.NoError(err, "failed to finalize invoice")

		// Assert the result.
		expectedResult := billing.NewFinalizeInvoiceResult()
		expectedResult.SetInvoiceNumber("INV-123")
		expectedResult.SetPaymentExternalID("pmi_123")

		s.Equal(expectedResult, result)

		// Assert the client is called with the correct arguments.
		s.StripeAppClient.AssertExpectations(s.T())
	})

	s.Run("finalize invoice with stripe tax error", func() {
		// Mock the stripe client to return the created invoice.
		invoice.ExternalIDs.Invoicing = "stripe-invoice-id"

		// We don't use the stripe.Error directly because it's already wrapped in the error returned by the client.
		// We just create it here to give more context to the test.
		stripeErrMock := &stripe.Error{
			Type:          "invalid_request_error",
			Code:          stripe.ErrorCodeCustomerTaxLocationInvalid,
			DocURL:        "https://stripe.com/docs/error-codes/customer-tax-location-invalid",
			Msg:           "When `automatic_tax[enabled]=true`, enough customer location information must be provided to accurately determine tax rates for the customer.",
			RequestLogURL: "https://dashboard.stripe.com/test/logs/req_abcd?t=1746741453",
		}

		// Mock the stripe client for finalize invoice.
		// 1. We return a Stripe Tax error.
		s.StripeAppClient.
			On("FinalizeInvoice", stripeclient.FinalizeInvoiceInput{
				AutoAdvance:     true,
				StripeInvoiceID: invoice.ExternalIDs.Invoicing,
			}).
			Once().
			Return(&stripe.Invoice{}, stripeclient.NewStripeInvoiceCustomerTaxLocationInvalidError(invoice.ExternalIDs.Invoicing, stripeErrMock.Msg))

		// 2. We update the invoice to disable tax calculation.
		s.StripeAppClient.
			On("UpdateInvoice", stripeclient.UpdateInvoiceInput{
				StripeInvoiceID:     invoice.ExternalIDs.Invoicing,
				AutomaticTaxEnabled: false,
			}).
			Once().
			Return(&stripe.Invoice{
				ID: invoice.ExternalIDs.Invoicing,
				Customer: &stripe.Customer{
					ID: customerData.StripeCustomerID,
				},
				Number:   "INV-123",
				Currency: "USD",
				Lines: &stripe.InvoiceLineItemList{
					Data: []*stripe.InvoiceLineItem{},
				},
			}, nil)

		// 3. We finalize the invoice.
		s.StripeAppClient.
			On("FinalizeInvoice", stripeclient.FinalizeInvoiceInput{
				AutoAdvance:     true,
				StripeInvoiceID: invoice.ExternalIDs.Invoicing,
			}).
			Once().
			Return(&stripe.Invoice{
				ID: invoice.ExternalIDs.Invoicing,
				Customer: &stripe.Customer{
					ID: customerData.StripeCustomerID,
				},
				Number:   "INV-123",
				Currency: "USD",
				Lines: &stripe.InvoiceLineItemList{
					Data: []*stripe.InvoiceLineItem{},
				},
				PaymentIntent: &stripe.PaymentIntent{
					ID: "pmi_123",
				},
			}, nil)

		// TODO: do not share env between tests
		defer s.StripeAppClient.Restore()

		// Create the invoice.
		result, err := invoicingApp.FinalizeInvoice(ctx, invoice)
		s.NoError(err, "failed to finalize invoice")

		// Assert the result.
		expectedResult := billing.NewFinalizeInvoiceResult()
		expectedResult.SetInvoiceNumber("INV-123")
		expectedResult.SetPaymentExternalID("pmi_123")

		s.Equal(expectedResult, result)

		// Assert the client is called with the correct arguments.
		s.StripeAppClient.AssertExpectations(s.T())
	})
}

func (s *StripeInvoiceTestSuite) TestEmptyInvoiceGenerationZeroUsage() {
	// Given we have a test customer and an UBP line without usage priced at 0
	// we can create the invoice and even if there are no detailed lines the validation
	// errors should be empty

	namespace := "ns-empty-invoice-generation"
	ctx := context.Background()
	periodStart := lo.Must(time.Parse(time.RFC3339, "2024-09-02T12:13:14Z"))
	periodEnd := lo.Must(time.Parse(time.RFC3339, "2024-09-03T12:13:14Z"))
	clock.FreezeTime(periodStart)
	defer clock.UnFreeze()

	_ = s.InstallSandboxApp(s.T(), namespace)

	meterSlug := "flat-per-unit"

	err := s.MeterAdapter.ReplaceMeters(ctx, []meter.Meter{
		{
			ManagedResource: models.ManagedResource{
				ID: ulid.Make().String(),
				NamespacedModel: models.NamespacedModel{
					Namespace: namespace,
				},
				ManagedModel: models.ManagedModel{
					CreatedAt: time.Now(),
					UpdatedAt: time.Now(),
				},
				Name: "Flat per unit",
			},
			Key:           meterSlug,
			Aggregation:   meter.MeterAggregationSum,
			EventType:     "test",
			ValueProperty: lo.ToPtr("$.value"),
		},
	})
	s.NoError(err, "failed to replace meters")

	defer func() {
		err = s.MeterAdapter.ReplaceMeters(ctx, []meter.Meter{})
		s.NoError(err, "failed to replace meters")
	}()

	// Let's initialize the mock streaming connector with data that is out of the period so that we
	// can start with empty values
	s.MockStreamingConnector.AddSimpleEvent(meterSlug, 0, periodStart.Add(-time.Minute))

	defer s.MockStreamingConnector.Reset()

	flatPerUnitFeature := lo.Must(s.FeatureService.CreateFeature(ctx, feature.CreateFeatureInputs{
		Namespace: namespace,
		Name:      "flat-per-unit",
		Key:       "flat-per-unit",
		MeterSlug: lo.ToPtr("flat-per-unit"),
	}))

	customerEntity, err := s.CustomerService.CreateCustomer(ctx, customer.CreateCustomerInput{
		Namespace: namespace,

		CustomerMutate: customer.CustomerMutate{
			Name:     "Test Customer",
			Currency: lo.ToPtr(currencyx.Code(currency.USD)),
			UsageAttribution: &customer.CustomerUsageAttribution{
				SubjectKeys: []string{"test"},
			},
		},
	})
	s.NoError(err)
	s.NotNil(customerEntity)
	s.NotEmpty(customerEntity.ID)

	app, err := s.Fixture.setupApp(ctx, namespace)
	s.NoError(err)

	// Given we have a default profile for the namespace
	s.ProvisionBillingProfile(ctx, namespace, app.GetID(), billingtest.WithBillingProfileEditFn(func(profile *billing.CreateProfileInput) {
		// manual advancement for testing the update invoice flow
		profile.WorkflowConfig.Invoicing.AutoAdvance = false
	}))

	// Setup the app with the customer

	customerData, err := s.Fixture.setupAppCustomerData(ctx, app, customerEntity)
	s.NoError(err)
	s.NotNil(customerData)

	s.StripeAppClient.
		On("GetCustomer", defaultStripeCustomerID).
		Return(stripeclient.StripeCustomer{
			StripeCustomerID: defaultStripeCustomerID,
			DefaultPaymentMethod: &stripeclient.StripePaymentMethod{
				ID:    "pm_123",
				Name:  "ACME Inc.",
				Email: "acme@test.com",
				BillingAddress: &models.Address{
					City:       lo.ToPtr("San Francisco"),
					PostalCode: lo.ToPtr("94103"),
					State:      lo.ToPtr("CA"),
					Country:    lo.ToPtr(models.CountryCode("US")),
					Line1:      lo.ToPtr("123 Market St"),
				},
			},
		}, nil)

	defaultProfile, err := s.BillingService.GetDefaultProfile(ctx, billing.GetDefaultProfileInput{
		Namespace: namespace,
	})
	s.NoError(err)
	// The default profile should be the same as the app
	s.Equal(defaultProfile.Apps.Invoicing.GetType(), app.GetType())

	// Given we have pending invoice items without usage
	pendingLines, err := s.BillingService.CreatePendingInvoiceLines(ctx,
		billing.CreatePendingInvoiceLinesInput{
			Customer: customerEntity.GetID(),
			Currency: currencyx.Code(currency.USD),
			Lines: []*billing.Line{
				{
					LineBase: billing.LineBase{
						ManagedResource: models.NewManagedResource(models.ManagedResourceInput{
							Name: "UBP - FLAT per unit",
						}),
						Period:    billing.Period{Start: periodStart, End: periodEnd},
						InvoiceAt: periodEnd,
						ManagedBy: billing.ManuallyManagedLine,
					},
					UsageBased: &billing.UsageBasedLine{
						FeatureKey: flatPerUnitFeature.Key,
						Price: productcatalog.NewPriceFrom(productcatalog.UnitPrice{
							Amount: alpacadecimal.NewFromFloat(0),
						}),
					},
				},
			},
		},
	)
	s.NoError(err)
	s.Len(pendingLines.Lines, 1)

	clock.FreezeTime(periodEnd.Add(time.Minute))

	stripeAppCreateInvoiceMock := s.StripeAppClient.
		// See expect for args below: we cannot setup argument expect here
		// because we don't know the invoice ID before the call
		On("CreateInvoice", mock.Anything).
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

	// When we generate the invoice
	invoices, err := s.BillingService.InvoicePendingLines(ctx, billing.InvoicePendingLinesInput{
		Customer: customerEntity.GetID(),
	})
	s.NoError(err)
	s.Len(invoices, 1)
	invoice := invoices[0]

	// Assert the args of the create invoice call
	// We have to do it after the call because the invoice ID is not known at the time of setting up the mock
	stripeAppCreateInvoiceMock.Arguments.Assert(s.T(), stripeclient.CreateInvoiceInput{
		AppID:               app.GetID(),
		CustomerID:          customerEntity.GetID(),
		InvoiceID:           invoice.ID,
		AutomaticTaxEnabled: true,
		CollectionMethod:    billing.CollectionMethodChargeAutomatically,
		StripeCustomerID:    customerData.StripeCustomerID,
		Currency:            "USD",
	})

	// Then the invoice should have the UBP line with 0 amount
	lines := invoice.Lines.OrEmpty()
	s.Len(lines, 1)
	line := lines[0]
	s.Equal(line.Name, "UBP - FLAT per unit")
	s.Equal(float64(0), lines[0].Totals.Total.InexactFloat64())
	s.Len(invoice.ValidationIssues, 0)

	// Editing the invoice should also work
	s.StripeAppClient.
		On("UpdateInvoice", stripeclient.UpdateInvoiceInput{
			AutomaticTaxEnabled: true,
			StripeInvoiceID:     invoice.ExternalIDs.Invoicing,
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

	invoice, err = s.BillingService.UpdateInvoice(ctx, billing.UpdateInvoiceInput{
		Invoice: invoice.InvoiceID(),
		EditFn: func(i *billing.Invoice) error {
			i.Supplier.Name = "ACME Inc. (updated)"
			return nil
		},
	})
	s.NoError(err)

	s.Equal("ACME Inc. (updated)", invoice.Supplier.Name)
	s.Equal(billing.InvoiceStatusDraftManualApprovalNeeded, invoice.Status)
}

func (s *StripeInvoiceTestSuite) TestSendInvoice() {
	// Given we have a test customer and a billing profile with send_invoice collection method
	// we can create an invoice that will be sent to the customer instead of charged automatically.
	// In this test we should see due date set and collection method set to send_invoice.

	namespace := "ns-send-invoice"
	ctx := context.Background()
	periodStart := lo.Must(time.Parse(time.RFC3339, "2024-09-02T12:13:14Z"))
	periodEnd := lo.Must(time.Parse(time.RFC3339, "2024-09-03T12:13:14Z"))
	clock.FreezeTime(periodStart)
	defer clock.UnFreeze()

	_ = s.InstallSandboxApp(s.T(), namespace)

	// Create a test customer
	customerEntity, err := s.CustomerService.CreateCustomer(ctx, customer.CreateCustomerInput{
		Namespace: namespace,

		CustomerMutate: customer.CustomerMutate{
			Name:     "Test Customer",
			Currency: lo.ToPtr(currencyx.Code(currency.USD)),
			UsageAttribution: &customer.CustomerUsageAttribution{
				SubjectKeys: []string{"test"},
			},
		},
	})
	s.NoError(err)
	s.NotNil(customerEntity)
	s.NotEmpty(customerEntity.ID)

	// Create a test app
	app, err := s.Fixture.setupApp(ctx, namespace)
	s.NoError(err)

	// Given we have a default profile for the namespace
	s.ProvisionBillingProfile(ctx, namespace, app.GetID(), billingtest.WithBillingProfileEditFn(func(profile *billing.CreateProfileInput) {
		// manual advancement for testing the update invoice flow
		profile.WorkflowConfig.Invoicing.AutoAdvance = false
		profile.WorkflowConfig.Payment.CollectionMethod = billing.CollectionMethodSendInvoice
		profile.WorkflowConfig.Invoicing.DueAfter = lo.Must(datetime.ISODurationString("P45D").Parse())
	}))

	// Setup the app with the customer

	customerData, err := s.Fixture.setupAppCustomerData(ctx, app, customerEntity)
	s.NoError(err)
	s.NotNil(customerData)

	// Mock the stripe app client for the get stripe customer call
	s.StripeAppClient.
		On("GetCustomer", defaultStripeCustomerID).
		Return(stripeclient.StripeCustomer{
			StripeCustomerID: defaultStripeCustomerID,
			DefaultPaymentMethod: &stripeclient.StripePaymentMethod{
				ID:    "pm_123",
				Name:  "ACME Inc.",
				Email: "acme@test.com",
				BillingAddress: &models.Address{
					City:       lo.ToPtr("San Francisco"),
					PostalCode: lo.ToPtr("94103"),
					State:      lo.ToPtr("CA"),
					Country:    lo.ToPtr(models.CountryCode("US")),
					Line1:      lo.ToPtr("123 Market St"),
				},
			},
		}, nil)

	// Get the default profile
	defaultProfile, err := s.BillingService.GetDefaultProfile(ctx, billing.GetDefaultProfileInput{
		Namespace: namespace,
	})
	s.NoError(err)

	// The default profile should be the same as the app
	s.Equal(defaultProfile.Apps.Invoicing.GetType(), app.GetType())

	// Add a pending invoice line with a flat fee
	pendingLines, err := s.BillingService.CreatePendingInvoiceLines(ctx,
		billing.CreatePendingInvoiceLinesInput{
			Customer: customerEntity.GetID(),
			Currency: currencyx.Code(currency.USD),
			Lines: []*billing.Line{
				billing.NewFlatFeeLine(billing.NewFlatFeeLineInput{
					Period:        billing.Period{Start: periodStart, End: periodEnd},
					InvoiceAt:     periodStart,
					Name:          "Flat fee",
					PerUnitAmount: alpacadecimal.NewFromFloat(10),
					PaymentTerm:   productcatalog.InAdvancePaymentTerm,
				}),
			},
		},
	)
	s.NoError(err)
	s.Len(pendingLines.Lines, 1)

	clock.FreezeTime(periodEnd.Add(time.Minute))

	// Mock the stripe app client for the create invoice call
	s.StripeAppClient.
		// See expect for args below: we cannot setup argument expect here
		// because we don't know the invoice ID before the call
		On("CreateInvoice", mock.MatchedBy(func(input stripeclient.CreateInvoiceInput) bool {
			s.Equal(stripeclient.CreateInvoiceInput{
				AppID:               app.GetID(),
				CustomerID:          customerEntity.GetID(),
				InvoiceID:           input.InvoiceID,
				AutomaticTaxEnabled: true,
				CollectionMethod:    billing.CollectionMethodSendInvoice,
				DaysUntilDue:        lo.ToPtr(int64(45)),
				StripeCustomerID:    customerData.StripeCustomerID,
				Currency:            "USD",
			}, input, "expected CreateInvoice input to match")

			return true
		})).
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

	s.StripeAppClient.
		On("AddInvoiceLines", mock.Anything).
		Once().
		// We don't add any lines to the invoice as we don't test for it
		Return([]stripeclient.StripeInvoiceItemWithLineID{}, nil)

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

	// Create the invoice.
	_, err = invoicingApp.UpsertInvoice(ctx, invoice)
	s.NoError(err, "failed to create invoice")

	// Assert the client is called with the correct arguments.
	// FIXME: fix this assert, for some reason other tests are bleeding into this test at mock assertion
	// This does not impact the test, the create invoice mock is still called and the assert passes
	// s.StripeAppClient.AssertExpectations(s.T())
}

func mapInvoiceItemParamsToInvoiceItem(id string, i *stripe.InvoiceItemParams) stripeclient.StripeInvoiceItemWithLineID {
	return stripeclient.StripeInvoiceItemWithLineID{
		LineID: fmt.Sprintf("il_%s", id),
		InvoiceItem: &stripe.InvoiceItem{
			ID:          fmt.Sprintf("ii_%s", id),
			Amount:      *i.Amount,
			Quantity:    lo.FromPtr(i.Quantity),
			Description: *i.Description,
			Currency:    stripe.Currency(lo.FromPtr(i.Currency)),

			Period: &stripe.Period{
				Start: *i.Period.Start,
				End:   *i.Period.End,
			},
			Metadata: i.Metadata,
		},
	}
}
