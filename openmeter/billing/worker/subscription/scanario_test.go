package billingworkersubscription

import (
	"context"
	"log/slog"
	"testing"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/invopop/gobl/currency"
	"github.com/samber/lo"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/credit"
	grantrepo "github.com/openmeterio/openmeter/openmeter/credit/adapter"
	customerentity "github.com/openmeterio/openmeter/openmeter/customer/entity"
	enttx "github.com/openmeterio/openmeter/openmeter/ent/tx"
	"github.com/openmeterio/openmeter/openmeter/entitlement"
	entitlementrepo "github.com/openmeterio/openmeter/openmeter/entitlement/adapter"
	booleanentitlement "github.com/openmeterio/openmeter/openmeter/entitlement/boolean"
	meteredentitlement "github.com/openmeterio/openmeter/openmeter/entitlement/metered"
	staticentitlement "github.com/openmeterio/openmeter/openmeter/entitlement/static"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/feature"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/plan"
	planadapter "github.com/openmeterio/openmeter/openmeter/productcatalog/plan/adapter"
	planservice "github.com/openmeterio/openmeter/openmeter/productcatalog/plan/service"
	plansubscription "github.com/openmeterio/openmeter/openmeter/productcatalog/subscription"
	productcatalogsubscription "github.com/openmeterio/openmeter/openmeter/productcatalog/subscription"
	"github.com/openmeterio/openmeter/openmeter/subscription"
	subscriptionentitlementadatapter "github.com/openmeterio/openmeter/openmeter/subscription/adapters/entitlement"
	subscriptionrepo "github.com/openmeterio/openmeter/openmeter/subscription/repo"
	subscriptionservice "github.com/openmeterio/openmeter/openmeter/subscription/service"
	"github.com/openmeterio/openmeter/openmeter/watermill/eventbus"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/datex"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/pagination"
	billingtest "github.com/openmeterio/openmeter/test/billing"
)

type SubscriptionHandlerTestSuite struct {
	billingtest.BaseSuite

	PlanService                 plan.Service
	SubscriptionService         subscription.Service
	SubscrpiptionPlanAdapter    plansubscription.Adapter
	SubscriptionWorkflowService subscription.WorkflowService

	Handler *Handler
}

func (s *SubscriptionHandlerTestSuite) SetupSuite() {
	s.BaseSuite.SetupSuite()

	planAdapter, err := planadapter.New(planadapter.Config{
		Client: s.DBClient,
		Logger: slog.Default(),
	})
	s.NoError(err)

	planService, err := planservice.New(planservice.Config{
		Feature: s.FeatureService,
		Adapter: planAdapter,
		Logger:  slog.Default(),
	})
	s.NoError(err)

	s.PlanService = planService

	subsRepo := subscriptionrepo.NewSubscriptionRepo(s.DBClient)
	subsItemRepo := subscriptionrepo.NewSubscriptionItemRepo(s.DBClient)

	s.SubscriptionService = subscriptionservice.New(subscriptionservice.ServiceConfig{
		SubscriptionRepo:      subsRepo,
		SubscriptionPhaseRepo: subscriptionrepo.NewSubscriptionPhaseRepo(s.DBClient),
		SubscriptionItemRepo:  subsItemRepo,
		// connectors
		CustomerService: s.CustomerService,
		// adapters
		EntitlementAdapter: subscriptionentitlementadatapter.NewSubscriptionEntitlementAdapter(
			s.SetupEntitlements(),
			subsItemRepo,
			subsRepo,
		),
		// framework
		TransactionManager: subsRepo,
		// events
		Publisher: eventbus.NewMock(s.T()),
	})

	s.SubscrpiptionPlanAdapter = plansubscription.NewPlanSubscriptionAdapter(plansubscription.PlanSubscriptionAdapterConfig{
		PlanService: planService,
		Logger:      slog.Default(),
	})

	s.SubscriptionWorkflowService = subscriptionservice.NewWorkflowService(subscriptionservice.WorkflowServiceConfig{
		Service:            s.SubscriptionService,
		CustomerService:    s.CustomerService,
		TransactionManager: subsRepo,
	})

	handler, err := New(Config{
		BillingService: s.BillingService,
		Logger:         slog.Default(),
		TxCreator:      s.BillingAdapter,
	})
	s.NoError(err)

	s.Handler = handler
}

