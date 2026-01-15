package apps

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/stripe/stripe-go/v80"
	"github.com/stripe/stripe-go/v80/webhook"

	api "github.com/openmeterio/openmeter/api/v3"
	"github.com/openmeterio/openmeter/api/v3/apierrors"
	"github.com/openmeterio/openmeter/openmeter/app"
	appstripeentity "github.com/openmeterio/openmeter/openmeter/app/stripe/entity"
	appstripehttpdriver "github.com/openmeterio/openmeter/openmeter/app/stripe/httpdriver"
	"github.com/openmeterio/openmeter/pkg/framework/commonhttp"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport"
	"github.com/openmeterio/openmeter/pkg/models"
)

type (
	HandleStripeWebhookRequest struct {
		AppId app.AppID
		Event stripe.Event
	}
	HandleStripeWebhookResponse = api.BillingAppStripeWebhookResponse
	HandleStripeWebhookParams   struct {
		AppID   api.ULID
		Payload []byte
	}
	HandleStripeWebhookHandler httptransport.HandlerWithArgs[HandleStripeWebhookRequest, HandleStripeWebhookResponse, HandleStripeWebhookParams]
)

func (h *handler) HandleStripeWebhook() HandleStripeWebhookHandler {
	return httptransport.NewHandlerWithArgs(
		func(ctx context.Context, r *http.Request, params HandleStripeWebhookParams) (HandleStripeWebhookRequest, error) {
			// Note that the webhook handler has no namespace resolver
			// We only know the namespace from the app id. Which we trust because
			// we validate the payload signature with the app's webhook secret.

			// Get the webhook secret for the app
			secret, err := h.stripeService.GetWebhookSecret(ctx, appstripeentity.GetWebhookSecretInput{
				AppID: params.AppID,
			})
			if err != nil {
				return HandleStripeWebhookRequest{}, err
			}

			// Validate the webhook event
			event, err := webhook.ConstructEventWithTolerance(params.Payload, r.Header.Get("Stripe-Signature"), secret.Value, time.Hour*10000)
			if err != nil {
				return HandleStripeWebhookRequest{}, models.NewGenericValidationError(
					fmt.Errorf("failed to construct webhook event: %w", err),
				)
			}

			appID := app.AppID{
				Namespace: secret.SecretID.Namespace,
				ID:        params.AppID,
			}

			req := HandleStripeWebhookRequest{
				AppId: appID,
				Event: event,
			}

			return req, nil
		},
		func(ctx context.Context, request HandleStripeWebhookRequest) (HandleStripeWebhookResponse, error) {
			appStripeWebhookRequest := ConvertStripeWebhookRequest(request)

			response, err := appstripehttpdriver.HandleAppStripeWebhookRequest(ctx, appStripeWebhookRequest, h.stripeService)
			if err != nil {
				return HandleStripeWebhookResponse{}, err
			}

			return ConvertStripeWebhookResponse(response), nil
		},
		commonhttp.JSONResponseEncoder[HandleStripeWebhookResponse],
		httptransport.AppendOptions(
			h.options,
			httptransport.WithOperationName("handle-stripe-webhook"),
			httptransport.WithErrorEncoder(apierrors.GenericErrorEncoder()),
		)...,
	)
}
