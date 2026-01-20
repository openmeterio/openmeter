package customersbilling

import (
	"context"
	"net/http"

	"github.com/samber/lo"

	api "github.com/openmeterio/openmeter/api/v3"
	"github.com/openmeterio/openmeter/api/v3/apierrors"
	"github.com/openmeterio/openmeter/api/v3/request"
	appstripeentity "github.com/openmeterio/openmeter/openmeter/app/stripe/entity"
	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/pkg/framework/commonhttp"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport"
)

type (
	CreateCustomerStripeCheckoutSessionRequest  = appstripeentity.CreateCheckoutSessionInput
	CreateCustomerStripeCheckoutSessionResponse = api.BillingAppStripeCreateCheckoutSessionResult
	CreateCustomerStripeCheckoutSessionHandler  httptransport.HandlerWithArgs[CreateCustomerStripeCheckoutSessionRequest, CreateCustomerStripeCheckoutSessionResponse, string]
)

func (h *handler) CreateCustomerStripeCheckoutSession() CreateCustomerStripeCheckoutSessionHandler {
	return httptransport.NewHandlerWithArgs(
		func(ctx context.Context, r *http.Request, customerIdParam string) (CreateCustomerStripeCheckoutSessionRequest, error) {
			body := api.BillingCustomerStripeCreateCheckoutSessionRequest{}
			if err := request.ParseBody(r, &body); err != nil {
				return CreateCustomerStripeCheckoutSessionRequest{}, err
			}

			namespace, err := h.resolveNamespace(ctx)
			if err != nil {
				return CreateCustomerStripeCheckoutSessionRequest{}, err
			}

			customerId := lo.ToPtr(customer.CustomerID{
				Namespace: namespace,
				ID:        customerIdParam,
			})

			appId, err := h.billingService.ResolveAppIDFromBillingProfile(ctx, namespace, customerId)
			if err != nil {
				return CreateCustomerStripeCheckoutSessionRequest{}, err
			}

			options, err := ConvertToCreateStripeCheckoutSessionRequestOptions(body.StripeOptions)
			if err != nil {
				return CreateCustomerStripeCheckoutSessionRequest{}, err
			}

			// Create request
			req := CreateCustomerStripeCheckoutSessionRequest{
				Namespace:  namespace,
				AppID:      appId,
				CustomerID: customerId,
				Options:    options,
			}

			return req, nil
		},
		func(ctx context.Context, request CreateCustomerStripeCheckoutSessionRequest) (CreateCustomerStripeCheckoutSessionResponse, error) {
			out, err := h.stripeService.CreateCheckoutSession(ctx, request)
			if err != nil {
				return CreateCustomerStripeCheckoutSessionResponse{}, err
			}

			response := ConvertCreateCheckoutSessionOutputToBillingAppStripeCreateCheckoutSessionResult(out)

			return response, nil
		},
		commonhttp.JSONResponseEncoderWithStatus[CreateCustomerStripeCheckoutSessionResponse](http.StatusCreated),
		httptransport.AppendOptions(
			h.options,
			httptransport.WithOperationName("create-customer-stripe-checkout-session"),
			httptransport.WithErrorEncoder(apierrors.GenericErrorEncoder()),
		)...,
	)
}