func (s *SubscriptionHandlerTestSuite) SetupEntitlements() entitlement.Connector {
	// Init grants/credit
	grantRepo := grantrepo.NewPostgresGrantRepo(s.DBClient)
	balanceSnapshotRepo := grantrepo.NewPostgresBalanceSnapshotRepo(s.DBClient)

	// Init entitlements
	entitlementRepo := entitlementrepo.NewPostgresEntitlementRepo(s.DBClient)
	usageResetRepo := entitlementrepo.NewPostgresUsageResetRepo(s.DBClient)

	mockPublisher := eventbus.NewMock(s.T())

	owner := meteredentitlement.NewEntitlementGrantOwnerAdapter(
		s.FeatureRepo,
		entitlementRepo,
		usageResetRepo,
		s.MeterRepo,
		slog.Default(),
	)

	transactionManager := enttx.NewCreator(s.DBClient)

	creditConnector := credit.NewCreditConnector(
		grantRepo,
		balanceSnapshotRepo,
		owner,
		s.MockStreamingConnector,
		slog.Default(),
		time.Minute,
		mockPublisher,
		transactionManager,
	)

	meteredEntitlementConnector := meteredentitlement.NewMeteredEntitlementConnector(
		s.MockStreamingConnector,
		owner,
		creditConnector,
		creditConnector,
		grantRepo,
		entitlementRepo,
		mockPublisher,
	)

	staticEntitlementConnector := staticentitlement.NewStaticEntitlementConnector()
	booleanEntitlementConnector := booleanentitlement.NewBooleanEntitlementConnector()

	return entitlement.NewEntitlementConnector(
		entitlementRepo,
		s.FeatureService,
		s.MeterRepo,
		meteredEntitlementConnector,
		staticEntitlementConnector,
		booleanEntitlementConnector,
		mockPublisher,
	)
}

func TestSubscriptionHandlerScenarios(t *testing.T) {
	suite.Run(t, new(SubscriptionHandlerTestSuite))
}

func (s *SubscriptionHandlerTestSuite) mustParseTime(t string) time.Time {
	return lo.Must(time.Parse(time.RFC3339, t))
}

