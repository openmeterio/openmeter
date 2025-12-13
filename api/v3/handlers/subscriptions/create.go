package subscriptions

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	api "github.com/openmeterio/openmeter/api/v3"
	"github.com/openmeterio/openmeter/api/v3/apierrors"
	"github.com/openmeterio/openmeter/api/v3/request"
	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/plan"
	plansubscription "github.com/openmeterio/openmeter/openmeter/productcatalog/subscription"
	"github.com/openmeterio/openmeter/pkg/framework/commonhttp"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport"
	models "github.com/openmeterio/openmeter/pkg/models"
)

type (
	CreateSubscriptionResponse = api.BillingSubscription
	CreateSubscriptionHandler  httptransport.Handler[CreateSubscriptionRequest, CreateSubscriptionResponse]
)

type CreateSubscriptionRequest struct {
	plansubscription.CreateSubscriptionRequest
}

// CreateSubscription returns a new httptransport.Handler for creating a subscription.
func (h *handler) CreateSubscription() CreateSubscriptionHandler {
	return httptransport.NewHandler(
		func(ctx context.Context, r *http.Request) (CreateSubscriptionRequest, error) {
			// Parse the request body
			body := api.BillingSubscriptionCreate{}
			if err := request.ParseBody(r, &body); err != nil {
				return CreateSubscriptionRequest{}, err
			}

			// Resolve the namespace
			ns, err := h.resolveNamespace(ctx)
			if err != nil {
				return CreateSubscriptionRequest{}, err
			}

			// Get the customer to validate it exists
			_, err = h.customerService.GetCustomer(ctx, customer.GetCustomerInput{
				CustomerID: &customer.CustomerID{
					Namespace: ns,
					ID:        body.CustomerId,
				},
			})
			if err != nil {
				return CreateSubscriptionRequest{}, fmt.Errorf("failed to get customer: %w", err)
			}

			// TODO: implement custom subscription creation
			if body.PlanId == nil && body.PlanKey == nil {
				// We use bad request error because not implemented does not provide the error context
				return CreateSubscriptionRequest{}, apierrors.NewBadRequestError(ctx,
					errors.New("custom subscription creation is not supported, provide a plan by key or by ID"),
					[]apierrors.InvalidParameter{
						{
							Field:  "plan",
							Reason: "plan is required",
							Source: apierrors.InvalidParamSourceBody,
							Rule:   "required",
						},
					},
				)
			}

			// Get the plan entity by ID or key to validate it exists
			planEntity, err := h.getPlanByIDOrKey(ctx, ns, body.PlanId, body.PlanKey, body.PlanVersion)
			if err != nil {
				return CreateSubscriptionRequest{}, fmt.Errorf("failed to get plan: %w", err)
			}

			// Convert the plan entity to a plan input
			planInput := plansubscription.PlanInput{}
			planInput.FromRef(&plansubscription.PlanRefInput{
				Key:     planEntity.Key,
				Version: &planEntity.Version,
			})

			// Convert the request to a create subscription workflow input
			subscriptionName := fmt.Sprintf("%s v%d", planEntity.Key, planEntity.Version)
			workflowInput, err := ConvertFromCreateSubscriptionRequestToCreateSubscriptionWorkflowInput(ns, subscriptionName, body)
			if err != nil {
				return CreateSubscriptionRequest{}, err
			}

			return CreateSubscriptionRequest{
				CreateSubscriptionRequest: plansubscription.CreateSubscriptionRequest{
					WorkflowInput: workflowInput,
					PlanInput:     planInput,
				},
			}, nil
		},
		func(ctx context.Context, request CreateSubscriptionRequest) (CreateSubscriptionResponse, error) {
			// Create the subscription from a plan
			m, err := h.planSubscriptionService.Create(ctx, request.CreateSubscriptionRequest)
			if err != nil {
				return CreateSubscriptionResponse{}, err
			}

			// Convert the subscription to an API subscription
			return ConvertSubscriptionToAPISubscription(m), nil
		},
		commonhttp.JSONResponseEncoderWithStatus[CreateSubscriptionResponse](http.StatusCreated),
		httptransport.AppendOptions(
			h.options,
			httptransport.WithOperationName("create-subscription"),
			httptransport.WithErrorEncoder(apierrors.GenericErrorEncoder()),
		)...,
	)
}

// getPlanByIDOrKey gets a plan by ID or key helper function
func (h *handler) getPlanByIDOrKey(ctx context.Context, namespace string, planID *string, planKey *string, planVersion *int) (*plan.Plan, error) {
	// Get the plan entity, to validate it exists
	var getPlanInput plan.GetPlanInput

	if planID != nil {
		getPlanInput = plan.GetPlanInput{
			NamespacedID: models.NamespacedID{
				Namespace: namespace,
				ID:        *planID,
			},
		}
	} else if planKey != nil {
		getPlanInput = plan.GetPlanInput{}
		// We use setters because namespace only exists on namespaced ID
		// But here we don't have a namespaced ID
		getPlanInput.Namespace = namespace
		getPlanInput.Key = *planKey

		if planVersion != nil {
			getPlanInput.Version = *planVersion
		} else {
			getPlanInput.IncludeLatest = true
		}
	} else {
		return nil, errors.New("plan id or plan key must be set")
	}

	// Get the plan entity
	return h.planService.GetPlan(ctx, getPlanInput)
}
