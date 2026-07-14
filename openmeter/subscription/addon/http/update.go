package httpdriver

import (
	"context"
	"errors"
	"net/http"

	"github.com/openmeterio/openmeter/api"
	subscriptionhttp "github.com/openmeterio/openmeter/openmeter/productcatalog/subscription/http"
	subscriptionworkflow "github.com/openmeterio/openmeter/openmeter/subscription/workflow"
	"github.com/openmeterio/openmeter/pkg/framework/commonhttp"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport"
	"github.com/openmeterio/openmeter/pkg/models"
)

type (
	UpdateSubscriptionAddonParams = struct {
		SubscriptionID      string
		SubscriptionAddonID string
	}
	UpdateSubscriptionAddonRequest = struct {
		WorkflowInput  subscriptionworkflow.ChangeAddonQuantityWorkflowInput
		SubscriptionID models.NamespacedID
	}
	UpdateSubscriptionAddonResponse = api.SubscriptionAddon
	UpdateSubscriptionAddonHandler  = httptransport.HandlerWithArgs[UpdateSubscriptionAddonRequest, UpdateSubscriptionAddonResponse, UpdateSubscriptionAddonParams]
)

func (h *handler) UpdateSubscriptionAddon() UpdateSubscriptionAddonHandler {
	return httptransport.NewHandlerWithArgs(
		func(ctx context.Context, r *http.Request, params UpdateSubscriptionAddonParams) (UpdateSubscriptionAddonRequest, error) {
			ns, err := h.resolveNamespace(ctx)
			if err != nil {
				return UpdateSubscriptionAddonRequest{}, err
			}

			var body api.UpdateSubscriptionAddonJSONRequestBody
			if err := commonhttp.JSONRequestBodyDecoder(r, &body); err != nil {
				return UpdateSubscriptionAddonRequest{}, err
			}

			if body.Timing == nil {
				return UpdateSubscriptionAddonRequest{}, errors.New("timing is required")
			}
			if body.Quantity == nil {
				return UpdateSubscriptionAddonRequest{}, errors.New("quantity is required")
			}

			timing, err := subscriptionhttp.MapAPITimingToTiming(*body.Timing)
			if err != nil {
				return UpdateSubscriptionAddonRequest{}, err
			}

			return UpdateSubscriptionAddonRequest{
				WorkflowInput: subscriptionworkflow.ChangeAddonQuantityWorkflowInput{
					SubscriptionAddonID: models.NamespacedID{
						Namespace: ns,
						ID:        params.SubscriptionAddonID,
					},
					Quantity: *body.Quantity,
					Timing:   timing,
				},
				SubscriptionID: models.NamespacedID{
					Namespace: ns,
					ID:        params.SubscriptionID,
				},
			}, nil
		},
		func(ctx context.Context, request UpdateSubscriptionAddonRequest) (UpdateSubscriptionAddonResponse, error) {
			view, addon, err := h.SubscriptionWorkflowService.ChangeAddonQuantity(ctx, request.SubscriptionID, request.WorkflowInput)
			if err != nil {
				return UpdateSubscriptionAddonResponse{}, err
			}

			return MapSubscriptionAddonToResponse(view, addon)
		},
		commonhttp.JSONResponseEncoderWithStatus[UpdateSubscriptionAddonResponse](http.StatusOK),
		httptransport.AppendOptions(
			h.Options,
			httptransport.WithOperationName("updateSubscriptionAddon"),
		)...,
	)
}
