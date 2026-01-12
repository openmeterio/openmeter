package apps

import (
	"context"
	"net/http"

	api "github.com/openmeterio/openmeter/api/v3"
	"github.com/openmeterio/openmeter/api/v3/apierrors"
	"github.com/openmeterio/openmeter/api/v3/request"
	"github.com/openmeterio/openmeter/pkg/framework/commonhttp"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport"
)

type (
	HandleStripeWebhookRequest struct {
		AppId api.ULID
		Body  api.BillingAppStripeWebhookEvent
	}
	HandleStripeWebhookResponse = *struct{}
	HandleStripeWebhookParams   = api.ULID
	HandleStripeWebhookHandler  httptransport.HandlerWithArgs[HandleStripeWebhookRequest, HandleStripeWebhookResponse, HandleStripeWebhookParams]
)

func (h *handler) HandleStripeWebhook() HandleStripeWebhookHandler {
	return httptransport.NewHandlerWithArgs(
		func(ctx context.Context, r *http.Request, appId HandleStripeWebhookParams) (HandleStripeWebhookRequest, error) {
			body := api.BillingAppStripeWebhookEvent{}
			if err := request.ParseBody(r, &body); err != nil {
				return HandleStripeWebhookRequest{}, err
			}

			return HandleStripeWebhookRequest{
				AppId: appId,
				Body:  body,
			}, nil
		},
		func(ctx context.Context, request HandleStripeWebhookRequest) (HandleStripeWebhookResponse, error) {
			return nil, apierrors.NewNotImplementedError(ctx, nil)
		},
		commonhttp.EmptyResponseEncoder[HandleStripeWebhookResponse](http.StatusNoContent),
		httptransport.AppendOptions(
			h.options,
			httptransport.WithOperationName("handle-stripe-webhook"),
			httptransport.WithErrorEncoder(apierrors.GenericErrorEncoder()),
		)...,
	)
}
