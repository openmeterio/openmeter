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
	"github.com/openmeterio/openmeter/openmeter/productcatalog/plan"
	planrepo "github.com/openmeterio/openmeter/openmeter/productcatalog/plan/adapter"
	planservice "github.com/openmeterio/openmeter/openmeter/productcatalog/plan/service"
	"github.com/openmeterio/openmeter/openmeter/registry"
	registrybuilder "github.com/openmeterio/openmeter/openmeter/registry/builder"
	streamingtestutils "github.com/openmeterio/openmeter/openmeter/streaming/testutils"
	"github.com/openmeterio/openmeter/openmeter/subscription"
	subscriptionentitlement "github.com/openmeterio/openmeter/openmeter/subscription/entitlement"
	"github.com/openmeterio/openmeter/openmeter/subscription/service"
	subscriptionworkflow "github.com/openmeterio/openmeter/openmeter/subscription/workflow"
	subscriptionworkflowservice "github.com/openmeterio/openmeter/openmeter/subscription/workflow/service"
	"github.com/openmeterio/openmeter/openmeter/testutils"
	"github.com/openmeterio/openmeter/openmeter/watermill/eventbus"
	"github.com/openmeterio/openmeter/pkg/isodate"
	"github.com/openmeterio/openmeter/pkg/models"
)

type SubscriptionDependencies struct {
	ItemRepo            subscription.SubscriptionItemRepository
	CustomerAdapter     *testCustomerRepo
	CustomerService     customer.Service
	FeatureConnector    *testFeatureConnector
	EntitlementAdapter  subscription.EntitlementAdapter
	PlanHelper          *planHelper
	PlanService         plan.Service
	DBDeps              *DBDeps
	EntitlementRegistry *registry.Entitlement
	SubscriptionService subscription.Service
	WorkflowService     subscriptionworkflow.Service
}

func NewService(t *testing.T, dbDeps *DBDeps) SubscriptionDependencies {
	t.Helper()
	logger := testutils.NewLogger(t)
	subRepo := NewSubscriptionRepo(t, dbDeps)
	subPhaseRepo := NewSubscriptionPhaseRepo(t, dbDeps)
	subItemRepo := NewSubscriptionItemRepo(t, dbDeps)
	publisher := eventbus.NewMock(t)

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

	entitlementRegistry := registrybuilder.GetEntitlementRegistry(registrybuilder.EntitlementOptions{
		DatabaseClient:     dbDeps.DBClient,
		StreamingConnector: streamingtestutils.NewMockStreamingConnector(t),
		Logger:             logger,
		Tracer:             noop.NewTracerProvider().Tracer("test"),
		MeterService:       meterAdapter,
		Publisher:          publisher,
		EntitlementsConfiguration: config.EntitlementsConfiguration{
			GracePeriod: isodate.String("P1D"),
		},
	})

	entitlementAdapter := subscriptionentitlement.NewSubscriptionEntitlementAdapter(
		entitlementRegistry.Entitlement,
		subItemRepo,
		subPhaseRepo,
	)

	customerAdapter := NewCustomerAdapter(t, dbDeps)
	customer := NewCustomerService(t, dbDeps)

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
	})

	workflowSvc := subscriptionworkflowservice.NewWorkflowService(subscriptionworkflowservice.WorkflowServiceConfig{
		Service:            svc,
		CustomerService:    customer,
		TransactionManager: subItemRepo,
	})

	return SubscriptionDependencies{
		WorkflowService:     workflowSvc,
		SubscriptionService: svc,
		CustomerAdapter:     customerAdapter,
		CustomerService:     customer,
		FeatureConnector:    NewTestFeatureConnector(entitlementRegistry.Feature),
		EntitlementAdapter:  entitlementAdapter,
		DBDeps:              dbDeps,
		PlanHelper:          planHelper,
		PlanService:         planService,
		ItemRepo:            subItemRepo,
		EntitlementRegistry: entitlementRegistry,
	}
}
