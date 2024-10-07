package httpdriver

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/stripe/stripe-go/v80"
	"github.com/stripe/stripe-go/v80/webhook"

	"github.com/openmeterio/openmeter/api"
	appentitybase "github.com/openmeterio/openmeter/openmeter/app/entity/base"
	"github.com/openmeterio/openmeter/openmeter/appstripe"
	stripeclient "github.com/openmeterio/openmeter/openmeter/appstripe/client"
	appstripeentity "github.com/openmeterio/openmeter/openmeter/appstripe/entity"
	"github.com/openmeterio/openmeter/pkg/framework/commonhttp"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport"
)

type AppStripeWebhookRequest struct {
	AppID appentitybase.AppID
	Event stripe.Event
}

type (
	AppStripeWebhookResponse = api.StripeWebhookResponse
	AppStripeWebhookHandler  httptransport.HandlerWithArgs[AppStripeWebhookRequest, AppStripeWebhookResponse, api.ULID]
)

// AppStripeWebhook returns a new httptransport.Handler for creating a customer.
func (h *handler) AppStripeWebhook() AppStripeWebhookHandler {
	return httptransport.NewHandlerWithArgs(
		func(ctx context.Context, r *http.Request, id api.ULID) (AppStripeWebhookRequest, error) {
			// Note that the webhook handler has no namespace resolver
			// We only know the namespace from the app id. Which we trust because
			// we validate the payload signature with the app's webhook secret.

			// Get the webhook secret for the app
			secret, err := h.service.GetWebhookSecret(ctx, appstripeentity.GetWebhookSecretInput{
				AppID: id,
			})
			if err != nil {
				return AppStripeWebhookRequest{}, fmt.Errorf("failed to get webhook secret: %w", err)
			}

			// Validate the webhook payload
			payload, err := io.ReadAll(r.Body)
			if err != nil {
				return AppStripeWebhookRequest{}, fmt.Errorf("failed to read request body: %w", err)
			}

			event, err := webhook.ConstructEvent(payload, r.Header.Get("Stripe-Signature"), secret.Value)
			if err != nil {
				return AppStripeWebhookRequest{}, appstripe.ValidationError{
					Err: fmt.Errorf("failed to construct webhook event: %w", err),
				}
			}

			appID := appentitybase.AppID{
				Namespace: secret.SecretID.Namespace,
				ID:        id,
			}

			req := AppStripeWebhookRequest{
				AppID: appID,
				Event: event,
			}

			return req, nil
		},
		func(ctx context.Context, request AppStripeWebhookRequest) (AppStripeWebhookResponse, error) {
			// In the response, we return what resources took action
			response := AppStripeWebhookResponse{
				NamespaceId: request.AppID.Namespace,
				AppId:       request.AppID.ID,
			}

			switch request.Event.Type {
			case stripeclient.WebhookEventTypeSetupIntentSucceeded:
				// Unmarshal to payment intent object
				var paymentIntent stripe.PaymentIntent

				err := json.Unmarshal(request.Event.Data.Raw, &paymentIntent)
				if err != nil {
					return AppStripeWebhookResponse{}, appstripe.ValidationError{
						Err: fmt.Errorf("failed to unmarshal payment intent: %w", err),
					}
				}

				// Validate the payment intent object
				if paymentIntent.Customer == nil {
					return AppStripeWebhookResponse{}, appstripe.ValidationError{
						Err: fmt.Errorf("payment intent customer is required"),
					}
				}

				if paymentIntent.PaymentMethod == nil {
					return AppStripeWebhookResponse{}, appstripe.ValidationError{
						Err: fmt.Errorf("payment intent payment method is required"),
					}
				}

				// Set the default payment method for the customer
				out, err := h.service.SetCustomerDefaultPaymentMethod(ctx, appstripeentity.SetCustomerDefaultPaymentMethodInput{
					AppID:            request.AppID,
					StripeCustomerID: paymentIntent.Customer.ID,
					PaymentMethodID:  paymentIntent.PaymentMethod.ID,
				})
				if err != nil {
					return AppStripeWebhookResponse{}, err
				}

				// Decroate the response with the customer id
				response.CustomerId = &out.CustomerID.ID

			default:
				return AppStripeWebhookResponse{}, appstripe.ValidationError{
					Err: fmt.Errorf("unsupported event type: %s", request.Event.Type),
				}
			}

			return response, nil
		},
		commonhttp.JSONResponseEncoderWithStatus[AppStripeWebhookResponse](http.StatusCreated),
		httptransport.AppendOptions(
			h.options,
			httptransport.WithOperationName("appStripeWebhook"),
			httptransport.WithErrorEncoder(errorEncoder()),
		)...,
	)
}
