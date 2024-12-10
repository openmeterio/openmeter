package subscriptionhandler

import (
	"context"
	"fmt"
	"log/slog"
	"testing"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/invopop/gobl/currency"
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
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/datex"
	"github.com/openmeterio/openmeter/pkg/models"
	billingtest "github.com/openmeterio/openmeter/test/billing"
	"github.com/samber/lo"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"gopkg.in/yaml.v3"
)

type SubscriptionHandlerTestSuite struct {
	billingtest.BaseSuite

	PlanService                 plan.Service
	SubscriptionService         subscription.Service
	SubscrpiptionPlanAdapter    plansubscription.Adapter
	SubscriptionWorkflowService subscription.WorkflowService
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

func (s *SubscriptionHandlerTestSuite) TestSubscriptionHappyPath() {
	ctx := context.Background()
	namespace := "test-subs-happy-path"
	start := time.Now()

	s.MeterRepo.ReplaceMeters(ctx, []models.Meter{
		{
			Namespace:   namespace,
			Slug:        "api-requests-total",
			WindowSize:  models.WindowSizeMinute,
			Aggregation: models.MeterAggregationSum,
		},
	})

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
					// TODO: let's add discount handling (as this could be a 100% discount for the first month)
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
						Key:        "discounted phase",
						StartAfter: datex.MustParse(s.T(), "P1M"),
					},
					// TODO: 50% discount
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
						Key:        "final phase",
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
		Version: lo.ToPtr(1), // TODO: what is the expected behavior if version is nil?, right now it just throws an
	})

	subsView, err := s.SubscriptionWorkflowService.CreateFromPlan(ctx, subscription.CreateSubscriptionWorkflowInput{
		Namespace:  namespace,
		ActiveFrom: start,
		CustomerID: customerEntity.ID,
		Name:       "subs-1",
	}, subscriptionPlan)

	s.NoError(err)
	s.NotNil(subsView)

	freeTierPhase := getPhraseByKey(s.T(), subsView, "free-trial")
	s.Equal(lo.ToPtr(datex.MustParse(s.T(), "P1M")), freeTierPhase.ItemsByKey[apiRequestsTotalFeatureKey][0].Spec.RateCard.BillingCadence)

	upcomingLineItems, err := GetUpcomingLineItems(ctx, GetUpcomingLineItemsInput{
		SubscriptionView: subsView,
		// StartFrom:        start,
	})

	linesString, err := yaml.Marshal(upcomingLineItems)
	s.NoError(err)
	fmt.Println(string(linesString))

	// TODO: remove, this is just debugging output
	json, err := yaml.Marshal(subsView)
	s.NoError(err)
	fmt.Println(string(json))

	s.T().Fail()
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
