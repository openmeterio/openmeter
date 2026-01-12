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
	CreateStripeCheckoutSessionRequest  = api.BillingAppStripeCreateCheckoutSessionRequest
	CreateStripeCheckoutSessionResponse = api.BillingAppStripeCreateCheckoutSessionResult
	CreateStripeCheckoutSessionHandler  httptransport.Handler[CreateStripeCheckoutSessionRequest, CreateStripeCheckoutSessionResponse]
)

func (h *handler) CreateStripeCheckoutSession() CreateStripeCheckoutSessionHandler {
	return httptransport.NewHandler(
		func(ctx context.Context, r *http.Request) (CreateStripeCheckoutSessionRequest, error) {
			body := api.BillingAppStripeCreateCheckoutSessionRequest{}
			if err := request.ParseBody(r, &body); err != nil {
				return CreateStripeCheckoutSessionRequest{}, err
			}

			return body, nil
		},
		func(ctx context.Context, request CreateStripeCheckoutSessionRequest) (CreateStripeCheckoutSessionResponse, error) {
			return CreateStripeCheckoutSessionResponse{}, apierrors.NewNotImplementedError(ctx, nil)
		},
		commonhttp.JSONResponseEncoderWithStatus[CreateStripeCheckoutSessionResponse](http.StatusCreated),
		httptransport.AppendOptions(
			h.options,
			httptransport.WithOperationName("create-stripe-checkout-session"),
			httptransport.WithErrorEncoder(apierrors.GenericErrorEncoder()),
		)...,
	)
}
