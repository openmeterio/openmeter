package subscriptions

import (
	"context"
	"errors"
	"net/http"

	api "github.com/openmeterio/openmeter/api/v3"
	"github.com/openmeterio/openmeter/api/v3/apierrors"
	"github.com/openmeterio/openmeter/api/v3/request"
	plansubscription "github.com/openmeterio/openmeter/openmeter/productcatalog/subscription"
	subscriptionworkflow "github.com/openmeterio/openmeter/openmeter/subscription/workflow"
	"github.com/openmeterio/openmeter/pkg/framework/commonhttp"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport"
	models "github.com/openmeterio/openmeter/pkg/models"
)

type (
	ChangeSubscriptionRequest struct {
		ID            models.NamespacedID
		PlanInput     plansubscription.PlanInput
		WorkflowInput subscriptionworkflow.ChangeSubscriptionWorkflowInput
	}
	ChangeSubscriptionResponse = api.BillingSubscriptionChangeResponse
	ChangeSubscriptionParams   = string
	ChangeSubscriptionHandler  httptransport.HandlerWithArgs[ChangeSubscriptionRequest, ChangeSubscriptionResponse, ChangeSubscriptionParams]
)

func (h *handler) ChangeSubscription() ChangeSubscriptionHandler {
	return httptransport.NewHandlerWithArgs(
		func(ctx context.Context, r *http.Request, subscriptionID ChangeSubscriptionParams) (ChangeSubscriptionRequest, error) {
			// Parse body
			body := api.BillingSubscriptionChange{}
			if err := request.ParseBody(r, &body); err != nil {
				return ChangeSubscriptionRequest{}, err
			}

			// Resolve namespace
			ns, err := h.resolveNamespace(ctx)
			if err != nil {
				return ChangeSubscriptionRequest{}, err
			}

			id := models.NamespacedID{
				Namespace: ns,
				ID:        subscriptionID,
			}

			// Fetch current subscription for defaults (name/desc/metadata)
			curr, err := h.subscriptionService.Get(ctx, id)
			if err != nil {
				return ChangeSubscriptionRequest{}, err
			}

			// Currently only plan-based subscription change is supported, so plan ref is required.
			if body.PlanId == nil && body.PlanKey == nil {
				reason := "one of plan_id or plan_key is required"
				return ChangeSubscriptionRequest{}, apierrors.NewBadRequestError(ctx,
					errors.New(reason),
					[]apierrors.InvalidParameter{
						{
							Field:  "plan_id",
							Reason: reason,
							Source: apierrors.InvalidParamSourceBody,
							Rule:   "required",
						},
						{
							Field:  "plan_key",
							Reason: reason,
							Source: apierrors.InvalidParamSourceBody,
							Rule:   "required",
						},
					},
				)
			}

			// Validate that plan exists and resolve to a concrete version
			planEntity, err := h.getPlanByIDOrKey(ctx, ns, body.PlanId, body.PlanKey, body.PlanVersion)
			if err != nil {
				return ChangeSubscriptionRequest{}, err
			}

			planInput := plansubscription.PlanInput{}
			planInput.FromRef(&plansubscription.PlanRefInput{
				Key:     planEntity.Key,
				Version: &planEntity.Version,
			})

			timing, err := ConvertBillingSubscriptionEditTimingToSubscriptionTiming(body.Timing)
			if err != nil {
				return ChangeSubscriptionRequest{}, err
			}

			metadataModel := curr.MetadataModel
			if body.Labels != nil {
				metadataModel = models.MetadataModel{
					Metadata: models.Metadata(*body.Labels),
				}
			}

			workflowInput := subscriptionworkflow.ChangeSubscriptionWorkflowInput{
				Timing:        timing,
				MetadataModel: metadataModel,
				Name:          curr.Name,
				Description:   curr.Description,
				BillingAnchor: body.BillingAnchor,
			}

			return ChangeSubscriptionRequest{
				ID:            id,
				PlanInput:     planInput,
				WorkflowInput: workflowInput,
			}, nil
		},
		func(ctx context.Context, req ChangeSubscriptionRequest) (ChangeSubscriptionResponse, error) {
			resp, err := h.planSubscriptionService.Change(ctx, plansubscription.ChangeSubscriptionRequest{
				ID:            req.ID,
				WorkflowInput: req.WorkflowInput,
				PlanInput:     req.PlanInput,
			})
			if err != nil {
				return ChangeSubscriptionResponse{}, err
			}

			return ChangeSubscriptionResponse{
				Current: ConvertSubscriptionToAPISubscription(resp.Current),
				Next:    ConvertSubscriptionToAPISubscription(resp.Next.Subscription),
			}, nil
		},
		commonhttp.JSONResponseEncoderWithStatus[ChangeSubscriptionResponse](http.StatusOK),
		httptransport.AppendOptions(
			h.options,
			httptransport.WithOperationName("change-subscription"),
			httptransport.WithErrorEncoder(apierrors.GenericErrorEncoder()),
		)...,
	)
}
