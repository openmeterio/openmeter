package service

import (
	"context"
	"fmt"

	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	plansubscription "github.com/openmeterio/openmeter/openmeter/productcatalog/subscription"
	"github.com/openmeterio/openmeter/openmeter/subscription"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/models"
)

func (s *service) Create(ctx context.Context, request plansubscription.CreateSubscriptionRequest) (subscription.Subscription, error) {
	var def subscription.Subscription

	if err := request.PlanInput.Validate(); err != nil {
		return def, models.NewGenericValidationError(err)
	}

	// Get the plan by reference
	planRef := request.PlanInput.AsRef()
	if planRef == nil {
		return def, fmt.Errorf("plan reference must be provided")
	}

	p, err := s.getPlanByVersion(ctx, request.WorkflowInput.Namespace, *planRef)
	if err != nil {
		return def, err
	}

	now := clock.Now()

	if p.DeletedAt != nil && !now.Before(*p.DeletedAt) {
		return def, models.NewGenericValidationError(
			fmt.Errorf("plan is deleted [namespace=%s, key=%s, version=%d, deleted_at=%s]", p.Namespace, p.Key, p.Version, p.DeletedAt),
		)
	}

	if p.StatusAt(now) != productcatalog.PlanStatusActive {
		return def, models.NewGenericValidationError(
			fmt.Errorf("plan %s@%d is not active at %s", p.Key, p.Version, now),
		)
	}

	if request.StartingPhase != nil {
		if err := s.removePhasesBeforeStartingPhase(p, *request.StartingPhase); err != nil {
			return def, err
		}
	}

	plan := PlanFromPlan(*p)

	// Create the subscription from the plan
	subView, err := s.WorkflowService.CreateFromPlan(ctx, request.WorkflowInput, plan)
	if err != nil {
		return def, err
	}

	return subView.Subscription, nil
}
