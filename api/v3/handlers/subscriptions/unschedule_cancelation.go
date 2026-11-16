package subscriptions

import (
	"context"
	"net/http"

	"github.com/openmeterio/openmeter/api/v3/apierrors"
	apiv3models "github.com/openmeterio/openmeter/api/v3/models"
	"github.com/openmeterio/openmeter/pkg/framework/commonhttp"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport"
	models "github.com/openmeterio/openmeter/pkg/models"
)

type (
	UnscheduleCancelationRequest  = models.NamespacedID
	UnscheduleCancelationResponse = apiv3models.BillingSubscription
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

			view, err := h.subscriptionService.GetView(ctx, sub.NamespacedID)
			if err != nil {
				return UnscheduleCancelationResponse{}, err
			}

			return ToAPIBillingSubscription(view)
		},
		commonhttp.JSONResponseEncoderWithStatus[UnscheduleCancelationResponse](http.StatusOK),
		httptransport.AppendOptions(
			h.options,
			httptransport.WithOperationName("unschedule-cancelation"),
			httptransport.WithErrorEncoder(apierrors.GenericErrorEncoder()),
		)...,
	)
}
