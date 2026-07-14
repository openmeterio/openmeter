package httpdriver

import (
	"context"
	"fmt"
	"net/http"

	"github.com/openmeterio/openmeter/api"
	subscriptionaddon "github.com/openmeterio/openmeter/openmeter/subscription/addon"
	"github.com/openmeterio/openmeter/pkg/framework/commonhttp"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/slicesx"
)

type (
	ListSubscriptionAddonsParams = struct {
		SubscriptionID string
	}
	ListSubscriptionAddonsRequest = struct {
		SubscriptionID models.NamespacedID
	}
	ListSubscriptionAddonsResponse = []api.SubscriptionAddon
	ListSubscriptionAddonsHandler  = httptransport.HandlerWithArgs[ListSubscriptionAddonsRequest, ListSubscriptionAddonsResponse, ListSubscriptionAddonsParams]
)

func (h *handler) ListSubscriptionAddons() ListSubscriptionAddonsHandler {
	return httptransport.NewHandlerWithArgs(
		func(ctx context.Context, r *http.Request, params ListSubscriptionAddonsParams) (ListSubscriptionAddonsRequest, error) {
			ns, err := h.resolveNamespace(ctx)
			if err != nil {
				return ListSubscriptionAddonsRequest{}, fmt.Errorf("failed to resolve namespace: %w", err)
			}

			return ListSubscriptionAddonsRequest{
				SubscriptionID: models.NamespacedID{
					Namespace: ns,
					ID:        params.SubscriptionID,
				},
			}, nil
		},
		func(ctx context.Context, req ListSubscriptionAddonsRequest) (ListSubscriptionAddonsResponse, error) {
			res, err := h.SubscriptionAddonService.List(ctx, req.SubscriptionID.Namespace, subscriptionaddon.ListSubscriptionAddonsInput{
				SubscriptionID: req.SubscriptionID.ID,
			})
			if err != nil {
				return nil, err
			}

			view, err := h.SubscriptionService.GetView(ctx, req.SubscriptionID)
			if err != nil {
				return nil, err
			}

			return slicesx.MapWithErr(res.Items, func(item subscriptionaddon.SubscriptionAddon) (api.SubscriptionAddon, error) {
				return MapSubscriptionAddonToResponse(view, item)
			})
		},
		commonhttp.JSONResponseEncoderWithStatus[ListSubscriptionAddonsResponse](http.StatusOK),
		httptransport.AppendOptions(
			h.Options,
			httptransport.WithOperationName("listSubscriptionAddons"),
		)...,
	)
}
