package client

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
)

type StripeClientFactory = func(config StripeClientConfig) (StripeClient, error)

type StripeClient interface {
	GetAccount(ctx context.Context) (StripeAccount, error)
	SetupWebhook(ctx context.Context, input SetupWebhookInput) (StripeWebhookEndpoint, error)
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
	client    *client.API
	namespace string
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
		client:    client,
		namespace: config.Namespace,
	}, nil
}

// SetupWebhook setups a stripe webhook to handle setup intents and save the payment method
func (c *stripeClient) SetupWebhook(ctx context.Context, input SetupWebhookInput) (StripeWebhookEndpoint, error) {
	if err := input.Validate(); err != nil {
		return StripeWebhookEndpoint{}, fmt.Errorf("invalid input: %w", err)
	}

	webhookURL, err := url.JoinPath(input.BaseURL, "/api/v1/apps/%s/stripe/webhook", input.AppID.ID)
	if err != nil {
		return StripeWebhookEndpoint{}, fmt.Errorf("failed to join url path: %w", err)
	}

	params := &stripe.WebhookEndpointParams{
		EnabledEvents: []*string{
			lo.ToPtr(WebhookEventTypeSetupIntentSucceeded),
		},
		URL:         lo.ToPtr(webhookURL),
		Description: lo.ToPtr("OpenMeter Stripe Webhook, do not delete or modify manually"),
		Metadata: map[string]string{
			SetupIntentDataMetadataNamespace: input.AppID.Namespace,
			SetupIntentDataMetadataAppID:     input.AppID.ID,
		},
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

// providerError returns a typed error for stripe provider errors
func (c *stripeClient) providerError(err error) error {
	if stripeErr, ok := err.(*stripe.Error); ok {
		if stripeErr.HTTPStatusCode == http.StatusUnauthorized {
			return app.AppProviderAuthenticationError{
				Namespace:     c.namespace,
				ProviderError: errors.New(stripeErr.Msg),
			}
		}

		return app.AppProviderError{
			Namespace:     c.namespace,
			ProviderError: errors.New(stripeErr.Msg),
		}
	}

	return err
}
