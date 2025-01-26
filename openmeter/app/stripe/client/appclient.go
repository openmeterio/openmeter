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
	appentity "github.com/openmeterio/openmeter/openmeter/app/entity"
	appentitybase "github.com/openmeterio/openmeter/openmeter/app/entity/base"
	"github.com/openmeterio/openmeter/pkg/models"
)

const (
	SetupIntentDataMetadataNamespace  = "om_namespace"
	SetupIntentDataMetadataAppID      = "om_app_id"
	SetupIntentDataMetadataCustomerID = "om_customer_id"

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
	// Invoice
	CreateInvoice(ctx context.Context, input CreateInvoiceInput) (*stripe.Invoice, error)
	UpdateInvoice(ctx context.Context, input UpdateInvoiceInput) (*stripe.Invoice, error)
	DeleteInvoice(ctx context.Context, input DeleteInvoiceInput) error
	FinalizeInvoice(ctx context.Context, input FinalizeInvoiceInput) (*stripe.Invoice, error)
	// Invoice Line
	AddInvoiceLines(ctx context.Context, input AddInvoiceLinesInput) ([]*stripe.InvoiceItem, error)
	UpdateInvoiceLines(ctx context.Context, input UpdateInvoiceLinesInput) ([]*stripe.InvoiceItem, error)
	RemoveInvoiceLines(ctx context.Context, input RemoveInvoiceLinesInput) error
}

// StripeAppClientFactory is a factory for creating a StripeAppClient for an installed app.
type StripeAppClientFactory = func(config StripeAppClientConfig) (StripeAppClient, error)

type StripeAppClientConfig struct {
	AppService app.Service
	AppID      appentitybase.AppID
	APIKey     string
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

	return nil
}

type stripeAppClient struct {
	appService app.Service
	appID      appentitybase.AppID
	client     *client.API
}

