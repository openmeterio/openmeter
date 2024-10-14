package httpdriver

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/stripe/stripe-go/v80"
	"github.com/stripe/stripe-go/v80/webhook"

	"github.com/openmeterio/openmeter/api"
	appentitybase "github.com/openmeterio/openmeter/openmeter/app/entity/base"
	appstripe "github.com/openmeterio/openmeter/openmeter/app/stripe"
	stripeclient "github.com/openmeterio/openmeter/openmeter/app/stripe/client"
	appstripeentity "github.com/openmeterio/openmeter/openmeter/app/stripe/entity"
	"github.com/openmeterio/openmeter/pkg/framework/commonhttp"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport"
)

type AppStripeWebhookParams struct {
	AppID   string
	Payload []byte
}

type AppStripeWebhookRequest struct {
	AppID appentitybase.AppID
	Event stripe.Event
}

type (
	AppStripeWebhookResponse = api.StripeWebhookResponse
	AppStripeWebhookHandler  httptransport.HandlerWithArgs[AppStripeWebhookRequest, AppStripeWebhookResponse, AppStripeWebhookParams]
)

// AppStripeWebhook returns a new httptransport.Handler for creating a customer.
func (h *handler) AppStripeWebhook() AppStripeWebhookHandler {
	return httptransport.NewHandlerWithArgs(
		func(ctx context.Context, r *http.Request, params AppStripeWebhookParams) (AppStripeWebhookRequest, error) {
			// Note that the webhook handler has no namespace resolver
			// We only know the namespace from the app id. Which we trust because
			// we validate the payload signature with the app's webhook secret.

			// Get the webhook secret for the app
			secret, err := h.service.GetWebhookSecret(ctx, appstripeentity.GetWebhookSecretInput{
				AppID: params.AppID,
			})
			if err != nil {
				return AppStripeWebhookRequest{}, err
			}

			// Validate the webhook event
			event, err := webhook.ConstructEventWithTolerance(params.Payload, r.Header.Get("Stripe-Signature"), secret.Value, time.Hour*10000)
			if err != nil {
				return AppStripeWebhookRequest{}, appstripe.ValidationError{
					Err: fmt.Errorf("failed to construct webhook event: %w", err),
				}
			}

			appID := appentitybase.AppID{
				Namespace: secret.SecretID.Namespace,
				ID:        params.AppID,
			}

			req := AppStripeWebhookRequest{
				AppID: appID,
				Event: event,
			}

			return req, nil
		},
		func(ctx context.Context, request AppStripeWebhookRequest) (AppStripeWebhookResponse, error) {
			// Handle the webhook event based on the event type
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

				// Validate the payment intent metadata
				if metadataAppId, ok := paymentIntent.Metadata[stripeclient.SetupIntentDataMetadataAppID]; !ok {
					// When the app id is set, it must match the app id in the API path
					if metadataAppId != "" && metadataAppId != request.AppID.ID {
						return AppStripeWebhookResponse{}, appstripe.ValidationError{
							Err: fmt.Errorf("appid mismatch: in request %s, in payment intent metadata %s", request.AppID.ID, metadataAppId),
						}
					}

					// If the app id is not set, we ignore the event as it's not initiated by the app
					// This can be the case when someone manually creates a payment intent
					return AppStripeWebhookResponse{}, nil
				}

				// This is an extra consistency check that should never fail as we skip manually created payment intents above
				if metadataNamespace, ok := paymentIntent.Metadata[stripeclient.SetupIntentDataMetadataNamespace]; !ok || metadataNamespace != request.AppID.Namespace {
					return AppStripeWebhookResponse{}, appstripe.ValidationError{
						Err: fmt.Errorf("namespace mismatch: in request %s, in payment intent metadata %s", request.AppID.Namespace, metadataNamespace),
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

				// In the response, we return what resources took action
				return AppStripeWebhookResponse{
					NamespaceId: request.AppID.Namespace,
					AppId:       request.AppID.ID,
					CustomerId:  &out.CustomerID.ID,
				}, nil
			}

			return AppStripeWebhookResponse{}, appstripe.ValidationError{
				Err: fmt.Errorf("unsupported event type: %s", request.Event.Type),
			}
		},
		commonhttp.JSONResponseEncoderWithStatus[AppStripeWebhookResponse](http.StatusCreated),
		httptransport.AppendOptions(
			h.options,
			httptransport.WithOperationName("appStripeWebhook"),
			httptransport.WithErrorEncoder(errorEncoder()),
		)...,
	)
}
