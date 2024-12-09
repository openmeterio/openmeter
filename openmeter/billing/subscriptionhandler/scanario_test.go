package subscriptionhandler

import (
	"log/slog"
	"testing"
	"time"

	"github.com/openmeterio/openmeter/openmeter/credit"
	grantrepo "github.com/openmeterio/openmeter/openmeter/credit/adapter"
	enttx "github.com/openmeterio/openmeter/openmeter/ent/tx"
	"github.com/openmeterio/openmeter/openmeter/entitlement"
	entitlementrepo "github.com/openmeterio/openmeter/openmeter/entitlement/adapter"
	booleanentitlement "github.com/openmeterio/openmeter/openmeter/entitlement/boolean"
	meteredentitlement "github.com/openmeterio/openmeter/openmeter/entitlement/metered"
	staticentitlement "github.com/openmeterio/openmeter/openmeter/entitlement/static"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/plan"
	planadapter "github.com/openmeterio/openmeter/openmeter/productcatalog/plan/adapter"
	planservice "github.com/openmeterio/openmeter/openmeter/productcatalog/plan/service"
	"github.com/openmeterio/openmeter/openmeter/subscription"
	subscriptionentitlementadatapter "github.com/openmeterio/openmeter/openmeter/subscription/adapters/entitlement"
	subscriptionrepo "github.com/openmeterio/openmeter/openmeter/subscription/repo"
	subscriptionservice "github.com/openmeterio/openmeter/openmeter/subscription/service"
	"github.com/openmeterio/openmeter/openmeter/watermill/eventbus"
	billingtest "github.com/openmeterio/openmeter/test/billing"
	"github.com/stretchr/testify/suite"
)

type SubscriptionHandlerTestSuite struct {
	billingtest.BaseSuite

	PlanService         plan.Service
	SubscriptionService subscription.Service
}

func (s *SubscriptionHandlerTestSuite) SetupSuite() {
	s.BaseSuite.SetupSuite()

	planAdapter, err := planadapter.New(planadapter.Config{
		Client: s.DBClient,
		Logger: slog.Default(),
	})

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
		EntitlementAdapter: subscriptionentitlementadatapter.NewEntitlementSubscriptionAdapter(
			s.SetupEntitlements(),
			subsItemRepo,
			subsRepo,
		),
		// framework
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

func (t *SubscriptionHandlerTestSuite) TestSubscriptionHappyPath() {
}
