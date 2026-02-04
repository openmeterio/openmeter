package custominvoicing

import (
	"context"
	"log/slog"
	"testing"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/invopop/gobl/currency"
	"github.com/oklog/ulid/v2"
	"github.com/samber/lo"
	"github.com/stretchr/testify/suite"

	"github.com/openmeterio/openmeter/openmeter/app"
	appcustominvoicing "github.com/openmeterio/openmeter/openmeter/app/custominvoicing"
	"github.com/openmeterio/openmeter/openmeter/app/custominvoicing/adapter"
	"github.com/openmeterio/openmeter/openmeter/app/custominvoicing/service"
	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/openmeter/meter"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/feature"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/datetime"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/timeutil"
	billingtest "github.com/openmeterio/openmeter/test/billing"
)

type CustomInvoicingTestSuite struct {
	billingtest.BaseSuite

	CustomInvoicingService appcustominvoicing.Service
}

func TestApp(t *testing.T) {
	suite.Run(t, &CustomInvoicingTestSuite{})
}

func (s *CustomInvoicingTestSuite) SetupSuite() {
	s.BaseSuite.SetupSuite()

	customInvoicingAdapter, err := adapter.New(adapter.Config{
		Client: s.DBClient,
		Logger: slog.Default(),
	})
	s.NoError(err, "failed to create custom invoicing adapter")

	svc, err := service.New(service.Config{
		Adapter:        customInvoicingAdapter,
		Logger:         slog.Default(),
		AppService:     s.AppService,
		BillingService: s.BillingService,
	})
	s.NoError(err, "failed to create custom invoicing service")

	s.CustomInvoicingService = svc

	// Let's register the app

	_, err = appcustominvoicing.NewFactory(appcustominvoicing.FactoryConfig{
		AppService:             s.AppService,
		CustomInvoicingService: svc,
		BillingService:         s.BillingService,
	})
	s.NoError(err, "failed to create custom invoicing factory")
}

func (s *CustomInvoicingTestSuite) setupDefaultBillingProfile(ctx context.Context, namespace string, customInvoicingConfig appcustominvoicing.Configuration) {
	// Install custom invoicing app
	customInvoicingApp, err := s.AppService.InstallMarketplaceListing(ctx, app.InstallAppInput{
		MarketplaceListingID: app.MarketplaceListingID{
			Type: app.AppTypeCustomInvoicing,
		},
		Namespace: namespace,
		Name:      "Custom Invoicing",
	})
	s.NoError(err, "failed to install custom invoicing app")

	// Let's set up the custom invoicing config
	_, err = s.AppService.UpdateApp(ctx, app.UpdateAppInput{
		AppID:           customInvoicingApp.GetID(),
		Name:            customInvoicingApp.GetName(),
		AppConfigUpdate: customInvoicingConfig,
	})
	s.NoError(err, "failed to upsert custom invoicing config")

	// Create billing profile
	s.ProvisionBillingProfile(ctx, namespace, customInvoicingApp.GetID(), billingtest.WithBillingProfileEditFn(func(profile *billing.CreateProfileInput) {
		profile.WorkflowConfig.Invoicing.DraftPeriod = lo.Must(datetime.ISODurationString("P0D").Parse())
	}))
}

