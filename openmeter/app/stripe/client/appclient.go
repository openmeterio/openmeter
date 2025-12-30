package client

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/samber/lo"
	"github.com/stripe/stripe-go/v80"
	"github.com/stripe/stripe-go/v80/client"

	app "github.com/openmeterio/openmeter/openmeter/app"
	"github.com/openmeterio/openmeter/pkg/models"
)

const (
	StripeMetadataNamespace  = "om_namespace"
	StripeMetadataAppID      = "om_app_id"
	StripeMetadataCustomerID = "om_customer_id"
	StripeMetadataInvoiceID  = "om_invoice_id"
)

// SetupIntentReservedMetadataKeys are the keys that are reserved for internal use by OpenMeter
// specifying these keys in the metadata will result in a validation error
var SetupIntentReservedMetadataKeys = []string{
	StripeMetadataNamespace,
	StripeMetadataAppID,
	StripeMetadataCustomerID,
}

const (
	// Stripe Webhook event types

	// Occurs when an SetupIntent has successfully setup a payment method.
	WebhookEventTypeSetupIntentSucceeded = "setup_intent.succeeded"
	// Occurs when an SetupIntent has failed to set up a payment method.
	WebhookEventTypeSetupIntentFailed = "setup_intent.setup_failed"
	// Occurs when a SetupIntent is in requires_action state.
	WebhookEventTypeSetupIntentRequiresAction = "setup_intent.requires_action"

	// Occurs whenever a draft invoice cannot be finalized
	WebhookEventTypeInvoiceFinalizationFailed = "invoice.finalization_failed"
	// Occurs whenever an invoice is marked uncollectible
	WebhookEventTypeInvoiceMarkedUncollectible = "invoice.marked_uncollectible"
	// Occurs X number of days after an invoice becomes due—where X is determined by Automations
	WebhookEventTypeInvoiceOverdue = "invoice.overdue"
	// Occurs whenever an invoice payment attempt succeeds or an invoice is marked as paid out-of-band.
	WebhookEventTypeInvoicePaid = "invoice.paid"
	// Occurs whenever an invoice payment attempt requires further user action to complete.
	WebhookEventTypeInvoicePaymentActionRequired = "invoice.payment_action_required"
	// Occurs whenever an invoice payment attempt fails, due either to a declined payment or to the lack of a stored payment method.
	WebhookEventTypeInvoicePaymentFailed = "invoice.payment_failed"
	// Occurs whenever an invoice payment attempt succeeds.
	WebhookEventTypeInvoicePaymentSucceeded = "invoice.payment_succeeded"
	// Occurs whenever an invoice email is sent out.
	WebhookEventTypeInvoiceSent = "invoice.sent"
	// Occurs whenever an invoice is voided.
	WebhookEventTypeInvoiceVoided = "invoice.voided"
)

// StripeAppClient is a client for the stripe API for an installed app.
// It is useful to call the Stripe API after the app is installed.
type StripeAppClient interface {
	DeleteWebhook(ctx context.Context, input DeleteWebhookInput) error
	GetAccount(ctx context.Context) (StripeAccount, error)
	GetCustomer(ctx context.Context, stripeCustomerID string) (StripeCustomer, error)
	CreateCustomer(ctx context.Context, input CreateStripeCustomerInput) (StripeCustomer, error)
	CreateCheckoutSession(ctx context.Context, input CreateCheckoutSessionInput) (StripeCheckoutSession, error)
	GetPaymentMethod(ctx context.Context, stripePaymentMethodID string) (StripePaymentMethod, error)
	CreatePortalSession(ctx context.Context, input CreatePortalSessionInput) (PortalSession, error)
	// Invoice
	GetInvoice(ctx context.Context, input GetInvoiceInput) (*stripe.Invoice, error)
	CreateInvoice(ctx context.Context, input CreateInvoiceInput) (*stripe.Invoice, error)
	UpdateInvoice(ctx context.Context, input UpdateInvoiceInput) (*stripe.Invoice, error)
	DeleteInvoice(ctx context.Context, input DeleteInvoiceInput) error
	FinalizeInvoice(ctx context.Context, input FinalizeInvoiceInput) (*stripe.Invoice, error)
	// Invoice Line
	ListInvoiceLineItems(ctx context.Context, stripeInvoiceID string) ([]*stripe.InvoiceLineItem, error)
	AddInvoiceLines(ctx context.Context, input AddInvoiceLinesInput) ([]StripeInvoiceItemWithLineID, error)
	UpdateInvoiceLines(ctx context.Context, input UpdateInvoiceLinesInput) ([]*stripe.InvoiceItem, error)
	RemoveInvoiceLines(ctx context.Context, input RemoveInvoiceLinesInput) error
}

// StripeAppClientFactory is a factory for creating a StripeAppClient for an installed app.
type StripeAppClientFactory = func(config StripeAppClientConfig) (StripeAppClient, error)

type StripeAppClientConfig struct {
	AppService app.Service
	AppID      app.AppID
	APIKey     string
	Logger     *slog.Logger
}

func (c *StripeAppClientConfig) Validate() error {
	if c.AppService == nil {
		return fmt.Errorf("app stripe servive is required")
	}

	if err := c.AppID.Validate(); err != nil {
		return fmt.Errorf("app id is required")
	}

	if c.APIKey == "" {
		return fmt.Errorf("api key is required")
	}

	if c.Logger == nil {
		return fmt.Errorf("logger is required")
	}

	return nil
}

