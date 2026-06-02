package subscriptionaddons

import (
	"context"
	"net/http"

	apiv3 "github.com/openmeterio/openmeter/api/v3"
	"github.com/openmeterio/openmeter/api/v3/apierrors"
	"github.com/openmeterio/openmeter/api/v3/request"
	subscriptionworkflow "github.com/openmeterio/openmeter/openmeter/subscription/workflow"
	"github.com/openmeterio/openmeter/pkg/framework/commonhttp"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport"
	"github.com/openmeterio/openmeter/pkg/models"
)

type (
	UpdateSubscriptionAddonRequest = struct {
		SubscriptionID models.NamespacedID
		WorkflowInput  subscriptionworkflow.ChangeAddonQuantityWorkflowInput
	}
	UpdateSubscriptionAddonResponse = apiv3.SubscriptionAddon
	UpdateSubscriptionAddonParams   struct {
		SubscriptionID      string
		SubscriptionAddonID string
	}
	UpdateSubscriptionAddonHandler = httptransport.HandlerWithArgs[UpdateSubscriptionAddonRequest, UpdateSubscriptionAddonResponse, UpdateSubscriptionAddonParams]
)

func (h *handler) UpdateSubscriptionAddon() UpdateSubscriptionAddonHandler {
	return httptransport.NewHandlerWithArgs(
		func(ctx context.Context, r *http.Request, arg UpdateSubscriptionAddonParams) (UpdateSubscriptionAddonRequest, error) {
			body := apiv3.UpdateSubscriptionAddonRequest{}
			if err := request.ParseBody(r, &body); err != nil {
				return UpdateSubscriptionAddonRequest{}, err
			}

			ns, err := h.resolveNamespace(ctx)
			if err != nil {
				return UpdateSubscriptionAddonRequest{}, err
			}

			workflowInput, err := toUpdateSubscriptionAddon(models.NamespacedID{
				Namespace: ns,
				ID:        arg.SubscriptionAddonID,
			}, body)
			if err != nil {
				return UpdateSubscriptionAddonRequest{}, err
			}

			return UpdateSubscriptionAddonRequest{
				SubscriptionID: models.NamespacedID{
					Namespace: ns,
					ID:        arg.SubscriptionID,
				},
				WorkflowInput: workflowInput,
			}, nil
		},
		func(ctx context.Context, request UpdateSubscriptionAddonRequest) (UpdateSubscriptionAddonResponse, error) {
			_, changedAddon, err := h.subscriptionWorkflowService.ChangeAddonQuantity(ctx, request.SubscriptionID, request.WorkflowInput)
			if err != nil {
				return UpdateSubscriptionAddonResponse{}, err
			}
			// TODO add view
			return toAPISubscriptionAddon(changedAddon)
		},
		commonhttp.JSONResponseEncoderWithStatus[UpdateSubscriptionAddonResponse](http.StatusCreated),
		httptransport.AppendOptions(
			h.options,
			httptransport.WithOperationName("update-subscription-addon"),
			httptransport.WithErrorEncoder(apierrors.GenericErrorEncoder()),
		)...,
	)
}