func NewStripeAppClient(config StripeAppClientConfig) (StripeAppClient, error) {
	if err := config.Validate(); err != nil {
		return nil, err
	}

	backend := stripe.GetBackendWithConfig(stripe.APIBackend, &stripe.BackendConfig{
		LeveledLogger: leveledLogger{
			logger: slog.Default(),
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

// GetCustomer returns the stripe customer by stripe customer ID
func (c *stripeAppClient) GetCustomer(ctx context.Context, stripeCustomerID string) (StripeCustomer, error) {
	stripeCustomer, err := c.client.Customers.Get(stripeCustomerID, &stripe.CustomerParams{
		Expand: []*string{lo.ToPtr("invoice_settings.default_payment_method")},
	})
	if err != nil {
		// Stripe customer not found error
		if stripeErr, ok := err.(*stripe.Error); ok && stripeErr.Code == stripe.ErrorCodeResourceMissing {
			if stripeErr.HTTPStatusCode == http.StatusUnauthorized {
				return StripeCustomer{}, StripeCustomerNotFoundError{
					StripeCustomerID: stripeCustomerID,
				}
			}
		}

		return StripeCustomer{}, c.providerError(err)
	}

	customer := StripeCustomer{
		StripeCustomerID: stripeCustomer.ID,
	}

	if stripeCustomer.Currency != "" {
		customer.Currency = lo.ToPtr(string(stripeCustomer.Currency))
	}

	if stripeCustomer.InvoiceSettings != nil {
		invoiceSettings := *stripeCustomer.InvoiceSettings

		if stripeCustomer.InvoiceSettings.DefaultPaymentMethod != nil {
			customer.DefaultPaymentMethod = lo.ToPtr(toStripePaymentMethod(invoiceSettings.DefaultPaymentMethod))
		}
	}

	return customer, nil
}

// GetPaymentMethod returns the stripe payment method by stripe payment method ID
func (c *stripeAppClient) GetPaymentMethod(ctx context.Context, stripePaymentMethodID string) (StripePaymentMethod, error) {
	stripePaymentMethod, err := c.client.PaymentMethods.Get(stripePaymentMethodID, nil)
	if err != nil {
		// Stripe customer not found error
		if stripeErr, ok := err.(*stripe.Error); ok && stripeErr.Code == stripe.ErrorCodeResourceMissing {
			if stripeErr.HTTPStatusCode == http.StatusUnauthorized {
				return StripePaymentMethod{}, StripePaymentMethodNotFoundError{
					StripePaymentMethodID: stripePaymentMethodID,
				}
			}
		}

		return StripePaymentMethod{}, c.providerError(err)
	}

	return toStripePaymentMethod(stripePaymentMethod), nil
}

// CreateCustomer creates a stripe customer
func (c *stripeAppClient) CreateCustomer(ctx context.Context, input CreateStripeCustomerInput) (StripeCustomer, error) {
	if err := input.Validate(); err != nil {
		return StripeCustomer{}, err
	}

	// Create customer
	stripeCustomer, err := c.client.Customers.New(&stripe.CustomerParams{
		Name: input.Name,
		Metadata: map[string]string{
			SetupIntentDataMetadataNamespace:  input.AppID.Namespace,
			SetupIntentDataMetadataCustomerID: input.CustomerID.ID,
		},
	})
	if err != nil {
		return StripeCustomer{}, c.providerError(err)
	}

	out := StripeCustomer{
		StripeCustomerID: stripeCustomer.ID,
	}

	return out, nil
}

// CreateCheckoutSession creates a checkout session
func (c *stripeAppClient) CreateCheckoutSession(ctx context.Context, input CreateCheckoutSessionInput) (StripeCheckoutSession, error) {
	if err := input.Validate(); err != nil {
		return StripeCheckoutSession{}, err
	}

	// Enrich user sent metadata with openmeter metadata
	metadata := map[string]string{}

	if len(input.Options.Metadata) > 0 {
		for k, v := range input.Options.Metadata {
			metadata[k] = v
		}
	}

	metadata[SetupIntentDataMetadataNamespace] = input.AppID.Namespace
	metadata[SetupIntentDataMetadataAppID] = input.AppID.ID
	metadata[SetupIntentDataMetadataCustomerID] = input.CustomerID.ID

	// Create checkout session
	params := &stripe.CheckoutSessionParams{
		Customer:                 lo.ToPtr(input.StripeCustomerID),
		Currency:                 input.Options.Currency,
		Mode:                     lo.ToPtr(string(stripe.CheckoutSessionModeSetup)),
		BillingAddressCollection: lo.ToPtr(string(stripe.CheckoutSessionBillingAddressCollectionAuto)),
		CustomerUpdate: &stripe.CheckoutSessionCustomerUpdateParams{
			Address: lo.ToPtr("auto"),
			Name:    lo.ToPtr("auto"),
		},
		TaxIDCollection: &stripe.CheckoutSessionTaxIDCollectionParams{
			Enabled:  lo.ToPtr(true),
			Required: lo.ToPtr("if_supported"),
		},
		SetupIntentData: &stripe.CheckoutSessionSetupIntentDataParams{
			Metadata: metadata,
		},
	}

	if input.Options.SuccessURL != nil {
		params.SuccessURL = input.Options.SuccessURL
	}

	if input.Options.CancelURL != nil {
		params.CancelURL = input.Options.CancelURL
	}

	if input.Options.ReturnURL != nil {
		params.ReturnURL = input.Options.ReturnURL
	}

	if input.Options.ClientReferenceID != nil {
		params.ClientReferenceID = input.Options.ClientReferenceID
	}

	if input.Options.CustomText != nil {
		params.CustomText = input.Options.CustomText
	}

	if input.Options.UIMode != nil {
		params.UIMode = lo.ToPtr(string(*input.Options.UIMode))
	}

	if input.Options.PaymentMethodTypes != nil {
		params.PaymentMethodTypes = *input.Options.PaymentMethodTypes
	}

	session, err := c.client.CheckoutSessions.New(params)
	if err != nil {
		return StripeCheckoutSession{}, c.providerError(err)
	}

	// Create output
	if session.SetupIntent == nil {
		return StripeCheckoutSession{}, errors.New("setup intent is required")
	}

	stripeCheckoutSession := StripeCheckoutSession{
		SessionID:     session.ID,
		SetupIntentID: session.SetupIntent.ID,
		Mode:          session.Mode,
	}

	if session.URL != "" {
		stripeCheckoutSession.URL = &session.URL
	}

	if session.CancelURL != "" {
		stripeCheckoutSession.CancelURL = &session.CancelURL
	}

	if session.ReturnURL != "" {
		stripeCheckoutSession.ReturnURL = &session.ReturnURL
	}

	if session.SuccessURL != "" {
		stripeCheckoutSession.SuccessURL = &session.SuccessURL
	}

	return stripeCheckoutSession, nil
}

// StripePaymentMethod converts a Stripe API payment method to a StripePaymentMethod
func toStripePaymentMethod(stripePaymentMethod *stripe.PaymentMethod) StripePaymentMethod {
	paymentMethod := StripePaymentMethod{
		ID: stripePaymentMethod.ID,
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
		if stripeErr.HTTPStatusCode == http.StatusUnauthorized {
			// Update app status to unauthorized
			status := appentitybase.AppStatusUnauthorized

			err = c.appService.UpdateAppStatus(context.Background(), appentity.UpdateAppStatusInput{
				ID:     c.appID,
				Status: status,
			})
			if err != nil {
				return fmt.Errorf("failed to update app status to %s for app %s: %w", c.appID.ID, status, err)
			}

			return app.AppProviderAuthenticationError{
				AppID:         &c.appID,
				ProviderError: errors.New(stripeErr.Msg),
			}
		}

		return app.AppProviderError{
			AppID:         &c.appID,
			ProviderError: errors.New(stripeErr.Msg),
		}
	}

	return err
}
