package service

import (
	"context"
	"fmt"

	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	plansubscription "github.com/openmeterio/openmeter/openmeter/productcatalog/subscription"
	"github.com/openmeterio/openmeter/openmeter/subscription"
	"github.com/openmeterio/openmeter/pkg/models"
)

func (s *service) Change(ctx context.Context, request plansubscription.ChangeSubscriptionRequest) (plansubscription.SubscriptionChangeResponse, error) {
	var def plansubscription.SubscriptionChangeResponse
	var plan subscription.Plan

	if err := request.PlanInput.Validate(); err != nil {
		return def, &models.GenericUserError{Inner: err}
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

		if p.Status() != productcatalog.ActiveStatus {
			return def, &models.GenericUserError{Inner: fmt.Errorf("plan is not active")}
		}

		pp, err := PlanFromPlan(*p)
		if err != nil {
			return def, err
		}

		plan = pp
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
