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
	GetSubscriptionRequest = struct {
		ID    models.NamespacedID
		Query api.GetSubscriptionParams
	}
	GetSubscriptionResponse = api.SubscriptionExpanded
	GetSubscriptionParams   = struct {
		Query api.GetSubscriptionParams
		ID    string
	}
	GetSubscriptionHandler = httptransport.HandlerWithArgs[GetSubscriptionRequest, GetSubscriptionResponse, GetSubscriptionParams]
)

func (h *handler) GetSubscription() GetSubscriptionHandler {
	return httptransport.NewHandlerWithArgs(
		func(ctx context.Context, r *http.Request, params GetSubscriptionParams) (GetSubscriptionRequest, error) {
			ns, err := h.resolveNamespace(ctx)
			if err != nil {
				return GetSubscriptionRequest{}, fmt.Errorf("failed to resolve namespace: %w", err)
			}

			return GetSubscriptionRequest{
				ID: models.NamespacedID{
					Namespace: ns,
					ID:        params.ID,
				},
				Query: params.Query,
			}, nil
		},
		func(ctx context.Context, req GetSubscriptionRequest) (GetSubscriptionResponse, error) {
			var def GetSubscriptionResponse

			if req.Query.At != nil {
				return def, commonhttp.NewHTTPError(http.StatusNotImplemented, fmt.Errorf("historical queries are not supported"))
			}

			view, err := h.SubscriptionService.GetView(ctx, req.ID)
			if err != nil {
				return def, err
			}

			return MapSubscriptionViewToAPI(view.WithoutItemHistory())
		},
		commonhttp.JSONResponseEncoderWithStatus[GetSubscriptionResponse](http.StatusOK),
		httptransport.AppendOptions(
			h.Options,
			httptransport.WithOperationName("getSubscription"),
			httptransport.WithErrorEncoder(errorEncoder()),
		)...,
	)
}