type stripeAppClient struct {
	appService app.Service
	appID      app.AppID
	client     *client.API
}

func NewStripeAppClient(config StripeAppClientConfig) (StripeAppClient, error) {
	if err := config.Validate(); err != nil {
		return nil, err
	}

	backend := stripe.GetBackendWithConfig(stripe.APIBackend, &stripe.BackendConfig{
		LeveledLogger: leveledLogger{
			logger: config.Logger,
		},
	})
	client := &client.API{}
	client.Init(config.APIKey, &stripe.Backends{
		API:     backend,
		Connect: backend,
		Uploads: backend,
	})

	return &stripeAppClient{
		appService: config.AppService,
		appID:      config.AppID,
		client:     client,
	}, nil
}

// DeleteWebhook setups a stripe webhook to handle setup intents and save the payment method
func (c *stripeAppClient) DeleteWebhook(ctx context.Context, input DeleteWebhookInput) error {
	_, err := c.client.WebhookEndpoints.Del(input.StripeWebhookID, nil)
	if err != nil {
		if stripeErr, ok := err.(*stripe.Error); ok {
			// Ignore error if user already removed the webhook
			if stripeErr.HTTPStatusCode == http.StatusNotFound {
				return nil
			}

			// Ignore error if user already revoked access
			if stripeErr.HTTPStatusCode == http.StatusUnauthorized {
				return nil
			}
		}

		return c.providerError(err)
	}
	return nil
}

// GetAccount returns the authorized stripe account
func (c *stripeAppClient) GetAccount(ctx context.Context) (StripeAccount, error) {
	stripeAccount, err := c.client.Accounts.Get()
	if err != nil {
		return StripeAccount{}, c.providerError(err)
	}

	return StripeAccount{
		StripeAccountID: stripeAccount.ID,
	}, nil
}

// GetPaymentMethod returns the stripe payment method by stripe payment method ID
func (c *stripeAppClient) GetPaymentMethod(ctx context.Context, stripePaymentMethodID string) (StripePaymentMethod, error) {
	stripePaymentMethod, err := c.client.PaymentMethods.Get(stripePaymentMethodID, nil)
	if err != nil {
		// Stripe customer not found error
		if stripeErr, ok := err.(*stripe.Error); ok && stripeErr.Code == stripe.ErrorCodeResourceMissing {
			if stripeErr.HTTPStatusCode == http.StatusUnauthorized {
				return StripePaymentMethod{}, NewStripePaymentMethodNotFoundError(stripePaymentMethodID)
			}
		}

		return StripePaymentMethod{}, c.providerError(err)
	}

	return toStripePaymentMethod(stripePaymentMethod), nil
}

// StripePaymentMethod converts a Stripe API payment method to a StripePaymentMethod
func toStripePaymentMethod(stripePaymentMethod *stripe.PaymentMethod) StripePaymentMethod {
	paymentMethod := StripePaymentMethod{
		ID: stripePaymentMethod.ID,
	}

	if stripePaymentMethod.Customer != nil {
		paymentMethod.StripeCustomerID = &stripePaymentMethod.Customer.ID
	}

	if stripePaymentMethod.BillingDetails != nil && stripePaymentMethod.BillingDetails.Address != nil {
		address := *stripePaymentMethod.BillingDetails.Address

		paymentMethod.Name = stripePaymentMethod.BillingDetails.Name
		paymentMethod.Email = stripePaymentMethod.BillingDetails.Email

		paymentMethod.BillingAddress = &models.Address{
			Country:     lo.ToPtr(models.CountryCode(address.Country)),
			City:        lo.ToPtr(address.City),
			State:       lo.ToPtr(address.State),
			PostalCode:  lo.ToPtr(address.PostalCode),
			Line1:       lo.ToPtr(address.Line1),
			Line2:       lo.ToPtr(address.Line2),
			PhoneNumber: lo.ToPtr(stripePaymentMethod.BillingDetails.Phone),
		}
	}

	return paymentMethod
}

// providerError returns a typed error for stripe provider errors
func (c *stripeAppClient) providerError(err error) error {
	if stripeErr, ok := err.(*stripe.Error); ok {
		switch stripeErr.HTTPStatusCode {
		// Let's reflect back invalid request errors to the client.
		case http.StatusBadRequest:
			return models.NewGenericValidationError(
				fmt.Errorf("stripe error: %s, request log url: %s", stripeErr.Msg, stripeErr.RequestLogURL),
			)
		// Let's reflect back unauthorized errors to the client.
		// We also update the app status to unauthorized.
		case http.StatusUnauthorized:
			status := app.AppStatusUnauthorized

			err = c.appService.UpdateAppStatus(context.Background(), app.UpdateAppStatusInput{
				ID:     c.appID,
				Status: status,
			})
			if err != nil {
				return fmt.Errorf("failed to update app status to %s for app %s: %w", c.appID.ID, status, err)
			}

			return app.NewAppProviderAuthenticationError(
				&c.appID,
				c.appID.Namespace,
				errors.New(stripeErr.Msg),
			)
		default:
			return app.NewAppProviderError(
				&c.appID,
				c.appID.Namespace,
				errors.New(stripeErr.Msg),
			)
		}
	}

	return err
}
