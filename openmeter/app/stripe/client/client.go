package client

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"

	"github.com/samber/lo"
	"github.com/stripe/stripe-go/v80"
	"github.com/stripe/stripe-go/v80/client"

	app "github.com/openmeterio/openmeter/openmeter/app"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/models"
)

// StripeClient is a client for the stripe API without an installed app
// It is useful to call the Stripe API before the app is installed,
// for example during the app installation process.
type StripeClient interface {
	GetAccount(ctx context.Context) (StripeAccount, error)
	SetupWebhook(ctx context.Context, input SetupWebhookInput) (StripeWebhookEndpoint, error)
}

// StripeClientFactory is a factory function to create a StripeClient.
type StripeClientFactory = func(config StripeClientConfig) (StripeClient, error)

type StripeClientConfig struct {
	Namespace string
	APIKey    string
	Logger    *slog.Logger
}

func (c *StripeClientConfig) Validate() error {
	if c.Namespace == "" {
		return fmt.Errorf("namespace is required")
	}

	if c.APIKey == "" {
		return fmt.Errorf("api key is required")
	}

	if c.Logger == nil {
		return fmt.Errorf("logger is required")
	}

	return nil
}

type stripeClient struct {
	client    *client.API
	namespace string
}

func NewStripeClient(config StripeClientConfig) (StripeClient, error) {
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

	return &stripeClient{
		client:    client,
		namespace: config.Namespace,
	}, nil
}

// SetupWebhook setups a stripe webhook to handle setup intents and save the payment method
func (c *stripeClient) SetupWebhook(ctx context.Context, input SetupWebhookInput) (StripeWebhookEndpoint, error) {
	if err := input.Validate(); err != nil {
		return StripeWebhookEndpoint{}, fmt.Errorf("invalid input: %w", err)
	}

	params := &stripe.WebhookEndpointParams{
		EnabledEvents: []*string{
			// Setup intents
			lo.ToPtr(WebhookEventTypeSetupIntentSucceeded),
			lo.ToPtr(WebhookEventTypeSetupIntentFailed),
			lo.ToPtr(WebhookEventTypeSetupIntentRequiresAction),

			// Invoices
			lo.ToPtr(WebhookEventTypeInvoiceFinalizationFailed),
			lo.ToPtr(WebhookEventTypeInvoiceMarkedUncollectible),
			lo.ToPtr(WebhookEventTypeInvoiceOverdue),
			lo.ToPtr(WebhookEventTypeInvoicePaid),
			lo.ToPtr(WebhookEventTypeInvoicePaymentActionRequired),
			lo.ToPtr(WebhookEventTypeInvoicePaymentFailed),
			lo.ToPtr(WebhookEventTypeInvoicePaymentSucceeded),
			lo.ToPtr(WebhookEventTypeInvoiceSent),
			lo.ToPtr(WebhookEventTypeInvoiceVoided),
		},
		URL:         lo.ToPtr(input.WebhookURL),
		Description: lo.ToPtr("OpenMeter Stripe Webhook, do not delete or modify manually"),
		Metadata: map[string]string{
			StripeMetadataNamespace: input.AppID.Namespace,
			StripeMetadataAppID:     input.AppID.ID,
		},
		// We set the API version to a specific date to ensure that
		// the webhook is compatible with the Stripe client's API version.
		// https://docs.stripe.com/sdks/set-version
		APIVersion: lo.ToPtr(stripe.APIVersion),
	}
	result, err := c.client.WebhookEndpoints.New(params)
	if err != nil {
		return StripeWebhookEndpoint{}, c.providerError(err)
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
		Country:         models.CountryCode(stripeAccount.Country),
		BusinessProfile: stripeAccount.BusinessProfile,
	}, nil
}

// providerError returns a typed error for stripe provider errors
func (c *stripeClient) providerError(err error) error {
	if stripeErr, ok := err.(*stripe.Error); ok {
		switch stripeErr.HTTPStatusCode {
		// Let's reflect back invalid request errors to the client.
		case http.StatusBadRequest:
			return models.NewGenericValidationError(
				fmt.Errorf("stripe error: %s, request log url: %s", stripeErr.Msg, stripeErr.RequestLogURL),
			)
		// Let's reflect back unauthorized errors to the client.
		case http.StatusUnauthorized:
			return app.NewAppProviderAuthenticationError(
				nil,
				c.namespace,
				errors.New(stripeErr.Msg),
			)
		default:
			return app.NewAppProviderError(
				nil,
				c.namespace,
				errors.New(stripeErr.Msg),
			)
		}
	}

	return err
}

// Stripe uses lowercase three-letter ISO codes for currency codes.
// See: https://docs.stripe.com/currencies
func Currency(c currencyx.Code) string {
	return strings.ToLower(string(c))
}

// CurrencyPtr is a helper function for pointer currency codes.
func CurrencyPtr(c *currencyx.Code) *string {
	return lo.ToPtr(Currency(*c))
}

// FromStripeCurrency converts a stripe currency code to a currencyx code.
func FromStripeCurrency(c stripe.Currency) currencyx.Code {
	return currencyx.Code(strings.ToUpper(string(c)))
}
