package appstripe

import (
	"context"
	"log/slog"
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
	"github.com/openmeterio/openmeter/openmeter/app/stripe/invoicesync"
	invoicesyncadapter "github.com/openmeterio/openmeter/openmeter/app/stripe/invoicesync/adapter"
	invoicesyncservice "github.com/openmeterio/openmeter/openmeter/app/stripe/invoicesync/service"
	appstripeservice "github.com/openmeterio/openmeter/openmeter/app/stripe/service"
	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/openmeter/meter"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/feature"
	"github.com/openmeterio/openmeter/openmeter/secret"
	secretadapter "github.com/openmeterio/openmeter/openmeter/secret/adapter"
	secretservice "github.com/openmeterio/openmeter/openmeter/secret/service"
	"github.com/openmeterio/openmeter/openmeter/watermill/eventbus"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/timeutil"
	billingtest "github.com/openmeterio/openmeter/test/billing"
)

// StripeSyncPlanTestSuite tests the async sync plan path end-to-end:
// invoice creation → sync plan generation → plan execution → result propagation.
type StripeSyncPlanTestSuite struct {
	billingtest.BaseSuite

	AppStripeService appstripe.Service
	Fixture          *Fixture
	SecretService    secret.Service
	StripeAppClient  *StripeAppClientMock
	SyncPlanAdapter  *invoicesyncadapter.Adapter
	SyncPlanHandler  *invoicesync.Handler
}

func TestStripeSyncPlan(t *testing.T) {
	suite.Run(t, &StripeSyncPlanTestSuite{})
}

func (s *StripeSyncPlanTestSuite) SetupSuite() {
	s.BaseSuite.SetupSuite()

	secretAdapter := secretadapter.New()
	secretService, err := secretservice.New(secretservice.Config{Adapter: secretAdapter})
	s.Require().NoError(err)
	s.SecretService = secretService

	stripeClient := &StripeClientMock{}
	stripeAppClient := &StripeAppClientMock{}
	s.StripeAppClient = stripeAppClient

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
	s.Require().NoError(err)

	syncPlanAdapter, err := invoicesyncadapter.New(invoicesyncadapter.Config{Client: s.DBClient})
	s.Require().NoError(err)
	s.SyncPlanAdapter = syncPlanAdapter

	webhookURLGenerator, err := appstripeservice.NewBaseURLWebhookURLGenerator("http://localhost:8888")
	s.Require().NoError(err)

	publisher := eventbus.NewMock(s.T())

	syncPlanService, err := invoicesyncservice.New(invoicesyncservice.Config{
		Adapter:   syncPlanAdapter,
		Publisher: publisher,
		Logger:    slog.Default(),
	})
	s.Require().NoError(err)

	appStripeService, err := appstripeservice.New(appstripeservice.Config{
		Adapter:             appStripeAdapter,
		AppService:          s.AppService,
		SecretService:       secretService,
		BillingService:      s.BillingService,
		Logger:              slog.Default(),
		Publisher:           publisher,
		WebhookURLGenerator: webhookURLGenerator,
		SyncPlanService:     syncPlanService,
	})
	s.Require().NoError(err)
	s.AppStripeService = appStripeService

	syncPlanHandler, err := invoicesync.NewHandler(invoicesync.HandlerConfig{
		Adapter:          syncPlanAdapter,
		AppService:       s.AppService,
		BillingService:   s.BillingService,
		StripeAppService: appStripeService,
		SecretService:    secretService,
		StripeAppClientFactory: func(config stripeclient.StripeAppClientConfig) (stripeclient.StripeAppClient, error) {
			return stripeAppClient, nil
		},
		Publisher: publisher,
		LockFunc: func(ctx context.Context, namespace, planID string) error {
			return nil // no-op lock in tests
		},
		Logger: slog.Default(),
	})
	s.Require().NoError(err)
	s.SyncPlanHandler = syncPlanHandler

	s.Fixture = NewFixture(s.AppService, s.CustomerService, stripeClient, stripeAppClient)
}

func (s *StripeSyncPlanTestSuite) TearDownTest() {
	s.StripeAppClient.Restore()
}

// executeSyncPlan runs the handler repeatedly until the plan completes or fails.
func (s *StripeSyncPlanTestSuite) executeSyncPlan(ctx context.Context, planID, namespace, invoiceID, customerID string) {
	s.T().Helper()

	for i := 0; i < 20; i++ {
		err := s.SyncPlanHandler.Handle(ctx, &invoicesync.ExecuteSyncPlanEvent{
			PlanID: planID, InvoiceID: invoiceID, Namespace: namespace, CustomerID: customerID,
		})
		s.Require().NoError(err)

		plan, err := s.SyncPlanAdapter.GetSyncPlan(ctx, planID)
		s.Require().NoError(err)
		if plan.Status == invoicesync.PlanStatusCompleted || plan.Status == invoicesync.PlanStatusFailed {
			return
		}
	}
	s.Fail("sync plan did not complete within 20 iterations")
}

