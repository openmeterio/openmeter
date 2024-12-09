package subscriptiontestutils

import (
	"context"
	"testing"
	"time"

	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/openmeter/meter"
	registrybuilder "github.com/openmeterio/openmeter/openmeter/registry/builder"
	streamingtestutils "github.com/openmeterio/openmeter/openmeter/streaming/testutils"
	"github.com/openmeterio/openmeter/openmeter/subscription"
	subscriptionentitlement "github.com/openmeterio/openmeter/openmeter/subscription/adapters/entitlement"
	"github.com/openmeterio/openmeter/openmeter/subscription/service"
	"github.com/openmeterio/openmeter/openmeter/testutils"
	"github.com/openmeterio/openmeter/openmeter/watermill/eventbus"
	"github.com/openmeterio/openmeter/pkg/models"
)

type ExposedServiceDeps struct {
	CustomerAdapter    *testCustomerRepo
	CustomerService    customer.Service
	FeatureConnector   *testFeatureConnector
	EntitlementAdapter subscription.EntitlementAdapter
	PlanAdapter        *planAdapter
	DBDeps             *DBDeps
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
	meterRepo := meter.NewInMemoryRepository([]models.Meter{{
		Slug:        ExampleFeatureMeterSlug,
		Namespace:   ExampleNamespace,
		Aggregation: models.MeterAggregationSum,
		WindowSize:  models.WindowSizeMinute,
	}})

	entitlementRegistry := registrybuilder.GetEntitlementRegistry(registrybuilder.EntitlementOptions{
		DatabaseClient:     dbDeps.dbClient,
		StreamingConnector: streamingtestutils.NewMockStreamingConnector(t),
		Logger:             logger,
		MeterRepository:    meterRepo,
		Publisher:          eventbus.NewMock(t),
	})

	entitlementAdapter := subscriptionentitlement.NewSubscriptionEntitlementAdapter(
		entitlementRegistry.Entitlement,
		subItemRepo,
		subPhaseRepo,
	)

	customerAdapter := NewCustomerAdapter(t, dbDeps)
	customer := NewCustomerService(t, dbDeps)

	planAdapter := NewMockPlanAdapter(t)

	svc := service.New(service.ServiceConfig{
		SubscriptionRepo:      subRepo,
		SubscriptionPhaseRepo: subPhaseRepo,
		SubscriptionItemRepo:  subItemRepo,
		CustomerService:       customer,
		EntitlementAdapter:    entitlementAdapter,
		TransactionManager:    subItemRepo,
	})

	workflowSvc := service.NewWorkflowService(service.WorkflowServiceConfig{
		Service:            svc,
		CustomerService:    customer,
		PlanAdapter:        planAdapter,
		TransactionManager: subItemRepo,
	})

	return services{
			Service:         svc,
			WorkflowService: workflowSvc,
		}, ExposedServiceDeps{
			CustomerAdapter:    customerAdapter,
			CustomerService:    customer,
			FeatureConnector:   NewTestFeatureConnector(entitlementRegistry.Feature),
			EntitlementAdapter: entitlementAdapter,
			PlanAdapter:        planAdapter,
			DBDeps:             dbDeps,
		}
}

type MockService struct {
	CreateFn   func(ctx context.Context, namespace string, spec subscription.SubscriptionSpec) (subscription.Subscription, error)
	UpdateFn   func(ctx context.Context, subscriptionID models.NamespacedID, target subscription.SubscriptionSpec) (subscription.Subscription, error)
	CancelFn   func(ctx context.Context, subscriptionID models.NamespacedID, at time.Time) (subscription.Subscription, error)
	ContinueFn func(ctx context.Context, subscriptionID models.NamespacedID) (subscription.Subscription, error)
	GetFn      func(ctx context.Context, subscriptionID models.NamespacedID) (subscription.Subscription, error)
	GetViewFn  func(ctx context.Context, subscriptionID models.NamespacedID) (subscription.SubscriptionView, error)
}

var _ subscription.Service = &MockService{}

func (s *MockService) Create(ctx context.Context, namespace string, spec subscription.SubscriptionSpec) (subscription.Subscription, error) {
	return s.CreateFn(ctx, namespace, spec)
}

func (s *MockService) Update(ctx context.Context, subscriptionID models.NamespacedID, target subscription.SubscriptionSpec) (subscription.Subscription, error) {
	return s.UpdateFn(ctx, subscriptionID, target)
}

func (s *MockService) Cancel(ctx context.Context, subscriptionID models.NamespacedID, at time.Time) (subscription.Subscription, error) {
	return s.CancelFn(ctx, subscriptionID, at)
}

func (s *MockService) Continue(ctx context.Context, subscriptionID models.NamespacedID) (subscription.Subscription, error) {
	return s.ContinueFn(ctx, subscriptionID)
}

func (s *MockService) Get(ctx context.Context, subscriptionID models.NamespacedID) (subscription.Subscription, error) {
	return s.GetFn(ctx, subscriptionID)
}

func (s *MockService) GetView(ctx context.Context, subscriptionID models.NamespacedID) (subscription.SubscriptionView, error) {
	return s.GetViewFn(ctx, subscriptionID)
}
