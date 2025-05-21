package service

import (
	"context"
	"fmt"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	plansubscription "github.com/openmeterio/openmeter/openmeter/productcatalog/subscription"
	"github.com/openmeterio/openmeter/openmeter/subscription"
	subscriptionworkflow "github.com/openmeterio/openmeter/openmeter/subscription/workflow"
	"github.com/openmeterio/openmeter/pkg/clock"
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
			fmt.Errorf("subscription %s has no plan, cannot be migrated", request.ID.ID),
		)
	}

	// Let's fetch the version of the p we should migrate to
	p, err := s.getPlanByVersion(ctx, request.ID.Namespace, plansubscription.PlanRefInput{
		Key:     sub.PlanRef.Key,
		Version: request.TargetVersion,
	})
	if err != nil {
		return def, err
	}

	if p == nil {
		return def, fmt.Errorf("plan is nil")
	}

	if p.Version <= sub.PlanRef.Version {
		return def, models.NewGenericValidationError(
			fmt.Errorf("subscription %s is already at version %d, cannot migrate to version %d", request.ID.ID, sub.PlanRef.Version, request.TargetVersion),
		)
	}

	now := clock.Now()

	if p.DeletedAt != nil && !now.Before(*p.DeletedAt) {
		return def, models.NewGenericValidationError(
			fmt.Errorf("plan is deleted [namespace=%s, key=%s, version=%d, deleted_at=%s]",
				p.Namespace, p.Key, p.Version, p.DeletedAt),
		)
	}

	if !lo.Contains([]productcatalog.PlanStatus{
		productcatalog.PlanStatusActive,
		productcatalog.PlanStatusArchived,
	}, p.StatusAt(now)) {
		return def, models.NewGenericValidationError(
			fmt.Errorf("plan %s@%d is not active or archived at %s", p.Key, p.Version, now),
		)
	}

	// Let's find the starting phase
	if request.StartingPhase != nil {
		if err := s.removePhasesBeforeStartingPhase(p, *request.StartingPhase); err != nil {
			return def, err
		}
	}

	pp := PlanFromPlan(*p)

	var timing subscription.Timing

	if request.Timing != nil {
		timing = *request.Timing
	} else {
		currView, err := s.SubscriptionService.GetView(ctx, request.ID)
		if err != nil {
			return def, err
		}

		// If we can, we want to migrate immediately.
		timing = subscription.Timing{Enum: lo.ToPtr(subscription.TimingImmediate)}

		// If we cannot, we want to migrate at the end of the current billing period
		if err := timing.ValidateForAction(subscription.SubscriptionActionCancel, &currView); err != nil {
			timing = subscription.Timing{Enum: lo.ToPtr(subscription.TimingNextBillingCycle)}
		}
	}

	// Then let's create the subscription from the plan
	workflowInput := subscriptionworkflow.ChangeSubscriptionWorkflowInput{
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
