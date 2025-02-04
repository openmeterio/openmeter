package billingworkersubscription

import (
	"context"
	"fmt"
	"log/slog"
	"testing"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/invopop/gobl/currency"
	"github.com/oklog/ulid/v2"
	"github.com/samber/lo"
	"github.com/samber/mo"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/credit"
	grantrepo "github.com/openmeterio/openmeter/openmeter/credit/adapter"
	"github.com/openmeterio/openmeter/openmeter/customer"
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
	productcatalogsubscription "github.com/openmeterio/openmeter/openmeter/productcatalog/subscription"
	subscriptiontestutils "github.com/openmeterio/openmeter/openmeter/productcatalog/subscription/testutils"
	"github.com/openmeterio/openmeter/openmeter/subscription"
	subscriptionentitlementadatapter "github.com/openmeterio/openmeter/openmeter/subscription/adapters/entitlement"
	"github.com/openmeterio/openmeter/openmeter/subscription/patch"
	subscriptionrepo "github.com/openmeterio/openmeter/openmeter/subscription/repo"
	subscriptionservice "github.com/openmeterio/openmeter/openmeter/subscription/service"
	"github.com/openmeterio/openmeter/openmeter/testutils"
	"github.com/openmeterio/openmeter/openmeter/watermill/eventbus"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/datex"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/pagination"
	billingtest "github.com/openmeterio/openmeter/test/billing"
)

type SubscriptionHandlerTestSuite struct {
	billingtest.BaseSuite

	PlanService                 plan.Service
	SubscriptionService         subscription.Service
	SubscriptionPlanAdapter     subscriptiontestutils.PlanSubscriptionAdapter
	SubscriptionWorkflowService subscription.WorkflowService

	Namespace               string
	Customer                *customer.Customer
	APIRequestsTotalFeature feature.Feature
	Context                 context.Context

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

	s.SubscriptionPlanAdapter = subscriptiontestutils.NewPlanSubscriptionAdapter(subscriptiontestutils.PlanSubscriptionAdapterConfig{
		PlanService: planService,
		Logger:      slog.Default(),
	})

	s.SubscriptionWorkflowService = subscriptionservice.NewWorkflowService(subscriptionservice.WorkflowServiceConfig{
		Service:            s.SubscriptionService,
		CustomerService:    s.CustomerService,
		TransactionManager: subsRepo,
	})

	handler, err := New(Config{
		BillingService:      s.BillingService,
		Logger:              slog.Default(),
		TxCreator:           s.BillingAdapter,
		SubscriptionService: s.SubscriptionService,
	})
	s.NoError(err)

	s.Handler = handler
}

func (s *SubscriptionHandlerTestSuite) BeforeTest(suiteName, testName string) {
	s.Namespace = "test-subs-update-" + ulid.Make().String()
	// TODO: go 1.24, let's use T()'s context
	s.Context = context.Background()

	ctx := s.Context

	_ = s.InstallSandboxApp(s.T(), s.Namespace)

	minimalCreateProfileInput := billingtest.MinimalCreateProfileInputTemplate
	minimalCreateProfileInput.Namespace = s.Namespace

	profile, err := s.BillingService.CreateProfile(ctx, minimalCreateProfileInput)
	s.NoError(err)
	s.NotNil(profile)

	apiRequestsTotalMeterSlug := "api-requests-total"

	s.MeterRepo.ReplaceMeters(ctx, []models.Meter{
		{
			Namespace:   s.Namespace,
			Slug:        apiRequestsTotalMeterSlug,
			WindowSize:  models.WindowSizeMinute,
			Aggregation: models.MeterAggregationSum,
		},
	})

	apiRequestsTotalFeatureKey := "api-requests-total"

	apiRequestsTotalFeature, err := s.FeatureService.CreateFeature(ctx, feature.CreateFeatureInputs{
		Namespace: s.Namespace,
		Name:      "api-requests-total",
		Key:       apiRequestsTotalFeatureKey,
		MeterSlug: lo.ToPtr("api-requests-total"),
	})
	s.NoError(err)
	s.APIRequestsTotalFeature = apiRequestsTotalFeature

	customerEntity := s.CreateTestCustomer(s.Namespace, "test")
	require.NotNil(s.T(), customerEntity)
	require.NotEmpty(s.T(), customerEntity.ID)

	s.Customer = customerEntity
}

func (s *SubscriptionHandlerTestSuite) AfterTest(suiteName, testName string) {
	clock.UnFreeze()
	clock.ResetTime()
	s.MeterRepo.ReplaceMeters(s.Context, []models.Meter{})
	s.MockStreamingConnector.Reset()
	s.Handler.featureFlags = FeatureFlags{}
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
	s.T().Helper()
	return lo.Must(time.Parse(time.RFC3339, t))
}

