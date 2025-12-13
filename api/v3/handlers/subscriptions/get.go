package subscriptions

import (
	"context"
	"net/http"

	api "github.com/openmeterio/openmeter/api/v3"
	"github.com/openmeterio/openmeter/api/v3/apierrors"
	"github.com/openmeterio/openmeter/pkg/framework/commonhttp"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport"
	models "github.com/openmeterio/openmeter/pkg/models"
)

type (
	GetSubscriptionRequest  = models.NamespacedID
	GetSubscriptionResponse = api.BillingSubscription
	GetSubscriptionParams   = string
	GetSubscriptionHandler  httptransport.HandlerWithArgs[GetSubscriptionRequest, GetSubscriptionResponse, GetSubscriptionParams]
)

// GetSubscription returns a handler for getting a subscription.
func (h *handler) GetSubscription() GetSubscriptionHandler {
	return httptransport.NewHandlerWithArgs(
		func(ctx context.Context, r *http.Request, subscriptionID GetSubscriptionParams) (GetSubscriptionRequest, error) {
			ns, err := h.resolveNamespace(ctx)
			if err != nil {
				return GetSubscriptionRequest{}, err
			}

			return GetSubscriptionRequest{
				Namespace: ns,
				ID:        subscriptionID,
			}, nil
		},
		func(ctx context.Context, request GetSubscriptionRequest) (GetSubscriptionResponse, error) {
			// Get the subscription
			m, err := h.subscriptionService.Get(ctx, request)
			if err != nil {
				return GetSubscriptionResponse{}, err
			}

			return ConvertSubscriptionToAPISubscription(m), nil
		},
		commonhttp.JSONResponseEncoderWithStatus[GetSubscriptionResponse](http.StatusOK),
		httptransport.AppendOptions(
			h.options,
			httptransport.WithOperationName("get-subscription"),
			httptransport.WithErrorEncoder(apierrors.GenericErrorEncoder()),
		)...,
	)
}
