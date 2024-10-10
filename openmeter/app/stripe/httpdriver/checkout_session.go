package httpdriver

import (
	"context"
	"fmt"
	"net/http"

	"github.com/samber/lo"
	"github.com/stripe/stripe-go/v80"

	"github.com/openmeterio/openmeter/api"
	appentitybase "github.com/openmeterio/openmeter/openmeter/app/entity/base"
	stripeclient "github.com/openmeterio/openmeter/openmeter/app/stripe/client"
	appstripeentity "github.com/openmeterio/openmeter/openmeter/app/stripe/entity"
	customerentity "github.com/openmeterio/openmeter/openmeter/customer/entity"
	"github.com/openmeterio/openmeter/pkg/framework/commonhttp"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport"
)

type (
	CreateAppStripeCheckoutSessionRequest  = appstripeentity.CreateCheckoutSessionInput
	CreateAppStripeCheckoutSessionResponse = api.CreateStripeCheckoutSessionResponse
	CreateAppStripeCheckoutSessionHandler  httptransport.HandlerWithArgs[CreateAppStripeCheckoutSessionRequest, CreateAppStripeCheckoutSessionResponse, string]
)

// CreateAppStripeCheckoutSession returns a handler for creating a checkout session.
func (h *handler) CreateAppStripeCheckoutSession() CreateAppStripeCheckoutSessionHandler {
	return httptransport.NewHandlerWithArgs(
		func(ctx context.Context, r *http.Request, appID string) (CreateAppStripeCheckoutSessionRequest, error) {
			body := api.CreateStripeCheckoutSessionRequest{}
			if err := commonhttp.JSONRequestBodyDecoder(r, &body); err != nil {
				return CreateAppStripeCheckoutSessionRequest{}, fmt.Errorf("field to decode create app stripe checkout session request: %w", err)
			}

			namespace, err := h.resolveNamespace(ctx)
			if err != nil {
				return CreateAppStripeCheckoutSessionRequest{}, fmt.Errorf("failed to resolve namespace: %w", err)
			}

			req := CreateAppStripeCheckoutSessionRequest{
				AppID:            appentitybase.AppID{Namespace: namespace, ID: appID},
				CustomerID:       customerentity.CustomerID{Namespace: namespace, ID: body.CustomerId},
				StripeCustomerID: body.StripeCustomerId,
				Options: stripeclient.StripeCheckoutSessionOptions{
					Currency:          body.Options.Currency,
					CancelURL:         body.Options.CancelURL,
					ClientReferenceID: body.Options.ClientReferenceID,
					ReturnURL:         body.Options.ReturnURL,
					SuccessURL:        body.Options.SuccessURL,
				},
			}

			if body.Options.UiMode != nil {
				req.Options.UIMode = lo.ToPtr(stripe.CheckoutSessionUIMode(*body.Options.UiMode))
			}

			if body.Options.PaymentMethodTypes != nil {
				req.Options.PaymentMethodTypes = lo.ToPtr(
					lo.Map(
						*body.Options.PaymentMethodTypes,
						func(paymentMethodType string, _ int) *string {
							return &paymentMethodType
						},
					),
				)
			}

			if body.Options.Metadata != nil {
				req.Options.Metadata = *body.Options.Metadata
			}

			if body.Options.CustomText != nil {
				req.Options.CustomText = &stripe.CheckoutSessionCustomTextParams{}

				// AfterSubmit
				if body.Options.CustomText.AfterSubmit != nil {
					req.Options.CustomText.AfterSubmit = &stripe.CheckoutSessionCustomTextAfterSubmitParams{}
				}

				if body.Options.CustomText.AfterSubmit.Message != nil {
					req.Options.CustomText.AfterSubmit.Message = body.Options.CustomText.AfterSubmit.Message
				}

				// ShippingAddress
				if body.Options.CustomText.ShippingAddress != nil {
					req.Options.CustomText.ShippingAddress = &stripe.CheckoutSessionCustomTextShippingAddressParams{}
				}

				if body.Options.CustomText.ShippingAddress.Message != nil {
					req.Options.CustomText.ShippingAddress.Message = body.Options.CustomText.ShippingAddress.Message
				}

				// BeforeSubmit
				if body.Options.CustomText.Submit != nil {
					req.Options.CustomText.Submit = &stripe.CheckoutSessionCustomTextSubmitParams{}
				}

				if body.Options.CustomText.Submit.Message != nil {
					req.Options.CustomText.Submit.Message = body.Options.CustomText.Submit.Message
				}

				// TermsOfAcceptance
				if body.Options.CustomText.TermsOfServiceAcceptance != nil {
					req.Options.CustomText.TermsOfServiceAcceptance = &stripe.CheckoutSessionCustomTextTermsOfServiceAcceptanceParams{}
				}

				if body.Options.CustomText.TermsOfServiceAcceptance.Message != nil {
					req.Options.CustomText.TermsOfServiceAcceptance.Message = body.Options.CustomText.TermsOfServiceAcceptance.Message
				}
			}

			return req, nil
		},
		func(ctx context.Context, request CreateAppStripeCheckoutSessionRequest) (CreateAppStripeCheckoutSessionResponse, error) {
			out, err := h.service.CreateCheckoutSession(ctx, request)
			if err != nil {
				return CreateAppStripeCheckoutSessionResponse{}, fmt.Errorf("failed to create app stripe checkout session: %w", err)
			}

			return CreateAppStripeCheckoutSessionResponse{
				CancelURL:        out.CancelURL,
				CustomerId:       out.CustomerID.ID,
				Mode:             api.StripeCheckoutSessionMode(out.Mode),
				ReturnURL:        out.ReturnURL,
				SessionId:        out.SessionID,
				SetupIntentId:    out.SetupIntentID,
				StripeCustomerId: out.StripeCustomerID,
				SuccessURL:       out.SuccessURL,
				Url:              out.URL,
			}, nil
		},
		commonhttp.JSONResponseEncoderWithStatus[CreateAppStripeCheckoutSessionResponse](http.StatusOK),
		httptransport.AppendOptions(
			h.options,
			httptransport.WithOperationName("createAppStripeCheckoutSession"),
			httptransport.WithErrorEncoder(errorEncoder()),
		)...,
	)
}
