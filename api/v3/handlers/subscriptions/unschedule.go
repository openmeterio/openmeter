package subscriptions

import (
	"context"
	"net/http"

	"github.com/openmeterio/openmeter/pkg/framework/commonhttp"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport"
	models "github.com/openmeterio/openmeter/pkg/models"
)

type (
	UnscheduleSubscriptionRequest  = models.NamespacedID
	UnscheduleSubscriptionResponse = any
	UnscheduleSubscriptionParams   = string
	UnscheduleSubscriptionHandler  httptransport.HandlerWithArgs[UnscheduleSubscriptionRequest, UnscheduleSubscriptionResponse, UnscheduleSubscriptionParams]
)

// UnscheduleSubscription deletes a scheduled (not-yet-active) subscription. The
// domain Delete is guarded by the subscription state machine to the Scheduled
// state only, so attempting to unschedule an active or already-started
// subscription surfaces a GenericForbiddenError (mapped to 403) rather than
// being treated as a cancel.
func (h *handler) UnscheduleSubscription() UnscheduleSubscriptionHandler {
	return httptransport.NewHandlerWithArgs(
		func(ctx context.Context, r *http.Request, subscriptionID UnscheduleSubscriptionParams) (UnscheduleSubscriptionRequest, error) {
			ns, err := h.resolveNamespace(ctx)
			if err != nil {
				return UnscheduleSubscriptionRequest{}, err
			}

			return UnscheduleSubscriptionRequest{
				Namespace: ns,
				ID:        subscriptionID,
			}, nil
		},
		func(ctx context.Context, req UnscheduleSubscriptionRequest) (UnscheduleSubscriptionResponse, error) {
			if err := h.subscriptionService.Delete(ctx, req); err != nil {
				return nil, err
			}

			return nil, nil
		},
		commonhttp.EmptyResponseEncoder[UnscheduleSubscriptionResponse](http.StatusNoContent),
		httptransport.AppendOptions(
			h.options,
			httptransport.WithOperationName("unschedule-subscription"),
			httptransport.WithErrorEncoder(subscriptionGenericErrorEncoder()),
		)...,
	)
}
