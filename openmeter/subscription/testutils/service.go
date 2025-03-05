package subscriptiontestutils

import (
	"testing"

	"github.com/samber/lo"
	"github.com/stretchr/testify/require"

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
	subscriptionentitlement "github.com/openmeterio/openmeter/openmeter/subscription/adapters/entitlement"
	"github.com/openmeterio/openmeter/openmeter/subscription/service"
	"github.com/openmeterio/openmeter/openmeter/testutils"
	"github.com/openmeterio/openmeter/openmeter/watermill/eventbus"
)

type ExposedServiceDeps struct {
	ItemRepo            subscription.SubscriptionItemRepository
	CustomerAdapter     *testCustomerRepo
	CustomerService     customer.Service
	FeatureConnector    *testFeatureConnector
	EntitlementAdapter  subscription.EntitlementAdapter
	PlanHelper          *planHelper
	PlanService         plan.Service
	DBDeps              *DBDeps
	EntitlementRegistry *registry.Entitlement
}

type services struct {
	Service         subscription.Service
	WorkflowService subscription.WorkflowService
}

func NewService(t *testing.T, dbDeps *DBDeps) (services, ExposedServiceDeps) {
	t.Helper()
	logger := testutils.NewLogger(t)
	subRepo := NewSubscriptionRepo(t, dbDeps)
	subPhaseRepo := NewSubscriptionPhaseRepo(t, dbDeps)
	subItemRepo := NewSubscriptionItemRepo(t, dbDeps)

	meterAdapter, err := meteradapter.New([]meter.Meter{{
		Key:           ExampleFeatureMeterSlug,
		Namespace:     ExampleNamespace,
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
		MeterService:       meterAdapter,
		Publisher:          eventbus.NewMock(t),
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
		Feature: entitlementRegistry.Feature,
		Logger:  logger,
		Adapter: planRepo,
	})
	require.NoError(t, err)

	planHelper := NewPlanHelper(planService)

	svc := service.New(service.ServiceConfig{
		SubscriptionRepo:      subRepo,
		SubscriptionPhaseRepo: subPhaseRepo,
		SubscriptionItemRepo:  subItemRepo,
		CustomerService:       customer,
		EntitlementAdapter:    entitlementAdapter,
		TransactionManager:    subItemRepo,
		Publisher:             eventbus.NewMock(t),
	})

	workflowSvc := service.NewWorkflowService(service.WorkflowServiceConfig{
		Service:            svc,
		CustomerService:    customer,
		TransactionManager: subItemRepo,
	})

	return services{
			Service:         svc,
			WorkflowService: workflowSvc,
		}, ExposedServiceDeps{
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
