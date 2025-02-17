package service

import (
	"context"
	"fmt"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/customer"
	plansubscription "github.com/openmeterio/openmeter/openmeter/productcatalog/subscription"
	"github.com/openmeterio/openmeter/openmeter/subscription"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/pagination"
)

func (s *service) Create(ctx context.Context, request plansubscription.CreateSubscriptionRequest) (subscription.Subscription, error) {
	var def subscription.Subscription

	// Let's resolve the customer
	customerID := lo.CoalesceOrEmpty(request.WorkflowInput.CustomerID, request.CustomerRef.ID)
	if customerID == "" {
		cust, err := s.CustomerService.ListCustomers(ctx, customer.ListCustomersInput{
			Key:            lo.ToPtr(request.CustomerRef.Key),
			Namespace:      request.WorkflowInput.Namespace,
			Page:           pagination.NewPage(1, 1),
			IncludeDeleted: false,
		})
		if err != nil {
			return def, err
		}

		if cust.TotalCount != 1 {
			return def, &models.GenericConflictError{
				Inner: fmt.Errorf("%d customers found with key %s", cust.TotalCount, request.CustomerRef.Key),
			}
		}

		customerID = cust.Items[0].ID
	}

	request.WorkflowInput.CustomerID = customerID

	// Let's build the plan input
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
