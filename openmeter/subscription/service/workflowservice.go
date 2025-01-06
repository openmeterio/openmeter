package service

import (
	"context"
	"fmt"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/customer"
	customerentity "github.com/openmeterio/openmeter/openmeter/customer/entity"
	"github.com/openmeterio/openmeter/openmeter/subscription"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/framework/transaction"
	"github.com/openmeterio/openmeter/pkg/models"
)

type WorkflowServiceConfig struct {
	Service subscription.Service
	// connectors
	CustomerService customer.Service
	// framework
	TransactionManager transaction.Creator
}

type workflowService struct {
	WorkflowServiceConfig
}

func NewWorkflowService(cfg WorkflowServiceConfig) subscription.WorkflowService {
	return &workflowService{
		WorkflowServiceConfig: cfg,
	}
}

var _ subscription.WorkflowService = &workflowService{}

func (s *workflowService) CreateFromPlan(ctx context.Context, inp subscription.CreateSubscriptionWorkflowInput, plan subscription.Plan) (subscription.SubscriptionView, error) {
	var def subscription.SubscriptionView

	// Let's validate the customer exists
	cust, err := s.CustomerService.GetCustomer(ctx, customerentity.GetCustomerInput{
		Namespace: inp.Namespace,
		ID:        inp.CustomerID,
	})
	if err != nil {
		return def, fmt.Errorf("failed to fetch customer: %w", err)
	}

	if cust == nil {
		return def, fmt.Errorf("unexpected nil customer")
	}

	// Let's create the new Spec
	spec, err := subscription.NewSpecFromPlan(plan, subscription.CreateSubscriptionCustomerInput{
		CustomerId:     cust.ID,
		Currency:       plan.Currency(),
		ActiveFrom:     inp.ActiveFrom,
		AnnotatedModel: inp.AnnotatedModel,
		Name:           inp.Name,
		Description:    inp.Description,
	})
	if err != nil {
		return def, fmt.Errorf("failed to create spec from plan: %w", err)
	}

	// Finally, let's create the subscription
	return transaction.Run(ctx, s.TransactionManager, func(ctx context.Context) (subscription.SubscriptionView, error) {
		sub, err := s.Service.Create(ctx, inp.Namespace, spec)
		if err != nil {
			return def, fmt.Errorf("failed to create subscription: %w", err)
		}

		return s.Service.GetView(ctx, sub.NamespacedID)
	})
}

func (s *workflowService) EditRunning(ctx context.Context, subscriptionID models.NamespacedID, customizations []subscription.Patch) (subscription.SubscriptionView, error) {
	// First, let's fetch the current state of the Subscription
	curr, err := s.Service.GetView(ctx, subscriptionID)
	if err != nil {
		return subscription.SubscriptionView{}, fmt.Errorf("failed to fetch subscription: %w", err)
	}

	// Let's validate the patches
	for i, patch := range customizations {
		if err := patch.Validate(); err != nil {
			return subscription.SubscriptionView{}, &models.GenericUserError{Inner: fmt.Errorf("invalid patch at index %d: %s", i, err.Error())}
		}
	}

	// Let's apply the customizations
	spec := curr.AsSpec()

	if err := spec.ApplyPatches(lo.Map(customizations, subscription.ToApplies), subscription.ApplyContext{
		Operation:   subscription.SpecOperationEdit,
		CurrentTime: clock.Now(),
	}); err != nil {
		return subscription.SubscriptionView{}, fmt.Errorf("failed to apply customizations: %w", err)
	}

	// Finally, let's update the subscription
	return transaction.Run(ctx, s.TransactionManager, func(ctx context.Context) (subscription.SubscriptionView, error) {
		sub, err := s.Service.Update(ctx, subscriptionID, spec)
		if err != nil {
			return subscription.SubscriptionView{}, fmt.Errorf("failed to update subscription: %w", err)
		}

		return s.Service.GetView(ctx, sub.NamespacedID)
	})
}

func (s *workflowService) ChangeToPlan(ctx context.Context, subscriptionID models.NamespacedID, inp subscription.ChangeSubscriptionWorkflowInput, plan subscription.Plan) (subscription.Subscription, subscription.SubscriptionView, error) {
	// typing helper
	type res struct {
		curr subscription.Subscription
		new  subscription.SubscriptionView
	}

	// Changing the plan means canceling the current subscription and creating a new one with the provided timestamp
	r, err := transaction.Run(ctx, s.TransactionManager, func(ctx context.Context) (res, error) {
		// First, let's try to cancel the current subscription
		curr, err := s.Service.Cancel(ctx, subscriptionID, inp.ActiveFrom)
		if err != nil {
			return res{}, fmt.Errorf("failed to end current subscription: %w", err)
		}

		// Now, let's create a new subscription with the new plan
		new, err := s.CreateFromPlan(ctx, subscription.CreateSubscriptionWorkflowInput{
			ChangeSubscriptionWorkflowInput: inp,
			Namespace:                       curr.Namespace,
			CustomerID:                      curr.CustomerId,
		}, plan)
		if err != nil {
			return res{}, fmt.Errorf("failed to create new subscription: %w", err)
		}

		// Let's just return after a great success
		return res{curr, new}, nil
	})

	return r.curr, r.new, err
}
