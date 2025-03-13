package httpdriver

import (
	"context"
	"net/http"

	"github.com/openmeterio/openmeter/pkg/framework/commonhttp"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport"
	"github.com/openmeterio/openmeter/pkg/models"
)

type (
	DeleteSubscriptionParams = struct {
		ID string
	}
	DeleteSubscriptionRequest = struct {
		ID models.NamespacedID
	}
	DeleteSubscriptionResponse = interface{}
	DeleteSubscriptionHandler  = httptransport.HandlerWithArgs[DeleteSubscriptionRequest, DeleteSubscriptionResponse, DeleteSubscriptionParams]
)

func (h *handler) DeleteSubscription() DeleteSubscriptionHandler {
	return httptransport.NewHandlerWithArgs(
		func(ctx context.Context, r *http.Request, params DeleteSubscriptionParams) (DeleteSubscriptionRequest, error) {
			ns, err := h.resolveNamespace(ctx)
			if err != nil {
				return DeleteSubscriptionRequest{}, err
			}

			return DeleteSubscriptionRequest{
				ID: models.NamespacedID{
					Namespace: ns,
					ID:        params.ID,
				},
			}, nil
		},
		func(ctx context.Context, request DeleteSubscriptionRequest) (DeleteSubscriptionResponse, error) {
			err := h.SubscriptionService.Delete(ctx, request.ID)
			if err != nil {
				return nil, err
			}

			return nil, nil
		},
		commonhttp.EmptyResponseEncoder[DeleteSubscriptionResponse](http.StatusNoContent),
		httptransport.AppendOptions(
			h.Options,
			httptransport.WithOperationName("deleteSubscription"),
			httptransport.WithErrorEncoder(errorEncoder()),
		)...,
	)
}
