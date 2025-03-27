package service

import (
	"context"
	"fmt"

	plansubscription "github.com/openmeterio/openmeter/openmeter/productcatalog/subscription"
	"github.com/openmeterio/openmeter/openmeter/subscription"
	"github.com/openmeterio/openmeter/pkg/models"
)

func (s *service) Create(ctx context.Context, request plansubscription.CreateSubscriptionRequest) (subscription.Subscription, error) {
	var def subscription.Subscription

	// Let's build the plan input
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
		p, err := s.getPlanByVersion(ctx, request.WorkflowInput.Namespace, *request.PlanInput.AsRef())
		if err != nil {
			return def, err
		}

		pp, err := PlanFromPlan(*p)
		if err != nil {
			return def, err
		}

		plan = pp
	} else {
		return def, fmt.Errorf("plan or plan reference must be provided, should have validated already")
	}

	// Then let's create the subscription form the plan
	subView, err := s.WorkflowService.CreateFromPlan(ctx, request.WorkflowInput, plan)
	if err != nil {
		return def, err
	}

	return subView.Subscription, nil
}
