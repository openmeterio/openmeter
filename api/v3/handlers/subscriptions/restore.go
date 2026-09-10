package subscriptions

import (
	"context"
	"net/http"

	api "github.com/openmeterio/openmeter/api/v3"
	"github.com/openmeterio/openmeter/pkg/framework/commonhttp"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport"
	models "github.com/openmeterio/openmeter/pkg/models"
)

type (
	RestoreSubscriptionRequest  = models.NamespacedID
	RestoreSubscriptionResponse = api.BillingSubscription
	RestoreSubscriptionParams   = string
	RestoreSubscriptionHandler  httptransport.HandlerWithArgs[RestoreSubscriptionRequest, RestoreSubscriptionResponse, RestoreSubscriptionParams]
)

// RestoreSubscription deletes any later-scheduled successor subscriptions and
// continues the current subscription indefinitely, undoing a future-dated
// change. Restore is not available when multi-subscription is enabled, in which
// case the workflow service returns a GenericForbiddenError (mapped to 403).
func (h *handler) RestoreSubscription() RestoreSubscriptionHandler {
	return httptransport.NewHandlerWithArgs(
		func(ctx context.Context, r *http.Request, subscriptionID RestoreSubscriptionParams) (RestoreSubscriptionRequest, error) {
			ns, err := h.resolveNamespace(ctx)
			if err != nil {
				return RestoreSubscriptionRequest{}, err
			}

			return RestoreSubscriptionRequest{
				Namespace: ns,
				ID:        subscriptionID,
			}, nil
		},
		func(ctx context.Context, req RestoreSubscriptionRequest) (RestoreSubscriptionResponse, error) {
			sub, err := h.subscriptionWorkflowService.Restore(ctx, req)
			if err != nil {
				return RestoreSubscriptionResponse{}, err
			}

			view, err := h.subscriptionService.GetView(ctx, sub.NamespacedID)
			if err != nil {
				return RestoreSubscriptionResponse{}, err
			}

			return ToAPIBillingSubscription(view)
		},
		commonhttp.JSONResponseEncoderWithStatus[RestoreSubscriptionResponse](http.StatusOK),
		httptransport.AppendOptions(
			h.options,
			httptransport.WithOperationName("restore-subscription"),
			httptransport.WithErrorEncoder(subscriptionGenericErrorEncoder()),
		)...,
	)
}
