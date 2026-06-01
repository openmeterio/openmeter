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
	CreateSubscriptionAddonRequest = struct {
		SubscriptionID models.NamespacedID
		AddonInput     subscriptionworkflow.AddAddonWorkflowInput
	}
	CreateSubscriptionAddonResponse = apiv3.SubscriptionAddon
	CreateSubscriptionAddonParams   = string
	CreateSubscriptionAddonHandler  = httptransport.HandlerWithArgs[CreateSubscriptionAddonRequest, CreateSubscriptionAddonResponse, CreateSubscriptionAddonParams]
)

func (h *handler) CreateSubscriptionAddon() CreateSubscriptionAddonHandler {
	return httptransport.NewHandlerWithArgs(
		func(ctx context.Context, r *http.Request, subscriptionID CreateSubscriptionAddonParams) (CreateSubscriptionAddonRequest, error) {
			body := apiv3.CreateSubscriptionAddonRequest{}
			if err := request.ParseBody(r, &body); err != nil {
				return CreateSubscriptionAddonRequest{}, err
			}

			ns, err := h.resolveNamespace(ctx)
			if err != nil {
				return CreateSubscriptionAddonRequest{}, err
			}

			addonInput, err := mapCreateSubscriptionAddonRequestToInput(body)
			if err != nil {
				return CreateSubscriptionAddonRequest{}, err
			}

			return CreateSubscriptionAddonRequest{
				SubscriptionID: models.NamespacedID{
					Namespace: ns,
					ID:        subscriptionID,
				},
				AddonInput: addonInput,
			}, nil
		},
		func(ctx context.Context, request CreateSubscriptionAddonRequest) (CreateSubscriptionAddonResponse, error) {
			view, added, err := h.SubscriptionWorkflowService.AddAddon(ctx, request.SubscriptionID, request.AddonInput)
			if err != nil {
				return CreateSubscriptionAddonResponse{}, err
			}

			return toAPISubscriptionAddon(view, added)
		},
		commonhttp.JSONResponseEncoderWithStatus[CreateSubscriptionAddonResponse](http.StatusCreated),
		httptransport.AppendOptions(
			h.options,
			httptransport.WithOperationName("create-subscription-addon"),
			httptransport.WithErrorEncoder(apierrors.GenericErrorEncoder()),
		)...,
	)
}