func (s *CustomInvoicingTestSuite) TestInvoicingFlowHooksEnabled() {
	ctx := context.Background()
	namespace := "ns-custom-invoicing-flow"

	now := time.Now().Truncate(time.Microsecond).In(time.UTC)
	periodEnd := now.Add(-time.Hour)
	periodStart := periodEnd.Add(-time.Hour * 24 * 30)
	issueAt := now.Add(-time.Minute)

	s.setupDefaultBillingProfile(ctx, namespace, appcustominvoicing.Configuration{
		EnableDraftSyncHook:   true,
		EnableIssuingSyncHook: true,
	})

	customerEntity, err := s.CustomerService.CreateCustomer(ctx, customer.CreateCustomerInput{
		Namespace: namespace,

		CustomerMutate: customer.CustomerMutate{
			Name:         "Test Customer",
			PrimaryEmail: lo.ToPtr("test@test.com"),
			UsageAttribution: &customer.CustomerUsageAttribution{
				SubjectKeys: []string{"test"},
			},
		},
	})
	s.NoError(err, "failed to create customer")
	s.NotNil(customerEntity, "customer should not be nil")

	// Let's set up a meter for ubp testing
	err = s.MeterAdapter.ReplaceMeters(ctx, []meter.Meter{
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
				Name: "Test Meter",
			},
			Key:           "test",
			Aggregation:   meter.MeterAggregationSum,
			EventType:     "test",
			ValueProperty: lo.ToPtr("$.value"),
		},
	})
	s.NoError(err, "meter adapter should be able to replace meters")

	defer func() {
		err = s.MeterAdapter.ReplaceMeters(ctx, []meter.Meter{})
		s.NoError(err, "meter adapter should be able to replace meters")
	}()

	// Streaming adapter
	s.MockStreamingConnector.AddSimpleEvent("test", 0o0, periodStart.Add(-time.Minute))
	s.MockStreamingConnector.AddSimpleEvent("test", 100, periodStart.Add(time.Minute))

	defer s.MockStreamingConnector.Reset()

	_, err = s.FeatureService.CreateFeature(ctx, feature.CreateFeatureInputs{
		Namespace: namespace,
		Name:      "test",
		Key:       "test",
		MeterSlug: lo.ToPtr("test"),
	})
	s.NoError(err)

	// Let's create a gathering invoice
	s.Run("gathering invoice can be created", func() {
		res, err := s.BillingService.CreatePendingInvoiceLines(ctx,
			billing.CreatePendingInvoiceLinesInput{
				Customer: customerEntity.GetID(),
				Currency: currencyx.Code(currency.HUF),
				Lines: []billing.GatheringLine{
					billing.NewFlatFeeGatheringLine(billing.NewFlatFeeLineInput{
						Period: billing.Period{Start: periodStart, End: periodEnd},

						InvoiceAt: issueAt,
						ManagedBy: billing.ManuallyManagedLine,

						Name: "Test item - HUF",

						PerUnitAmount: alpacadecimal.NewFromFloat(200),
						PaymentTerm:   productcatalog.InAdvancePaymentTerm,
					}),
					{
						GatheringLineBase: billing.GatheringLineBase{
							ManagedResource: models.NewManagedResource(models.ManagedResourceInput{
								Name: "Test item - HUF",
							}),
							ServicePeriod: timeutil.ClosedPeriod{From: periodStart, To: periodEnd},

							InvoiceAt: issueAt,
							ManagedBy: billing.ManuallyManagedLine,
							Price: lo.FromPtr(productcatalog.NewPriceFrom(productcatalog.TieredPrice{
								Mode: productcatalog.GraduatedTieredPrice,
								Tiers: []productcatalog.PriceTier{
									{
										UpToAmount: lo.ToPtr(alpacadecimal.NewFromFloat(100)),
										UnitPrice: &productcatalog.PriceTierUnitPrice{
											Amount: alpacadecimal.NewFromFloat(10),
										},
									},
									{
										UnitPrice: &productcatalog.PriceTierUnitPrice{
											Amount: alpacadecimal.NewFromFloat(100),
										},
									},
								},
							})),
							FeatureKey: "test",
						},
					},
				},
			})
		s.NoError(err, "failed to create pending invoice lines")
		s.NotNil(res, "result should not be nil")
	})

	var invoice billing.StandardInvoice
	// When there are lines to be invoiced, we can create the invoice, and
	// it will end up in draft.syncing state
	s.Run("invoice can be created and will end up in draft.syncing state", func() {
		invoices, err := s.BillingService.InvoicePendingLines(ctx,
			billing.InvoicePendingLinesInput{
				Customer: customerEntity.GetID(),
				AsOf:     lo.ToPtr(issueAt),
			})
		s.NoError(err, "failed to invoice pending lines")
		s.NotNil(invoices, "result should not be nil")
		s.Len(invoices, 1, "should have one invoice")
		invoice = invoices[0]
		s.Len(invoice.Lines.OrEmpty(), 2, "invoice should have two lines")

		s.Equal(billing.StandardInvoiceStatusDraftSyncing, invoice.Status, "invoice should be in draft.sync state")
	})

	// When calling the service's SyncDraftInvoice, it should advance the invoice to issuing.syncing state
	s.Run("syncing the invoice should advance it to issuing.syncing state", func() {
		upsertResults := billing.NewUpsertStandardInvoiceResult().
			SetInvoiceNumber("DRAFT-123").
			SetExternalID("ext-123").
			AddLineExternalID(invoice.Lines.OrEmpty()[0].ID, "ext-123")

		draftSyncedInvoice, err := s.CustomInvoicingService.SyncDraftInvoice(ctx, appcustominvoicing.SyncDraftInvoiceInput{
			InvoiceID:            invoice.InvoiceID(),
			UpsertInvoiceResults: upsertResults,
		})
		s.NoError(err, "failed to sync draft invoice")
		s.Equal(billing.StandardInvoiceStatusIssuingSyncing, draftSyncedInvoice.Status, "invoice should be in issuing.sync state")

		// Let's validate the external IDs
		s.Equal("ext-123", draftSyncedInvoice.ExternalIDs.Invoicing, "invoice external ID should be set")
		s.Equal("ext-123", draftSyncedInvoice.Lines.OrEmpty()[0].ExternalIDs.Invoicing, "line external ID should be set")
		s.Equal("DRAFT-123", draftSyncedInvoice.Number, "invoice number should be set")

		invoice = draftSyncedInvoice
	})

	// When calling the service's SyncIssuingInvoice, it should advance the invoice to payment-processing.pending state
	s.Run("syncing the invoice should advance it to payment-processing.pending state", func() {
		finalizeResults := billing.NewFinalizeStandardInvoiceResult().
			SetPaymentExternalID("issuing-ext-123").
			SetInvoiceNumber("ISSUING-123")

		issuingSyncedInvoice, err := s.CustomInvoicingService.SyncIssuingInvoice(ctx, appcustominvoicing.SyncIssuingInvoiceInput{
			InvoiceID:             invoice.InvoiceID(),
			FinalizeInvoiceResult: finalizeResults,
		})
		s.NoError(err, "failed to sync issuing invoice")
		s.Equal(billing.StandardInvoiceStatusPaymentProcessingPending, issuingSyncedInvoice.Status, "invoice should be in issued state")

		// Let's validate the external IDs
		s.Equal("issuing-ext-123", issuingSyncedInvoice.ExternalIDs.Payment, "invoice external ID should be set")
		s.Equal("ISSUING-123", issuingSyncedInvoice.Number, "invoice number should be set")
	})

	// Payment status handling: we can transition the invoice to paid state
	s.Run("invoice can be transitioned to uncollectible state", func() {
		invoice, err := s.CustomInvoicingService.HandlePaymentTrigger(ctx, appcustominvoicing.HandlePaymentTriggerInput{
			InvoiceID: invoice.InvoiceID(),
			Trigger:   billing.TriggerPaid,
		})
		s.NoError(err, "failed to handle payment trigger")
		s.Equal(billing.StandardInvoiceStatusPaid, invoice.Status, "invoice should be in paid state")
		s.NotNil(invoice.IssuedAt, "invoice should have an issued at time")
	})

	// Payment status handling: we cannot transition the invoice to uncollectible state (full mesh transitions)
	s.Run("invoice cannot be transitioned to uncollectible state", func() {
		invoice, err := s.BillingService.GetInvoiceByID(ctx, billing.GetInvoiceByIdInput{
			Invoice: invoice.InvoiceID(),
		})
		s.NoError(err, "failed to get invoice")
		s.Len(invoice.ValidationIssues, 0, "invoice should have no validation issues")

		invoice, err = s.CustomInvoicingService.HandlePaymentTrigger(ctx, appcustominvoicing.HandlePaymentTriggerInput{
			InvoiceID: invoice.InvoiceID(),
			Trigger:   billing.TriggerPaymentUncollectible,
		})
		s.Error(err, "failed to handle payment trigger")
		s.ErrorAs(err, &billing.ValidationError{}, "error should be a validation error")
	})
}

