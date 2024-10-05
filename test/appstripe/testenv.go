package appstripe

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/samber/lo"
	"github.com/stripe/stripe-go/v80"

	"github.com/openmeterio/openmeter/openmeter/app"
	appadapter "github.com/openmeterio/openmeter/openmeter/app/adapter"
	appservice "github.com/openmeterio/openmeter/openmeter/app/service"
	"github.com/openmeterio/openmeter/openmeter/appstripe"
	appstripeadapter "github.com/openmeterio/openmeter/openmeter/appstripe/adapter"
	appstripeentity "github.com/openmeterio/openmeter/openmeter/appstripe/entity"
	appstripeservice "github.com/openmeterio/openmeter/openmeter/appstripe/service"
	"github.com/openmeterio/openmeter/openmeter/customer"
	customeradapter "github.com/openmeterio/openmeter/openmeter/customer/adapter"
	customerservice "github.com/openmeterio/openmeter/openmeter/customer/service"
	"github.com/openmeterio/openmeter/openmeter/secret"
	secretadapter "github.com/openmeterio/openmeter/openmeter/secret/adapter"
	secretservice "github.com/openmeterio/openmeter/openmeter/secret/service"
	"github.com/openmeterio/openmeter/pkg/defaultx"
	entdriver "github.com/openmeterio/openmeter/pkg/framework/entutils/entdriver"
	"github.com/openmeterio/openmeter/pkg/framework/pgdriver"
	"github.com/openmeterio/openmeter/pkg/models"
)

const (
	TestNamespace = "default"

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

func NewTestEnv(ctx context.Context) (TestEnv, error) {
	logger := slog.Default().WithGroup("stripe app")

	postgresHost := defaultx.IfZero(os.Getenv("POSTGRES_HOST"), DefaultPostgresHost)

	postgresDriver, err := pgdriver.NewPostgresDriver(ctx, fmt.Sprintf(PostgresURLTemplate, postgresHost))
	if err != nil {
		return nil, fmt.Errorf("failed to initialize postgres driver: %w", err)
	}

	entPostgresDriver := entdriver.NewEntPostgresDriver(postgresDriver.DB())
	entClient := entPostgresDriver.Client()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err = entClient.Schema.Create(ctx); err != nil {
		return nil, fmt.Errorf("failed to create database schema: %w", err)
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

	// Marketplace
	marketplaceAdapter := appadapter.NewMarketplaceAdapter()

	// App
	appAdapter, err := appadapter.New(appadapter.Config{
		Client:      entClient,
		Marketplace: marketplaceAdapter,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create app adapter: %w", err)
	}

	appService, err := appservice.New(appservice.Config{
		Adapter:     appAdapter,
		Marketplace: marketplaceAdapter,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create app service: %w", err)
	}

	// App Stripe
	appStripeAdapter, err := appstripeadapter.New(appstripeadapter.Config{
		Client:          entClient,
		AppService:      appService,
		CustomerService: customerService,
		Marketplace:     marketplaceAdapter,
		SecretService:   secretService,
		StripeClientFactory: func(config appstripeentity.StripeClientConfig) (appstripeentity.StripeClient, error) {
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

		if err = entPostgresDriver.Close(); err != nil {
			errs = errors.Join(errs, fmt.Errorf("failed to close ent driver: %w", err))
		}

		if err = postgresDriver.Close(); err != nil {
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

func (c *StripeClientMock) GetAccount(ctx context.Context) (appstripeentity.StripeAccount, error) {
	return appstripeentity.StripeAccount{
		StripeAccountID: c.StripeAccountID,
	}, nil
}

func (c *StripeClientMock) GetCustomer(ctx context.Context, stripeCustomerID string) (appstripeentity.StripeCustomer, error) {
	return appstripeentity.StripeCustomer{
		StripeCustomerID: stripeCustomerID,
		DefaultPaymentMethod: &appstripeentity.StripePaymentMethod{
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

func (c *StripeClientMock) CreateCustomer(ctx context.Context, input appstripeentity.StripeClientCreateStripeCustomerInput) (appstripeentity.StripeCustomer, error) {
	if err := input.Validate(); err != nil {
		return appstripeentity.StripeCustomer{}, err
	}

	return appstripeentity.StripeCustomer{
		StripeCustomerID: "cus_123",
	}, nil
}

func (c *StripeClientMock) CreateCheckoutSession(ctx context.Context, input appstripeentity.StripeClientCreateCheckoutSessionInput) (appstripeentity.StripeCheckoutSession, error) {
	if err := input.Validate(); err != nil {
		return appstripeentity.StripeCheckoutSession{}, err
	}

	return appstripeentity.StripeCheckoutSession{
		SessionID:     "cs_123",
		SetupIntentID: "seti_123",
		Mode:          stripe.CheckoutSessionModeSetup,
		URL:           "https://checkout.stripe.com/cs_123/test",
	}, nil
}
