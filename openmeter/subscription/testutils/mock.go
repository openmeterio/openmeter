package subscriptiontestutils

import (
	"context"
	"time"

	"github.com/openmeterio/openmeter/openmeter/subscription"
	"github.com/openmeterio/openmeter/pkg/models"
)

type MockService struct {
	CreateFn   func(ctx context.Context, namespace string, spec subscription.SubscriptionSpec) (subscription.Subscription, error)
	UpdateFn   func(ctx context.Context, subscriptionID models.NamespacedID, target subscription.SubscriptionSpec) (subscription.Subscription, error)
	DeleteFn   func(ctx context.Context, subscriptionID models.NamespacedID) error
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

func (s *MockService) Delete(ctx context.Context, subscriptionID models.NamespacedID) error {
	return s.DeleteFn(ctx, subscriptionID)
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

type MockWorkflowService struct {
	CreateFromPlanFn func(ctx context.Context, inp subscription.CreateSubscriptionWorkflowInput, plan subscription.Plan) (subscription.SubscriptionView, error)
	EditRunningFn    func(ctx context.Context, subscriptionID models.NamespacedID, customizations []subscription.Patch) (subscription.SubscriptionView, error)
	ChangeToPlanFn   func(ctx context.Context, subscriptionID models.NamespacedID, inp subscription.ChangeSubscriptionWorkflowInput, plan subscription.Plan) (current subscription.Subscription, new subscription.SubscriptionView, err error)
}

func (s *MockWorkflowService) CreateFromPlan(ctx context.Context, inp subscription.CreateSubscriptionWorkflowInput, plan subscription.Plan) (subscription.SubscriptionView, error) {
	return s.CreateFromPlanFn(ctx, inp, plan)
}

func (s *MockWorkflowService) EditRunning(ctx context.Context, subscriptionID models.NamespacedID, customizations []subscription.Patch) (subscription.SubscriptionView, error) {
	return s.EditRunningFn(ctx, subscriptionID, customizations)
}

func (s *MockWorkflowService) ChangeToPlan(ctx context.Context, subscriptionID models.NamespacedID, inp subscription.ChangeSubscriptionWorkflowInput, plan subscription.Plan) (current subscription.Subscription, new subscription.SubscriptionView, err error) {
	return s.ChangeToPlanFn(ctx, subscriptionID, inp, plan)
}