func (s *SubscriptionHandlerTestSuite) TestSubscriptionHappyPath() {
	ctx := context.Background()
	namespace := "test-subs-happy-path"
	start := s.mustParseTime("2024-01-01T00:00:00Z")
	clock.SetTime(start)
	defer clock.ResetTime()

	_ = s.InstallSandboxApp(s.T(), namespace)

	minimalCreateProfileInput := billingtest.MinimalCreateProfileInputTemplate
	minimalCreateProfileInput.Namespace = namespace

	profile, err := s.BillingService.CreateProfile(ctx, minimalCreateProfileInput)
	s.NoError(err)
	s.NotNil(profile)

	apiRequestsTotalMeterSlug := "api-requests-total"

	s.MeterRepo.ReplaceMeters(ctx, []models.Meter{
		{
			Namespace:   namespace,
			Slug:        apiRequestsTotalMeterSlug,
			WindowSize:  models.WindowSizeMinute,
			Aggregation: models.MeterAggregationSum,
		},
	})
	defer s.MeterRepo.ReplaceMeters(ctx, []models.Meter{})

	apiRequestsTotalFeatureKey := "api-requests-total"

	apiRequestsTotalFeature, err := s.FeatureService.CreateFeature(ctx, feature.CreateFeatureInputs{
		Namespace: namespace,
		Name:      "api-requests-total",
		Key:       apiRequestsTotalFeatureKey,
		MeterSlug: lo.ToPtr("api-requests-total"),
	})
	s.NoError(err)

	customerEntity, err := s.CustomerService.CreateCustomer(ctx, customerentity.CreateCustomerInput{
		Namespace: namespace,

		CustomerMutate: customerentity.CustomerMutate{
			Name:         "Test Customer",
			PrimaryEmail: lo.ToPtr("test@test.com"),
			BillingAddress: &models.Address{
				Country: lo.ToPtr(models.CountryCode("US")),
			},
			Currency: lo.ToPtr(currencyx.Code(currency.USD)),
			UsageAttribution: customerentity.CustomerUsageAttribution{
				SubjectKeys: []string{"test"},
			},
		},
	})
	require.NoError(s.T(), err)
	require.NotNil(s.T(), customerEntity)
	require.NotEmpty(s.T(), customerEntity.ID)

	plan, err := s.PlanService.CreatePlan(ctx, plan.CreatePlanInput{
		NamespacedModel: models.NamespacedModel{
			Namespace: namespace,
		},
		Plan: productcatalog.Plan{
			PlanMeta: productcatalog.PlanMeta{
				Name:     "Test Plan",
				Key:      "test-plan",
				Version:  1,
				Currency: currency.USD,
			},

			Phases: []productcatalog.Phase{
				{
					PhaseMeta: productcatalog.PhaseMeta{
						Name:       "free trial",
						Key:        "free-trial",
						StartAfter: datex.MustParse(s.T(), "P0D"),
					},
					// TODO[OM-1031]: let's add discount handling (as this could be a 100% discount for the first month)
					RateCards: productcatalog.RateCards{
						&productcatalog.UsageBasedRateCard{
							RateCardMeta: productcatalog.RateCardMeta{
								Key:     apiRequestsTotalFeatureKey,
								Name:    apiRequestsTotalFeatureKey,
								Feature: &apiRequestsTotalFeature,
							},
							BillingCadence: datex.MustParse(s.T(), "P1M"),
						},
					},
				},
				{
					PhaseMeta: productcatalog.PhaseMeta{
						Name:       "discounted phase",
						Key:        "discounted-phase",
						StartAfter: datex.MustParse(s.T(), "P1M"),
					},
					// TODO[OM-1031]: 50% discount
					RateCards: productcatalog.RateCards{
						&productcatalog.UsageBasedRateCard{
							RateCardMeta: productcatalog.RateCardMeta{
								Key:     apiRequestsTotalFeatureKey,
								Name:    apiRequestsTotalFeatureKey,
								Feature: &apiRequestsTotalFeature,
								Price: productcatalog.NewPriceFrom(productcatalog.UnitPrice{
									Amount: alpacadecimal.NewFromFloat(5),
								}),
							},
							BillingCadence: datex.MustParse(s.T(), "P1M"),
						},
					},
				},
				{
					PhaseMeta: productcatalog.PhaseMeta{
						Name:       "final phase",
						Key:        "final-phase",
						StartAfter: datex.MustParse(s.T(), "P3M"),
					},
					RateCards: productcatalog.RateCards{
						&productcatalog.UsageBasedRateCard{
							RateCardMeta: productcatalog.RateCardMeta{
								Key:     apiRequestsTotalFeatureKey,
								Name:    apiRequestsTotalFeatureKey,
								Feature: &apiRequestsTotalFeature,
								Price: productcatalog.NewPriceFrom(productcatalog.UnitPrice{
									Amount: alpacadecimal.NewFromFloat(10),
								}),
							},
							BillingCadence: datex.MustParse(s.T(), "P1M"),
						},
					},
				},
			},
		},
	})

	s.NoError(err)
	s.NotNil(plan)

	subscriptionPlan, err := s.SubscrpiptionPlanAdapter.GetVersion(ctx, namespace, productcatalogsubscription.PlanRefInput{
		Key:     plan.Key,
		Version: lo.ToPtr(1),
	})
	s.NoError(err)

	subsView, err := s.SubscriptionWorkflowService.CreateFromPlan(ctx, subscription.CreateSubscriptionWorkflowInput{
		ChangeSubscriptionWorkflowInput: subscription.ChangeSubscriptionWorkflowInput{
			ActiveFrom: start,
			Name:       "subs-1",
		},
		Namespace:  namespace,
		CustomerID: customerEntity.ID,
	}, subscriptionPlan)

	s.NoError(err)
	s.NotNil(subsView)

	freeTierPhase := getPhraseByKey(s.T(), subsView, "free-trial")
	s.Equal(lo.ToPtr(datex.MustParse(s.T(), "P1M")), freeTierPhase.ItemsByKey[apiRequestsTotalFeatureKey][0].Spec.RateCard.BillingCadence)

	discountedPhase := getPhraseByKey(s.T(), subsView, "discounted-phase")
	var gatheringInvoiceID billing.InvoiceID

	// let's provision the first set of items
	s.Run("provision first set of items", func() {
		s.NoError(s.Handler.SyncronizeSubscription(ctx, subsView, clock.Now()))

		// then there should be a gathering invoice
		invoices, err := s.BillingService.ListInvoices(ctx, billing.ListInvoicesInput{
			Namespace: namespace,
			Customers: []string{customerEntity.ID},
			Page: pagination.Page{
				PageSize:   10,
				PageNumber: 1,
			},
			Expand: billing.InvoiceExpandAll,
		})
		s.NoError(err)
		s.Len(invoices.Items, 1)

		invoice := invoices.Items[0]
		s.Equal(billing.InvoiceStatusGathering, invoice.Status)
		s.Len(invoice.Lines.OrEmpty(), 1)

		line := invoice.Lines.OrEmpty()[0]
		s.Equal(line.Subscription.SubscriptionID, subsView.Subscription.ID)
		s.Equal(line.Subscription.PhaseID, discountedPhase.SubscriptionPhase.ID)
		s.Equal(line.Subscription.ItemID, discountedPhase.ItemsByKey[apiRequestsTotalFeatureKey][0].SubscriptionItem.ID)
		// 1 month free tier + in arrears billing with 1 month cadence
		s.Equal(line.InvoiceAt, s.mustParseTime("2024-03-01T00:00:00Z"))

		// When we advance the clock the invoice doesn't get changed
		clock.SetTime(s.mustParseTime("2024-02-01T00:00:00Z"))

		s.NoError(s.Handler.SyncronizeSubscription(ctx, subsView, clock.Now()))

		gatheringInvoice, err := s.BillingService.GetInvoiceByID(ctx, billing.GetInvoiceByIdInput{
			Invoice: invoice.InvoiceID(),
			Expand:  billing.InvoiceExpandAll,
		})
		s.NoError(err)
		gatheringInvoiceID = gatheringInvoice.InvoiceID()

		gatheringLine := gatheringInvoice.Lines.OrEmpty()[0]

		// TODO[OM-1039]: the invoice's updated at gets updated even if the invoice is not changed
		s.Equal(billing.InvoiceStatusGathering, gatheringInvoice.Status)
		s.Equal(line.UpdatedAt, gatheringLine.UpdatedAt)
	})

	s.NoError(gatheringInvoiceID.Validate())

	// Progressive billing updates
	s.Run("progressive billing updates", func() {
		s.MockStreamingConnector.AddSimpleEvent(
			apiRequestsTotalMeterSlug,
			100,
			s.mustParseTime("2024-02-02T00:00:00Z"))
		clock.SetTime(s.mustParseTime("2024-02-15T00:00:00Z"))

		// we invoice the customer
		invoices, err := s.BillingService.InvoicePendingLines(ctx, billing.InvoicePendingLinesInput{
			Customer: customerentity.CustomerID{
				ID:        customerEntity.ID,
				Namespace: namespace,
			},
		})
		s.NoError(err)
		s.Len(invoices, 1)
		invoice := invoices[0]

		s.Equal(billing.InvoiceStatusDraftWaitingAutoApproval, invoice.Status)
		s.Equal(float64(5*100), invoice.Totals.Total.InexactFloat64())

		s.Len(invoice.Lines.OrEmpty(), 1)
		line := invoice.Lines.OrEmpty()[0]
		s.Equal(line.Subscription.SubscriptionID, subsView.Subscription.ID)
		s.Equal(line.Subscription.PhaseID, discountedPhase.SubscriptionPhase.ID)
		s.Equal(line.Subscription.ItemID, discountedPhase.ItemsByKey[apiRequestsTotalFeatureKey][0].SubscriptionItem.ID)
		s.Equal(line.InvoiceAt, s.mustParseTime("2024-02-15T00:00:00Z"))
		s.Equal(line.Period, billing.Period{
			Start: s.mustParseTime("2024-02-01T00:00:00Z"),
			End:   s.mustParseTime("2024-02-15T00:00:00Z"),
		})

		// let's fetch the gathering invoice
		gatheringInvoice, err := s.BillingService.GetInvoiceByID(ctx, billing.GetInvoiceByIdInput{
			Invoice: gatheringInvoiceID,
			Expand:  billing.InvoiceExpandAll,
		})
		s.NoError(err)

		s.Len(gatheringInvoice.Lines.OrEmpty(), 1)
		gatheringLine := gatheringInvoice.Lines.OrEmpty()[0]
		s.Equal(gatheringLine.Subscription.SubscriptionID, subsView.Subscription.ID)
		s.Equal(gatheringLine.Subscription.PhaseID, discountedPhase.SubscriptionPhase.ID)
		s.Equal(gatheringLine.Subscription.ItemID, discountedPhase.ItemsByKey[apiRequestsTotalFeatureKey][0].SubscriptionItem.ID)
		s.Equal(gatheringLine.InvoiceAt, s.mustParseTime("2024-03-01T00:00:00Z"))
		s.Equal(gatheringLine.Period, billing.Period{
			Start: s.mustParseTime("2024-02-15T00:00:00Z"),
			End:   s.mustParseTime("2024-03-01T00:00:00Z"),
		})

		// TODO[OM-1037]: let's add/change some items of the subscription then expect that the new item appears on the gathering
		// invoice, but the draft invoice is untouched.
	})

	s.Run("subscription cancellation", func() {
		clock.SetTime(s.mustParseTime("2024-02-20T00:00:00Z"))

		cancelAt := s.mustParseTime("2024-02-22T00:00:00Z")
		subs, err := s.SubscriptionService.Cancel(ctx, models.NamespacedID{
			Namespace: namespace,
			ID:        subsView.Subscription.ID,
		}, cancelAt)
		s.NoError(err)

		subsView, err = s.SubscriptionService.GetView(ctx, models.NamespacedID{
			Namespace: namespace,
			ID:        subs.ID,
		})
		s.NoError(err)

		// Subscription has set the cancellation date, and the view's subscription items are updated to have the cadence
		// set properly up to the cancellation date.

		// If we are now resyncing the subscription, the gathering invoice should be updated to reflect the new cadence.

		s.NoError(s.Handler.SyncronizeSubscription(ctx, subsView, clock.Now()))

		gatheringInvoice, err := s.BillingService.GetInvoiceByID(ctx, billing.GetInvoiceByIdInput{
			Invoice: gatheringInvoiceID,
			Expand:  billing.InvoiceExpandAll.SetSplitLines(true),
		})
		s.NoError(err)

		s.Len(gatheringInvoice.Lines.OrEmpty(), 2)
		gatheringLinesByType := lo.GroupBy(gatheringInvoice.Lines.OrEmpty(), func(line *billing.Line) billing.InvoiceLineStatus {
			return line.Status
		})

		s.Len(gatheringLinesByType[billing.InvoiceLineStatusValid], 1)
		gatheringLine := gatheringLinesByType[billing.InvoiceLineStatusValid][0]

		s.Equal(gatheringLine.Subscription.SubscriptionID, subsView.Subscription.ID)
		s.Equal(gatheringLine.Subscription.PhaseID, discountedPhase.SubscriptionPhase.ID)
		s.Equal(gatheringLine.Subscription.ItemID, discountedPhase.ItemsByKey[apiRequestsTotalFeatureKey][0].SubscriptionItem.ID)

		s.Equal(gatheringLine.Period, billing.Period{
			Start: s.mustParseTime("2024-02-15T00:00:00Z"),
			End:   cancelAt,
		})
		s.Equal(gatheringLine.InvoiceAt, cancelAt)

		// split line
		s.Len(gatheringLinesByType[billing.InvoiceLineStatusSplit], 1)
		splitLine := gatheringLinesByType[billing.InvoiceLineStatusSplit][0]

		s.Equal(splitLine.Subscription.SubscriptionID, subsView.Subscription.ID)
		s.Equal(splitLine.Period, billing.Period{
			Start: s.mustParseTime("2024-02-01T00:00:00Z"),
			End:   s.mustParseTime("2024-02-22T00:00:00Z"),
		})
	})

	s.Run("continue subscription", func() {
		clock.SetTime(s.mustParseTime("2024-02-21T00:00:00Z"))

		subs, err := s.SubscriptionService.Continue(ctx, models.NamespacedID{
			Namespace: namespace,
			ID:        subsView.Subscription.ID,
		})
		s.NoError(err)

		subsView, err = s.SubscriptionService.GetView(ctx, models.NamespacedID{
			Namespace: namespace,
			ID:        subs.ID,
		})
		s.NoError(err)

		// If we are now resyncing the subscription, the gathering invoice should be updated to reflect the original cadence

		s.NoError(s.Handler.SyncronizeSubscription(ctx, subsView, clock.Now()))

		gatheringInvoice, err := s.BillingService.GetInvoiceByID(ctx, billing.GetInvoiceByIdInput{
			Invoice: gatheringInvoiceID,
			Expand:  billing.InvoiceExpandAll.SetSplitLines(true),
		})
		s.NoError(err)

		s.Len(gatheringInvoice.Lines.OrEmpty(), 2)
		gatheringLinesByType := lo.GroupBy(gatheringInvoice.Lines.OrEmpty(), func(line *billing.Line) billing.InvoiceLineStatus {
			return line.Status
		})

		s.Len(gatheringLinesByType[billing.InvoiceLineStatusValid], 1)
		gatheringLine := gatheringLinesByType[billing.InvoiceLineStatusValid][0]

		s.Equal(gatheringLine.Subscription.SubscriptionID, subsView.Subscription.ID)
		s.Equal(gatheringLine.Subscription.PhaseID, discountedPhase.SubscriptionPhase.ID)
		s.Equal(gatheringLine.Subscription.ItemID, discountedPhase.ItemsByKey[apiRequestsTotalFeatureKey][0].SubscriptionItem.ID)

		s.Equal(gatheringLine.Period, billing.Period{
			Start: s.mustParseTime("2024-02-15T00:00:00Z"),
			End:   s.mustParseTime("2024-03-01T00:00:00Z"),
		})
		s.Equal(gatheringLine.InvoiceAt, s.mustParseTime("2024-03-01T00:00:00Z"))

		// split line
		s.Len(gatheringLinesByType[billing.InvoiceLineStatusSplit], 1)
		splitLine := gatheringLinesByType[billing.InvoiceLineStatusSplit][0]

		s.Equal(splitLine.Subscription.SubscriptionID, subsView.Subscription.ID)
		s.Equal(splitLine.Period, billing.Period{
			Start: s.mustParseTime("2024-02-01T00:00:00Z"),
			End:   s.mustParseTime("2024-03-01T00:00:00Z"),
		})
	})
}

