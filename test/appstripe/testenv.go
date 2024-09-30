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
	"github.com/openmeterio/openmeter/openmeter/appcustomer"
	appcustomeradapter "github.com/openmeterio/openmeter/openmeter/appcustomer/adapter"
	appcustomerservice "github.com/openmeterio/openmeter/openmeter/appcustomer/service"
	"github.com/openmeterio/openmeter/openmeter/appstripe"
	appstripeadapter "github.com/openmeterio/openmeter/openmeter/appstripe/adapter"
	appstripeobserver "github.com/openmeterio/openmeter/openmeter/appstripe/observer"
	appstripeservice "github.com/openmeterio/openmeter/openmeter/appstripe/service"
	"github.com/openmeterio/openmeter/openmeter/customer"
	customerrepository "github.com/openmeterio/openmeter/openmeter/customer/repository"
	"github.com/openmeterio/openmeter/pkg/defaultx"
	entdriver "github.com/openmeterio/openmeter/pkg/framework/entutils/entdriver"
	"github.com/openmeterio/openmeter/pkg/framework/pgdriver"
)

const (
	TestNamespace = "default"

	PostgresURLTemplate = "postgres://postgres:postgres@%s:5432/postgres?sslmode=disable"
)

type TestEnv interface {
	AppStripeAdapter() appstripe.Adapter
	AppStripe() appstripe.Service

	CustomerAdapter() customer.Repository
	Customer() customer.Service

	Close() error
}

var _ TestEnv = (*testEnv)(nil)

type testEnv struct {
	adapter   appstripe.Adapter
	appstripe appstripe.Service

	appAdapter         app.Adapter
	app                app.Service
	appCustomerAdapter appcustomer.Adapter
	appCustomerService appcustomer.Service
	customerAdapter    customer.Repository
	customer           customer.Service

	closerFunc func() error
}

func (n testEnv) Close() error {
	return n.closerFunc()
}

func (n testEnv) AppStripeAdapter() appstripe.Adapter {
	return n.adapter
}

func (n testEnv) AppStripe() appstripe.Service {
	return n.appstripe
}

func (n testEnv) CustomerAdapter() customer.Repository {
	return n.customerAdapter
}

func (n testEnv) Customer() customer.Service {
	return n.customer
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
	customerRepo, err := customerrepository.New(customerrepository.Config{
		Client: entClient,
		Logger: logger.WithGroup("postgres"),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create customer repo: %w", err)
	}

	customerService, err := customer.NewService(customer.ServiceConfig{
		Repository: customerRepo,
	})
	if err != nil {
		return nil, err
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
		return nil, err
	}

	// App Customer
	appCustomerAdapter, err := appcustomeradapter.New(appcustomeradapter.Config{
		Client: entClient,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create appcustomer adapter: %w", err)
	}

	appCustomerService, err := appcustomerservice.New(appcustomerservice.Config{
		Adapter: appCustomerAdapter,
	})
	if err != nil {
		return nil, err
	}

	// App Stripe
	appStripeAdapter, err := appstripeadapter.New(appstripeadapter.Config{
		Client:             entClient,
		AppService:         appService,
		AppCustomerService: appCustomerService,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create appstripe adapter: %w", err)
	}

	appStripeService, err := appstripeservice.New(appstripeservice.Config{
		Adapter: appStripeAdapter,
	})
	if err != nil {
		return nil, err
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
		adapter:            appStripeAdapter,
		appstripe:          appStripeService,
		appAdapter:         appAdapter,
		app:                appService,
		appCustomerAdapter: appCustomerAdapter,
		appCustomerService: appCustomerService,
		customerAdapter:    customerRepo,
		customer:           customerService,
		closerFunc:         closerFunc,
	}, nil
}
