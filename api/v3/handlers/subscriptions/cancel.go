package subscriptions

import (
	"context"
	"net/http"

	"github.com/samber/lo"

	api "github.com/openmeterio/openmeter/api/v3"
	"github.com/openmeterio/openmeter/api/v3/apierrors"
	"github.com/openmeterio/openmeter/api/v3/request"
	"github.com/openmeterio/openmeter/openmeter/subscription"
	"github.com/openmeterio/openmeter/pkg/framework/commonhttp"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport"
	models "github.com/openmeterio/openmeter/pkg/models"
)

type (
	CancelSubscriptionRequest struct {
		ID     models.NamespacedID
		Timing subscription.Timing
	}
	CancelSubscriptionResponse = api.BillingSubscription
	CancelSubscriptionParams   = string
	CancelSubscriptionHandler  httptransport.HandlerWithArgs[CancelSubscriptionRequest, CancelSubscriptionResponse, CancelSubscriptionParams]
)

func (h *handler) CancelSubscription() CancelSubscriptionHandler {
	return httptransport.NewHandlerWithArgs(
		func(ctx context.Context, r *http.Request, subscriptionID CancelSubscriptionParams) (CancelSubscriptionRequest, error) {
			// Parse body
			body := api.BillingSubscriptionCancel{}
			if err := request.ParseBody(r, &body); err != nil {
				return CancelSubscriptionRequest{}, err
			}

			// Resolve namespace
			ns, err := h.resolveNamespace(ctx)
			if err != nil {
				return CancelSubscriptionRequest{}, err
			}

			// Timing (defaults to immediate)
			timing := subscription.Timing{}
			if body.Timing == nil {
				timing.Enum = lo.ToPtr(subscription.TimingImmediate)
			} else {
				timing, err = ConvertBillingSubscriptionEditTimingToSubscriptionTiming(*body.Timing)
				if err != nil {
					return CancelSubscriptionRequest{}, err
				}
			}

			return CancelSubscriptionRequest{
				ID: models.NamespacedID{
					Namespace: ns,
					ID:        subscriptionID,
				},
				Timing: timing,
			}, nil
		},
		func(ctx context.Context, req CancelSubscriptionRequest) (CancelSubscriptionResponse, error) {
			sub, err := h.subscriptionService.Cancel(ctx, req.ID, req.Timing)
			if err != nil {
				return CancelSubscriptionResponse{}, err
			}

			return ConvertSubscriptionToAPISubscription(sub), nil
		},
		commonhttp.JSONResponseEncoderWithStatus[CancelSubscriptionResponse](http.StatusOK),
		httptransport.AppendOptions(
			h.options,
			httptransport.WithOperationName("cancel-subscription"),
			httptransport.WithErrorEncoder(apierrors.GenericErrorEncoder()),
		)...,
	)
}
