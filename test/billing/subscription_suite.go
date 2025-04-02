package billing

import (
	"errors"
	"log/slog"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/trace/noop"

	"github.com/openmeterio/openmeter/openmeter/credit"
	grantrepo "github.com/openmeterio/openmeter/openmeter/credit/adapter"
	"github.com/openmeterio/openmeter/openmeter/credit/balance"
	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/openmeter/ent/db"
	enttx "github.com/openmeterio/openmeter/openmeter/ent/tx"
	"github.com/openmeterio/openmeter/openmeter/entitlement"
	entitlementrepo "github.com/openmeterio/openmeter/openmeter/entitlement/adapter"
	booleanentitlement "github.com/openmeterio/openmeter/openmeter/entitlement/boolean"
	meteredentitlement "github.com/openmeterio/openmeter/openmeter/entitlement/metered"
	staticentitlement "github.com/openmeterio/openmeter/openmeter/entitlement/static"
	meteradapter "github.com/openmeterio/openmeter/openmeter/meter/mockadapter"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/feature"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/plan"
	planadapter "github.com/openmeterio/openmeter/openmeter/productcatalog/plan/adapter"
	planservice "github.com/openmeterio/openmeter/openmeter/productcatalog/plan/service"
	subscriptiontestutils "github.com/openmeterio/openmeter/openmeter/productcatalog/subscription/testutils"
	streamingtestutils "github.com/openmeterio/openmeter/openmeter/streaming/testutils"
	"github.com/openmeterio/openmeter/openmeter/subscription"
	subscriptionentitlementadatapter "github.com/openmeterio/openmeter/openmeter/subscription/entitlement"
	subscriptionrepo "github.com/openmeterio/openmeter/openmeter/subscription/repo"
	subscriptionservice "github.com/openmeterio/openmeter/openmeter/subscription/service"
	"github.com/openmeterio/openmeter/openmeter/watermill/eventbus"
	"github.com/openmeterio/openmeter/pkg/isodate"
)

type SubscriptionMixin struct {
	PlanService                 plan.Service
	SubscriptionService         subscription.Service
	SubscriptionPlanAdapter     subscriptiontestutils.PlanSubscriptionAdapter
	SubscriptionWorkflowService subscription.WorkflowService
}

type SubscriptionMixInDependencies struct {
	DBClient *db.Client

	FeatureRepo     feature.FeatureRepo
	FeatureService  feature.FeatureConnector
	CustomerService customer.Service

	MeterAdapter           *meteradapter.TestAdapter
	MockStreamingConnector *streamingtestutils.MockStreamingConnector
}

func (d SubscriptionMixInDependencies) Validate() error {
	if d.DBClient == nil {
		return errors.New("DBClient is required")
	}

	if d.FeatureRepo == nil {
		return errors.New("FeatureRepo is required")
	}

	if d.FeatureService == nil {
		return errors.New("FeatureService is required")
	}

	if d.CustomerService == nil {
		return errors.New("CustomerService is required")
	}

	if d.MeterAdapter == nil {
		return errors.New("MeterAdapter is required")
	}

	if d.MockStreamingConnector == nil {
		return errors.New("MockStreamingConnector is required")
	}

	return nil
}