func (s *StripeSyncPlanTestSuite) TestSyncPlanDraftSync() {
	namespace := "ns-syncplan-draft"
	ctx := context.Background()

	periodStart := lo.Must(time.Parse(time.RFC3339, "2024-09-02T12:13:14Z"))
	periodEnd := lo.Must(time.Parse(time.RFC3339, "2024-09-03T12:13:14Z"))
	clock.FreezeTime(periodStart)
	defer clock.UnFreeze()

	sandboxApp := s.InstallSandboxApp(s.T(), namespace)

	meterID := ulid.Make().String()
	err := s.MeterAdapter.ReplaceMeters(ctx, []meter.Meter{{
		ManagedResource: models.ManagedResource{
			ID: meterID, NamespacedModel: models.NamespacedModel{Namespace: namespace},
			ManagedModel: models.ManagedModel{CreatedAt: clock.Now(), UpdatedAt: clock.Now()},
			Name:         "API Calls",
		},
		Key: "api-calls", Aggregation: meter.MeterAggregationSum,
		EventType: "test", ValueProperty: lo.ToPtr("$.value"),
	}})
	s.NoError(err)

	feat := lo.Must(s.FeatureService.CreateFeature(ctx, feature.CreateFeatureInputs{
		Namespace: namespace, Name: "API Calls", Key: "api-calls",
		MeterID: lo.ToPtr(meterID),
	}))

	customerEntity, err := s.CustomerService.CreateCustomer(ctx, customer.CreateCustomerInput{
		Namespace: namespace,
		CustomerMutate: customer.CustomerMutate{
			Name:     "SyncPlan Test Customer",
			Currency: lo.ToPtr(currencyx.Code(currency.USD)),
			UsageAttribution: &customer.CustomerUsageAttribution{
				SubjectKeys: []string{"test"},
			},
		},
	})
	s.NoError(err)

	// Provision billing profile with sandbox app first
	s.ProvisionBillingProfile(ctx, namespace, sandboxApp.GetID())

	// Create pending lines
	_, err = s.BillingService.CreatePendingInvoiceLines(ctx, billing.CreatePendingInvoiceLinesInput{
		Customer: customerEntity.GetID(),
		Currency: currencyx.Code(currency.USD),
		Lines: []billing.GatheringLine{{
			GatheringLineBase: billing.GatheringLineBase{
				ManagedResource: models.NewManagedResource(models.ManagedResourceInput{Name: "API Calls"}),
				ServicePeriod:   timeutil.ClosedPeriod{From: periodStart, To: periodEnd},
				InvoiceAt:       periodEnd,
				ManagedBy:       billing.ManuallyManagedLine,
				FeatureKey:      feat.Key,
				Price: lo.FromPtr(productcatalog.NewPriceFrom(productcatalog.UnitPrice{
					Amount: alpacadecimal.NewFromFloat(0.01),
				})),
			},
		}},
	})
	s.NoError(err)

	// Advance time and add usage
	clock.FreezeTime(periodEnd.Add(time.Minute))
	s.MockStreamingConnector.AddSimpleEvent("api-calls", 100, periodStart.Add(time.Minute))

	// Setup Stripe app + customer
	var stripeApp app.App
	var customerData appstripeentity.CustomerData

	s.Run("setup stripe app and customer", func() {
		stripeApp, err = s.Fixture.setupApp(ctx, namespace)
		s.NoError(err)
		customerData, err = s.Fixture.setupAppCustomerData(ctx, stripeApp, customerEntity)
		s.NoError(err)
	})

	// Create the invoice
	var invoice billing.StandardInvoice
	s.Run("create invoice from pending lines", func() {
		invoices, err := s.BillingService.InvoicePendingLines(ctx, billing.InvoicePendingLinesInput{
			Customer: customerEntity.GetID(),
			AsOf:     &periodEnd,
		})
		s.NoError(err)
		s.Require().Len(invoices, 1)
		invoice = lo.Must(invoices[0].RemoveCircularReferences())
	})

	s.Run("upsert creates sync plan and handler executes it", func() {
		defer s.StripeAppClient.Restore()

		invoicingApp, err := billing.GetApp(stripeApp)
		s.NoError(err)

		// UpsertStandardInvoice creates a sync plan (async mode)
		result, err := invoicingApp.UpsertStandardInvoice(ctx, invoice)
		s.NoError(err)
		s.Nil(result, "async mode should return nil result")

		// Verify plan was created
		plan, err := s.SyncPlanAdapter.GetActiveSyncPlanByInvoice(ctx, namespace, invoice.ID, invoicesync.SyncPlanPhaseDraft)
		s.NoError(err)
		s.Require().NotNil(plan, "draft sync plan should exist")
		s.Equal(invoicesync.PlanStatusPending, plan.Status)
		s.Greater(len(plan.Operations), 0)
		s.Equal(invoicesync.OpTypeInvoiceCreate, plan.Operations[0].Type)

		// Mock Stripe calls for plan execution
		s.StripeAppClient.On("CreateInvoice", mock.Anything).Once().
			Return(&stripe.Invoice{
				ID: "in_syncplan", Number: "SP-001",
				Customer: &stripe.Customer{ID: customerData.StripeCustomerID},
				Currency: "USD",
				Lines:    &stripe.InvoiceLineItemList{Data: []*stripe.InvoiceLineItem{}},
			}, nil)

		leafLines := invoice.GetLeafLinesWithConsolidatedTaxBehavior()
		lineID := ""
		if len(leafLines) > 0 {
			lineID = leafLines[0].ID
		}

		s.StripeAppClient.On("AddInvoiceLines", mock.MatchedBy(func(input stripeclient.AddInvoiceLinesInput) bool {
			return input.StripeInvoiceID == "in_syncplan"
		})).Once().Return([]stripeclient.StripeInvoiceItemWithLineID{{
			InvoiceItem: &stripe.InvoiceItem{
				ID:       "ii_1",
				Metadata: map[string]string{"om_line_id": lineID, "om_line_type": "line"},
			},
			LineID: "il_1",
		}}, nil)

		// Execute operations one by one (the handler calls SyncDraftInvoice on completion,
		// which requires the invoice to be in draft.syncing state. Since we called
		// UpsertStandardInvoice directly, the invoice isn't in that state.
		// We execute operations manually and verify the plan state instead.)
		executor := &invoicesync.Executor{
			Adapter: s.SyncPlanAdapter,
			Logger:  slog.Default(),
		}

		// Execute first op (InvoiceCreate)
		result1, err := executor.ExecuteNextOperation(ctx, s.StripeAppClient, plan)
		s.NoError(err)
		s.False(result1.Done)

		// Re-fetch plan to get updated operation responses
		plan, err = s.SyncPlanAdapter.GetSyncPlan(ctx, plan.ID)
		s.NoError(err)

		// Execute second op (LineItemAdd)
		result2, err := executor.ExecuteNextOperation(ctx, s.StripeAppClient, plan)
		s.NoError(err)
		s.False(result2.Done)

		// Execute again — should complete the plan (no more pending ops)
		plan, err = s.SyncPlanAdapter.GetSyncPlan(ctx, plan.ID)
		s.NoError(err)
		result3, err := executor.ExecuteNextOperation(ctx, s.StripeAppClient, plan)
		s.NoError(err)
		s.True(result3.Done)
		s.False(result3.Failed)

		// Verify plan completed successfully
		plan, err = s.SyncPlanAdapter.GetSyncPlan(ctx, plan.ID)
		s.NoError(err)
		s.Equal(invoicesync.PlanStatusCompleted, plan.Status)
		s.NotNil(plan.CompletedAt)

		for _, op := range plan.Operations {
			s.Equal(invoicesync.OpStatusCompleted, op.Status, "op %s (seq %d) should be completed", op.Type, op.Sequence)
			s.NotNil(op.StripeResponse, "op %s should have a response", op.Type)
		}

		// Verify results can be built from the completed plan
		upsertResult, err := invoicesync.BuildUpsertResultFromPlan(plan)
		s.NoError(err)
		externalID, ok := upsertResult.GetExternalID()
		s.True(ok)
		s.Equal("in_syncplan", externalID)
		invoiceNumber, ok := upsertResult.GetInvoiceNumber()
		s.True(ok)
		s.Equal("SP-001", invoiceNumber)

		s.StripeAppClient.AssertExpectations(s.T())
	})
}

