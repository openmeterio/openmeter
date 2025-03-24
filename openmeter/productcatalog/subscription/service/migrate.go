package service

import (
	"context"
	"fmt"

	"github.com/samber/lo"

	plansubscription "github.com/openmeterio/openmeter/openmeter/productcatalog/subscription"
	"github.com/openmeterio/openmeter/openmeter/subscription"
	"github.com/openmeterio/openmeter/pkg/models"
)

func (s *service) Migrate(ctx context.Context, request plansubscription.MigrateSubscriptionRequest) (plansubscription.SubscriptionChangeResponse, error) {
	var def plansubscription.SubscriptionChangeResponse

	// Let's fetch the current sub
	sub, err := s.SubscriptionService.Get(ctx, request.ID)
	if err != nil {
		return def, err
	}

	if sub.PlanRef == nil {
		return def, models.NewGenericValidationError(
			fmt.Errorf("Subscription %s has no plan, cannot be migrated", request.ID.ID),
		)
	}

	// Let's fetch the version of the plan we should migrate to
	plan, err := s.getPlanByVersion(ctx, request.ID.Namespace, plansubscription.PlanRefInput{
		Key:     sub.PlanRef.Key,
		Version: request.TargetVersion,
	})
	if err != nil {
		return def, err
	}

	if plan.Version <= sub.PlanRef.Version {
		return def, models.NewGenericValidationError(
			fmt.Errorf("Subscription %s is already at version %d, cannot migrate to version %d", request.ID.ID, sub.PlanRef.Version, request.TargetVersion),
		)
	}

	if plan == nil {
		return def, fmt.Errorf("plan is nil")
	}

	pp, err := PlanFromPlan(*plan)
	if err != nil {
		return def, err
	}

	currView, err := s.SubscriptionService.GetView(ctx, request.ID)
	if err != nil {
		return def, err
	}

	// If we can, we want to migrate immediately.
	timing := subscription.Timing{Enum: lo.ToPtr(subscription.TimingImmediate)}

	// If we cannot, we want to migrate at the end of the current billing period
	if err := timing.ValidateForAction(subscription.SubscriptionActionCancel, &currView); err != nil {
		timing = subscription.Timing{Enum: lo.ToPtr(subscription.TimingNextBillingCycle)}
	}

	// Then let's create the subscription from the plan
	workflowInput := subscription.ChangeSubscriptionWorkflowInput{
		Timing:        timing,
		MetadataModel: sub.MetadataModel,
		Name:          sub.Name,
		Description:   sub.Description,
	}
	curr, new, err := s.WorkflowService.ChangeToPlan(ctx, request.ID, workflowInput, pp)
	if err != nil {
		return def, err
	}

	return plansubscription.SubscriptionChangeResponse{
		Current: curr,
		Next:    new,
	}, nil
}
