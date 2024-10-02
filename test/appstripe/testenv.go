package appstripe

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/openmeterio/openmeter/openmeter/app"
	appadapter "github.com/openmeterio/openmeter/openmeter/app/adapter"
	appservice "github.com/openmeterio/openmeter/openmeter/app/service"
	"github.com/openmeterio/openmeter/openmeter/appstripe"
	appstripeadapter "github.com/openmeterio/openmeter/openmeter/appstripe/adapter"
	appstripeobserver "github.com/openmeterio/openmeter/openmeter/appstripe/observer"
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
		SecretService:   secretService,
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

	// Register stripe app type with marketplace
	err = appstripe.Register(appstripe.RegisterConfig{
		AppService:       appService,
		AppStripeService: appStripeService,
		Client:           entClient,
		Marketplace:      marketplaceAdapter,
		SecretService:    secretService,
		StripeClientFactory: func(apiKey string) appstripe.StripeClient {
			return &StripeClientMock{
				StripeAccountID: "acct_123",
			}
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to register stripe app: %w", err)
	}

	// App Stripe Customer
	appStripeObserver, err := appstripeobserver.NewCustomerObserver(appstripeobserver.CustomerObserverConfig{
		AppService:       appService,
		AppstripeService: appStripeService,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create app stripe observer: %w", err)
	}

	// Register app stripe observer on customer service
	err = customerService.Register(appStripeObserver)
	if err != nil {
		return nil, fmt.Errorf("failed to register app stripe observer on custoemr service: %w", err)
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

func (c *StripeClientMock) GetAccount(ctx context.Context) (appstripe.StripeAccount, error) {
	return appstripe.StripeAccount{
		StripeAccountID: c.StripeAccountID,
	}, nil
}

func (c *StripeClientMock) GetCustomer(ctx context.Context, stripeCustomerID string) (appstripe.StripeCustomer, error) {
	return appstripe.StripeCustomer{
		StripeCustomerID: stripeCustomerID,
	}, nil
}
