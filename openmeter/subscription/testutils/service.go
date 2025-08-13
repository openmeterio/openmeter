package subscriptiontestutils

import (
	"testing"
	"time"

	"github.com/oklog/ulid/v2"
	"github.com/samber/lo"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/trace/noop"

	"github.com/openmeterio/openmeter/app/config"
	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/openmeter/meter"
	meteradapter "github.com/openmeterio/openmeter/openmeter/meter/mockadapter"
	addonrepo "github.com/openmeterio/openmeter/openmeter/productcatalog/addon/adapter"
	addonservice "github.com/openmeterio/openmeter/openmeter/productcatalog/addon/service"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/plan"
	planrepo "github.com/openmeterio/openmeter/openmeter/productcatalog/plan/adapter"
	planservice "github.com/openmeterio/openmeter/openmeter/productcatalog/plan/service"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/planaddon"
	planaddonrepo "github.com/openmeterio/openmeter/openmeter/productcatalog/planaddon/adapter"
	planaddonservice "github.com/openmeterio/openmeter/openmeter/productcatalog/planaddon/service"
	"github.com/openmeterio/openmeter/openmeter/registry"
	registrybuilder "github.com/openmeterio/openmeter/openmeter/registry/builder"
	streamingtestutils "github.com/openmeterio/openmeter/openmeter/streaming/testutils"
	"github.com/openmeterio/openmeter/openmeter/subject"
	"github.com/openmeterio/openmeter/openmeter/subscription"
	subscriptionaddon "github.com/openmeterio/openmeter/openmeter/subscription/addon"
	subscriptionaddonrepo "github.com/openmeterio/openmeter/openmeter/subscription/addon/repo"
	subscriptionaddonservice "github.com/openmeterio/openmeter/openmeter/subscription/addon/service"
	subscriptionentitlement "github.com/openmeterio/openmeter/openmeter/subscription/entitlement"
	"github.com/openmeterio/openmeter/openmeter/subscription/service"
	subscriptionworkflow "github.com/openmeterio/openmeter/openmeter/subscription/workflow"
	subscriptionworkflowservice "github.com/openmeterio/openmeter/openmeter/subscription/workflow/service"
	"github.com/openmeterio/openmeter/openmeter/testutils"
	"github.com/openmeterio/openmeter/openmeter/watermill/eventbus"
	"github.com/openmeterio/openmeter/pkg/datetime"
	"github.com/openmeterio/openmeter/pkg/framework/lockr"
	"github.com/openmeterio/openmeter/pkg/models"
)

type SubscriptionDependencies struct {
	ItemRepo                 subscription.SubscriptionItemRepository
	CustomerAdapter          *testCustomerRepo
	CustomerService          customer.Service
	SubjectService           subject.Service
	FeatureConnector         *testFeatureConnector
	MeterService             meter.Service
	MockStreamingConnector   *streamingtestutils.MockStreamingConnector
	EntitlementAdapter       subscription.EntitlementAdapter
	PlanHelper               *planHelper
	PlanService              plan.Service
	DBDeps                   *DBDeps
	EntitlementRegistry      *registry.Entitlement
	SubscriptionService      subscription.Service
	WorkflowService          subscriptionworkflow.Service
	SubscriptionAddonService subscriptionaddon.Service
	AddonService             *testAddonService
	PlanAddonService         planaddon.Service
}