func (s *SubscriptionHandlerTestSuite) TestInArrearsProrating() {
	ctx := context.Background()
	namespace := "test-subs-pro-rating"
	start := s.mustParseTime("2024-01-01T00:00:00Z")
	clock.SetTime(start)
	defer clock.ResetTime()

	_ = s.InstallSandboxApp(s.T(), namespace)

	minimalCreateProfileInput := billingtest.MinimalCreateProfileInputTemplate
	minimalCreateProfileInput.Namespace = namespace

	profile, err := s.BillingService.CreateProfile(ctx, minimalCreateProfileInput)
	s.NoError(err)
	s.NotNil(profile)

	customerEntity, err := s.CustomerService.CreateCustomer(ctx, customerentity.CreateCustomerInput{
		Namespace: namespace,

		CustomerMutate: customerentity.CustomerMutate{
			Name:         "Test Customer",
			PrimaryEmail: lo.ToPtr("test@test.com"),
			BillingAddress: &models.Address{
				Country: lo.ToPtr(models.CountryCode("US")),
			},
			Currency: lo.ToPtr(currencyx.Code(currency.USD)),
			UsageAttribution: customerentity.CustomerUsageAttribution{
				SubjectKeys: []string{"test"},
			},
		},
	})
	require.NoError(s.T(), err)
	require.NotNil(s.T(), customerEntity)
	require.NotEmpty(s.T(), customerEntity.ID)

	plan, err := s.PlanService.CreatePlan(ctx, plan.CreatePlanInput{
		NamespacedModel: models.NamespacedModel{
			Namespace: namespace,
		},
		Plan: productcatalog.Plan{
			PlanMeta: productcatalog.PlanMeta{
				Name:     "Test Plan",
				Key:      "test-plan",
				Version:  1,
				Currency: currency.USD,
			},

			Phases: []productcatalog.Phase{
				{
					PhaseMeta: productcatalog.PhaseMeta{
						Name:       "first-phase",
						Key:        "first-phase",
						StartAfter: datex.MustParse(s.T(), "P0D"),
					},
					RateCards: productcatalog.RateCards{
						&productcatalog.UsageBasedRateCard{
							RateCardMeta: productcatalog.RateCardMeta{
								Key:  "in-arrears",
								Name: "in-arrears",
								Price: productcatalog.NewPriceFrom(productcatalog.FlatPrice{
									Amount:      alpacadecimal.NewFromFloat(5),
									PaymentTerm: productcatalog.InArrearsPaymentTerm,
								}),
							},
							BillingCadence: datex.MustParse(s.T(), "P1D"),
						},
					},
				},
			},
		},
	})

	s.NoError(err)
	s.NotNil(plan)

	subscriptionPlan, err := s.SubscrpiptionPlanAdapter.GetVersion(ctx, namespace, productcatalogsubscription.PlanRefInput{
		Key:     plan.Key,
		Version: lo.ToPtr(1),
	})
	s.NoError(err)

	subsView, err := s.SubscriptionWorkflowService.CreateFromPlan(ctx, subscription.CreateSubscriptionWorkflowInput{
		ChangeSubscriptionWorkflowInput: subscription.ChangeSubscriptionWorkflowInput{
			ActiveFrom: start,
			Name:       "subs-1",
		},
		Namespace:  namespace,
		CustomerID: customerEntity.ID,
	}, subscriptionPlan)

	s.NoError(err)
	s.NotNil(subsView)

	// let's provision the first set of items
	s.Run("provision first set of items", func() {
		s.NoError(s.Handler.SyncronizeSubscription(ctx, subsView, clock.Now()))

		// then there should be a gathering invoice
		invoices, err := s.BillingService.ListInvoices(ctx, billing.ListInvoicesInput{
			Namespace: namespace,
			Customers: []string{customerEntity.ID},
			Page: pagination.Page{
				PageSize:   10,
				PageNumber: 1,
			},
			Expand: billing.InvoiceExpandAll,
		})
		s.NoError(err)
		s.Len(invoices.Items, 1)

		lines := invoices.Items[0].Lines.OrEmpty()
		s.Len(lines, 1)

		flatFeeLine := lines[0]
		s.Equal(flatFeeLine.Subscription.SubscriptionID, subsView.Subscription.ID)
		s.Equal(flatFeeLine.Subscription.PhaseID, subsView.Phases[0].SubscriptionPhase.ID)
		s.Equal(flatFeeLine.Subscription.ItemID, subsView.Phases[0].ItemsByKey["in-arrears"][0].SubscriptionItem.ID)
		s.Equal(flatFeeLine.InvoiceAt, s.mustParseTime("2024-01-02T00:00:00Z"))
		s.Equal(flatFeeLine.Period, billing.Period{
			Start: s.mustParseTime("2024-01-01T00:00:00Z"),
			End:   s.mustParseTime("2024-01-02T00:00:00Z"),
		})
		s.Equal(flatFeeLine.FlatFee.PerUnitAmount.InexactFloat64(), 5.0)
		s.Equal(flatFeeLine.FlatFee.Quantity.InexactFloat64(), 1.0)
	})

	s.Run("canceling the subscription causes the existing item to be pro-rated", func() {
		clock.SetTime(s.mustParseTime("2024-01-01T10:00:00Z"))

		cancelAt := s.mustParseTime("2024-01-01T12:00:00Z")
		subs, err := s.SubscriptionService.Cancel(ctx, models.NamespacedID{
			Namespace: namespace,
			ID:        subsView.Subscription.ID,
		}, cancelAt)
		s.NoError(err)

		subsView, err = s.SubscriptionService.GetView(ctx, models.NamespacedID{
			Namespace: namespace,
			ID:        subs.ID,
		})
		s.NoError(err)

		s.NoError(s.Handler.SyncronizeSubscription(ctx, subsView, clock.Now()))

		// then there should be a gathering invoice
		invoices, err := s.BillingService.ListInvoices(ctx, billing.ListInvoicesInput{
			Namespace: namespace,
			Customers: []string{customerEntity.ID},
			Page: pagination.Page{
				PageSize:   10,
				PageNumber: 1,
			},
			Expand: billing.InvoiceExpandAll,
		})
		s.NoError(err)
		s.Len(invoices.Items, 1)

		lines := invoices.Items[0].Lines.OrEmpty()
		s.Len(lines, 1)

		flatFeeLine := lines[0]
		s.Equal(flatFeeLine.Subscription.SubscriptionID, subsView.Subscription.ID)
		s.Equal(flatFeeLine.InvoiceAt, cancelAt)
		s.Equal(flatFeeLine.Period, billing.Period{
			Start: s.mustParseTime("2024-01-01T00:00:00Z"),
			End:   cancelAt,
		})
		s.Equal(flatFeeLine.FlatFee.PerUnitAmount.InexactFloat64(), 2.5)
		s.Equal(flatFeeLine.FlatFee.Quantity.InexactFloat64(), 1.0)
	})
}

func getPhraseByKey(t *testing.T, subsView subscription.SubscriptionView, key string) subscription.SubscriptionPhaseView {
	for _, phase := range subsView.Phases {
		if phase.SubscriptionPhase.Key == key {
			return phase
		}
	}

	t.Fatalf("phase with key %s not found", key)
	return subscription.SubscriptionPhaseView{}
}
