package httpdriver

import (
	"context"
	"fmt"
	"net/http"

	apiv1models "github.com/openmeterio/openmeter/api/models"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	subscriptionaddon "github.com/openmeterio/openmeter/openmeter/subscription/addon"
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
	GetSubscriptionAddonResponse = apiv1models.SubscriptionAddon
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
			res, err := h.SubscriptionAddonService.Get(ctx, subscriptionaddon.GetSubscriptionAddonInput{
				NamespacedID: req.SubscriptionAddonID,
			})
			if err != nil {
				return GetSubscriptionAddonResponse{}, err
			}

			view, err := h.SubscriptionService.GetView(ctx, req.SubscriptionID)
			if err != nil {
				return GetSubscriptionAddonResponse{}, err
			}

			if res.Addon.AsProductCatalogAddon().HasUnitConfig() {
				return GetSubscriptionAddonResponse{}, productcatalog.ErrUnitConfigNotRepresentable
			}

			taxCodes, err := h.taxCodesForAddons(ctx, req.SubscriptionAddonID.Namespace, *res)
			if err != nil {
				return GetSubscriptionAddonResponse{}, err
			}

			return MapSubscriptionAddonToResponse(view, *res, taxCodes)
		},
		commonhttp.JSONResponseEncoderWithStatus[GetSubscriptionAddonResponse](http.StatusOK),
		httptransport.AppendOptions(
			h.Options,
			httptransport.WithOperationName("getSubscriptionAddon"),
		)...,
	)
}