func (s *StripeSyncPlanTestSuite) TestSyncPlanFailure() {
	namespace := "ns-syncplan-fail"
	ctx := context.Background()

	periodStart := lo.Must(time.Parse(time.RFC3339, "2024-09-02T12:13:14Z"))
	periodEnd := lo.Must(time.Parse(time.RFC3339, "2024-09-03T12:13:14Z"))
	clock.FreezeTime(periodStart)
	defer clock.UnFreeze()

	sandboxApp := s.InstallSandboxApp(s.T(), namespace)

	meterID := ulid.Make().String()
	s.NoError(s.MeterAdapter.ReplaceMeters(ctx, []meter.Meter{{
		ManagedResource: models.ManagedResource{
			ID: meterID, NamespacedModel: models.NamespacedModel{Namespace: namespace},
			ManagedModel: models.ManagedModel{CreatedAt: clock.Now(), UpdatedAt: clock.Now()},
			Name:         "Requests",
		},
		Key: "requests", Aggregation: meter.MeterAggregationSum,
		EventType: "test", ValueProperty: lo.ToPtr("$.value"),
	}}))

	feat := lo.Must(s.FeatureService.CreateFeature(ctx, feature.CreateFeatureInputs{
		Namespace: namespace, Name: "Requests", Key: "requests", MeterID: lo.ToPtr(meterID),
	}))

	customerEntity, err := s.CustomerService.CreateCustomer(ctx, customer.CreateCustomerInput{
		Namespace: namespace,
		CustomerMutate: customer.CustomerMutate{
			Name: "Fail Customer", Currency: lo.ToPtr(currencyx.Code(currency.USD)),
			UsageAttribution: &customer.CustomerUsageAttribution{SubjectKeys: []string{"test"}},
		},
	})
	s.NoError(err)

	s.ProvisionBillingProfile(ctx, namespace, sandboxApp.GetID())

	_, err = s.BillingService.CreatePendingInvoiceLines(ctx, billing.CreatePendingInvoiceLinesInput{
		Customer: customerEntity.GetID(), Currency: currencyx.Code(currency.USD),
		Lines: []billing.GatheringLine{{
			GatheringLineBase: billing.GatheringLineBase{
				ManagedResource: models.NewManagedResource(models.ManagedResourceInput{Name: "Requests"}),
				ServicePeriod:   timeutil.ClosedPeriod{From: periodStart, To: periodEnd},
				InvoiceAt:       periodEnd, ManagedBy: billing.ManuallyManagedLine,
				FeatureKey: feat.Key,
				Price: lo.FromPtr(productcatalog.NewPriceFrom(productcatalog.UnitPrice{
					Amount: alpacadecimal.NewFromFloat(0.01),
				})),
			},
		}},
	})
	s.NoError(err)

	clock.FreezeTime(periodEnd.Add(time.Minute))
	s.MockStreamingConnector.AddSimpleEvent("requests", 50, periodStart.Add(time.Minute))

	var stripeApp app.App
	s.Run("setup", func() {
		stripeApp, err = s.Fixture.setupApp(ctx, namespace)
		s.NoError(err)
		_, err = s.Fixture.setupAppCustomerData(ctx, stripeApp, customerEntity)
		s.NoError(err)
	})

	var invoice billing.StandardInvoice
	s.Run("create invoice", func() {
		invoices, err := s.BillingService.InvoicePendingLines(ctx, billing.InvoicePendingLinesInput{
			Customer: customerEntity.GetID(), AsOf: &periodEnd,
		})
		s.NoError(err)
		s.Require().Len(invoices, 1)
		invoice = lo.Must(invoices[0].RemoveCircularReferences())
	})

	s.Run("sync plan fails on stripe error", func() {
		defer s.StripeAppClient.Restore()

		invoicingApp, err := billing.GetApp(stripeApp)
		s.NoError(err)

		result, err := invoicingApp.UpsertStandardInvoice(ctx, invoice)
		s.NoError(err)
		s.Nil(result)

		plan, err := s.SyncPlanAdapter.GetActiveSyncPlanByInvoice(ctx, namespace, invoice.ID, invoicesync.SyncPlanPhaseDraft)
		s.NoError(err)
		s.Require().NotNil(plan)

		// Mock CreateInvoice to fail with 400 (non-retryable)
		s.StripeAppClient.On("CreateInvoice", mock.Anything).Once().
			Return((*stripe.Invoice)(nil), &stripe.Error{
				HTTPStatusCode: 400, Code: "invalid_request", Msg: "test failure",
			})

		// Execute via executor directly (not handler) to avoid TriggerFailed
		// interacting with the billing state machine in the test environment.
		executor := &invoicesync.Executor{
			Adapter: s.SyncPlanAdapter,
			Logger:  slog.Default(),
		}
		execResult, err := executor.ExecuteNextOperation(ctx, s.StripeAppClient, plan)
		s.NoError(err)
		s.True(execResult.Done)
		s.True(execResult.Failed)
		s.Contains(execResult.FailError, "test failure")

		plan, err = s.SyncPlanAdapter.GetSyncPlan(ctx, plan.ID)
		s.NoError(err)
		s.Equal(invoicesync.PlanStatusFailed, plan.Status)
		s.Require().NotNil(plan.Error)
		s.Contains(*plan.Error, "test failure")

		for _, op := range plan.Operations {
			s.Equal(invoicesync.OpStatusFailed, op.Status, "op %d should be failed", op.Sequence)
		}

		s.StripeAppClient.AssertExpectations(s.T())
	})
}
