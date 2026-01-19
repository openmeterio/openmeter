package customersbilling

import (
	"context"
	"fmt"
	"net/http"

	"github.com/samber/lo"

	api "github.com/openmeterio/openmeter/api/v3"
	"github.com/openmeterio/openmeter/api/v3/apierrors"
	"github.com/openmeterio/openmeter/api/v3/request"
	appstripeentity "github.com/openmeterio/openmeter/openmeter/app/stripe/entity"
	appstripehttpdriver "github.com/openmeterio/openmeter/openmeter/app/stripe/httpdriver"
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
				return CreateCustomerStripeCheckoutSessionRequest{}, fmt.Errorf("failed to resolve namespace: %w", err)
			}

			if customerIdParam == "" {
				return CreateCustomerStripeCheckoutSessionRequest{}, fmt.Errorf("customer is required")
			}

			customerId := lo.ToPtr(customer.CustomerID{
				Namespace: namespace,
				ID:        customerIdParam,
			})

			// Resolve app ID from request or from billing profile
			appId, err := appstripehttpdriver.ResolveAppIDFromBillingProfile(ctx, namespace, customerId, h.billingService)
			if err != nil {
				return CreateCustomerStripeCheckoutSessionRequest{}, fmt.Errorf("failed to resolve app id from billing profile: %w", err)
			}

			options, err := ConvertToCreateStripeCheckoutSessionRequestOptions(body.StripeOptions)
			if err != nil {
				return CreateCustomerStripeCheckoutSessionRequest{}, fmt.Errorf("failed to convert customer options to CreateStripeCheckoutSessionRequestOptions: %w", err)
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
				return CreateCustomerStripeCheckoutSessionResponse{}, fmt.Errorf("failed to create app stripe checkout session: %w", err)
			}

			response := CreateCustomerStripeCheckoutSessionResponse{
				CancelUrl:        out.CancelURL,
				CustomerId:       out.CustomerID.ID,
				Mode:             api.BillingAppStripeCheckoutSessionMode(out.Mode),
				ReturnUrl:        out.ReturnURL,
				SessionId:        out.SessionID,
				SetupIntentId:    out.SetupIntentID,
				StripeCustomerId: out.StripeCustomerID,
				SuccessUrl:       out.SuccessURL,
				Url:              out.URL,

				// Add new fields from the CreateCheckoutSessionOutput
				ClientSecret:      out.ClientSecret,
				ClientReferenceId: out.ClientReferenceID,
				CustomerEmail:     out.CustomerEmail,
				Currency:          (*api.CurrencyCode)(out.Currency),
				CreatedAt:         out.CreatedAt,
				Metadata:          out.Metadata,
				Status:            (*string)(out.Status),
				ExpiresAt:         out.ExpiresAt,
			}

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