func NewService(t *testing.T, dbDeps *DBDeps) SubscriptionDependencies {
	t.Helper()
	logger := testutils.NewLogger(t)
	subRepo := NewSubscriptionRepo(t, dbDeps)
	subPhaseRepo := NewSubscriptionPhaseRepo(t, dbDeps)
	subItemRepo := NewSubscriptionItemRepo(t, dbDeps)
	publisher := eventbus.NewMock(t)

	lockr, err := lockr.NewLocker(&lockr.LockerConfig{Logger: logger})
	require.NoError(t, err)

	meterAdapter, err := meteradapter.New([]meter.Meter{{
		ManagedResource: models.ManagedResource{
			ID: ulid.Make().String(),
			NamespacedModel: models.NamespacedModel{
				Namespace: ExampleNamespace,
			},
			ManagedModel: models.ManagedModel{
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			},
			Name: "Meter 1",
		},
		Key:           ExampleFeatureMeterSlug,
		Aggregation:   meter.MeterAggregationSum,
		ValueProperty: lo.ToPtr("$.value"),
		EventType:     "test",
	}})
	require.NoError(t, err)
	require.NotNil(t, meterAdapter)

	mockStreaming := streamingtestutils.NewMockStreamingConnector(t)

	entitlementRegistry := registrybuilder.GetEntitlementRegistry(registrybuilder.EntitlementOptions{
		DatabaseClient:     dbDeps.DBClient,
		StreamingConnector: mockStreaming,
		Logger:             logger,
		Tracer:             noop.NewTracerProvider().Tracer("test"),
		MeterService:       meterAdapter,
		Publisher:          publisher,
		EntitlementsConfiguration: config.EntitlementsConfiguration{
			GracePeriod: datetime.ISODurationString("P1D"),
		},
		Locker: lockr,
	})

	entitlementAdapter := subscriptionentitlement.NewSubscriptionEntitlementAdapter(
		entitlementRegistry.Entitlement,
		subItemRepo,
		subPhaseRepo,
	)

	customerAdapter := NewCustomerAdapter(t, dbDeps)
	customer := NewCustomerService(t, dbDeps)
	subjectService := NewSubjectService(t, dbDeps)

	planRepo, err := planrepo.New(planrepo.Config{
		Client: dbDeps.DBClient,
		Logger: logger,
	})
	require.NoError(t, err)

	planService, err := planservice.New(planservice.Config{
		Feature:   entitlementRegistry.Feature,
		Logger:    logger,
		Adapter:   planRepo,
		Publisher: publisher,
	})
	require.NoError(t, err)

	planHelper := NewPlanHelper(planService)

	svc := service.New(service.ServiceConfig{
		SubscriptionRepo:      subRepo,
		SubscriptionPhaseRepo: subPhaseRepo,
		SubscriptionItemRepo:  subItemRepo,
		CustomerService:       customer,
		EntitlementAdapter:    entitlementAdapter,
		FeatureService:        entitlementRegistry.Feature,
		TransactionManager:    subItemRepo,
		Publisher:             publisher,
		Lockr:                 lockr,
	})

	addonRepo, err := addonrepo.New(addonrepo.Config{
		Client: dbDeps.DBClient,
		Logger: logger,
	})
	require.NoError(t, err)

	addonService, err := addonservice.New(addonservice.Config{
		Adapter:   addonRepo,
		Logger:    logger,
		Publisher: publisher,
		Feature:   entitlementRegistry.Feature,
	})
	require.NoError(t, err)

	planAddonRepo, err := planaddonrepo.New(planaddonrepo.Config{
		Client: dbDeps.DBClient,
		Logger: logger,
	})
	require.NoError(t, err)

	planAddonService, err := planaddonservice.New(planaddonservice.Config{
		Adapter:   planAddonRepo,
		Logger:    logger,
		Plan:      planService,
		Addon:     addonService,
		Publisher: publisher,
	})
	require.NoError(t, err)
	subAddRepo := subscriptionaddonrepo.NewSubscriptionAddonRepo(dbDeps.DBClient)
	subAddQtyRepo := subscriptionaddonrepo.NewSubscriptionAddonQuantityRepo(dbDeps.DBClient)

	subAddSvc, err := subscriptionaddonservice.NewService(subscriptionaddonservice.Config{
		TxManager:        subItemRepo,
		Logger:           logger,
		AddonService:     addonService,
		SubService:       svc,
		SubAddRepo:       subAddRepo,
		SubAddQtyRepo:    subAddQtyRepo,
		PlanAddonService: planAddonService,
		Publisher:        publisher,
	})
	require.NoError(t, err)

	workflowSvc := subscriptionworkflowservice.NewWorkflowService(subscriptionworkflowservice.WorkflowServiceConfig{
		Service:            svc,
		CustomerService:    customer,
		TransactionManager: subItemRepo,
		AddonService:       subAddSvc,
		Logger:             logger.With("subsystem", "subscription.workflow.service"),
		Lockr:              lockr,
	})

	return SubscriptionDependencies{
		SubscriptionService:      svc,
		WorkflowService:          workflowSvc,
		CustomerAdapter:          customerAdapter,
		CustomerService:          customer,
		SubjectService:           subjectService,
		FeatureConnector:         NewTestFeatureConnector(entitlementRegistry.Feature),
		EntitlementAdapter:       entitlementAdapter,
		DBDeps:                   dbDeps,
		PlanHelper:               planHelper,
		PlanService:              planService,
		ItemRepo:                 subItemRepo,
		EntitlementRegistry:      entitlementRegistry,
		SubscriptionAddonService: subAddSvc,
		AddonService:             NewTestAddonService(addonService),
		PlanAddonService:         planAddonService,
		MeterService:             meterAdapter,
		MockStreamingConnector:   mockStreaming,
	}
}