func (s *SubscriptionHandlerTestSuite) TestSubscriptionHappyPath() {
	ctx := s.Context
	namespace := s.Namespace
	start := s.mustParseTime("2024-01-01T00:00:00Z")
	clock.SetTime(start)
	defer clock.ResetTime()
	defer s.MockStreamingConnector.Reset()

	_ = s.InstallSandboxApp(s.T(), namespace)

	s.enableProgressiveBilling()

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
						Name:     "free trial",
						Key:      "free-trial",
						Duration: lo.ToPtr(testutils.GetISODuration(s.T(), "P1M")),
					},
					// TODO[OM-1031]: let's add discount handling (as this could be a 100% discount for the first month)
					RateCards: productcatalog.RateCards{
						&productcatalog.UsageBasedRateCard{
							RateCardMeta: productcatalog.RateCardMeta{
								Key:     s.APIRequestsTotalFeature.Key,
								Name:    s.APIRequestsTotalFeature.Key,
								Feature: &s.APIRequestsTotalFeature,
							},
							BillingCadence: datex.MustParse(s.T(), "P1M"),
						},
					},
				},
				{
					PhaseMeta: productcatalog.PhaseMeta{
						Name:     "discounted phase",
						Key:      "discounted-phase",
						Duration: lo.ToPtr(testutils.GetISODuration(s.T(), "P2M")),
					},
					// TODO[OM-1031]: 50% discount
					RateCards: productcatalog.RateCards{
						&productcatalog.UsageBasedRateCard{
							RateCardMeta: productcatalog.RateCardMeta{
								Key:     s.APIRequestsTotalFeature.Key,
								Name:    s.APIRequestsTotalFeature.Key,
								Feature: &s.APIRequestsTotalFeature,
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
						Name:     "final phase",
						Key:      "final-phase",
						Duration: nil,
					},
					RateCards: productcatalog.RateCards{
						&productcatalog.UsageBasedRateCard{
							RateCardMeta: productcatalog.RateCardMeta{
								Key:     s.APIRequestsTotalFeature.Key,
								Name:    s.APIRequestsTotalFeature.Key,
								Feature: &s.APIRequestsTotalFeature,
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

	subscriptionPlan, err := s.SubscriptionPlanAdapter.GetVersion(ctx, namespace, productcatalogsubscription.PlanRefInput{
		Key:     plan.Key,
		Version: lo.ToPtr(1),
	})
	s.NoError(err)

	subsView, err := s.SubscriptionWorkflowService.CreateFromPlan(ctx, subscription.CreateSubscriptionWorkflowInput{
		ChangeSubscriptionWorkflowInput: subscription.ChangeSubscriptionWorkflowInput{
			Timing: subscription.Timing{
				Custom: lo.ToPtr(start),
			},
			Name: "subs-1",
		},
		Namespace:  namespace,
		CustomerID: s.Customer.ID,
	}, subscriptionPlan)

	s.NoError(err)
	s.NotNil(subsView)

	freeTierPhase := getPhraseByKey(s.T(), subsView, "free-trial")
	s.Equal(lo.ToPtr(datex.MustParse(s.T(), "P1M")), freeTierPhase.ItemsByKey[s.APIRequestsTotalFeature.Key][0].Spec.RateCard.BillingCadence)

	discountedPhase := getPhraseByKey(s.T(), subsView, "discounted-phase")
	var gatheringInvoiceID billing.InvoiceID

	// let's provision the first set of items
	s.Run("provision first set of items", func() {
		s.NoError(s.Handler.SyncronizeSubscription(ctx, subsView, clock.Now()))

		// then there should be a gathering invoice
		invoice := s.gatheringInvoice(ctx, namespace, s.Customer.ID)
		invoiceUpdatedAt := invoice.UpdatedAt

		s.Len(invoice.Lines.OrEmpty(), 1)

		line := invoice.Lines.OrEmpty()[0]
		s.Equal(line.Subscription.SubscriptionID, subsView.Subscription.ID)
		s.Equal(line.Subscription.PhaseID, discountedPhase.SubscriptionPhase.ID)
		s.Equal(line.Subscription.ItemID, discountedPhase.ItemsByKey[s.APIRequestsTotalFeature.Key][0].SubscriptionItem.ID)

		// 1 month free tier + in arrears billing with 1 month cadence
		s.Equal(line.InvoiceAt, s.mustParseTime("2024-03-01T00:00:00Z"))

		// When we advance the clock the invoice doesn't get changed
		clock.SetTime(s.mustParseTime("2024-02-01T00:00:00Z"))
		s.NoError(s.Handler.SyncronizeSubscription(ctx, subsView, clock.Now()))

		gatheringInvoice := s.gatheringInvoice(ctx, namespace, s.Customer.ID)
		s.NoError(err)
		gatheringInvoiceID = gatheringInvoice.InvoiceID()

		s.DebugDumpInvoice("gathering invoice - 2nd update", gatheringInvoice)

		gatheringLine := gatheringInvoice.Lines.OrEmpty()[0]

		s.Equal(invoiceUpdatedAt, gatheringInvoice.UpdatedAt)
		s.Equal(billing.InvoiceStatusGathering, gatheringInvoice.Status)
		s.Equal(line.UpdatedAt, gatheringLine.UpdatedAt)
	})

	s.NoError(gatheringInvoiceID.Validate())

	// Progressive billing updates
	s.Run("progressive billing updates", func() {
		s.MockStreamingConnector.AddSimpleEvent(
			*s.APIRequestsTotalFeature.MeterSlug,
			100,
			s.mustParseTime("2024-02-02T00:00:00Z"))
		clock.SetTime(s.mustParseTime("2024-02-15T00:00:01Z"))

		// we invoice the customer
		invoices, err := s.BillingService.InvoicePendingLines(ctx, billing.InvoicePendingLinesInput{
			Customer: customer.CustomerID{
				ID:        s.Customer.ID,
				Namespace: namespace,
			},
			AsOf: lo.ToPtr(s.mustParseTime("2024-02-15T00:00:00Z")),
		})
		if err != nil {
			fmt.Printf("current time: %s\n", clock.Now().Format(time.RFC3339))
		}
		s.NoError(err)
		s.Len(invoices, 1)
		invoice := invoices[0]

		s.Equal(billing.InvoiceStatusDraftWaitingAutoApproval, invoice.Status)
		s.Equal(float64(5*100), invoice.Totals.Total.InexactFloat64())

		s.Len(invoice.Lines.OrEmpty(), 1)
		line := invoice.Lines.OrEmpty()[0]
		s.Equal(line.Subscription.SubscriptionID, subsView.Subscription.ID)
		s.Equal(line.Subscription.PhaseID, discountedPhase.SubscriptionPhase.ID)
		s.Equal(line.Subscription.ItemID, discountedPhase.ItemsByKey[s.APIRequestsTotalFeature.Key][0].SubscriptionItem.ID)
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
		s.Equal(gatheringLine.Subscription.ItemID, discountedPhase.ItemsByKey[s.APIRequestsTotalFeature.Key][0].SubscriptionItem.ID)
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
		}, subscription.Timing{
			Custom: lo.ToPtr(cancelAt),
		})
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
		s.Equal(gatheringLine.Subscription.ItemID, discountedPhase.ItemsByKey[s.APIRequestsTotalFeature.Key][0].SubscriptionItem.ID)

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
		s.Equal(gatheringLine.Subscription.ItemID, discountedPhase.ItemsByKey[s.APIRequestsTotalFeature.Key][0].SubscriptionItem.ID)

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
	s.enableProrating()

	_ = s.InstallSandboxApp(s.T(), namespace)

	minimalCreateProfileInput := billingtest.MinimalCreateProfileInputTemplate
	minimalCreateProfileInput.Namespace = namespace

	profile, err := s.BillingService.CreateProfile(ctx, minimalCreateProfileInput)
	s.NoError(err)
	s.NotNil(profile)

	customerEntity := s.CreateTestCustomer(namespace, "test")
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
						Name:     "first-phase",
						Key:      "first-phase",
						Duration: nil,
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

	subscriptionPlan, err := s.SubscriptionPlanAdapter.GetVersion(ctx, namespace, productcatalogsubscription.PlanRefInput{
		Key:     plan.Key,
		Version: lo.ToPtr(1),
	})
	s.NoError(err)

	subsView, err := s.SubscriptionWorkflowService.CreateFromPlan(ctx, subscription.CreateSubscriptionWorkflowInput{
		ChangeSubscriptionWorkflowInput: subscription.ChangeSubscriptionWorkflowInput{
			Timing: subscription.Timing{
				Custom: lo.ToPtr(start),
			},
			Name: "subs-1",
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
			Namespaces: []string{namespace},
			Customers:  []string{customerEntity.ID},
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
		}, subscription.Timing{
			Custom: lo.ToPtr(cancelAt),
		})
		s.NoError(err)

		subsView, err = s.SubscriptionService.GetView(ctx, models.NamespacedID{
			Namespace: namespace,
			ID:        subs.ID,
		})
		s.NoError(err)

		s.NoError(s.Handler.SyncronizeSubscription(ctx, subsView, clock.Now()))

		// then there should be a gathering invoice
		invoices, err := s.BillingService.ListInvoices(ctx, billing.ListInvoicesInput{
			Namespaces: []string{namespace},
			Customers:  []string{customerEntity.ID},
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

func (s *SubscriptionHandlerTestSuite) TestInAdvanceGatheringSyncNonBillableAmountProrated() {
	ctx := s.Context
	clock.FreezeTime(s.mustParseTime("2024-01-01T00:00:00Z"))
	s.enableProrating()

	// Given
	//  we have a subscription with a single phase with a single static fee
	// When
	//  we edit the subscription quite fast to change the fee
	// Then
	//  the gathering invoice will only contain the new version of the fee, as the old one was
	//  pro-rated and the total amount is 0

	subsView := s.createSubscriptionFromPlanPhases([]productcatalog.Phase{
		{
			PhaseMeta: s.phaseMeta("first-phase", ""),
			RateCards: productcatalog.RateCards{
				&productcatalog.UsageBasedRateCard{
					RateCardMeta: productcatalog.RateCardMeta{
						Key:  "in-advance",
						Name: "in-advance",
						Price: productcatalog.NewPriceFrom(productcatalog.FlatPrice{
							Amount:      alpacadecimal.NewFromFloat(5),
							PaymentTerm: productcatalog.InAdvancePaymentTerm,
						}),
					},
					BillingCadence: datex.MustParse(s.T(), "P1D"),
				},
			},
		},
	})

	s.NoError(s.Handler.SyncronizeSubscription(ctx, subsView, s.mustParseTime("2024-01-05T12:00:00Z")))
	s.DebugDumpInvoice("gathering invoice", s.gatheringInvoice(ctx, s.Namespace, s.Customer.ID))

	clock.FreezeTime(s.mustParseTime("2024-01-01T00:00:40Z"))

	updatedSubsView, err := s.SubscriptionWorkflowService.EditRunning(ctx, subsView.Subscription.NamespacedID, []subscription.Patch{
		patch.PatchRemoveItem{
			PhaseKey: "first-phase",
			ItemKey:  "in-advance",
		},
		subscriptionAddItem{
			PhaseKey: "first-phase",
			ItemKey:  "in-advance",
			Price: productcatalog.NewPriceFrom(productcatalog.FlatPrice{
				Amount:      alpacadecimal.NewFromFloat(10),
				PaymentTerm: productcatalog.InAdvancePaymentTerm,
			}),
			BillingCadence: lo.ToPtr(datex.MustParse(s.T(), "P1D")),
		}.AsPatch(),
	})
	s.NoError(err)
	s.NotNil(updatedSubsView)

	s.NoError(s.Handler.SyncronizeSubscription(ctx, updatedSubsView, s.mustParseTime("2024-01-05T12:00:00Z")))

	gatheringInvoice := s.gatheringInvoice(ctx, s.Namespace, s.Customer.ID)
	s.DebugDumpInvoice("gathering invoice - 2nd sync", gatheringInvoice)

	s.expectLines(gatheringInvoice, subsView.Subscription.ID, []expectedLine{
		{
			Matcher: recurringLineMatcher{
				PhaseKey:  "first-phase",
				ItemKey:   "in-advance",
				Version:   1,
				PeriodMin: 0,
				PeriodMax: 4,
			},

			Qty:       mo.Some[float64](1),
			UnitPrice: mo.Some[float64](10),
			Periods:   s.generatePeriods("2024-01-01T00:00:40Z", "2024-01-02T00:00:40Z", "P1D", 5),
			InvoiceAt: s.generateDailyTimestamps("2024-01-01T00:00:40Z", 5),
		},
	})
}

func (s *SubscriptionHandlerTestSuite) TestInAdvanceGatheringSyncNonBillableAmount() {
	ctx := s.Context
	clock.FreezeTime(s.mustParseTime("2024-01-01T00:00:00Z"))

	// Given
	//  we have a subscription with a single phase with a single static fee
	// When
	//  we edit the subscription quite fast to change the fee
	// Then
	//  the gathering invoice will only contain both versions of the fee as we are not
	//  doing any pro-rating logic

	subsView := s.createSubscriptionFromPlanPhases([]productcatalog.Phase{
		{
			PhaseMeta: s.phaseMeta("first-phase", ""),
			RateCards: productcatalog.RateCards{
				&productcatalog.UsageBasedRateCard{
					RateCardMeta: productcatalog.RateCardMeta{
						Key:  "in-advance",
						Name: "in-advance",
						Price: productcatalog.NewPriceFrom(productcatalog.FlatPrice{
							Amount:      alpacadecimal.NewFromFloat(5),
							PaymentTerm: productcatalog.InAdvancePaymentTerm,
						}),
					},
					BillingCadence: datex.MustParse(s.T(), "P1D"),
				},
			},
		},
	})

	s.NoError(s.Handler.SyncronizeSubscription(ctx, subsView, s.mustParseTime("2024-01-05T12:00:00Z")))
	s.DebugDumpInvoice("gathering invoice", s.gatheringInvoice(ctx, s.Namespace, s.Customer.ID))

	clock.FreezeTime(s.mustParseTime("2024-01-01T00:00:40Z"))

	updatedSubsView, err := s.SubscriptionWorkflowService.EditRunning(ctx, subsView.Subscription.NamespacedID, []subscription.Patch{
		patch.PatchRemoveItem{
			PhaseKey: "first-phase",
			ItemKey:  "in-advance",
		},
		subscriptionAddItem{
			PhaseKey: "first-phase",
			ItemKey:  "in-advance",
			Price: productcatalog.NewPriceFrom(productcatalog.FlatPrice{
				Amount:      alpacadecimal.NewFromFloat(10),
				PaymentTerm: productcatalog.InAdvancePaymentTerm,
			}),
			BillingCadence: lo.ToPtr(datex.MustParse(s.T(), "P1D")),
		}.AsPatch(),
	})
	s.NoError(err)
	s.NotNil(updatedSubsView)

	s.NoError(s.Handler.SyncronizeSubscription(ctx, updatedSubsView, s.mustParseTime("2024-01-05T12:00:00Z")))

	gatheringInvoice := s.gatheringInvoice(ctx, s.Namespace, s.Customer.ID)
	s.DebugDumpInvoice("gathering invoice - 2nd sync", gatheringInvoice)

	s.expectLines(gatheringInvoice, subsView.Subscription.ID, []expectedLine{
		{
			Matcher: recurringLineMatcher{
				PhaseKey:  "first-phase",
				ItemKey:   "in-advance",
				Version:   0,
				PeriodMin: 0,
				PeriodMax: 0,
			},

			Qty:       mo.Some[float64](1),
			UnitPrice: mo.Some[float64](5),
			Periods: []billing.Period{
				{
					Start: s.mustParseTime("2024-01-01T00:00:00Z"),
					End:   s.mustParseTime("2024-01-01T00:00:40Z"),
				},
			},
			InvoiceAt: []time.Time{s.mustParseTime("2024-01-01T00:00:00Z")},
		},
		{
			Matcher: recurringLineMatcher{
				PhaseKey:  "first-phase",
				ItemKey:   "in-advance",
				Version:   1,
				PeriodMin: 0,
				PeriodMax: 4,
			},

			Qty:       mo.Some[float64](1),
			UnitPrice: mo.Some[float64](10),
			Periods:   s.generatePeriods("2024-01-01T00:00:40Z", "2024-01-02T00:00:40Z", "P1D", 5),
			InvoiceAt: s.generateDailyTimestamps("2024-01-01T00:00:40Z", 5),
		},
	})
}

func (s *SubscriptionHandlerTestSuite) TestInArrearsGatheringSyncNonBillableAmount() {
	ctx := s.Context
	clock.FreezeTime(s.mustParseTime("2024-01-01T00:00:00Z"))

	// Given
	//  we have a subscription with a single phase with a single static fee in arrears
	// When
	//  we edit the subscription quite fast to change the fee
	// Then
	//  the gathering invoice will only contain both versions of the fee as we are not
	//  doing any pro-rating logic

	subsView := s.createSubscriptionFromPlanPhases([]productcatalog.Phase{
		{
			PhaseMeta: s.phaseMeta("first-phase", ""),
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
	})

	s.NoError(s.Handler.SyncronizeSubscription(ctx, subsView, s.mustParseTime("2024-01-05T12:00:00Z")))
	s.DebugDumpInvoice("gathering invoice", s.gatheringInvoice(ctx, s.Namespace, s.Customer.ID))

	clock.FreezeTime(s.mustParseTime("2024-01-01T00:00:40Z"))

	updatedSubsView, err := s.SubscriptionWorkflowService.EditRunning(ctx, subsView.Subscription.NamespacedID, []subscription.Patch{
		patch.PatchRemoveItem{
			PhaseKey: "first-phase",
			ItemKey:  "in-arrears",
		},
		subscriptionAddItem{
			PhaseKey: "first-phase",
			ItemKey:  "in-arrears",
			Price: productcatalog.NewPriceFrom(productcatalog.FlatPrice{
				Amount:      alpacadecimal.NewFromFloat(10),
				PaymentTerm: productcatalog.InArrearsPaymentTerm,
			}),
			BillingCadence: lo.ToPtr(datex.MustParse(s.T(), "P1D")),
		}.AsPatch(),
	})
	s.NoError(err)
	s.NotNil(updatedSubsView)

	s.NoError(s.Handler.SyncronizeSubscription(ctx, updatedSubsView, s.mustParseTime("2024-01-05T12:00:00Z")))

	gatheringInvoice := s.gatheringInvoice(ctx, s.Namespace, s.Customer.ID)
	s.DebugDumpInvoice("gathering invoice - 2nd sync", gatheringInvoice)

	s.expectLines(gatheringInvoice, subsView.Subscription.ID, []expectedLine{
		{
			Matcher: recurringLineMatcher{
				PhaseKey:  "first-phase",
				ItemKey:   "in-arrears",
				Version:   0,
				PeriodMin: 0,
				PeriodMax: 0,
			},

			Qty:       mo.Some[float64](1),
			UnitPrice: mo.Some[float64](5),
			Periods: []billing.Period{
				{
					Start: s.mustParseTime("2024-01-01T00:00:00Z"),
					End:   s.mustParseTime("2024-01-01T00:00:40Z"),
				},
			},
			InvoiceAt: []time.Time{s.mustParseTime("2024-01-01T00:00:40Z")},
		},
		{
			Matcher: recurringLineMatcher{
				PhaseKey:  "first-phase",
				ItemKey:   "in-arrears",
				Version:   1,
				PeriodMin: 0,
				PeriodMax: 4,
			},

			Qty:       mo.Some[float64](1),
			UnitPrice: mo.Some[float64](10),
			Periods:   s.generatePeriods("2024-01-01T00:00:40Z", "2024-01-02T00:00:40Z", "P1D", 5),
			InvoiceAt: s.generateDailyTimestamps("2024-01-02T00:00:40Z", 5),
		},
	})
}

func (s *SubscriptionHandlerTestSuite) TestInAdvanceGatheringSyncBillableAmountProrated() {
	ctx := s.Context
	clock.FreezeTime(s.mustParseTime("2024-01-01T00:00:00Z"))
	s.enableProrating()

	// Given
	//  we have a subscription with a single phase with a single static fee
	// When
	//  we edit the subscription later
	// Then
	//  the gathering invoice will contain the pro-rated previous fee and the new fee
	//  with shifted periods

	subsView := s.createSubscriptionFromPlanPhases([]productcatalog.Phase{
		{
			PhaseMeta: s.phaseMeta("first-phase", ""),
			RateCards: productcatalog.RateCards{
				&productcatalog.UsageBasedRateCard{
					RateCardMeta: productcatalog.RateCardMeta{
						Key:  "in-advance",
						Name: "in-advance",
						Price: productcatalog.NewPriceFrom(productcatalog.FlatPrice{
							Amount:      alpacadecimal.NewFromFloat(6),
							PaymentTerm: productcatalog.InAdvancePaymentTerm,
						}),
					},
					BillingCadence: datex.MustParse(s.T(), "P1D"),
				},
			},
		},
	})

	s.NoError(s.Handler.SyncronizeSubscription(ctx, subsView, s.mustParseTime("2024-01-05T12:00:00Z")))
	s.DebugDumpInvoice("gathering invoice", s.gatheringInvoice(ctx, s.Namespace, s.Customer.ID))

	clock.FreezeTime(s.mustParseTime("2024-01-01T12:00:00Z"))

	updatedSubsView, err := s.SubscriptionWorkflowService.EditRunning(ctx, subsView.Subscription.NamespacedID, []subscription.Patch{
		patch.PatchRemoveItem{
			PhaseKey: "first-phase",
			ItemKey:  "in-advance",
		},
		subscriptionAddItem{
			PhaseKey: "first-phase",
			ItemKey:  "in-advance",
			Price: productcatalog.NewPriceFrom(productcatalog.FlatPrice{
				Amount:      alpacadecimal.NewFromFloat(10),
				PaymentTerm: productcatalog.InAdvancePaymentTerm,
			}),
			BillingCadence: lo.ToPtr(datex.MustParse(s.T(), "P1D")),
		}.AsPatch(),
	})
	s.NoError(err)
	s.NotNil(updatedSubsView)

	s.NoError(s.Handler.SyncronizeSubscription(ctx, updatedSubsView, s.mustParseTime("2024-01-05T12:00:00Z")))

	gatheringInvoice := s.gatheringInvoice(ctx, s.Namespace, s.Customer.ID)
	s.DebugDumpInvoice("gathering invoice - 2nd sync", gatheringInvoice)

	s.expectLines(gatheringInvoice, subsView.Subscription.ID, []expectedLine{
		{
			Matcher: recurringLineMatcher{
				PhaseKey:  "first-phase",
				ItemKey:   "in-advance",
				Version:   0,
				PeriodMin: 0,
				PeriodMax: 0,
			},

			Qty:       mo.Some[float64](1),
			UnitPrice: mo.Some[float64](3),
			Periods: []billing.Period{
				{
					Start: s.mustParseTime("2024-01-01T00:00:00Z"),
					End:   s.mustParseTime("2024-01-01T12:00:00Z"),
				},
			},
			InvoiceAt: []time.Time{s.mustParseTime("2024-01-01T00:00:00Z")},
		},
		{
			Matcher: recurringLineMatcher{
				PhaseKey:  "first-phase",
				ItemKey:   "in-advance",
				Version:   1,
				PeriodMin: 0,
				PeriodMax: 3,
			},

			Qty:       mo.Some[float64](1),
			UnitPrice: mo.Some[float64](10),
			Periods:   s.generatePeriods("2024-01-01T12:00:00Z", "2024-01-02T12:00:00Z", "P1D", 4),
			InvoiceAt: s.generateDailyTimestamps("2024-01-01T12:00:00Z", 4),
		},
	})
}

func (s *SubscriptionHandlerTestSuite) TestInAdvanceGatheringSyncDraftInvoiceProrated() {
	ctx := s.Context
	clock.FreezeTime(s.mustParseTime("2024-01-01T00:00:00Z"))
	s.enableProrating()

	// Given
	//  we have a subscription with a single phase with a single static fee
	// When
	//  we have an outstanding draft invoice and we edit the subscription later
	// Then
	//  then the draft invoice gets updated with the new pro-rated fee and the new fee
	//  item will be available as a gathering invoice

	subsView := s.createSubscriptionFromPlanPhases([]productcatalog.Phase{
		{
			PhaseMeta: s.phaseMeta("first-phase", ""),
			RateCards: productcatalog.RateCards{
				&productcatalog.UsageBasedRateCard{
					RateCardMeta: productcatalog.RateCardMeta{
						Key:  "in-advance",
						Name: "in-advance",
						Price: productcatalog.NewPriceFrom(productcatalog.FlatPrice{
							Amount:      alpacadecimal.NewFromFloat(6),
							PaymentTerm: productcatalog.InAdvancePaymentTerm,
						}),
					},
					BillingCadence: datex.MustParse(s.T(), "P1D"),
				},
			},
		},
	})

	s.NoError(s.Handler.SyncronizeSubscription(ctx, subsView, s.mustParseTime("2024-01-05T12:00:00Z")))
	s.DebugDumpInvoice("gathering invoice", s.gatheringInvoice(ctx, s.Namespace, s.Customer.ID))

	clock.FreezeTime(s.mustParseTime("2024-01-01T12:00:00Z"))

	draftInvoices, err := s.BillingService.InvoicePendingLines(ctx, billing.InvoicePendingLinesInput{
		Customer: s.Customer.GetID(),
		AsOf:     lo.ToPtr(clock.Now()),
	})
	s.NoError(err)
	s.Require().Len(draftInvoices, 1)

	s.DebugDumpInvoice("draft invoice", draftInvoices[0])

	draftInvoice := draftInvoices[0]
	s.expectLines(draftInvoice, subsView.Subscription.ID, []expectedLine{
		{
			Matcher: recurringLineMatcher{
				PhaseKey:  "first-phase",
				ItemKey:   "in-advance",
				Version:   0,
				PeriodMin: 0,
				PeriodMax: 0,
			},
			Qty:       mo.Some[float64](1),
			UnitPrice: mo.Some[float64](6),
			Periods: []billing.Period{
				{
					Start: s.mustParseTime("2024-01-01T00:00:00Z"),
					End:   s.mustParseTime("2024-01-02T00:00:00Z"),
				},
			},
			InvoiceAt: []time.Time{s.mustParseTime("2024-01-01T00:00:00Z")},
		},
	})

	updatedSubsView, err := s.SubscriptionWorkflowService.EditRunning(ctx, subsView.Subscription.NamespacedID, []subscription.Patch{
		patch.PatchRemoveItem{
			PhaseKey: "first-phase",
			ItemKey:  "in-advance",
		},
		subscriptionAddItem{
			PhaseKey: "first-phase",
			ItemKey:  "in-advance",
			Price: productcatalog.NewPriceFrom(productcatalog.FlatPrice{
				Amount:      alpacadecimal.NewFromFloat(10),
				PaymentTerm: productcatalog.InAdvancePaymentTerm,
			}),
			BillingCadence: lo.ToPtr(datex.MustParse(s.T(), "P1D")),
		}.AsPatch(),
	})
	s.NoError(err)
	s.NotNil(updatedSubsView)

	s.NoError(s.Handler.SyncronizeSubscription(ctx, updatedSubsView, s.mustParseTime("2024-01-05T12:00:00Z")))

	// gathering invoice
	gatheringInvoice := s.gatheringInvoice(ctx, s.Namespace, s.Customer.ID)
	s.DebugDumpInvoice("gathering invoice - 2nd sync", gatheringInvoice)

	s.expectLines(gatheringInvoice, subsView.Subscription.ID, []expectedLine{
		{
			Matcher: recurringLineMatcher{
				PhaseKey:  "first-phase",
				ItemKey:   "in-advance",
				Version:   1,
				PeriodMin: 0,
				PeriodMax: 3,
			},

			Qty:       mo.Some[float64](1),
			UnitPrice: mo.Some[float64](10),
			Periods:   s.generatePeriods("2024-01-01T12:00:00Z", "2024-01-02T12:00:00Z", "P1D", 4),
			InvoiceAt: s.generateDailyTimestamps("2024-01-01T12:00:00Z", 4),
		},
	})

	// draft invoice
	draftInvoice, err = s.BillingService.GetInvoiceByID(ctx, billing.GetInvoiceByIdInput{
		Invoice: draftInvoice.InvoiceID(),
		Expand:  billing.InvoiceExpandAll,
	})
	s.NoError(err)

	s.expectLines(draftInvoice, subsView.Subscription.ID, []expectedLine{
		{
			Matcher: recurringLineMatcher{
				PhaseKey:  "first-phase",
				ItemKey:   "in-advance",
				Version:   0,
				PeriodMin: 0,
				PeriodMax: 0,
			},

			Qty:       mo.Some[float64](1),
			UnitPrice: mo.Some[float64](3),
			Periods: []billing.Period{
				{
					Start: s.mustParseTime("2024-01-01T00:00:00Z"),
					End:   s.mustParseTime("2024-01-01T12:00:00Z"),
				},
			},
			InvoiceAt: []time.Time{s.mustParseTime("2024-01-01T00:00:00Z")},
		},
	})
}

func (s *SubscriptionHandlerTestSuite) TestInAdvanceGatheringSyncIssuedInvoiceProrated() {
	ctx := s.Context
	clock.FreezeTime(s.mustParseTime("2024-01-01T00:00:00Z"))
	s.enableProrating()

	// Given
	//  we have a subscription with a single phase with a single static fee
	// When
	//  we have an outstanding invoice that has been already finalized and we edit the subscription later
	// Then
	//  then the finalized invoice doesn't get updated with the new pro-rated fee, but we
	//  add a warning to the invoice

	subsView := s.createSubscriptionFromPlanPhases([]productcatalog.Phase{
		{
			PhaseMeta: s.phaseMeta("first-phase", ""),
			RateCards: productcatalog.RateCards{
				&productcatalog.UsageBasedRateCard{
					RateCardMeta: productcatalog.RateCardMeta{
						Key:  "in-advance",
						Name: "in-advance",
						Price: productcatalog.NewPriceFrom(productcatalog.FlatPrice{
							Amount:      alpacadecimal.NewFromFloat(6),
							PaymentTerm: productcatalog.InAdvancePaymentTerm,
						}),
					},
					BillingCadence: datex.MustParse(s.T(), "P1D"),
				},
			},
		},
	})

	s.NoError(s.Handler.SyncronizeSubscription(ctx, subsView, s.mustParseTime("2024-01-05T12:00:00Z")))
	s.DebugDumpInvoice("gathering invoice", s.gatheringInvoice(ctx, s.Namespace, s.Customer.ID))

	clock.FreezeTime(s.mustParseTime("2024-01-01T12:00:00Z"))

	draftInvoices, err := s.BillingService.InvoicePendingLines(ctx, billing.InvoicePendingLinesInput{
		Customer: s.Customer.GetID(),
		AsOf:     lo.ToPtr(clock.Now()),
	})
	s.NoError(err)
	s.Require().Len(draftInvoices, 1)

	draftInvoice := draftInvoices[0]
	s.Equal(billing.InvoiceStatusDraftWaitingAutoApproval, draftInvoice.Status)

	approvedInvoice, err := s.BillingService.ApproveInvoice(ctx, draftInvoice.InvoiceID())
	s.NoError(err)
	s.Equal(billing.InvoiceStatusPaid, approvedInvoice.Status)

	s.expectLines(approvedInvoice, subsView.Subscription.ID, []expectedLine{
		{
			Matcher: recurringLineMatcher{
				PhaseKey:  "first-phase",
				ItemKey:   "in-advance",
				Version:   0,
				PeriodMin: 0,
				PeriodMax: 0,
			},
			Qty:       mo.Some[float64](1),
			UnitPrice: mo.Some[float64](6),
			Periods: []billing.Period{
				{
					Start: s.mustParseTime("2024-01-01T00:00:00Z"),
					End:   s.mustParseTime("2024-01-02T00:00:00Z"),
				},
			},
			InvoiceAt: []time.Time{s.mustParseTime("2024-01-01T00:00:00Z")},
		},
	})

	updatedSubsView, err := s.SubscriptionWorkflowService.EditRunning(ctx, subsView.Subscription.NamespacedID, []subscription.Patch{
		patch.PatchRemoveItem{
			PhaseKey: "first-phase",
			ItemKey:  "in-advance",
		},
		subscriptionAddItem{
			PhaseKey: "first-phase",
			ItemKey:  "in-advance",
			Price: productcatalog.NewPriceFrom(productcatalog.FlatPrice{
				Amount:      alpacadecimal.NewFromFloat(10),
				PaymentTerm: productcatalog.InAdvancePaymentTerm,
			}),
			BillingCadence: lo.ToPtr(datex.MustParse(s.T(), "P1D")),
		}.AsPatch(),
	})
	s.NoError(err)
	s.NotNil(updatedSubsView)

	s.NoError(s.Handler.SyncronizeSubscription(ctx, updatedSubsView, s.mustParseTime("2024-01-05T12:00:00Z")))

	// gathering invoice
	gatheringInvoice := s.gatheringInvoice(ctx, s.Namespace, s.Customer.ID)
	s.DebugDumpInvoice("gathering invoice - 2nd sync", gatheringInvoice)

	s.expectLines(gatheringInvoice, subsView.Subscription.ID, []expectedLine{
		{
			Matcher: recurringLineMatcher{
				PhaseKey:  "first-phase",
				ItemKey:   "in-advance",
				Version:   1,
				PeriodMin: 0,
				PeriodMax: 3,
			},

			Qty:       mo.Some[float64](1),
			UnitPrice: mo.Some[float64](10),
			Periods:   s.generatePeriods("2024-01-01T12:00:00Z", "2024-01-02T12:00:00Z", "P1D", 4),
			InvoiceAt: s.generateDailyTimestamps("2024-01-01T12:00:00Z", 4),
		},
	})

	// issued invoice
	approvedInvoice, err = s.BillingService.GetInvoiceByID(ctx, billing.GetInvoiceByIdInput{
		Invoice: draftInvoice.InvoiceID(),
		Expand:  billing.InvoiceExpandAll,
	})
	s.NoError(err)

	s.expectLines(approvedInvoice, subsView.Subscription.ID, []expectedLine{
		{
			Matcher: recurringLineMatcher{
				PhaseKey:  "first-phase",
				ItemKey:   "in-advance",
				Version:   0,
				PeriodMin: 0,
				PeriodMax: 0,
			},

			Qty:       mo.Some[float64](1),
			UnitPrice: mo.Some[float64](6),
			Periods: []billing.Period{
				{
					Start: s.mustParseTime("2024-01-01T00:00:00Z"),
					End:   s.mustParseTime("2024-01-02T00:00:00Z"),
				},
			},
			InvoiceAt: []time.Time{s.mustParseTime("2024-01-01T00:00:00Z")},
		},
	})
	s.Len(approvedInvoice.ValidationIssues, 1)

	s.expectValidationIssueForLine(approvedInvoice.Lines.OrEmpty()[0], approvedInvoice.ValidationIssues[0])
}

func (s *SubscriptionHandlerTestSuite) TestInAdvanceOneTimeFeeSyncing() {
	ctx := s.Context
	clock.FreezeTime(s.mustParseTime("2024-01-01T00:00:00Z"))

	// Given
	//  we have a subscription with a single phase with a single one-time fee in advance
	// When
	//  we we provision the lines
	// Then
	//  the gathering invoice will contain the generated item

	subsView := s.createSubscriptionFromPlanPhases([]productcatalog.Phase{
		{
			PhaseMeta: s.phaseMeta("first-phase", ""),
			RateCards: productcatalog.RateCards{
				&productcatalog.FlatFeeRateCard{
					RateCardMeta: productcatalog.RateCardMeta{
						Key:  "in-advance",
						Name: "in-advance",
						Price: productcatalog.NewPriceFrom(productcatalog.FlatPrice{
							Amount:      alpacadecimal.NewFromFloat(5),
							PaymentTerm: productcatalog.InAdvancePaymentTerm,
						}),
					},
				},
			},
		},
	})

	s.NoError(s.Handler.SyncronizeSubscription(ctx, subsView, s.mustParseTime("2024-01-05T12:00:00Z")))
	gatheringInvoice := s.gatheringInvoice(ctx, s.Namespace, s.Customer.ID)
	s.DebugDumpInvoice("gathering invoice", gatheringInvoice)

	s.expectLines(gatheringInvoice, subsView.Subscription.ID, []expectedLine{
		{
			Matcher: oneTimeLineMatcher{
				PhaseKey: "first-phase",
				ItemKey:  "in-advance",
				Version:  0,
			},

			Qty:       mo.Some[float64](1),
			UnitPrice: mo.Some[float64](5),
			Periods: []billing.Period{
				{
					Start: s.mustParseTime("2024-01-01T00:00:00Z"),
					End:   s.mustParseTime("2024-01-01T00:00:00Z"),
				},
			},
			InvoiceAt: []time.Time{s.mustParseTime("2024-01-01T00:00:00Z")},
		},
	})
}

func (s *SubscriptionHandlerTestSuite) TestInArrearsOneTimeFeeSyncing() {
	ctx := s.Context
	clock.FreezeTime(s.mustParseTime("2024-01-01T00:00:00Z"))

	// Given
	//  we have a subscription with a single phase with a single one-time fee in arrears
	// When
	//  we we provision the lines
	// Then
	//  there will be no gathering invoice, as we don't know what is in arrears

	// When
	//  we cancel the subscription
	// Then
	//  the gathering invoice will contain the generated item schedule to the cancellation's timestamp

	subsView := s.createSubscriptionFromPlanPhases([]productcatalog.Phase{
		{
			PhaseMeta: s.phaseMeta("first-phase", ""),
			RateCards: productcatalog.RateCards{
				&productcatalog.FlatFeeRateCard{
					RateCardMeta: productcatalog.RateCardMeta{
						Key:  "in-arrears",
						Name: "in-arrears",
						Price: productcatalog.NewPriceFrom(productcatalog.FlatPrice{
							Amount:      alpacadecimal.NewFromFloat(5),
							PaymentTerm: productcatalog.InArrearsPaymentTerm,
						}),
					},
				},
			},
		},
	})

	s.NoError(s.Handler.SyncronizeSubscription(ctx, subsView, s.mustParseTime("2024-01-05T12:00:00Z")))
	s.expectNoGatheringInvoice(ctx, s.Namespace, s.Customer.ID)

	// let's cancel the subscription
	cancelAt := s.mustParseTime("2024-01-04T12:00:00Z")

	subs, err := s.SubscriptionService.Cancel(ctx, subsView.Subscription.NamespacedID, subscription.Timing{
		Custom: &cancelAt,
	})
	s.NoError(err)

	subsView, err = s.SubscriptionService.GetView(ctx, subs.NamespacedID)
	s.NoError(err)

	s.NoError(s.Handler.SyncronizeSubscription(ctx, subsView, s.mustParseTime("2024-01-05T12:00:00Z")))

	gatheringInvoice := s.gatheringInvoice(ctx, s.Namespace, s.Customer.ID)
	s.DebugDumpInvoice("gathering invoice", gatheringInvoice)

	s.expectLines(gatheringInvoice, subsView.Subscription.ID, []expectedLine{
		{
			Matcher: oneTimeLineMatcher{
				PhaseKey: "first-phase",
				ItemKey:  "in-arrears",
				Version:  0,
			},

			Qty:       mo.Some[float64](1),
			UnitPrice: mo.Some[float64](5),
			Periods: []billing.Period{
				{
					Start: s.mustParseTime("2024-01-01T00:00:00Z"),
					End:   s.mustParseTime("2024-01-04T12:00:00Z"),
				},
			},
			InvoiceAt: []time.Time{s.mustParseTime("2024-01-04T12:00:00Z")},
		},
	})
}

func (s *SubscriptionHandlerTestSuite) TestUsageBasedGatheringUpdate() {
	ctx := s.Context
	clock.FreezeTime(s.mustParseTime("2024-01-01T00:00:00Z"))

	// Given
	//  we have a subscription with a single phase with an usage based price, and the gathering invoice contains the items
	// When
	//  when we add a new phase, that disrupts the period of previous items with a new usage based price for the same feature
	// Then
	//  then the gathering invoice is updated, the period of the previous items are updated accordingly

	subsView := s.createSubscriptionFromPlanPhases([]productcatalog.Phase{
		{
			PhaseMeta: s.phaseMeta("first-phase", ""),
			RateCards: productcatalog.RateCards{
				&productcatalog.UsageBasedRateCard{
					RateCardMeta: productcatalog.RateCardMeta{
						Key:     s.APIRequestsTotalFeature.Key,
						Name:    s.APIRequestsTotalFeature.Key,
						Feature: &s.APIRequestsTotalFeature,
						Price: productcatalog.NewPriceFrom(productcatalog.UnitPrice{
							Amount: alpacadecimal.NewFromFloat(10),
						}),
					},
					BillingCadence: datex.MustParse(s.T(), "P1D"),
				},
			},
		},
	})

	s.NoError(s.Handler.SyncronizeSubscription(ctx, subsView, s.mustParseTime("2024-01-05T12:00:00Z")))
	gatheringInvoice := s.gatheringInvoice(ctx, s.Namespace, s.Customer.ID)
	s.DebugDumpInvoice("gathering invoice", gatheringInvoice)

	s.expectLines(gatheringInvoice, subsView.Subscription.ID, []expectedLine{
		{
			Matcher: recurringLineMatcher{
				PhaseKey:  "first-phase",
				ItemKey:   s.APIRequestsTotalFeature.Key,
				Version:   0,
				PeriodMin: 0,
				PeriodMax: 4,
			},
			Price: mo.Some(productcatalog.NewPriceFrom(productcatalog.UnitPrice{
				Amount: alpacadecimal.NewFromFloat(10),
			})),
			Periods:   s.generatePeriods("2024-01-01T00:00:00Z", "2024-01-02T00:00:00Z", "P1D", 5),
			InvoiceAt: s.generateDailyTimestamps("2024-01-02T00:00:00Z", 5),
		},
	})

	updatedSubsView, err := s.SubscriptionWorkflowService.EditRunning(ctx, subsView.Subscription.NamespacedID, []subscription.Patch{
		patch.PatchAddPhase{
			PhaseKey: "second-phase",
			CreateInput: subscription.CreateSubscriptionPhaseInput{
				CreateSubscriptionPhasePlanInput: subscription.CreateSubscriptionPhasePlanInput{
					PhaseKey:   "second-phase",
					Name:       "second-phase",
					StartAfter: datex.MustParse(s.T(), "P1DT12H"),
				},
			},
		},
		subscriptionAddItem{
			PhaseKey: "second-phase",
			ItemKey:  s.APIRequestsTotalFeature.Key,
			Price: productcatalog.NewPriceFrom(productcatalog.UnitPrice{
				Amount: alpacadecimal.NewFromFloat(5),
			}),
			FeatureKey:     s.APIRequestsTotalFeature.Key,
			BillingCadence: lo.ToPtr(datex.MustParse(s.T(), "P1D")),
		}.AsPatch(),
	})
	s.NoError(err)
	s.NotNil(updatedSubsView)

	s.NoError(s.Handler.SyncronizeSubscription(ctx, updatedSubsView, s.mustParseTime("2024-01-05T12:00:00Z")))

	// gathering invoice
	gatheringInvoice = s.gatheringInvoice(ctx, s.Namespace, s.Customer.ID)
	s.DebugDumpInvoice("gathering invoice - 2nd sync", gatheringInvoice)

	s.expectLines(gatheringInvoice, subsView.Subscription.ID, []expectedLine{
		{
			Matcher: recurringLineMatcher{
				PhaseKey:  "first-phase",
				ItemKey:   s.APIRequestsTotalFeature.Key,
				Version:   0,
				PeriodMin: 0,
				PeriodMax: 1,
			},

			Price: mo.Some(productcatalog.NewPriceFrom(productcatalog.UnitPrice{
				Amount: alpacadecimal.NewFromFloat(10),
			})),
			Periods: []billing.Period{
				{
					Start: s.mustParseTime("2024-01-01T00:00:00Z"),
					End:   s.mustParseTime("2024-01-02T00:00:00Z"),
				},
				{
					Start: s.mustParseTime("2024-01-02T00:00:00Z"),
					End:   s.mustParseTime("2024-01-02T12:00:00Z"),
				},
			},
			InvoiceAt: []time.Time{s.mustParseTime("2024-01-02T00:00:00Z"), s.mustParseTime("2024-01-02T12:00:00Z")},
		},
		{
			Matcher: recurringLineMatcher{
				PhaseKey:  "second-phase",
				ItemKey:   s.APIRequestsTotalFeature.Key,
				Version:   0,
				PeriodMin: 0,
				PeriodMax: 2,
			},

			Price: mo.Some(productcatalog.NewPriceFrom(productcatalog.UnitPrice{
				Amount: alpacadecimal.NewFromFloat(5),
			})),
			Periods:   s.generatePeriods("2024-01-02T12:00:00Z", "2024-01-03T12:00:00Z", "P1D", 3),
			InvoiceAt: s.generateDailyTimestamps("2024-01-03T12:00:00Z", 3),
		},
	})
}

func (s *SubscriptionHandlerTestSuite) TestUsageBasedGatheringUpdateDraftInvoice() {
	ctx := s.Context
	clock.FreezeTime(s.mustParseTime("2024-01-01T00:00:00Z"))

	// Given
	//  we have a subscription with a single phase with an usage based price, and the gathering invoice contains the items
	//  a draft invoice has been created.
	// When
	//  when we add a new phase, that disrupts the period of previous items with a new usage based qty due to the period changes for the same feature
	// Then
	//  then the gathering invoice is updated, the period of the previous items are updated accordingly in the draft invoice
	//
	// NOTE: this simulates late event processing when we are severely behind the real time in billing worker (~1 day), this should not
	// happen, but we support this scenario

	// Initialize events
	s.MockStreamingConnector.AddSimpleEvent(*s.APIRequestsTotalFeature.MeterSlug, 0, s.mustParseTime("2023-01-01T00:00:00Z"))
	s.MockStreamingConnector.AddSimpleEvent(*s.APIRequestsTotalFeature.MeterSlug, 2, s.mustParseTime("2024-01-01T00:00:00Z"))
	s.MockStreamingConnector.AddSimpleEvent(*s.APIRequestsTotalFeature.MeterSlug, 3, s.mustParseTime("2024-01-01T12:00:00Z"))
	s.MockStreamingConnector.AddSimpleEvent(*s.APIRequestsTotalFeature.MeterSlug, 6, s.mustParseTime("2024-01-02T00:00:00Z"))

	subsView := s.createSubscriptionFromPlanPhases([]productcatalog.Phase{
		{
			PhaseMeta: s.phaseMeta("first-phase", ""),
			RateCards: productcatalog.RateCards{
				&productcatalog.UsageBasedRateCard{
					RateCardMeta: productcatalog.RateCardMeta{
						Key:     s.APIRequestsTotalFeature.Key,
						Name:    s.APIRequestsTotalFeature.Key,
						Feature: &s.APIRequestsTotalFeature,
						Price: productcatalog.NewPriceFrom(productcatalog.UnitPrice{
							Amount: alpacadecimal.NewFromFloat(10),
						}),
					},
					BillingCadence: datex.MustParse(s.T(), "P1D"),
				},
			},
		},
	})

	s.NoError(s.Handler.SyncronizeSubscription(ctx, subsView, s.mustParseTime("2024-01-05T12:00:00Z")))

	clock.FreezeTime(s.mustParseTime("2024-01-02T12:00:00Z"))
	draftInvoices, err := s.BillingService.InvoicePendingLines(ctx, billing.InvoicePendingLinesInput{
		Customer: s.Customer.GetID(),
	})
	s.NoError(err)
	s.Len(draftInvoices, 1)

	draftInvoice := draftInvoices[0]
	s.DebugDumpInvoice("draft invoice", draftInvoice)
	s.expectLines(draftInvoice, subsView.Subscription.ID, []expectedLine{
		{
			Matcher: recurringLineMatcher{
				PhaseKey: "first-phase",
				ItemKey:  s.APIRequestsTotalFeature.Key,
			},
			Qty: mo.Some[float64](5),
			Price: mo.Some(productcatalog.NewPriceFrom(productcatalog.UnitPrice{
				Amount: alpacadecimal.NewFromFloat(10),
			})),
			Periods: []billing.Period{
				{
					Start: s.mustParseTime("2024-01-01T00:00:00Z"),
					End:   s.mustParseTime("2024-01-02T00:00:00Z"),
				},
			},
			InvoiceAt: []time.Time{s.mustParseTime("2024-01-02T00:00:00Z")},
		},
	})

	gatheringInvoice := s.gatheringInvoice(ctx, s.Namespace, s.Customer.ID)
	s.DebugDumpInvoice("gathering invoice", gatheringInvoice)

	s.expectLines(gatheringInvoice, subsView.Subscription.ID, []expectedLine{
		{
			Matcher: recurringLineMatcher{
				PhaseKey:  "first-phase",
				ItemKey:   s.APIRequestsTotalFeature.Key,
				Version:   0,
				PeriodMin: 1,
				PeriodMax: 4,
			},
			Price: mo.Some(productcatalog.NewPriceFrom(productcatalog.UnitPrice{
				Amount: alpacadecimal.NewFromFloat(10),
			})),
			Periods:   s.generatePeriods("2024-01-02T00:00:00Z", "2024-01-03T00:00:00Z", "P1D", 4),
			InvoiceAt: s.generateDailyTimestamps("2024-01-03T00:00:00Z", 4),
		},
	})

	clock.FreezeTime(s.mustParseTime("2024-01-01T11:00:00Z"))

	updatedSubsView, err := s.SubscriptionWorkflowService.EditRunning(ctx, subsView.Subscription.NamespacedID, []subscription.Patch{
		patch.PatchAddPhase{
			PhaseKey: "second-phase",
			CreateInput: subscription.CreateSubscriptionPhaseInput{
				CreateSubscriptionPhasePlanInput: subscription.CreateSubscriptionPhasePlanInput{
					PhaseKey:   "second-phase",
					Name:       "second-phase",
					StartAfter: datex.MustParse(s.T(), "PT12H"),
				},
			},
		},
		subscriptionAddItem{
			PhaseKey: "second-phase",
			ItemKey:  s.APIRequestsTotalFeature.Key,
			Price: productcatalog.NewPriceFrom(productcatalog.UnitPrice{
				Amount: alpacadecimal.NewFromFloat(5),
			}),
			FeatureKey:     s.APIRequestsTotalFeature.Key,
			BillingCadence: lo.ToPtr(datex.MustParse(s.T(), "P1D")),
		}.AsPatch(),
	})
	s.NoError(err)
	s.NotNil(updatedSubsView)

	// Let's reset back the clock to the last sync's time
	clock.FreezeTime(s.mustParseTime("2024-01-02T12:00:00Z"))
	s.NoError(s.Handler.SyncronizeSubscription(ctx, updatedSubsView, s.mustParseTime("2024-01-05T12:00:00Z")))

	// gathering invoice
	gatheringInvoice = s.gatheringInvoice(ctx, s.Namespace, s.Customer.ID)
	s.DebugDumpInvoice("gathering invoice - 2nd sync", gatheringInvoice)

	s.expectLines(gatheringInvoice, subsView.Subscription.ID, []expectedLine{
		{
			Matcher: recurringLineMatcher{
				PhaseKey:  "second-phase",
				ItemKey:   s.APIRequestsTotalFeature.Key,
				Version:   0,
				PeriodMin: 0,
				PeriodMax: 3,
			},

			Price: mo.Some(productcatalog.NewPriceFrom(productcatalog.UnitPrice{
				Amount: alpacadecimal.NewFromFloat(5),
			})),
			Periods:   s.generatePeriods("2024-01-01T12:00:00Z", "2024-01-02T12:00:00Z", "P1D", 4),
			InvoiceAt: s.generateDailyTimestamps("2024-01-02T12:00:00Z", 4),
		},
	})

	updatedDraftInvoice, err := s.BillingService.GetInvoiceByID(ctx, billing.GetInvoiceByIdInput{
		Invoice: draftInvoice.InvoiceID(),
		Expand:  billing.InvoiceExpandAll,
	})
	s.NoError(err)
	s.DebugDumpInvoice("draft invoice - 2nd sync", updatedDraftInvoice)

	s.expectLines(updatedDraftInvoice, subsView.Subscription.ID, []expectedLine{
		{
			Matcher: recurringLineMatcher{
				PhaseKey: "first-phase",
				ItemKey:  s.APIRequestsTotalFeature.Key,
			},

			Qty: mo.Some[float64](2),
			Price: mo.Some(productcatalog.NewPriceFrom(productcatalog.UnitPrice{
				Amount: alpacadecimal.NewFromFloat(10),
			})),
			Periods: []billing.Period{
				{
					Start: s.mustParseTime("2024-01-01T00:00:00Z"),
					End:   s.mustParseTime("2024-01-01T12:00:00Z"),
				},
			},
			InvoiceAt: []time.Time{s.mustParseTime("2024-01-01T12:00:00Z")},
		},
	})
}

func (s *SubscriptionHandlerTestSuite) TestUsageBasedGatheringUpdateIssuedInvoice() {
	ctx := s.Context
	clock.FreezeTime(s.mustParseTime("2024-01-01T00:00:00Z"))

	// Given
	//  we have a subscription with a single phase with an usage based price, and the gathering invoice contains the items
	//  an issued invoice has been created.
	// When
	//  when we add a new phase, that disrupts the period of previous items with a new usage based qty due to the period changes for
	//  the same feature
	// Then
	//  then the gathering invoice is updated, the finalized invoice doesn't get updated with the periods, but a validation issue is added
	//
	// NOTE: this simulates late event processing when we are severely behind the real time in billing worker (~1 day), this should not
	// happen, but we support this scenario
	//
	// NOTE: This is variant of the TestUsageBasedGatheringUpdateDraftInvoice so we are keeping the checks at a minimum here

	// Initialize events
	s.MockStreamingConnector.AddSimpleEvent(*s.APIRequestsTotalFeature.MeterSlug, 0, s.mustParseTime("2023-01-01T00:00:00Z"))
	s.MockStreamingConnector.AddSimpleEvent(*s.APIRequestsTotalFeature.MeterSlug, 2, s.mustParseTime("2024-01-01T00:00:00Z"))
	s.MockStreamingConnector.AddSimpleEvent(*s.APIRequestsTotalFeature.MeterSlug, 3, s.mustParseTime("2024-01-01T12:00:00Z"))
	s.MockStreamingConnector.AddSimpleEvent(*s.APIRequestsTotalFeature.MeterSlug, 6, s.mustParseTime("2024-01-02T00:00:00Z"))

	subsView := s.createSubscriptionFromPlanPhases([]productcatalog.Phase{
		{
			PhaseMeta: s.phaseMeta("first-phase", ""),
			RateCards: productcatalog.RateCards{
				&productcatalog.UsageBasedRateCard{
					RateCardMeta: productcatalog.RateCardMeta{
						Key:     s.APIRequestsTotalFeature.Key,
						Name:    s.APIRequestsTotalFeature.Key,
						Feature: &s.APIRequestsTotalFeature,
						Price: productcatalog.NewPriceFrom(productcatalog.UnitPrice{
							Amount: alpacadecimal.NewFromFloat(10),
						}),
					},
					BillingCadence: datex.MustParse(s.T(), "P1D"),
				},
			},
		},
	})

	s.NoError(s.Handler.SyncronizeSubscription(ctx, subsView, s.mustParseTime("2024-01-05T12:00:00Z")))

	clock.FreezeTime(s.mustParseTime("2024-01-02T12:00:00Z"))
	draftInvoices, err := s.BillingService.InvoicePendingLines(ctx, billing.InvoicePendingLinesInput{
		Customer: s.Customer.GetID(),
	})
	s.NoError(err)
	s.Len(draftInvoices, 1)

	draftInvoice := draftInvoices[0]
	s.Equal(billing.InvoiceStatusDraftWaitingAutoApproval, draftInvoice.Status)

	issuedInvoice, err := s.BillingService.ApproveInvoice(ctx, draftInvoice.InvoiceID())
	s.NoError(err)
	s.Equal(billing.InvoiceStatusPaid, issuedInvoice.Status)
	s.Len(issuedInvoice.ValidationIssues, 0)
	s.DebugDumpInvoice("issued invoice", issuedInvoice)
	s.expectLines(issuedInvoice, subsView.Subscription.ID, []expectedLine{
		{
			Matcher: recurringLineMatcher{
				PhaseKey: "first-phase",
				ItemKey:  s.APIRequestsTotalFeature.Key,
			},

			Qty: mo.Some[float64](5),
			Price: mo.Some(productcatalog.NewPriceFrom(productcatalog.UnitPrice{
				Amount: alpacadecimal.NewFromFloat(10),
			})),
			Periods: []billing.Period{
				{
					Start: s.mustParseTime("2024-01-01T00:00:00Z"),
					End:   s.mustParseTime("2024-01-02T00:00:00Z"),
				},
			},
			InvoiceAt: []time.Time{s.mustParseTime("2024-01-02T00:00:00Z")},
		},
	})

	s.DebugDumpInvoice("gathering invoice", s.gatheringInvoice(ctx, s.Namespace, s.Customer.ID))

	clock.FreezeTime(s.mustParseTime("2024-01-01T11:00:00Z"))

	updatedSubsView, err := s.SubscriptionWorkflowService.EditRunning(ctx, subsView.Subscription.NamespacedID, []subscription.Patch{
		patch.PatchAddPhase{
			PhaseKey: "second-phase",
			CreateInput: subscription.CreateSubscriptionPhaseInput{
				CreateSubscriptionPhasePlanInput: subscription.CreateSubscriptionPhasePlanInput{
					PhaseKey:   "second-phase",
					Name:       "second-phase",
					StartAfter: datex.MustParse(s.T(), "PT12H"),
				},
			},
		},
		subscriptionAddItem{
			PhaseKey: "second-phase",
			ItemKey:  s.APIRequestsTotalFeature.Key,
			Price: productcatalog.NewPriceFrom(productcatalog.UnitPrice{
				Amount: alpacadecimal.NewFromFloat(5),
			}),
			FeatureKey:     s.APIRequestsTotalFeature.Key,
			BillingCadence: lo.ToPtr(datex.MustParse(s.T(), "P1D")),
		}.AsPatch(),
	})
	s.NoError(err)
	s.NotNil(updatedSubsView)

	// Let's reset back the clock to the last sync's time
	clock.FreezeTime(s.mustParseTime("2024-01-02T12:00:00Z"))
	s.NoError(s.Handler.SyncronizeSubscription(ctx, updatedSubsView, s.mustParseTime("2024-01-05T12:00:00Z")))

	// gathering invoice
	s.DebugDumpInvoice("gathering invoice - 2nd sync", s.gatheringInvoice(ctx, s.Namespace, s.Customer.ID))

	updatedIssuedInvoice, err := s.BillingService.GetInvoiceByID(ctx, billing.GetInvoiceByIdInput{
		Invoice: issuedInvoice.InvoiceID(),
		Expand:  billing.InvoiceExpandAll,
	})
	s.NoError(err)
	s.DebugDumpInvoice("issued invoice - 2nd sync", updatedIssuedInvoice)

	s.expectLines(updatedIssuedInvoice, subsView.Subscription.ID, []expectedLine{
		{
			Matcher: recurringLineMatcher{
				PhaseKey: "first-phase",
				ItemKey:  s.APIRequestsTotalFeature.Key,
			},

			Qty: mo.Some[float64](5),
			Price: mo.Some(productcatalog.NewPriceFrom(productcatalog.UnitPrice{
				Amount: alpacadecimal.NewFromFloat(10),
			})),
			Periods: []billing.Period{
				{
					Start: s.mustParseTime("2024-01-01T00:00:00Z"),
					End:   s.mustParseTime("2024-01-02T00:00:00Z"),
				},
			},
			InvoiceAt: []time.Time{s.mustParseTime("2024-01-02T00:00:00Z")},
		},
	})

	s.expectValidationIssueForLine(updatedIssuedInvoice.Lines.OrEmpty()[0], updatedIssuedInvoice.ValidationIssues[0])
}

func (s *SubscriptionHandlerTestSuite) TestUsageBasedUpdateWithLineSplits() {
	ctx := s.Context
	clock.FreezeTime(s.mustParseTime("2024-01-01T00:00:00Z"))

	// Given
	//  we have progressive billing enalbed
	//  we have a subscription with a single phase with an usage based price, and the gathering invoice contains the items
	//  invoice1 has been created for 2024-01-01T00:00:00Z - 2024-01-01T10:00:00Z, gets issued
	//  invoice2 has been created for 2024-01-01T10:00:00Z - 2024-01-01T13:00:00Z, remains in draft state
	// When
	//  when we add a new phase at 2024-01-10T09:00:00Z, that disrupts the period of previous items with a
	// new usage based qty due to the period changes for the same feature
	// Then
	//  then the gathering invoice is updated, the period of the previous items are updated accordingly in the draft invoice
	//  invoice1 remains the same, but a validation error has been added
	//  invoice2's line gets deleted, and the invoice goes to deleted state, as it doesn't have any line items
	//
	// NOTE: this simulates late event processing when we are severely behind the real time in billing worker (~1 day), but smaller differences
	// (minutes) can happen due to async nature of processing, thus we need to handle these scenarios

	// Initialize events
	s.MockStreamingConnector.AddSimpleEvent(*s.APIRequestsTotalFeature.MeterSlug, 0, s.mustParseTime("2023-01-01T00:00:00Z"))
	s.MockStreamingConnector.AddSimpleEvent(*s.APIRequestsTotalFeature.MeterSlug, 1, s.mustParseTime("2024-01-01T00:00:00Z"))
	s.MockStreamingConnector.AddSimpleEvent(*s.APIRequestsTotalFeature.MeterSlug, 1, s.mustParseTime("2024-01-01T09:30:00Z"))
	s.MockStreamingConnector.AddSimpleEvent(*s.APIRequestsTotalFeature.MeterSlug, 3, s.mustParseTime("2024-01-01T11:00:00Z"))
	s.MockStreamingConnector.AddSimpleEvent(*s.APIRequestsTotalFeature.MeterSlug, 7, s.mustParseTime("2024-01-01T12:30:00Z"))
	s.MockStreamingConnector.AddSimpleEvent(*s.APIRequestsTotalFeature.MeterSlug, 11, s.mustParseTime("2024-01-02T00:00:00Z"))

	s.enableProgressiveBilling()

	subsView := s.createSubscriptionFromPlanPhases([]productcatalog.Phase{
		{
			PhaseMeta: s.phaseMeta("first-phase", ""),
			RateCards: productcatalog.RateCards{
				&productcatalog.UsageBasedRateCard{
					RateCardMeta: productcatalog.RateCardMeta{
						Key:     s.APIRequestsTotalFeature.Key,
						Name:    s.APIRequestsTotalFeature.Key,
						Feature: &s.APIRequestsTotalFeature,
						Price: productcatalog.NewPriceFrom(productcatalog.UnitPrice{
							Amount: alpacadecimal.NewFromFloat(10),
						}),
					},
					BillingCadence: datex.MustParse(s.T(), "P1D"),
				},
			},
		},
	})

	s.NoError(s.Handler.SyncronizeSubscription(ctx, subsView, s.mustParseTime("2024-01-05T12:00:00Z")))

	// invoice 1: issued invoice creation
	clock.FreezeTime(s.mustParseTime("2024-01-01T14:00:00Z"))
	draftInvoices1, err := s.BillingService.InvoicePendingLines(ctx, billing.InvoicePendingLinesInput{
		Customer: s.Customer.GetID(),
		AsOf:     lo.ToPtr(s.mustParseTime("2024-01-01T10:00:00Z")),
	})
	s.NoError(err)
	s.Len(draftInvoices1, 1)

	invoice1, err := s.BillingService.ApproveInvoice(ctx, draftInvoices1[0].InvoiceID())
	s.NoError(err)
	s.Equal(billing.InvoiceStatusPaid, invoice1.Status)

	s.populateChildIDsFromParents(&invoice1)
	s.DebugDumpInvoice("issued invoice1", invoice1)

	s.expectLines(invoice1, subsView.Subscription.ID, []expectedLine{
		{
			Matcher: recurringLineMatcher{
				PhaseKey: "first-phase",
				ItemKey:  s.APIRequestsTotalFeature.Key,
			},
			Qty: mo.Some[float64](2),
			Price: mo.Some(productcatalog.NewPriceFrom(productcatalog.UnitPrice{
				Amount: alpacadecimal.NewFromFloat(10),
			})),
			Periods: []billing.Period{
				{
					Start: s.mustParseTime("2024-01-01T00:00:00Z"),
					End:   s.mustParseTime("2024-01-01T10:00:00Z"),
				},
			},
			InvoiceAt: []time.Time{s.mustParseTime("2024-01-01T10:00:00Z")},
		},
	})

	// invoice 2: draft invoice creation
	draftInvoices2, err := s.BillingService.InvoicePendingLines(ctx, billing.InvoicePendingLinesInput{
		Customer: s.Customer.GetID(),
		AsOf:     lo.ToPtr(s.mustParseTime("2024-01-01T13:00:00Z")),
	})
	s.NoError(err)
	s.Len(draftInvoices2, 1)

	draftInvoice2 := draftInvoices2[0]
	s.populateChildIDsFromParents(&draftInvoice2)
	s.DebugDumpInvoice("draft invoice2", draftInvoice2)
	s.Equal(billing.InvoiceStatusDraftWaitingAutoApproval, draftInvoice2.Status)

	s.expectLines(draftInvoice2, subsView.Subscription.ID, []expectedLine{
		{
			Matcher: recurringLineMatcher{
				PhaseKey: "first-phase",
				ItemKey:  s.APIRequestsTotalFeature.Key,
			},
			Qty: mo.Some[float64](10),
			Price: mo.Some(productcatalog.NewPriceFrom(productcatalog.UnitPrice{
				Amount: alpacadecimal.NewFromFloat(10),
			})),
			Periods: []billing.Period{
				{
					Start: s.mustParseTime("2024-01-01T10:00:00Z"),
					End:   s.mustParseTime("2024-01-01T13:00:00Z"),
				},
			},
			InvoiceAt: []time.Time{s.mustParseTime("2024-01-01T13:00:00Z")},
		},
	})

	// gathering invoice checks
	gatheringInvoice := s.gatheringInvoice(ctx, s.Namespace, s.Customer.ID)
	s.populateChildIDsFromParents(&gatheringInvoice)
	s.DebugDumpInvoice("gathering invoice", gatheringInvoice)

	s.expectLines(gatheringInvoice, subsView.Subscription.ID, []expectedLine{
		{
			Matcher: recurringLineMatcher{
				PhaseKey:  "first-phase",
				ItemKey:   s.APIRequestsTotalFeature.Key,
				Version:   0,
				PeriodMin: 0,
				PeriodMax: 4,
			},
			Price: mo.Some(productcatalog.NewPriceFrom(productcatalog.UnitPrice{
				Amount: alpacadecimal.NewFromFloat(10),
			})),
			Periods: append(
				[]billing.Period{
					{
						Start: s.mustParseTime("2024-01-01T13:00:00Z"),
						End:   s.mustParseTime("2024-01-02T00:00:00Z"),
					},
				},
				s.generatePeriods("2024-01-02T00:00:00Z", "2024-01-03T00:00:00Z", "P1D", 4)...,
			),
			InvoiceAt: s.generateDailyTimestamps("2024-01-02T00:00:00Z", 5),
		},
	})
	clock.FreezeTime(s.mustParseTime("2024-01-01T05:00:00Z"))

	updatedSubsView, err := s.SubscriptionWorkflowService.EditRunning(ctx, subsView.Subscription.NamespacedID, []subscription.Patch{
		patch.PatchAddPhase{
			PhaseKey: "second-phase",
			CreateInput: subscription.CreateSubscriptionPhaseInput{
				CreateSubscriptionPhasePlanInput: subscription.CreateSubscriptionPhasePlanInput{
					PhaseKey:   "second-phase",
					Name:       "second-phase",
					StartAfter: datex.MustParse(s.T(), "PT6H"),
				},
			},
		},
		subscriptionAddItem{
			PhaseKey: "second-phase",
			ItemKey:  s.APIRequestsTotalFeature.Key,
			Price: productcatalog.NewPriceFrom(productcatalog.UnitPrice{
				Amount: alpacadecimal.NewFromFloat(5),
			}),
			FeatureKey:     s.APIRequestsTotalFeature.Key,
			BillingCadence: lo.ToPtr(datex.MustParse(s.T(), "P1D")),
		}.AsPatch(),
	})

	s.NoError(err)
	s.NotNil(updatedSubsView)

	// THEN
	// Let's reset back the clock to the last sync's time
	clock.FreezeTime(s.mustParseTime("2024-01-01T14:00:00Z"))
	s.NoError(s.Handler.SyncronizeSubscription(ctx, updatedSubsView, s.mustParseTime("2024-01-05T12:00:00Z")))

	// gathering invoice
	gatheringInvoice = s.gatheringInvoice(ctx, s.Namespace, s.Customer.ID)
	s.populateChildIDsFromParents(&gatheringInvoice)
	s.DebugDumpInvoice("gathering invoice - 2nd sync", gatheringInvoice)

	s.expectLines(gatheringInvoice, subsView.Subscription.ID, []expectedLine{
		{
			Matcher: recurringLineMatcher{
				PhaseKey:  "second-phase",
				ItemKey:   s.APIRequestsTotalFeature.Key,
				Version:   0,
				PeriodMin: 0,
				PeriodMax: 4,
			},

			Price: mo.Some(productcatalog.NewPriceFrom(productcatalog.UnitPrice{
				Amount: alpacadecimal.NewFromFloat(5),
			})),
			Periods:   s.generatePeriods("2024-01-01T06:00:00Z", "2024-01-02T06:00:00Z", "P1D", 5),
			InvoiceAt: s.generateDailyTimestamps("2024-01-02T06:00:00Z", 5),
		},
	})

	// invoice 1 (issued) checks
	updatedIssuedInvoice, err := s.BillingService.GetInvoiceByID(ctx, billing.GetInvoiceByIdInput{
		Invoice: invoice1.InvoiceID(),
		Expand:  billing.InvoiceExpandAll,
	})
	s.NoError(err)

	s.populateChildIDsFromParents(&updatedIssuedInvoice)
	s.DebugDumpInvoice("invoice1 (issued) - 2nd sync", updatedIssuedInvoice)

	s.expectLines(updatedIssuedInvoice, subsView.Subscription.ID, []expectedLine{
		{
			Matcher: recurringLineMatcher{
				PhaseKey: "first-phase",
				ItemKey:  s.APIRequestsTotalFeature.Key,
			},
			Qty: mo.Some[float64](2),
			Price: mo.Some(productcatalog.NewPriceFrom(productcatalog.UnitPrice{
				Amount: alpacadecimal.NewFromFloat(10),
			})),
			Periods: []billing.Period{
				{
					Start: s.mustParseTime("2024-01-01T00:00:00Z"),
					End:   s.mustParseTime("2024-01-01T10:00:00Z"),
				},
			},
			InvoiceAt: []time.Time{s.mustParseTime("2024-01-01T10:00:00Z")},
		},
	})

	s.expectValidationIssueForLine(updatedIssuedInvoice.Lines.OrEmpty()[0], updatedIssuedInvoice.ValidationIssues[0])

	// invoice 2 (draft) checks
	updatedDraftInvoice, err := s.BillingService.GetInvoiceByID(ctx, billing.GetInvoiceByIdInput{
		Invoice: draftInvoice2.InvoiceID(),
		Expand:  billing.InvoiceExpandAll,
	})
	s.NoError(err)

	s.populateChildIDsFromParents(&updatedDraftInvoice)
	s.DebugDumpInvoice("draft invoice2 - 2nd sync", updatedDraftInvoice)
	s.Len(updatedDraftInvoice.Lines.OrEmpty(), 0)
	s.Equal(billing.InvoiceStatusDeleted, updatedDraftInvoice.Status)
}

func (s *SubscriptionHandlerTestSuite) TestGatheringManualEditSync() {
	ctx := s.Context
	clock.FreezeTime(s.mustParseTime("2024-01-01T00:00:00Z"))

	// Given
	//  we have a subscription with a single phase with recurring flat fee
	// When
	//  we have the gathering invoice created, and update an item (manually)
	// Then
	//  resyncing the subscription would not cause the item to be upserted again

	subsView := s.createSubscriptionFromPlanPhases([]productcatalog.Phase{
		{
			PhaseMeta: s.phaseMeta("first-phase", ""),
			RateCards: productcatalog.RateCards{
				&productcatalog.UsageBasedRateCard{
					RateCardMeta: productcatalog.RateCardMeta{
						Key:  "in-advance",
						Name: "in-advance",
						Price: productcatalog.NewPriceFrom(productcatalog.FlatPrice{
							Amount:      alpacadecimal.NewFromFloat(5),
							PaymentTerm: productcatalog.InAdvancePaymentTerm,
						}),
					},
					BillingCadence: datex.MustParse(s.T(), "P1D"),
				},
			},
		},
	})

	s.NoError(s.Handler.SyncronizeSubscription(ctx, subsView, s.mustParseTime("2024-01-05T12:00:00Z")))
	gatheringInvoice := s.gatheringInvoice(ctx, s.Namespace, s.Customer.ID)
	s.DebugDumpInvoice("gathering invoice", gatheringInvoice)

	var updatedLine *billing.Line
	editedInvoice, err := s.BillingService.UpdateInvoice(ctx, billing.UpdateInvoiceInput{
		Invoice: gatheringInvoice.InvoiceID(),
		EditFn: func(invoice *billing.Invoice) error {
			line := s.getLineByChildID(*invoice, fmt.Sprintf("%s/first-phase/in-advance/v[0]/period[0]", subsView.Subscription.ID))

			line.FlatFee.PaymentTerm = productcatalog.InArrearsPaymentTerm
			line.Period = billing.Period{
				Start: line.Period.Start.Add(time.Hour),
				End:   line.Period.End.Add(time.Hour),
			}
			line.InvoiceAt = line.Period.End
			line.ManagedBy = billing.ManuallyManagedLine

			updatedLine = line.Clone()
			return nil
		},
	})

	s.NoError(err)
	s.DebugDumpInvoice("edited invoice", editedInvoice)

	// When resyncing the subscription
	s.NoError(s.Handler.SyncronizeSubscription(ctx, subsView, s.mustParseTime("2024-01-05T12:00:00Z")))
	gatheringInvoice = s.gatheringInvoice(ctx, s.Namespace, s.Customer.ID)
	s.DebugDumpInvoice("gathering invoice - after sync", gatheringInvoice)

	// Then the line should not be updated
	invoiceLine := s.getLineByChildID(gatheringInvoice, *updatedLine.ChildUniqueReferenceID)
	s.True(invoiceLine.LineBase.Equal(updatedLine.LineBase), "line should not be updated")
}

func (s *SubscriptionHandlerTestSuite) TestSplitLineManualEditSync() {
	ctx := s.Context
	clock.FreezeTime(s.mustParseTime("2024-01-01T00:00:00Z"))
	s.enableProgressiveBilling()

	s.MockStreamingConnector.AddSimpleEvent(*s.APIRequestsTotalFeature.MeterSlug, 12, s.mustParseTime("2024-01-01T10:00:00Z"))

	// Given
	//  we have a subscription with a single phase with recurring flat fee
	//  we have the gathering invoice created
	//  we have a draft invoice with a split line
	// When
	//  the item on the draft invoice gets updated (manually)
	// Then
	//  editing the subscription will update fields, but period will be managed by the sync to ensure consistency between line and parent

	subsView := s.createSubscriptionFromPlanPhases([]productcatalog.Phase{
		{
			PhaseMeta: s.phaseMeta("first-phase", ""),
			RateCards: productcatalog.RateCards{
				&productcatalog.UsageBasedRateCard{
					RateCardMeta: productcatalog.RateCardMeta{
						Key:  s.APIRequestsTotalFeature.Key,
						Name: "ubp",
						Price: productcatalog.NewPriceFrom(productcatalog.UnitPrice{
							Amount: alpacadecimal.NewFromFloat(5),
						}),
						Feature: &s.APIRequestsTotalFeature,
					},
					BillingCadence: datex.MustParse(s.T(), "P1D"),
				},
			},
		},
	})

	s.NoError(s.Handler.SyncronizeSubscription(ctx, subsView, s.mustParseTime("2024-01-05T12:00:00Z")))
	s.DebugDumpInvoice("gathering invoice - pre invoicing", s.gatheringInvoice(ctx, s.Namespace, s.Customer.ID))

	clock.FreezeTime(s.mustParseTime("2024-01-01T12:00:00Z"))
	draftInvoices, err := s.BillingService.InvoicePendingLines(ctx, billing.InvoicePendingLinesInput{
		Customer: s.Customer.GetID(),
	})
	s.NoError(err)
	s.Len(draftInvoices, 1)
	draftInvoice := draftInvoices[0]

	s.DebugDumpInvoice("draft invoice", draftInvoice)

	var updatedLine *billing.Line
	editedInvoice, err := s.BillingService.UpdateInvoice(ctx, billing.UpdateInvoiceInput{
		Invoice: draftInvoice.InvoiceID(),
		EditFn: func(invoice *billing.Invoice) error {
			lines := invoice.Lines.OrEmpty()
			s.Len(lines, 1)

			line := lines[0]

			line.Name = "test"
			line.ManagedBy = billing.ManuallyManagedLine

			updatedLine = line.Clone()
			return nil
		},
	})

	s.NoError(err)
	s.DebugDumpInvoice("edited invoice", editedInvoice)
	s.DebugDumpInvoice("gathering invoice", s.gatheringInvoice(ctx, s.Namespace, s.Customer.ID))
	s.NotNil(updatedLine)

	clock.FreezeTime(s.mustParseTime("2024-01-01T11:00:00Z"))
	_, err = s.SubscriptionService.Cancel(ctx, subsView.Subscription.NamespacedID, subscription.Timing{
		Custom: lo.ToPtr(s.mustParseTime("2024-01-01T11:00:00Z")),
	})
	s.NoError(err)

	subsView, err = s.SubscriptionService.GetView(ctx, subsView.Subscription.NamespacedID)
	s.NoError(err)

	// When resyncing the subscription
	s.NoError(s.Handler.SyncronizeSubscription(ctx, subsView, s.mustParseTime("2024-01-05T12:00:00Z")))
	s.T().Log("-> Subscription canceled")

	s.DebugDumpInvoice("gathering invoice - after sync", s.gatheringInvoice(ctx, s.Namespace, s.Customer.ID))

	resyncedInvoice, err := s.BillingService.GetInvoiceByID(ctx, billing.GetInvoiceByIdInput{
		Invoice: editedInvoice.InvoiceID(),
		Expand:  billing.InvoiceExpandAll,
	})
	s.NoError(err)
	s.DebugDumpInvoice("draft invoice - after sync", resyncedInvoice)

	// Then the line should not be updated
	s.Len(resyncedInvoice.Lines.OrEmpty(), 1)
	resyncedInvoiceLine := resyncedInvoice.Lines.OrEmpty()[0]

	// Field updates are supported for manually managed lines
	s.Equal(resyncedInvoiceLine.LineBase.Name, updatedLine.Name)
	// Period however is managed by the sync to ensure consistency between line and parent (update endpoint does the filtering)
	s.Equal(billing.Period{
		Start: s.mustParseTime("2024-01-01T00:00:00Z"),
		End:   s.mustParseTime("2024-01-01T11:00:00Z"),
	}, resyncedInvoiceLine.Period)
}

func (s *SubscriptionHandlerTestSuite) TestGatheringManualDeleteSync() {
	ctx := s.Context
	clock.FreezeTime(s.mustParseTime("2024-01-01T00:00:00Z"))

	// Given
	//  we have a subscription with a single phase with recurring flat fee
	// When
	//  we have the gathering invoice created, and delete an item (manually)
	// Then
	//  resyncing the subscription would not cause the item to be upserted again

	subsView := s.createSubscriptionFromPlanPhases([]productcatalog.Phase{
		{
			PhaseMeta: s.phaseMeta("first-phase", ""),
			RateCards: productcatalog.RateCards{
				&productcatalog.UsageBasedRateCard{
					RateCardMeta: productcatalog.RateCardMeta{
						Key:  "in-advance",
						Name: "in-advance",
						Price: productcatalog.NewPriceFrom(productcatalog.FlatPrice{
							Amount:      alpacadecimal.NewFromFloat(5),
							PaymentTerm: productcatalog.InAdvancePaymentTerm,
						}),
					},
					BillingCadence: datex.MustParse(s.T(), "P1D"),
				},
			},
		},
	})

	s.NoError(s.Handler.SyncronizeSubscription(ctx, subsView, s.mustParseTime("2024-01-05T12:00:00Z")))
	gatheringInvoice := s.gatheringInvoice(ctx, s.Namespace, s.Customer.ID)
	s.DebugDumpInvoice("gathering invoice", gatheringInvoice)

	var updatedLine *billing.Line

	childUniqueReferenceID := fmt.Sprintf("%s/first-phase/in-advance/v[0]/period[0]", subsView.Subscription.ID)

	editedInvoice, err := s.BillingService.UpdateInvoice(ctx, billing.UpdateInvoiceInput{
		Invoice: gatheringInvoice.InvoiceID(),
		EditFn: func(invoice *billing.Invoice) error {
			line := s.getLineByChildID(*invoice, childUniqueReferenceID)

			line.DeletedAt = lo.ToPtr(clock.Now())
			line.ManagedBy = billing.ManuallyManagedLine

			updatedLine = line.Clone()
			return nil
		},
		IncludeDeletedLines: true,
	})

	updatedLineFromEditedInvoice := s.getLineByChildID(editedInvoice, childUniqueReferenceID)
	s.NotNil(updatedLineFromEditedInvoice.DeletedAt)
	s.Equal(billing.ManuallyManagedLine, updatedLineFromEditedInvoice.ManagedBy)

	s.NoError(err)
	s.DebugDumpInvoice("edited invoice", editedInvoice)

	// When resyncing the subscription
	s.NoError(s.Handler.SyncronizeSubscription(ctx, subsView, s.mustParseTime("2024-01-05T12:00:00Z")))
	gatheringInvoice = s.gatheringInvoice(ctx, s.Namespace, s.Customer.ID)
	s.DebugDumpInvoice("gathering invoice - after sync", gatheringInvoice)

	// Then the line should not be recreated
	s.expectNoLineWithChildID(gatheringInvoice, *updatedLine.ChildUniqueReferenceID)
}

func (s *SubscriptionHandlerTestSuite) TestSplitLineManualDeleteSync() {
	ctx := s.Context
	clock.FreezeTime(s.mustParseTime("2024-01-01T00:00:00Z"))
	s.enableProgressiveBilling()

	s.MockStreamingConnector.AddSimpleEvent(*s.APIRequestsTotalFeature.MeterSlug, 12, s.mustParseTime("2024-01-01T10:00:00Z"))

	// Given
	//  we have a subscription with a single phase with recurring flat fee
	//  we have the gathering invoice created
	//  we have a draft invoice with a split line
	// When
	//  the item on the draft invoice gets deleted (manually)
	// Then
	//  editing the subscription would not cause the item to be recreated, but periods are updated

	subsView := s.createSubscriptionFromPlanPhases([]productcatalog.Phase{
		{
			PhaseMeta: s.phaseMeta("first-phase", ""),
			RateCards: productcatalog.RateCards{
				&productcatalog.UsageBasedRateCard{
					RateCardMeta: productcatalog.RateCardMeta{
						Key:  s.APIRequestsTotalFeature.Key,
						Name: "ubp",
						Price: productcatalog.NewPriceFrom(productcatalog.UnitPrice{
							Amount: alpacadecimal.NewFromFloat(5),
						}),
						Feature: &s.APIRequestsTotalFeature,
					},
					BillingCadence: datex.MustParse(s.T(), "P1D"),
				},
			},
		},
	})

	s.NoError(s.Handler.SyncronizeSubscription(ctx, subsView, s.mustParseTime("2024-01-05T12:00:00Z")))
	s.DebugDumpInvoice("gathering invoice - pre invoicing", s.gatheringInvoice(ctx, s.Namespace, s.Customer.ID))

	clock.FreezeTime(s.mustParseTime("2024-01-01T12:00:00Z"))
	draftInvoices, err := s.BillingService.InvoicePendingLines(ctx, billing.InvoicePendingLinesInput{
		Customer: s.Customer.GetID(),
	})
	s.NoError(err)
	s.Len(draftInvoices, 1)
	draftInvoice := draftInvoices[0]

	s.DebugDumpInvoice("draft invoice", draftInvoice)

	var updatedLine *billing.Line
	editedInvoice, err := s.BillingService.UpdateInvoice(ctx, billing.UpdateInvoiceInput{
		Invoice: draftInvoice.InvoiceID(),
		EditFn: func(invoice *billing.Invoice) error {
			lines := invoice.Lines.OrEmpty()
			s.Len(lines, 1)

			line := lines[0]

			line.DeletedAt = lo.ToPtr(clock.Now())

			updatedLine = line.Clone()
			return nil
		},
	})

	s.NoError(err)
	s.DebugDumpInvoice("edited invoice", editedInvoice)
	s.DebugDumpInvoice("gathering invoice", s.gatheringInvoice(ctx, s.Namespace, s.Customer.ID))
	s.NotNil(updatedLine)

	clock.FreezeTime(s.mustParseTime("2024-01-01T11:00:00Z"))
	_, err = s.SubscriptionService.Cancel(ctx, subsView.Subscription.NamespacedID, subscription.Timing{
		Custom: lo.ToPtr(s.mustParseTime("2024-01-01T11:00:00Z")),
	})
	s.NoError(err)

	subsView, err = s.SubscriptionService.GetView(ctx, subsView.Subscription.NamespacedID)
	s.NoError(err)

	// When resyncing the subscription
	s.NoError(s.Handler.SyncronizeSubscription(ctx, subsView, s.mustParseTime("2024-01-05T12:00:00Z")))
	s.T().Log("-> Subscription canceled")

	s.DebugDumpInvoice("gathering invoice - after sync", s.gatheringInvoice(ctx, s.Namespace, s.Customer.ID))

	resyncedInvoice, err := s.BillingService.GetInvoiceByID(ctx, billing.GetInvoiceByIdInput{
		Invoice: editedInvoice.InvoiceID(),
		Expand:  billing.InvoiceExpandAll.SetDeletedLines(true),
	})
	s.NoError(err)
	s.DebugDumpInvoice("draft invoice - after sync", resyncedInvoice)

	// The line should still be deleted
	s.Len(resyncedInvoice.Lines.OrEmpty(), 1)

	line := resyncedInvoice.Lines.OrEmpty()[0]
	s.NotNil(line.DeletedAt)
	// Period is updated
	s.Equal(billing.Period{
		Start: s.mustParseTime("2024-01-01T00:00:00Z"),
		End:   s.mustParseTime("2024-01-01T11:00:00Z"),
	}, line.Period)

	s.NotNil(line.ParentLine)
	parentLine := line.ParentLine
	// Parent's period is in sync with the child
	s.Equal(billing.Period{
		Start: s.mustParseTime("2024-01-01T00:00:00Z"),
		End:   s.mustParseTime("2024-01-01T11:00:00Z"),
	}, parentLine.Period)
	s.Equal(fmt.Sprintf("%s/first-phase/api-requests-total/v[0]/period[0]", subsView.Subscription.ID), *parentLine.ChildUniqueReferenceID)
}

func (s *SubscriptionHandlerTestSuite) TestRateCardTaxSync() {
	ctx := s.Context
	clock.FreezeTime(s.mustParseTime("2024-01-01T00:00:00Z"))

	// Given
	//  we have tax information set in the rate card
	// When
	//  we syncronize the subscription phases
	// Then
	//  the gathering invoice will contain the tax details

	taxConfig := &productcatalog.TaxConfig{
		Behavior: lo.ToPtr(productcatalog.ExclusiveTaxBehavior),
		Stripe: &productcatalog.StripeTaxConfig{
			Code: "txcd_10000000",
		},
	}

	subsView := s.createSubscriptionFromPlanPhases([]productcatalog.Phase{
		{
			PhaseMeta: s.phaseMeta("first-phase", ""),
			RateCards: productcatalog.RateCards{
				&productcatalog.UsageBasedRateCard{
					RateCardMeta: productcatalog.RateCardMeta{
						Key:  "in-arrears",
						Name: "in-arrears",
						Price: productcatalog.NewPriceFrom(productcatalog.FlatPrice{
							Amount:      alpacadecimal.NewFromFloat(5),
							PaymentTerm: productcatalog.InArrearsPaymentTerm,
						}),
						TaxConfig: taxConfig,
					},
					BillingCadence: datex.MustParse(s.T(), "P1D"),
				},
			},
		},
	})

	s.NoError(s.Handler.SyncronizeSubscription(ctx, subsView, s.mustParseTime("2024-01-05T12:00:00Z")))

	gatheringInvoice := s.gatheringInvoice(ctx, s.Namespace, s.Customer.ID)
	s.DebugDumpInvoice("gathering invoice", gatheringInvoice)

	lines := gatheringInvoice.Lines.OrEmpty()
	for _, line := range lines {
		s.Equal(taxConfig, line.TaxConfig)
	}

	// Given we edit the subscription the tax config is carried over to the lines

	updatedSubsView, err := s.SubscriptionWorkflowService.EditRunning(ctx, subsView.Subscription.NamespacedID, []subscription.Patch{
		patch.PatchRemoveItem{
			PhaseKey: "first-phase",
			ItemKey:  "in-arrears",
		},
		subscriptionAddItem{
			PhaseKey: "first-phase",
			ItemKey:  "in-advance",
			Price: productcatalog.NewPriceFrom(productcatalog.FlatPrice{
				Amount:      alpacadecimal.NewFromFloat(10),
				PaymentTerm: productcatalog.InAdvancePaymentTerm,
			}),
			TaxConfig:      taxConfig,
			BillingCadence: lo.ToPtr(datex.MustParse(s.T(), "P1D")),
		}.AsPatch(),
	})
	s.NoError(err)
	s.NotNil(updatedSubsView)

	s.NoError(s.Handler.SyncronizeSubscription(ctx, subsView, s.mustParseTime("2024-01-05T12:00:00Z")))

	gatheringInvoice = s.gatheringInvoice(ctx, s.Namespace, s.Customer.ID)
	s.DebugDumpInvoice("gathering invoice - after edit", gatheringInvoice)

	lines = gatheringInvoice.Lines.OrEmpty()
	for _, line := range lines {
		s.Equal(taxConfig, line.TaxConfig)
	}
}

func (s *SubscriptionHandlerTestSuite) expectValidationIssueForLine(line *billing.Line, issue billing.ValidationIssue) {
	s.Equal(billing.ValidationIssueSeverityWarning, issue.Severity)
	s.Equal(billing.ImmutableInvoiceHandlingNotSupportedErrorCode, issue.Code)
	s.Equal(SubscriptionSyncComponentName, issue.Component)
	s.Equal(fmt.Sprintf("lines/%s", line.ID), issue.Path)
}

type expectedLine struct {
	Matcher   lineMatcher
	Qty       mo.Option[float64]
	UnitPrice mo.Option[float64]
	Price     mo.Option[*productcatalog.Price]
	Periods   []billing.Period
	InvoiceAt []time.Time
}

func (s *SubscriptionHandlerTestSuite) expectLines(invoice billing.Invoice, subscriptionID string, expectedLines []expectedLine) {
	s.T().Helper()

	lines := invoice.Lines.OrEmpty()

	existingLineChildIDs := lo.Map(lines, func(line *billing.Line, _ int) string {
		return lo.FromPtrOr(line.ChildUniqueReferenceID, line.ID)
	})

	expectedLineIds := lo.Flatten(lo.Map(expectedLines, func(expectedLine expectedLine, _ int) []string {
		return expectedLine.Matcher.ChildIDs(subscriptionID)
	}))

	s.ElementsMatch(expectedLineIds, existingLineChildIDs)

	for _, expectedLine := range expectedLines {
		childIDs := expectedLine.Matcher.ChildIDs(subscriptionID)
		for idx, childID := range childIDs {
			line, found := lo.Find(lines, func(line *billing.Line) bool {
				return lo.FromPtrOr(line.ChildUniqueReferenceID, line.ID) == childID
			})
			s.Truef(found, "line not found with child id %s", childID)
			s.NotNil(line)

			if expectedLine.Qty.IsPresent() {
				if line.Type == billing.InvoiceLineTypeFee {
					s.Equal(expectedLine.Qty.OrEmpty(), line.FlatFee.Quantity.InexactFloat64(), childID)
				} else {
					s.Equal(expectedLine.Qty.OrEmpty(), line.UsageBased.Quantity.InexactFloat64(), childID)
				}
			}

			if expectedLine.UnitPrice.IsPresent() {
				s.Equal(line.Type, billing.InvoiceLineTypeFee, childID)
				s.Equal(expectedLine.UnitPrice.OrEmpty(), line.FlatFee.PerUnitAmount.InexactFloat64(), childID)
			}

			if expectedLine.Price.IsPresent() {
				s.Equal(line.Type, billing.InvoiceLineTypeUsageBased)
				s.Equal(*expectedLine.Price.OrEmpty(), *line.UsageBased.Price, childID)
			}

			s.Equal(expectedLine.Periods[idx].Start, line.Period.Start, childID)
			s.Equal(expectedLine.Periods[idx].End, line.Period.End, childID)

			s.Equal(expectedLine.InvoiceAt[idx], line.InvoiceAt, childID)
		}
	}
}

type lineMatcher interface {
	ChildIDs(subsID string) []string
}

type recurringLineMatcher struct {
	PhaseKey  string
	ItemKey   string
	Version   int
	PeriodMin int
	PeriodMax int
}

func (m recurringLineMatcher) ChildIDs(subsID string) []string {
	out := []string{}
	for periodID := m.PeriodMin; periodID <= m.PeriodMax; periodID++ {
		out = append(out, fmt.Sprintf("%s/%s/%s/v[%d]/period[%d]", subsID, m.PhaseKey, m.ItemKey, m.Version, periodID))
	}

	return out
}

type oneTimeLineMatcher struct {
	PhaseKey string
	ItemKey  string
	Version  int
}

func (m oneTimeLineMatcher) ChildIDs(subsID string) []string {
	return []string{fmt.Sprintf("%s/%s/%s/v[%d]", subsID, m.PhaseKey, m.ItemKey, m.Version)}
}

// helpers

//nolint:unparam
func (s *SubscriptionHandlerTestSuite) phaseMeta(key string, duration string) productcatalog.PhaseMeta {
	out := productcatalog.PhaseMeta{
		Key:  key,
		Name: key,
	}

	if duration != "" {
		out.Duration = lo.ToPtr(datex.MustParse(s.T(), duration))
	}

	return out
}

func (s *SubscriptionHandlerTestSuite) enableProgressiveBilling() {
	defaultProfile, err := s.BillingService.GetDefaultProfile(s.Context, billing.GetDefaultProfileInput{
		Namespace: s.Namespace,
	})
	s.NoError(err)

	defaultProfile.WorkflowConfig.Invoicing.ProgressiveBilling = true
	defaultProfile.AppReferences = nil

	_, err = s.BillingService.UpdateProfile(s.Context, billing.UpdateProfileInput(defaultProfile.BaseProfile))
	s.NoError(err)
}

type subscriptionAddItem struct {
	PhaseKey       string
	ItemKey        string
	Price          *productcatalog.Price
	BillingCadence *datex.Period
	FeatureKey     string
	TaxConfig      *productcatalog.TaxConfig
}

func (i subscriptionAddItem) AsPatch() subscription.Patch {
	return patch.PatchAddItem{
		PhaseKey: i.PhaseKey,
		ItemKey:  i.ItemKey,
		CreateInput: subscription.SubscriptionItemSpec{
			CreateSubscriptionItemInput: subscription.CreateSubscriptionItemInput{
				CreateSubscriptionItemPlanInput: subscription.CreateSubscriptionItemPlanInput{
					PhaseKey: i.PhaseKey,
					ItemKey:  i.ItemKey,
					RateCard: subscription.RateCard{
						Name:           i.ItemKey,
						Price:          i.Price,
						BillingCadence: i.BillingCadence,
						FeatureKey:     lo.EmptyableToPtr(i.FeatureKey),
						TaxConfig:      i.TaxConfig,
					},
				},
			},
		},
	}
}

func (s *SubscriptionHandlerTestSuite) generatePeriods(startStr, endStr string, cadenceStr string, n int) []billing.Period { //nolint: unparam
	start := s.mustParseTime(startStr)
	end := s.mustParseTime(endStr)
	cadence := datex.MustParse(s.T(), cadenceStr)

	out := []billing.Period{}

	for {
		if n == 0 {
			break
		}

		out = append(out, billing.Period{
			Start: start,
			End:   end,
		})

		start, _ = cadence.AddTo(start)
		end, _ = cadence.AddTo(end)

		n--
	}
	return out
}

func (s *SubscriptionHandlerTestSuite) generateDailyTimestamps(startStr string, n int) []time.Time {
	start := s.mustParseTime(startStr)
	cadence := datex.MustParse(s.T(), "P1D")

	out := []time.Time{}

	for {
		if n == 0 {
			break
		}

		out = append(out, start)

		start, _ = cadence.AddTo(start)

		n--
	}
	return out
}

// populateChildIDsFromParents copies over the child ID from the parent line, if it's not already set
// as line splitting doesn't set the child ID on child lines to prevent conflicts if multiple split lines
// end up on a single invoice.
func (s *SubscriptionHandlerTestSuite) populateChildIDsFromParents(invoice *billing.Invoice) {
	for _, line := range invoice.Lines.OrEmpty() {
		if line.ChildUniqueReferenceID == nil && line.ParentLine != nil {
			line.ChildUniqueReferenceID = line.ParentLine.ChildUniqueReferenceID
		}
	}
}

// helpers
func (s *SubscriptionHandlerTestSuite) createSubscriptionFromPlanPhases(phases []productcatalog.Phase) subscription.SubscriptionView {
	ctx := s.Context

	plan, err := s.PlanService.CreatePlan(ctx, plan.CreatePlanInput{
		NamespacedModel: models.NamespacedModel{
			Namespace: s.Namespace,
		},
		Plan: productcatalog.Plan{
			PlanMeta: productcatalog.PlanMeta{
				Name:     "Test Plan",
				Key:      "test-plan",
				Version:  1,
				Currency: currency.USD,
			},
			Phases: phases,
		},
	})
	s.NoError(err)

	subscriptionPlan, err := s.SubscriptionPlanAdapter.GetVersion(ctx, s.Namespace, productcatalogsubscription.PlanRefInput{
		Key:     plan.Key,
		Version: lo.ToPtr(1),
	})
	s.NoError(err)

	subsView, err := s.SubscriptionWorkflowService.CreateFromPlan(ctx, subscription.CreateSubscriptionWorkflowInput{
		ChangeSubscriptionWorkflowInput: subscription.ChangeSubscriptionWorkflowInput{
			Timing: subscription.Timing{
				Custom: lo.ToPtr(clock.Now()),
			},
			Name: "subs-1",
		},
		Namespace:  s.Namespace,
		CustomerID: s.Customer.ID,
	}, subscriptionPlan)

	s.NoError(err)
	s.NotNil(subsView)
	return subsView
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

func (s *SubscriptionHandlerTestSuite) gatheringInvoice(ctx context.Context, namespace string, customerID string) billing.Invoice {
	s.T().Helper()

	invoices, err := s.BillingService.ListInvoices(ctx, billing.ListInvoicesInput{
		Namespaces: []string{namespace},
		Customers:  []string{customerID},
		Page: pagination.Page{
			PageSize:   10,
			PageNumber: 1,
		},
		Expand: billing.InvoiceExpandAll,
		Statuses: []string{
			string(billing.InvoiceStatusGathering),
		},
	})

	s.NoError(err)
	s.Len(invoices.Items, 1)
	return invoices.Items[0]
}

func (s *SubscriptionHandlerTestSuite) expectNoGatheringInvoice(ctx context.Context, namespace string, customerID string) {
	s.T().Helper()

	invoices, err := s.BillingService.ListInvoices(ctx, billing.ListInvoicesInput{
		Namespaces: []string{namespace},
		Customers:  []string{customerID},
		Page: pagination.Page{
			PageSize:   10,
			PageNumber: 1,
		},
		Expand: billing.InvoiceExpandAll,
		Statuses: []string{
			string(billing.InvoiceStatusGathering),
		},
	})

	s.NoError(err)
	s.Len(invoices.Items, 0)
}

func (s *SubscriptionHandlerTestSuite) enableProrating() {
	s.Handler.featureFlags.EnableFlatFeeInAdvanceProrating = true
	s.Handler.featureFlags.EnableFlatFeeInArrearsProrating = true
}

func (s *SubscriptionHandlerTestSuite) getLineByChildID(invoice billing.Invoice, childID string) *billing.Line {
	s.T().Helper()

	for _, line := range invoice.Lines.OrEmpty() {
		if line.ChildUniqueReferenceID != nil && *line.ChildUniqueReferenceID == childID {
			return line
		}
	}

	s.Failf("line not found", "line with child id %s not found", childID)

	return nil
}

func (s *SubscriptionHandlerTestSuite) expectNoLineWithChildID(invoice billing.Invoice, childID string) {
	s.T().Helper()

	for _, line := range invoice.Lines.OrEmpty() {
		if line.ChildUniqueReferenceID != nil && *line.ChildUniqueReferenceID == childID {
			s.Failf("line found", "line with child id %s found", childID)
		}
	}
}
