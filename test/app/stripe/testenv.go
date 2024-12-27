package appstripe

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"testing"

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
	"github.com/openmeterio/openmeter/openmeter/testutils"
)

const (
	PostgresURLTemplate = "postgres://postgres:postgres@%s:5432/postgres?sslmode=disable"
)

type TestEnv interface {
	App() app.Service
	AppStripe() appstripe.Service
	Customer() customer.Service
	Fixture() *Fixture
	Secret() *MockSecretService
	StripeClient() *StripeClientMock
	StripeAppClient() *StripeAppClientMock
	Close() error
}

var _ TestEnv = (*testEnv)(nil)

type testEnv struct {
	app             app.Service
	appstripe       appstripe.Service
	customer        customer.Service
	fixture         *Fixture
	secret          *MockSecretService
	stripeClient    *StripeClientMock
	stripeAppClient *StripeAppClientMock

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

func (n testEnv) Fixture() *Fixture {
	return n.fixture
}

func (n testEnv) Secret() *MockSecretService {
	return n.secret
}

func (n testEnv) StripeClient() *StripeClientMock {
	return n.stripeClient
}

func (n testEnv) StripeAppClient() *StripeAppClientMock {
	return n.stripeAppClient
}

const (
	DefaultPostgresHost = "127.0.0.1"
)

func NewTestEnv(t *testing.T, ctx context.Context) (TestEnv, error) {
	logger := slog.Default().WithGroup("stripe app")

	// Initialize postgres driver
	driver := testutils.InitPostgresDB(t)

	entClient := driver.EntDriver.Client()
	if err := entClient.Schema.Create(ctx); err != nil {
		t.Fatalf("failed to create schema: %v", err)
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
	secretService, err := NewMockSecretService()
	if err != nil {
		return nil, fmt.Errorf("failed to create secret service mock: %w", err)
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

	// Stripe Client
	stripeClientMock := &StripeClientMock{}
	stripeAppClientMock := &StripeAppClientMock{}

	// App Stripe
	appStripeAdapter, err := appstripeadapter.New(appstripeadapter.Config{
		Client:          entClient,
		AppService:      appService,
		CustomerService: customerService,
		SecretService:   secretService,
		StripeClientFactory: func(config stripeclient.StripeClientConfig) (stripeclient.StripeClient, error) {
			return stripeClientMock, nil
		},
		StripeAppClientFactory: func(config stripeclient.StripeAppClientConfig) (stripeclient.StripeAppClient, error) {
			return stripeAppClientMock, nil
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create appstripe adapter: %w", err)
	}

	appStripeService, err := appstripeservice.New(appstripeservice.Config{
		Adapter:       appStripeAdapter,
		AppService:    appService,
		SecretService: secretService,
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
		app:             appService,
		appstripe:       appStripeService,
		customer:        customerService,
		fixture:         NewFixture(appService, customerService, stripeClientMock),
		secret:          secretService,
		closerFunc:      closerFunc,
		stripeClient:    stripeClientMock,
		stripeAppClient: stripeAppClientMock,
	}, nil
}
