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
	// adapters
	PlanAdapter subscription.PlanAdapter
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

func (s *workflowService) CreateFromPlan(ctx context.Context, inp subscription.CreateFromPlanInput) (subscription.SubscriptionView, error) {
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

	// Let's validate the plan exists
	plan, err := s.PlanAdapter.GetVersion(ctx, inp.Namespace, inp.Plan.Key, inp.Plan.Version)
	if err != nil {
		return def, fmt.Errorf("failed to fetch plan: %w", err)
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
			return subscription.SubscriptionView{}, &models.GenericUserError{Message: fmt.Sprintf("invalid patch at index %d: %s", i, err.Error())}
		}
	}

	// Let's apply the customizations
	spec := curr.AsSpec()

	err = spec.ApplyPatches(lo.Map(customizations, subscription.ToApplies), subscription.ApplyContext{
		Operation:   subscription.SpecOperationEdit,
		CurrentTime: clock.Now(),
	})
	if sErr, ok := lo.ErrorsAs[*subscription.SpecValidationError](err); ok {
		// FIXME: error details are lost here
		return subscription.SubscriptionView{}, &models.GenericUserError{Message: sErr.Error()}
	} else if err != nil {
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
