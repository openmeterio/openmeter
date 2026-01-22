package httpdriver

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/samber/lo"
	"github.com/stripe/stripe-go/v80"
	"github.com/stripe/stripe-go/v80/webhook"

	"github.com/openmeterio/openmeter/api"
	"github.com/openmeterio/openmeter/openmeter/app"
	stripeclient "github.com/openmeterio/openmeter/openmeter/app/stripe/client"
	appstripeentity "github.com/openmeterio/openmeter/openmeter/app/stripe/entity"
	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/pkg/framework/commonhttp"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport"
	"github.com/openmeterio/openmeter/pkg/models"
)

type AppStripeWebhookParams struct {
	AppID   string
	Payload []byte
}

type AppStripeWebhookRequest struct {
	AppID app.AppID
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
				return AppStripeWebhookRequest{}, models.NewGenericValidationError(
					fmt.Errorf("failed to construct webhook event: %w", err),
				)
			}

			appID := app.AppID{
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
			ctx = context.WithValue(ctx, StripeEventIDAttributeName, request.Event.ID)
			ctx = context.WithValue(ctx, StripeEventTypeAttributeName, request.Event.Type)
			ctx = context.WithValue(ctx, AppIDAttributeName, request.AppID)

			// Handle the webhook event based on the event type
			switch request.Event.Type {
			case stripeclient.WebhookEventTypeSetupIntentSucceeded:
				// Unmarshal to payment intent object
				var paymentIntent stripe.PaymentIntent

				err := json.Unmarshal(request.Event.Data.Raw, &paymentIntent)
				if err != nil {
					return AppStripeWebhookResponse{}, models.NewGenericValidationError(
						fmt.Errorf("failed to unmarshal payment intent for app: %s in event: %s: %w", request.AppID.ID, request.Event.ID, err),
					)
				}

				// Validate the payment intent metadata
				metadataAppId, hasMetadataAppId := paymentIntent.Metadata[stripeclient.StripeMetadataAppID]

				// If the event has not app metadata it's not initiated by an OpenMeter app and we ignore it.
				// This can be the case when someone manually creates a payment intent.
				if !hasMetadataAppId {
					return AppStripeWebhookResponse{
						Message: lo.ToPtr("ignoring event as it was not initiated by the openmeter app"),
					}, nil
				}

				// When the OpenMeter app id is set it cannot be empty
				if metadataAppId == "" {
					return AppStripeWebhookResponse{}, models.NewGenericValidationError(
						fmt.Errorf("appid metadata cannot be empty if provided for app: %s in event: %s", request.AppID.ID, request.Event.ID),
					)
				}

				// If someone installs the same Stripe account in multiple apps, we need to ignore the event from other apps
				if metadataAppId != request.AppID.ID {
					// Ignore the event from other apps
					return AppStripeWebhookResponse{
						Message: lo.ToPtr("ignoring event as it was initiated by a different openmeter app"),
					}, nil
				}

				// Validate the namespace
				// At this point we know that the event is for this specific app so require the namespace.
				metadataNamespace, hasMetadataNamespace := paymentIntent.Metadata[stripeclient.StripeMetadataNamespace]
				if !hasMetadataNamespace {
					return AppStripeWebhookResponse{}, models.NewGenericValidationError(
						fmt.Errorf("namespace metadata is required for app: %s in event: %s", request.AppID.ID, request.Event.ID),
					)
				}

				// When the namespace is set it cannot be empty
				if metadataNamespace == "" {
					return AppStripeWebhookResponse{}, models.NewGenericValidationError(
						fmt.Errorf("namespace metadata cannot be empty if provided for app: %s in event: %s", request.AppID.ID, request.Event.ID),
					)
				}

				// As we already checked that this event is for this specific app we validate the namespace
				if metadataNamespace != request.AppID.Namespace {
					return AppStripeWebhookResponse{}, models.NewGenericValidationError(
						fmt.Errorf("namespace mismatch: in request %s, in payment intent metadata %s in event: %s", request.AppID.Namespace, metadataNamespace, request.Event.ID),
					)
				}

				// Validate the payment intent object
				if paymentIntent.Customer == nil {
					return AppStripeWebhookResponse{}, models.NewGenericValidationError(
						fmt.Errorf("payment intent customer is required for app: %s in event: %s", request.AppID.ID, request.Event.ID),
					)
				}

				if paymentIntent.PaymentMethod == nil {
					return AppStripeWebhookResponse{}, models.NewGenericValidationError(
						fmt.Errorf("payment intent payment method is required for app %s in event: %s", request.AppID.ID, request.Event.ID),
					)
				}

				// Set the default payment method for the customer
				out, err := h.service.HandleSetupIntentSucceeded(ctx,
					appstripeentity.HandleSetupIntentSucceededInput{
						SetCustomerDefaultPaymentMethodInput: appstripeentity.SetCustomerDefaultPaymentMethodInput{
							AppID:            request.AppID,
							StripeCustomerID: paymentIntent.Customer.ID,
							PaymentMethodID:  paymentIntent.PaymentMethod.ID,
						},
						PaymentIntentMetadata: paymentIntent.Metadata,
					})
				if err != nil {
					return AppStripeWebhookResponse{}, err
				}

				// In the response, we return what resources took action
				return AppStripeWebhookResponse{
					NamespaceId: request.AppID.Namespace,
					AppId:       request.AppID.ID,
					CustomerId:  &out.CustomerID.ID,
					Message:     lo.ToPtr("customer default payment method set"),
				}, nil

			case stripeclient.WebhookEventTypeSetupIntentFailed:
				return AppStripeWebhookResponse{
					NamespaceId: request.AppID.Namespace,
					AppId:       request.AppID.ID,
				}, nil
			case stripeclient.WebhookEventTypeSetupIntentRequiresAction:
				return AppStripeWebhookResponse{
					NamespaceId: request.AppID.Namespace,
					AppId:       request.AppID.ID,
				}, nil

			// Invoice events
			case stripeclient.WebhookEventTypeInvoiceFinalizationFailed:
				invoice, err := unmarshalInvoiceEvent(request.Event.Data.Raw)
				if err != nil {
					return AppStripeWebhookResponse{}, err
				}

				err = h.service.HandleInvoiceStateTransition(ctx, appstripeentity.HandleInvoiceStateTransitionInput{
					AppID:   request.AppID,
					Invoice: invoice,
					Trigger: billing.TriggerFailed,
					TargetStatuses: []billing.StandardInvoiceStatus{
						billing.StandardInvoiceStatusIssuingSyncFailed,
						billing.StandardInvoiceStatusPaymentProcessingFailed,
					},
					IgnoreInvoiceInStatus: []billing.StandardInvoiceStatusMatcher{
						billing.StandardInvoiceStatusCategoryPaymentProcessing,
						billing.StandardInvoiceStatusCategoryPaid,
						billing.StandardInvoiceStatusCategoryUncollectible,
					},
					ShouldTriggerOnEvent: func(stripeInvoice *stripe.Invoice) (bool, error) {
						return stripeInvoice.LastFinalizationError != nil, nil
					},
					GetValidationErrors: func(stripeInvoice *stripe.Invoice) (*appstripeentity.ValidationErrorsInput, error) {
						return &appstripeentity.ValidationErrorsInput{
							Op:     billing.StandardInvoiceOpFinalize,
							Errors: []*stripe.Error{stripeInvoice.LastFinalizationError},
						}, nil
					},
				})
				if err != nil {
					return AppStripeWebhookResponse{}, err
				}

				return AppStripeWebhookResponse{
					NamespaceId: request.AppID.Namespace,
					AppId:       request.AppID.ID,
				}, nil

			case stripeclient.WebhookEventTypeInvoiceSent:
				invoice, err := unmarshalInvoiceEvent(request.Event.Data.Raw)
				if err != nil {
					return AppStripeWebhookResponse{}, err
				}

				err = h.service.HandleInvoiceSentEvent(ctx, appstripeentity.HandleInvoiceSentEventInput{
					AppID:   request.AppID,
					Invoice: invoice,
					SentAt:  request.Event.Created,
				})
				if err != nil {
					return AppStripeWebhookResponse{}, err
				}

				return AppStripeWebhookResponse{
					NamespaceId: request.AppID.Namespace,
					AppId:       request.AppID.ID,
				}, nil

			case stripeclient.WebhookEventTypeInvoiceVoided:
				invoice, err := unmarshalInvoiceEvent(request.Event.Data.Raw)
				if err != nil {
					return AppStripeWebhookResponse{}, err
				}

				err = h.service.HandleInvoiceStateTransition(ctx, appstripeentity.HandleInvoiceStateTransitionInput{
					AppID:          request.AppID,
					Invoice:        invoice,
					Trigger:        billing.TriggerVoid,
					TargetStatuses: []billing.StandardInvoiceStatus{billing.StandardInvoiceStatusVoided},
					IgnoreInvoiceInStatus: []billing.StandardInvoiceStatusMatcher{
						billing.StandardInvoiceStatusCategoryPaid,
					},
					ShouldTriggerOnEvent: func(stripeInvoice *stripe.Invoice) (bool, error) {
						// Let's only invoke the state transition if the upstream invoice is voided
						return stripeInvoice.Status == stripe.InvoiceStatusVoid, nil
					},
				})
				if err != nil {
					return AppStripeWebhookResponse{}, err
				}

				return AppStripeWebhookResponse{
					NamespaceId: request.AppID.Namespace,
					AppId:       request.AppID.ID,
				}, nil

			case stripeclient.WebhookEventTypeInvoiceMarkedUncollectible:
				invoice, err := unmarshalInvoiceEvent(request.Event.Data.Raw)
				if err != nil {
					return AppStripeWebhookResponse{}, err
				}

				err = h.service.HandleInvoiceStateTransition(ctx, appstripeentity.HandleInvoiceStateTransitionInput{
					AppID:          request.AppID,
					Invoice:        invoice,
					Trigger:        billing.TriggerPaymentUncollectible,
					TargetStatuses: []billing.StandardInvoiceStatus{billing.StandardInvoiceStatusUncollectible},
					ShouldTriggerOnEvent: func(stripeInvoice *stripe.Invoice) (bool, error) {
						// Let's only invoke the state transition if the upstream invoice is uncollectible
						return stripeInvoice.Status == stripe.InvoiceStatusUncollectible, nil
					},
				})
				if err != nil {
					return AppStripeWebhookResponse{}, err
				}

				return AppStripeWebhookResponse{
					NamespaceId: request.AppID.Namespace,
					AppId:       request.AppID.ID,
				}, nil
			case stripeclient.WebhookEventTypeInvoiceOverdue:
				invoice, err := unmarshalInvoiceEvent(request.Event.Data.Raw)
				if err != nil {
					return AppStripeWebhookResponse{}, err
				}

				err = h.service.HandleInvoiceStateTransition(ctx, appstripeentity.HandleInvoiceStateTransitionInput{
					AppID:          request.AppID,
					Invoice:        invoice,
					Trigger:        billing.TriggerPaymentOverdue,
					TargetStatuses: []billing.StandardInvoiceStatus{billing.StandardInvoiceStatusOverdue},
					IgnoreInvoiceInStatus: []billing.StandardInvoiceStatusMatcher{
						billing.StandardInvoiceStatusCategoryUncollectible,
					},
					ShouldTriggerOnEvent: func(stripeInvoice *stripe.Invoice) (bool, error) {
						// Let's only invoke the state transition if the upstream invoice is still open
						return stripeInvoice.Status == stripe.InvoiceStatusOpen, nil
					},
				})
				if err != nil {
					return AppStripeWebhookResponse{}, err
				}

				return AppStripeWebhookResponse{
					NamespaceId: request.AppID.Namespace,
					AppId:       request.AppID.ID,
				}, nil
			case stripeclient.WebhookEventTypeInvoicePaid:
				invoice, err := unmarshalInvoiceEvent(request.Event.Data.Raw)
				if err != nil {
					return AppStripeWebhookResponse{}, err
				}

				err = h.service.HandleInvoiceStateTransition(ctx, appstripeentity.HandleInvoiceStateTransitionInput{
					AppID:          request.AppID,
					Invoice:        invoice,
					Trigger:        billing.TriggerPaid,
					TargetStatuses: []billing.StandardInvoiceStatus{billing.StandardInvoiceStatusPaid},
					ShouldTriggerOnEvent: func(stripeInvoice *stripe.Invoice) (bool, error) {
						// Let's only invoke the state transition if the upstream invoice is paid
						return stripeInvoice.Status == stripe.InvoiceStatusPaid, nil
					},
				})
				if err != nil {
					return AppStripeWebhookResponse{}, err
				}

				return AppStripeWebhookResponse{
					NamespaceId: request.AppID.Namespace,
					AppId:       request.AppID.ID,
				}, nil
			case stripeclient.WebhookEventTypeInvoicePaymentActionRequired:
				invoice, err := unmarshalInvoiceEvent(request.Event.Data.Raw)
				if err != nil {
					return AppStripeWebhookResponse{}, err
				}

				err = h.service.HandleInvoiceStateTransition(ctx, appstripeentity.HandleInvoiceStateTransitionInput{
					AppID:          request.AppID,
					Invoice:        invoice,
					Trigger:        billing.TriggerActionRequired,
					TargetStatuses: []billing.StandardInvoiceStatus{billing.StandardInvoiceStatusPaymentProcessingActionRequired},
					IgnoreInvoiceInStatus: []billing.StandardInvoiceStatusMatcher{
						billing.StandardInvoiceStatusCategoryPaid,
						billing.StandardInvoiceStatusCategoryUncollectible,
					},

					ShouldTriggerOnEvent: func(stripeInvoice *stripe.Invoice) (bool, error) {
						// Let's only invoke the state transition if the upstream invoice is still open
						return stripeInvoice.Status == stripe.InvoiceStatusOpen, nil
					},
				})
				if err != nil {
					return AppStripeWebhookResponse{}, err
				}

				return AppStripeWebhookResponse{
					NamespaceId: request.AppID.Namespace,
					AppId:       request.AppID.ID,
				}, nil
			case stripeclient.WebhookEventTypeInvoicePaymentFailed:
				invoice, err := unmarshalInvoiceEvent(request.Event.Data.Raw)
				if err != nil {
					return AppStripeWebhookResponse{}, err
				}

				err = h.service.HandleInvoiceStateTransition(ctx, appstripeentity.HandleInvoiceStateTransitionInput{
					AppID:   request.AppID,
					Invoice: invoice,
					Trigger: billing.TriggerFailed,

					TargetStatuses: []billing.StandardInvoiceStatus{
						billing.StandardInvoiceStatusPaymentProcessingFailed,
					},
					IgnoreInvoiceInStatus: []billing.StandardInvoiceStatusMatcher{
						billing.StandardInvoiceStatusCategoryPaid,
						billing.StandardInvoiceStatusCategoryUncollectible,
					},

					ShouldTriggerOnEvent: func(stripeInvoice *stripe.Invoice) (bool, error) {
						// Let's only invoke the state transition if the upstream invoice is still open
						return stripeInvoice.Status == stripe.InvoiceStatusOpen, nil
					},
				})
				if err != nil {
					return AppStripeWebhookResponse{}, err
				}

				return AppStripeWebhookResponse{
					NamespaceId: request.AppID.Namespace,
					AppId:       request.AppID.ID,
				}, nil
			case stripeclient.WebhookEventTypeInvoicePaymentSucceeded:
				// We ignore this event for now, as we handle the invoice.paid event instead

				// Details: https://docs.stripe.com/invoicing/integration

				// Successful invoice payments trigger both an invoice.paid and invoice.payment_succeeded event. Both event
				// types contain the same invoice data, so it’s only necessary to listen to one of them to be notified of successful
				// invoice payments. The difference is that invoice.payment_succeeded events are sent for successful invoice payments,
				// but aren’t sent when you mark an invoice as paid_out_of_band. invoice.paid events, on the other hand, are triggered for
				// both successful payments and out of band payments. Because invoice.paid covers both scenarios, we typically recommend
				// listening to invoice.paid rather than invoice.payment_succeeded.
				return AppStripeWebhookResponse{
					NamespaceId: request.AppID.Namespace,
					AppId:       request.AppID.ID,
				}, nil
			}

			return AppStripeWebhookResponse{}, models.NewGenericValidationError(
				fmt.Errorf("unsupported event type: %s", request.Event.Type),
			)
		},
		commonhttp.JSONResponseEncoderWithStatus[AppStripeWebhookResponse](http.StatusCreated),
		httptransport.AppendOptions(
			h.options,
			httptransport.WithOperationName("appStripeWebhook"),
		)...,
	)
}

func unmarshalInvoiceEvent(data []byte) (stripe.Invoice, error) {
	var invoice stripe.Invoice
	if err := json.Unmarshal(data, &invoice); err != nil {
		return stripe.Invoice{}, models.NewGenericValidationError(
			fmt.Errorf("failed to unmarshal invoice: %w", err),
		)
	}
	return invoice, nil
}
