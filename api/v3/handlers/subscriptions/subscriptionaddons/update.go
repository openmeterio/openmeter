package subscriptionaddons

import (
	"context"
	"net/http"

	apiv3 "github.com/openmeterio/openmeter/api/v3"
	"github.com/openmeterio/openmeter/api/v3/apierrors"
	"github.com/openmeterio/openmeter/api/v3/handlers/subscriptions"
	"github.com/openmeterio/openmeter/api/v3/request"
	subscriptionworkflow "github.com/openmeterio/openmeter/openmeter/subscription/workflow"
	"github.com/openmeterio/openmeter/pkg/framework/commonhttp"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport"
	"github.com/openmeterio/openmeter/pkg/models"
)

type (
	UpdateSubscriptionAddonRequest struct {
		WorkflowInput  subscriptionworkflow.ChangeAddonQuantityWorkflowInput
		SubscriptionID models.NamespacedID
	}
	UpdateSubscriptionAddonParams struct {
		SubscriptionID      string
		SubscriptionAddonID string
	}
	UpdateSubscriptionAddonResponse = apiv3.SubscriptionAddon
	UpdateSubscriptionAddonHandler  = httptransport.HandlerWithArgs[UpdateSubscriptionAddonRequest, UpdateSubscriptionAddonResponse, UpdateSubscriptionAddonParams]
)

// UpdateSubscriptionAddon changes the quantity of an existing subscription add-on.
// The v1 handler rejects add-ons with a unit-config price here; that restriction
// is intentionally omitted, since the resulting state is representable in v3.
func (h *handler) UpdateSubscriptionAddon() UpdateSubscriptionAddonHandler {
	return httptransport.NewHandlerWithArgs(
		func(ctx context.Context, r *http.Request, params UpdateSubscriptionAddonParams) (UpdateSubscriptionAddonRequest, error) {
			body := apiv3.UpdateSubscriptionAddonJSONRequestBody{}
			if err := request.ParseBody(r, &body); err != nil {
				return UpdateSubscriptionAddonRequest{}, err
			}

			ns, err := h.resolveNamespace(ctx)
			if err != nil {
				return UpdateSubscriptionAddonRequest{}, err
			}

			timing, err := subscriptions.FromAPIBillingSubscriptionEditTiming(body.Timing)
			if err != nil {
				return UpdateSubscriptionAddonRequest{}, err
			}

			return UpdateSubscriptionAddonRequest{
				WorkflowInput: subscriptionworkflow.ChangeAddonQuantityWorkflowInput{
					SubscriptionAddonID: models.NamespacedID{
						Namespace: ns,
						ID:        params.SubscriptionAddonID,
					},
					Quantity: body.Quantity,
					Timing:   timing,
				},
				SubscriptionID: models.NamespacedID{
					Namespace: ns,
					ID:        params.SubscriptionID,
				},
			}, nil
		},
		func(ctx context.Context, req UpdateSubscriptionAddonRequest) (UpdateSubscriptionAddonResponse, error) {
			view, addon, err := h.subscriptionWorkflowService.ChangeAddonQuantity(ctx, req.SubscriptionID, req.WorkflowInput)
			if err != nil {
				return UpdateSubscriptionAddonResponse{}, err
			}

			return toAPISubscriptionAddon(view, addon)
		},
		commonhttp.JSONResponseEncoderWithStatus[UpdateSubscriptionAddonResponse](http.StatusOK),
		httptransport.AppendOptions(
			h.options,
			httptransport.WithOperationName("update-subscription-addon"),
			httptransport.WithErrorEncoder(apierrors.GenericErrorEncoder()),
		)...,
	)
}
