package httpdriver

import (
	"context"
	"net/http"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/api"
	"github.com/openmeterio/openmeter/openmeter/subscription"
	"github.com/openmeterio/openmeter/pkg/framework/commonhttp"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport"
	"github.com/openmeterio/openmeter/pkg/models"
)

type (
	CancelSubscriptionRequest = struct {
		Timing subscription.Timing
		ID     models.NamespacedID
	}
	CancelSubscriptionResponse = api.Subscription
	CancelSubscriptionParams   = struct {
		ID string
	}
	CancelSubscriptionHandler = httptransport.HandlerWithArgs[CancelSubscriptionRequest, CancelSubscriptionResponse, CancelSubscriptionParams]
)

func (h *handler) CancelSubscription() CancelSubscriptionHandler {
	return httptransport.NewHandlerWithArgs(
		func(ctx context.Context, r *http.Request, params CancelSubscriptionParams) (CancelSubscriptionRequest, error) {
			ns, err := h.resolveNamespace(ctx)
			if err != nil {
				return CancelSubscriptionRequest{}, err
			}

			var body api.CancelSubscriptionJSONRequestBody

			if err := commonhttp.JSONRequestBodyDecoder(r, &body); err != nil {
				return CancelSubscriptionRequest{}, err
			}

			var timing subscription.Timing

			if body.Timing == nil {
				timing.Enum = lo.ToPtr(subscription.TimingImmediate)
			} else {
				timing, err = MapAPITimingToTiming(*body.Timing)
				if err != nil {
					return CancelSubscriptionRequest{}, models.NewGenericValidationError(err)
				}
			}

			return CancelSubscriptionRequest{
				Timing: timing,
				ID:     models.NamespacedID{Namespace: ns, ID: params.ID},
			}, nil
		},
		func(ctx context.Context, req CancelSubscriptionRequest) (CancelSubscriptionResponse, error) {
			sub, err := h.SubscriptionService.Cancel(ctx, req.ID, req.Timing)
			if err != nil {
				return CancelSubscriptionResponse{}, err
			}

			return MapSubscriptionToAPI(sub), nil
		},
		commonhttp.JSONResponseEncoderWithStatus[CancelSubscriptionResponse](http.StatusOK),
		httptransport.AppendOptions(
			h.Options,
			httptransport.WithOperationName("cancelSubscription"),
			httptransport.WithErrorEncoder(errorEncoder()),
		)...,
	)
}

type (
	ContinueSubscriptionRequest = struct {
		ID models.NamespacedID
	}
	ContinueSubscriptionResponse = api.Subscription
	ContinueSubscriptionParams   = struct {
		ID string
	}
	ContinueSubscriptionHandler = httptransport.HandlerWithArgs[ContinueSubscriptionRequest, ContinueSubscriptionResponse, ContinueSubscriptionParams]
)

func (h *handler) ContinueSubscription() ContinueSubscriptionHandler {
	return httptransport.NewHandlerWithArgs(
		func(ctx context.Context, r *http.Request, params ContinueSubscriptionParams) (ContinueSubscriptionRequest, error) {
			ns, err := h.resolveNamespace(ctx)
			if err != nil {
				return ContinueSubscriptionRequest{}, err
			}

			return ContinueSubscriptionRequest{
				ID: models.NamespacedID{Namespace: ns, ID: params.ID},
			}, nil
		},
		func(ctx context.Context, req ContinueSubscriptionRequest) (ContinueSubscriptionResponse, error) {
			sub, err := h.SubscriptionService.Continue(ctx, req.ID)
			if err != nil {
				return ContinueSubscriptionResponse{}, err
			}

			return MapSubscriptionToAPI(sub), nil
		},
		commonhttp.JSONResponseEncoderWithStatus[ContinueSubscriptionResponse](http.StatusOK),
		httptransport.AppendOptions(
			h.Options,
			httptransport.WithOperationName("continueSubscription"),
			httptransport.WithErrorEncoder(errorEncoder()),
		)...,
	)
}
