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
	UnscheduleCancelationRequest  = models.NamespacedID
	UnscheduleCancelationResponse = api.BillingSubscription
	UnscheduleCancelationParams   = string
	UnscheduleCancelationHandler  httptransport.HandlerWithArgs[UnscheduleCancelationRequest, UnscheduleCancelationResponse, UnscheduleCancelationParams]
)

func (h *handler) UnscheduleCancelation() UnscheduleCancelationHandler {
	return httptransport.NewHandlerWithArgs(
		func(ctx context.Context, r *http.Request, subscriptionID UnscheduleCancelationParams) (UnscheduleCancelationRequest, error) {
			ns, err := h.resolveNamespace(ctx)
			if err != nil {
				return UnscheduleCancelationRequest{}, err
			}

			return UnscheduleCancelationRequest{
				Namespace: ns,
				ID:        subscriptionID,
			}, nil
		},
		func(ctx context.Context, req UnscheduleCancelationRequest) (UnscheduleCancelationResponse, error) {
			sub, err := h.subscriptionService.Continue(ctx, req)
			if err != nil {
				return UnscheduleCancelationResponse{}, err
			}

			return ConvertSubscriptionToAPISubscription(sub), nil
		},
		commonhttp.JSONResponseEncoderWithStatus[UnscheduleCancelationResponse](http.StatusOK),
		httptransport.AppendOptions(
			h.options,
			httptransport.WithOperationName("unschedule-cancelation"),
			httptransport.WithErrorEncoder(apierrors.GenericErrorEncoder()),
		)...,
	)
}
