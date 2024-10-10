package appstripe

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"testing"

	"github.com/samber/lo"
	"github.com/stripe/stripe-go/v80"

	"github.com/openmeterio/openmeter/openmeter/app"
	appadapter "github.com/openmeterio/openmeter/openmeter/app/adapter"
	appservice "github.com/openmeterio/openmeter/openmeter/app/service"
	appstripe "github.com/openmeterio/openmeter/openmeter/app/stripe"
	appstripeadapter "github.com/openmeterio/openmeter/openmeter/app/stripe/adapter"
	stripeclient "github.com/openmeterio/openmeter/openmeter/app/stripe/client"
	appstripeservice "github.com/openmeterio/openmeter/openmeter/app/stripe/service"
	"github.com/openmeterio/openmeter/openmeter/customer"
	customeradapter "github.com/openmeterio/openmeter/openmeter/customer/adapter"
	customerservice "github.com/openmeterio/openmeter/openmeter/customer/service"
	"github.com/openmeterio/openmeter/openmeter/secret"
	secretadapter "github.com/openmeterio/openmeter/openmeter/secret/adapter"
	secretservice "github.com/openmeterio/openmeter/openmeter/secret/service"
	"github.com/openmeterio/openmeter/openmeter/testutils"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/tools/migrate"
)

const (
	PostgresURLTemplate = "postgres://postgres:postgres@%s:5432/postgres?sslmode=disable"
)

type TestEnv interface {
	App() app.Service
	AppStripe() appstripe.Service
	Customer() customer.Service
	Secret() secret.Service
	Close() error
}

var _ TestEnv = (*testEnv)(nil)

type testEnv struct {
	app       app.Service
	appstripe appstripe.Service
	customer  customer.Service
	secret    secret.Service

	closerFunc func() error
}

func (n testEnv) Close() error {
	return n.closerFunc()
}

func (n testEnv) App() app.Service {
	return n.app
}

func (n testEnv) AppStripe() appstripe.Service {
	return n.appstripe
}

func (n testEnv) Customer() customer.Service {
	return n.customer
}

func (n testEnv) Secret() secret.Service {
	return n.secret
}

const (
	DefaultPostgresHost = "127.0.0.1"
)

func NewTestEnv(t *testing.T, ctx context.Context) (TestEnv, error) {
	logger := slog.Default().WithGroup("stripe app")

	// Initialize postgres driver
	driver := testutils.InitPostgresDB(t)

	entClient := driver.EntDriver.Client()
	if err := migrate.Up(driver.URL); err != nil {
		t.Fatalf("failed to migrate db: %s", err.Error())
	}

	// Customer
	customerAdapter, err := customeradapter.New(customeradapter.Config{
		Client: entClient,
		Logger: logger.WithGroup("postgres"),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create customer repo: %w", err)
	}

	customerService, err := customerservice.New(customerservice.Config{
		Adapter: customerAdapter,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create customer service: %w", err)
	}

	// Secret
	secretAdapter := secretadapter.New()

	secretService, err := secretservice.New(secretservice.Config{
		Adapter: secretAdapter,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create secret service")
	}

	// App
	appAdapter, err := appadapter.New(appadapter.Config{
		Client:  entClient,
		BaseURL: "http://localhost:8888",
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create app adapter: %w", err)
	}

	appService, err := appservice.New(appservice.Config{
		Adapter: appAdapter,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create app service: %w", err)
	}

	// App Stripe
	appStripeAdapter, err := appstripeadapter.New(appstripeadapter.Config{
		Client:          entClient,
		AppService:      appService,
		CustomerService: customerService,
		SecretService:   secretService,
		StripeClientFactory: func(config stripeclient.StripeClientConfig) (stripeclient.StripeClient, error) {
			return &StripeClientMock{
				StripeAccountID: "acct_123",
			}, nil
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create appstripe adapter: %w", err)
	}

	appStripeService, err := appstripeservice.New(appstripeservice.Config{
		Adapter: appStripeAdapter,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create appstripe service: %w", err)
	}

	closerFunc := func() error {
		var errs error

		if err = entClient.Close(); err != nil {
			errs = errors.Join(errs, fmt.Errorf("failed to close ent driver: %w", err))
		}

		if err = driver.PGDriver.Close(); err != nil {
			errs = errors.Join(errs, fmt.Errorf("failed to close postgres driver: %w", err))
		}

		return errs
	}

	return &testEnv{
		app:        appService,
		appstripe:  appStripeService,
		customer:   customerService,
		secret:     secretService,
		closerFunc: closerFunc,
	}, nil
}

type StripeClientMock struct {
	StripeAccountID string
}

func (c *StripeClientMock) SetupWebhook(ctx context.Context, input stripeclient.SetupWebhookInput) (stripeclient.StripeWebhookEndpoint, error) {
	return stripeclient.StripeWebhookEndpoint{
		EndpointID: "we_123",
		Secret:     "whsec_123",
	}, nil
}

func (c *StripeClientMock) GetAccount(ctx context.Context) (stripeclient.StripeAccount, error) {
	return stripeclient.StripeAccount{
		StripeAccountID: c.StripeAccountID,
	}, nil
}

func (c *StripeClientMock) GetCustomer(ctx context.Context, stripeCustomerID string) (stripeclient.StripeCustomer, error) {
	return stripeclient.StripeCustomer{
		StripeCustomerID: stripeCustomerID,
		DefaultPaymentMethod: &stripeclient.StripePaymentMethod{
			ID:    "pm_123",
			Name:  "ACME Inc.",
			Email: "acme@test.com",
			BillingAddress: &models.Address{
				City:       lo.ToPtr("San Francisco"),
				PostalCode: lo.ToPtr("94103"),
				State:      lo.ToPtr("CA"),
				Country:    lo.ToPtr(models.CountryCode("US")),
				Line1:      lo.ToPtr("123 Market St"),
			},
		},
	}, nil
}

func (c *StripeClientMock) CreateCustomer(ctx context.Context, input stripeclient.CreateStripeCustomerInput) (stripeclient.StripeCustomer, error) {
	if err := input.Validate(); err != nil {
		return stripeclient.StripeCustomer{}, err
	}

	return stripeclient.StripeCustomer{
		StripeCustomerID: "cus_123",
	}, nil
}

func (c *StripeClientMock) CreateCheckoutSession(ctx context.Context, input stripeclient.CreateCheckoutSessionInput) (stripeclient.StripeCheckoutSession, error) {
	if err := input.Validate(); err != nil {
		return stripeclient.StripeCheckoutSession{}, err
	}

	return stripeclient.StripeCheckoutSession{
		SessionID:     "cs_123",
		SetupIntentID: "seti_123",
		Mode:          stripe.CheckoutSessionModeSetup,
		URL:           "https://checkout.stripe.com/cs_123/test",
	}, nil
}

func (c *StripeClientMock) GetPaymentMethod(ctx context.Context, paymentMethodID string) (stripeclient.StripePaymentMethod, error) {
	return stripeclient.StripePaymentMethod{
		ID:    "pm_123",
		Name:  "ACME Inc.",
		Email: "acme@test.com",
		BillingAddress: &models.Address{
			City:       lo.ToPtr("San Francisco"),
			PostalCode: lo.ToPtr("94103"),
			State:      lo.ToPtr("CA"),
			Country:    lo.ToPtr(models.CountryCode("US")),
			Line1:      lo.ToPtr("123 Market St"),
		},
	}, nil
}
