package subscriptionaddons

import (
	"context"
	"net/http"

	apiv3 "github.com/openmeterio/openmeter/api/v3"
	subscriptionaddon "github.com/openmeterio/openmeter/openmeter/subscription/addon"
	"github.com/openmeterio/openmeter/pkg/framework/commonhttp"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport"
	"github.com/openmeterio/openmeter/pkg/models"
)

type (
	GetSubscriptionAddonRequest = subscriptionaddon.GetSubscriptionAddonInput
	GetSubscriptionAddonParams  struct {
		SubscriptionID      string
		SubscriptionAddonID string
	}
	GetSubscriptionAddonResponse = apiv3.SubscriptionAddon
	GetSubscriptionAddonHandler  httptransport.HandlerWithArgs[GetSubscriptionAddonRequest, GetSubscriptionAddonResponse, GetSubscriptionAddonParams]
)

func (h *handler) GetSubscriptionAddons() GetSubscriptionAddonHandler {
	return httptransport.NewHandlerWithArgs(
		func(ctx context.Context, r *http.Request, params GetSubscriptionAddonParams) (GetSubscriptionAddonRequest, error) {
			ns, err := h.resolveNamespace(ctx)
			if err != nil {
				return GetSubscriptionAddonRequest{}, err
			}

			return GetSubscriptionAddonRequest{
				NamespacedID: models.NamespacedID{
					Namespace: ns,
					ID:        params.SubscriptionAddonID,
				},
				SubscriptionID: params.SubscriptionID,
			}, nil
		},
		func(ctx context.Context, request GetSubscriptionAddonRequest) (GetSubscriptionAddonResponse, error) {
			a, err := h.addonService.Get(ctx, request)
			if err != nil {
				return GetSubscriptionAddonResponse{}, err
			}

			return toAPISubscriptionAddon(*a)
		},
		commonhttp.JSONResponseEncoderWithStatus[GetSubscriptionAddonResponse](http.StatusOK),
		httptransport.AppendOptions(
			h.options,
			httptransport.WithOperationName("get-subscription-addon"),
		)...,
	)
}