func (s *SubscriptionMixin) SetupSuite(t *testing.T, deps SubscriptionMixInDependencies) {
	require.NoError(t, deps.Validate())

	publisher := eventbus.NewMock(t)

	planAdapter, err := planadapter.New(planadapter.Config{
		Client: deps.DBClient,
		Logger: slog.Default(),
	})
	require.NoError(t, err)

	planService, err := planservice.New(planservice.Config{
		Feature:   deps.FeatureService,
		Adapter:   planAdapter,
		Logger:    slog.Default(),
		Publisher: publisher,
	})
	require.NoError(t, err)

	s.PlanService = planService

	subsRepo := subscriptionrepo.NewSubscriptionRepo(deps.DBClient)
	subsItemRepo := subscriptionrepo.NewSubscriptionItemRepo(deps.DBClient)

	s.SubscriptionService = subscriptionservice.New(subscriptionservice.ServiceConfig{
		SubscriptionRepo:      subsRepo,
		SubscriptionPhaseRepo: subscriptionrepo.NewSubscriptionPhaseRepo(deps.DBClient),
		SubscriptionItemRepo:  subsItemRepo,
		// connectors
		CustomerService: deps.CustomerService,
		FeatureService:  deps.FeatureService,
		// adapters
		EntitlementAdapter: subscriptionentitlementadatapter.NewSubscriptionEntitlementAdapter(
			s.SetupEntitlements(t, deps),
			subsItemRepo,
			subsRepo,
		),
		// framework
		TransactionManager: subsRepo,
		// events
		Publisher: publisher,
	})

	s.SubscriptionPlanAdapter = subscriptiontestutils.NewPlanSubscriptionAdapter(subscriptiontestutils.PlanSubscriptionAdapterConfig{
		PlanService: planService,
		Logger:      slog.Default(),
	})

	s.SubscriptionWorkflowService = subscriptionservice.NewWorkflowService(subscriptionservice.WorkflowServiceConfig{
		Service:            s.SubscriptionService,
		CustomerService:    deps.CustomerService,
		TransactionManager: subsRepo,
	})
}

func (s *SubscriptionMixin) SetupEntitlements(t *testing.T, deps SubscriptionMixInDependencies) entitlement.Connector {
	tracer := noop.NewTracerProvider().Tracer("test")

	// Init grants/credit
	grantRepo := grantrepo.NewPostgresGrantRepo(deps.DBClient)
	balanceSnapshotRepo := grantrepo.NewPostgresBalanceSnapshotRepo(deps.DBClient)

	// Init entitlements
	entitlementRepo := entitlementrepo.NewPostgresEntitlementRepo(deps.DBClient)
	usageResetRepo := entitlementrepo.NewPostgresUsageResetRepo(deps.DBClient)

	mockPublisher := eventbus.NewMock(t)

	owner := meteredentitlement.NewEntitlementGrantOwnerAdapter(
		deps.FeatureRepo,
		entitlementRepo,
		usageResetRepo,
		deps.MeterAdapter,
		slog.Default(),
		tracer,
	)

	balanceSnapshotService := balance.NewSnapshotService(balance.SnapshotServiceConfig{
		OwnerConnector:     owner,
		StreamingConnector: deps.MockStreamingConnector,
		Repo:               balanceSnapshotRepo,
	})

	transactionManager := enttx.NewCreator(deps.DBClient)

	creditConnector := credit.NewCreditConnector(
		credit.CreditConnectorConfig{
			GrantRepo:              grantRepo,
			BalanceSnapshotService: balanceSnapshotService,
			OwnerConnector:         owner,
			StreamingConnector:     deps.MockStreamingConnector,
			Logger:                 slog.Default(),
			Tracer:                 tracer,
			Granularity:            time.Minute,
			Publisher:              mockPublisher,
			TransactionManager:     transactionManager,
			SnapshotGracePeriod:    isodate.MustParse(t, "P1W"),
		},
	)

	meteredEntitlementConnector := meteredentitlement.NewMeteredEntitlementConnector(
		deps.MockStreamingConnector,
		owner,
		creditConnector,
		creditConnector,
		grantRepo,
		entitlementRepo,
		mockPublisher,
		slog.Default(),
		tracer,
	)

	staticEntitlementConnector := staticentitlement.NewStaticEntitlementConnector()
	booleanEntitlementConnector := booleanentitlement.NewBooleanEntitlementConnector()

	return entitlement.NewEntitlementConnector(
		entitlementRepo,
		deps.FeatureService,
		deps.MeterAdapter,
		meteredEntitlementConnector,
		staticEntitlementConnector,
		booleanEntitlementConnector,
		mockPublisher,
	)
}
