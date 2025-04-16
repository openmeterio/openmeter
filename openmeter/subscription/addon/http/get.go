package httpdriver

import (
	"context"
	"fmt"
	"net/http"

	"github.com/openmeterio/openmeter/api"
	"github.com/openmeterio/openmeter/pkg/framework/commonhttp"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport"
	"github.com/openmeterio/openmeter/pkg/models"
)

type (
	GetSubscriptionAddonParams = struct {
		SubscriptionID      string
		SubscriptionAddonID string
	}
	GetSubscriptionAddonRequest = struct {
		SubscriptionID      models.NamespacedID
		SubscriptionAddonID models.NamespacedID
	}
	GetSubscriptionAddonResponse = api.SubscriptionAddon
	GetSubscriptionAddonHandler  = httptransport.HandlerWithArgs[GetSubscriptionAddonRequest, GetSubscriptionAddonResponse, GetSubscriptionAddonParams]
)

func (h *handler) GetSubscriptionAddon() GetSubscriptionAddonHandler {
	return httptransport.NewHandlerWithArgs(
		func(ctx context.Context, r *http.Request, params GetSubscriptionAddonParams) (GetSubscriptionAddonRequest, error) {
			ns, err := h.resolveNamespace(ctx)
			if err != nil {
				return GetSubscriptionAddonRequest{}, fmt.Errorf("failed to resolve namespace: %w", err)
			}

			return GetSubscriptionAddonRequest{
				SubscriptionID:      models.NamespacedID{Namespace: ns, ID: params.SubscriptionID},
				SubscriptionAddonID: models.NamespacedID{Namespace: ns, ID: params.SubscriptionAddonID},
			}, nil
		},
		func(ctx context.Context, req GetSubscriptionAddonRequest) (GetSubscriptionAddonResponse, error) {
			res, err := h.SubscriptionAddonService.Get(ctx, req.SubscriptionAddonID)
			if err != nil {
				return GetSubscriptionAddonResponse{}, err
			}

			view, err := h.SubscriptionService.GetView(ctx, req.SubscriptionID)
			if err != nil {
				return GetSubscriptionAddonResponse{}, err
			}

			return MapSubscriptionAddonToResponse(view, *res)
		},
		commonhttp.JSONResponseEncoderWithStatus[GetSubscriptionAddonResponse](http.StatusOK),
		httptransport.AppendOptions(
			h.Options,
			httptransport.WithOperationName("getSubscriptionAddon"),
		)...,
	)
}
