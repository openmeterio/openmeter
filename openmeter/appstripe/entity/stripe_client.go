package appstripeentity

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"

	"github.com/samber/lo"
	"github.com/stripe/stripe-go/v80"
	"github.com/stripe/stripe-go/v80/client"

	app "github.com/openmeterio/openmeter/openmeter/app"
	appentitybase "github.com/openmeterio/openmeter/openmeter/app/entity/base"
	"github.com/openmeterio/openmeter/pkg/models"
)

type StripeClientFactory = func(config StripeClientConfig) (StripeClient, error)

type StripeClient interface {
	SetupWebhook(ctx context.Context, input StripeClientSetupWebhookInput) (StripeWebhookEndpoint, error)
	GetAccount(ctx context.Context) (StripeAccount, error)
	GetCustomer(ctx context.Context, stripeCustomerID string) (StripeCustomer, error)
	CreateCustomer(ctx context.Context, input StripeClientCreateStripeCustomerInput) (StripeCustomer, error)
	CreateCheckoutSession(ctx context.Context, input StripeClientCreateCheckoutSessionInput) (StripeCheckoutSession, error)
}

type StripeClientConfig struct {
	Namespace string
	APIKey    string
}

func (c *StripeClientConfig) Validate() error {
	if c.Namespace == "" {
		return fmt.Errorf("namespace is required")
	}

	if c.APIKey == "" {
		return fmt.Errorf("api key is required")
	}

	return nil
}

type stripeClient struct {
	namespace string
	client    *client.API
}

func NewStripeClient(config StripeClientConfig) (StripeClient, error) {
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

	return &stripeClient{
		namespace: config.Namespace,
		client:    client,
	}, nil
}

// leveledLogger is a logger that implements the stripe LeveledLogger interface
var _ stripe.LeveledLoggerInterface = (*leveledLogger)(nil)

type leveledLogger struct {
	logger *slog.Logger
}

func (l leveledLogger) Debugf(format string, args ...interface{}) {
	l.logger.Debug(fmt.Sprintf(format, args...))
}

func (l leveledLogger) Infof(format string, args ...interface{}) {
	l.logger.Info(fmt.Sprintf(format, args...))
}

func (l leveledLogger) Warnf(format string, args ...interface{}) {
	l.logger.Warn(fmt.Sprintf(format, args...))
}

func (l leveledLogger) Errorf(format string, args ...interface{}) {
	l.logger.Error(fmt.Sprintf(format, args...))
}

// SetupWebhook setups a stripe webhook to handle setup intents and save the payment method
func (c *stripeClient) SetupWebhook(ctx context.Context, input StripeClientSetupWebhookInput) (StripeWebhookEndpoint, error) {
	if err := input.Validate(); err != nil {
		return StripeWebhookEndpoint{}, fmt.Errorf("invalid input: %w", err)
	}

	webhookURL, err := url.JoinPath(input.BaseURL, "/api/v1/apps/%s/stripe/webhook", input.AppID.ID)
	if err != nil {
		return StripeWebhookEndpoint{}, fmt.Errorf("failed to join url path: %w", err)
	}

	params := &stripe.WebhookEndpointParams{
		EnabledEvents: []*string{
			lo.ToPtr("setup_intent.succeeded"),
		},
		URL: lo.ToPtr(webhookURL),
	}
	result, err := c.client.WebhookEndpoints.New(params)
	if err != nil {
		return StripeWebhookEndpoint{}, fmt.Errorf("failed to create stripe webhook: %w", err)
	}

	out := StripeWebhookEndpoint{
		EndpointID: result.ID,
		Secret:     result.Secret,
	}

	return out, nil
}

// GetAccount returns the authorized stripe account
func (c *stripeClient) GetAccount(ctx context.Context) (StripeAccount, error) {
	stripeAccount, err := c.client.Accounts.Get()
	if err != nil {
		return StripeAccount{}, c.providerError(err)
	}

	return StripeAccount{
		StripeAccountID: stripeAccount.ID,
	}, nil
}

// GetCustomer returns the stripe customer by stripe customer ID
func (c *stripeClient) GetCustomer(ctx context.Context, stripeCustomerID string) (StripeCustomer, error) {
	stripeCustomer, err := c.client.Customers.Get(stripeCustomerID, nil)
	if err != nil {
		// Stripe customer not found error
		if stripeErr, ok := err.(*stripe.Error); ok && stripeErr.Code == stripe.ErrorCodeResourceMissing {
			if stripeErr.HTTPStatusCode == http.StatusUnauthorized {
				return StripeCustomer{}, stripeCustomerNotFoundError{
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

// CreateCustomer creates a stripe customer
func (c *stripeClient) CreateCustomer(ctx context.Context, input StripeClientCreateStripeCustomerInput) (StripeCustomer, error) {
	if err := input.Validate(); err != nil {
		return StripeCustomer{}, err
	}

	// Create customer
	params := &stripe.CustomerParams{}

	stripeCustomer, err := c.client.Customers.New(params)
	if err != nil {
		return StripeCustomer{}, c.providerError(err)
	}

	out := StripeCustomer{
		StripeCustomerID: stripeCustomer.ID,
	}

	return out, nil
}

// CreateCheckoutSession creates a checkout session
func (c *stripeClient) CreateCheckoutSession(ctx context.Context, input StripeClientCreateCheckoutSessionInput) (StripeCheckoutSession, error) {
	if err := input.Validate(); err != nil {
		return StripeCheckoutSession{}, err
	}

	// Create checkout session
	params := &stripe.CheckoutSessionParams{
		Customer:                 lo.ToPtr(input.StripeCustomerID),
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
			Metadata: map[string]string{
				"om_namespace":   input.AppID.Namespace,
				"om_app_id":      input.AppID.ID,
				"om_customer_id": input.CustomerID.ID,
			},
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

	if len(input.Options.Metadata) > 0 {
		params.Metadata = input.Options.Metadata
	}

	if input.Options.UIMode != nil {
		params.Mode = lo.ToPtr(string(*input.Options.UIMode))
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
		URL:           session.URL,
		SetupIntentID: session.SetupIntent.ID,
		Mode:          session.Mode,
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
func (c *stripeClient) providerError(err error) error {
	if stripeErr, ok := err.(*stripe.Error); ok {
		if stripeErr.HTTPStatusCode == http.StatusUnauthorized {
			return app.AppProviderAuthenticationError{
				Namespace:     c.namespace,
				Type:          appentitybase.AppTypeStripe,
				ProviderError: errors.New(stripeErr.Msg),
			}
		}

		return app.AppProviderError{
			Namespace:     c.namespace,
			Type:          appentitybase.AppTypeStripe,
			ProviderError: errors.New(stripeErr.Msg),
		}
	}

	return err
}

var _ error = (*stripeCustomerNotFoundError)(nil)

type stripeCustomerNotFoundError struct {
	StripeCustomerID string
}

func (e stripeCustomerNotFoundError) Error() string {
	return fmt.Sprintf("stripe customer %s not found", e.StripeCustomerID)
}