func (s *CustomInvoicingTestSuite) TestInvoicingFlowPaymentStatusOnly() {
	ctx := context.Background()
	namespace := "ns-custom-invoicing-flow-payment-status-only"

	now := time.Now().Truncate(time.Microsecond).In(time.UTC)
	periodEnd := now.Add(-time.Hour)
	periodStart := periodEnd.Add(-time.Hour * 24 * 30)
	issueAt := now.Add(-time.Minute)

	s.setupDefaultBillingProfile(ctx, namespace, appcustominvoicing.Configuration{
		EnableDraftSyncHook:   false,
		EnableIssuingSyncHook: false,
	})

	customerEntity, err := s.CustomerService.CreateCustomer(ctx, customer.CreateCustomerInput{
		Namespace: namespace,

		CustomerMutate: customer.CustomerMutate{
			Name:         "Test Customer",
			PrimaryEmail: lo.ToPtr("test@test.com"),
			UsageAttribution: &customer.CustomerUsageAttribution{
				SubjectKeys: []string{"test"},
			},
		},
	})
	s.NoError(err, "failed to create customer")
	s.NotNil(customerEntity, "customer should not be nil")

	s.Run("gathering invoice can be created", func() {
		res, err := s.BillingService.CreatePendingInvoiceLines(ctx,
			billing.CreatePendingInvoiceLinesInput{
				Customer: customerEntity.GetID(),
				Currency: currencyx.Code(currency.HUF),
				Lines: []billing.GatheringLine{
					billing.NewFlatFeeGatheringLine(billing.NewFlatFeeLineInput{
						Period: billing.Period{Start: periodStart, End: periodEnd},

						InvoiceAt: issueAt,
						ManagedBy: billing.ManuallyManagedLine,

						Name: "Test item - HUF",

						PerUnitAmount: alpacadecimal.NewFromFloat(600),
						PaymentTerm:   productcatalog.InAdvancePaymentTerm,
					}),
				},
			})
		s.NoError(err, "failed to create pending invoice lines")
		s.NotNil(res, "result should not be nil")
	})

	var invoice billing.StandardInvoice

	// When there are lines to be invoiced, we can create the invoice, and
	// it will end up in payment_processing.pending state
	s.Run("invoice can be created and will end up in draft.syncing state", func() {
		invoices, err := s.BillingService.InvoicePendingLines(ctx,
			billing.InvoicePendingLinesInput{
				Customer: customerEntity.GetID(),
				AsOf:     lo.ToPtr(issueAt),
			})
		s.NoError(err, "failed to invoice pending lines")
		s.NotNil(invoices, "result should not be nil")
		s.Len(invoices, 1, "should have one invoice")
		invoice = invoices[0]
		s.Len(invoice.Lines.OrEmpty(), 1, "invoice should have one line")

		// Let's validate the invoice's status

		// We end up in payment_processing.pending state because we don't have a draft sync hook
		s.Equal(billing.StandardInvoiceStatusPaymentProcessingPending, invoice.Status, "invoice should be in payment_processing.pending state")

		// Invoice should have a generic invoice number assigned
		s.Equal("INV-TECU-1", invoice.Number, "invoice number should be set")
	})
}
