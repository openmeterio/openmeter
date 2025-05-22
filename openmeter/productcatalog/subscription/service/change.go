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

func (s *service) Change(ctx context.Context, request plansubscription.ChangeSubscriptionRequest) (plansubscription.SubscriptionChangeResponse, error) {
	var def plansubscription.SubscriptionChangeResponse
	var plan subscription.Plan

	if err := request.PlanInput.Validate(); err != nil {
		return def, models.NewGenericValidationError(err)
	}

	if request.PlanInput.AsInput() != nil {
		p, err := PlanFromPlanInput(*request.PlanInput.AsInput())
		if err != nil {
			return def, err
		}

		plan = p
	} else if request.PlanInput.AsRef() != nil {
		p, err := s.getPlanByVersion(ctx, request.ID.Namespace, *request.PlanInput.AsRef())
		if err != nil {
			return def, err
		}

		now := clock.Now()

		pStatus := p.StatusAt(now)
		if pStatus != productcatalog.PlanStatusActive {
			return def, models.NewGenericValidationError(fmt.Errorf("plan %s@%d is not active at %s", p.Key, p.Version, now))
		}

		if p.DeletedAt != nil && !now.Before(*p.DeletedAt) {
			return def, models.NewGenericValidationError(
				fmt.Errorf("plan is deleted [namespace=%s, key=%s, version=%d, deleted_at=%s]", p.Namespace, p.Key, p.Version, p.DeletedAt),
			)
		}

		// Let's find the starting phase
		if request.StartingPhase != nil {
			if err := s.removePhasesBeforeStartingPhase(p, *request.StartingPhase); err != nil {
				return def, err
			}
		}

		plan = PlanFromPlan(*p)
	} else {
		return def, fmt.Errorf("plan or plan reference must be provided, input should already be validated")
	}

	// Then let's create the subscription from the plan
	curr, new, err := s.WorkflowService.ChangeToPlan(ctx, request.ID, request.WorkflowInput, plan)
	if err != nil {
		return def, err
	}

	return plansubscription.SubscriptionChangeResponse{
		Current: curr,
		Next:    new,
	}, nil
}
