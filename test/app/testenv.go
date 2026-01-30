package app

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/app/config"
	"github.com/openmeterio/openmeter/openmeter/app"
	appadapter "github.com/openmeterio/openmeter/openmeter/app/adapter"
	appservice "github.com/openmeterio/openmeter/openmeter/app/service"
	appstripeadapter "github.com/openmeterio/openmeter/openmeter/app/stripe/adapter"
	appstripeservice "github.com/openmeterio/openmeter/openmeter/app/stripe/service"
	"github.com/openmeterio/openmeter/openmeter/billing"
	billingadapter "github.com/openmeterio/openmeter/openmeter/billing/adapter"
	billingservice "github.com/openmeterio/openmeter/openmeter/billing/service"
	"github.com/openmeterio/openmeter/openmeter/customer"
	customeradapter "github.com/openmeterio/openmeter/openmeter/customer/adapter"
	customerservice "github.com/openmeterio/openmeter/openmeter/customer/service"
	"github.com/openmeterio/openmeter/openmeter/ent/db"
	meteradapter "github.com/openmeterio/openmeter/openmeter/meter/mockadapter"
	registrybuilder "github.com/openmeterio/openmeter/openmeter/registry/builder"
	secretadapter "github.com/openmeterio/openmeter/openmeter/secret/adapter"
	secretservice "github.com/openmeterio/openmeter/openmeter/secret/service"
	streamingtestutils "github.com/openmeterio/openmeter/openmeter/streaming/testutils"
	"github.com/openmeterio/openmeter/openmeter/testutils"
	"github.com/openmeterio/openmeter/openmeter/watermill/eventbus"
	"github.com/openmeterio/openmeter/pkg/datetime"
	"github.com/openmeterio/openmeter/pkg/framework/lockr"
	"github.com/openmeterio/openmeter/tools/migrate"
)

const (
	PostgresURLTemplate = "postgres://postgres:postgres@%s:5432/postgres?sslmode=disable"
)

type TestEnv interface {
	Adapter() app.Adapter
	App() app.Service

	Close() error
}

var _ TestEnv = (*testEnv)(nil)

type testEnv struct {
	adapter app.Adapter
	app     app.Service

	closerFunc func() error
}

func (n testEnv) Close() error {
	return n.closerFunc()
}

func (n testEnv) Adapter() app.Adapter {
	return n.adapter
}

func (n testEnv) App() app.Service {
	return n.app
}

const (
	DefaultPostgresHost = "127.0.0.1"
)

func NewTestEnv(t *testing.T, ctx context.Context) (TestEnv, error) {
	logger := slog.Default().WithGroup("app")
	publisher := eventbus.NewMock(t)

	// Initialize postgres driver
	driver := testutils.InitPostgresDB(t)

	entClient := driver.EntDriver.Client()
	migrator, err := migrate.New(migrate.MigrateOptions{
		ConnectionString: driver.URL,
		Migrations:       migrate.OMMigrationsConfig,
		Logger:           testutils.NewLogger(t),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create migrator: %w", err)
	}
	if err := migrator.Up(); err != nil {
		t.Fatalf("failed to create schema: %v", err)
	}

	defer migrator.CloseOrLogError()

	// Customer
	customerAdapter, err := customeradapter.New(customeradapter.Config{
		Client: entClient,
		Logger: logger.WithGroup("postgres"),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create customer adapter: %w", err)
	}

	customerService, err := customerservice.New(customerservice.Config{
		Adapter:   customerAdapter,
		Publisher: publisher,
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
		Client: entClient,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create app adapter: %w", err)
	}

	appService, err := appservice.New(appservice.Config{
		Adapter:   appAdapter,
		Publisher: publisher,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create app service: %w", err)
	}

	billingService, err := InitBillingService(t, ctx, InitBillingServiceInput{
		DBClient:        entClient,
		CustomerService: customerService,
		AppService:      appService,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create billing service: %w", err)
	}

	webhookURLGenerator, err := appstripeservice.NewBaseURLWebhookURLGenerator("http://localhost:8888")
	if err != nil {
		return nil, fmt.Errorf("failed to create webhook url generator: %w", err)
	}

	// App Stripe
	appStripeAdapter, err := appstripeadapter.New(appstripeadapter.Config{
		Client:          entClient,
		AppService:      appService,
		CustomerService: customerService,
		SecretService:   secretService,
		Logger:          logger,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create appstripe adapter: %w", err)
	}

	_, err = appstripeservice.New(appstripeservice.Config{
		Adapter:             appStripeAdapter,
		AppService:          appService,
		SecretService:       secretService,
		Logger:              logger,
		BillingService:      billingService,
		Publisher:           publisher,
		WebhookURLGenerator: webhookURLGenerator,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create appstripe service: %w", err)
	}

	closerFunc := func() error {
		var errs error

		if err = entClient.Close(); err != nil {
			errs = errors.Join(errs, fmt.Errorf("failed to close ent driver: %w", err))
		}

		if err = driver.EntDriver.Close(); err != nil {
			errs = errors.Join(errs, fmt.Errorf("failed to close postgres driver: %w", err))
		}

		return errs
	}

	return &testEnv{
		adapter:    appAdapter,
		app:        appService,
		closerFunc: closerFunc,
	}, nil
}

type InitBillingServiceInput struct {
	DBClient        *db.Client
	CustomerService customer.Service
	AppService      app.Service
}

func (i InitBillingServiceInput) Validate() error {
	if i.DBClient == nil {
		return fmt.Errorf("db client is required")
	}

	if i.CustomerService == nil {
		return fmt.Errorf("customer service is required")
	}

	if i.AppService == nil {
		return fmt.Errorf("app service is required")
	}

	return nil
}

func InitBillingService(t *testing.T, ctx context.Context, in InitBillingServiceInput) (billing.Service, error) {
	if err := in.Validate(); err != nil {
		return nil, fmt.Errorf("failed to validate input: %w", err)
	}

	mockStreamingConnector := streamingtestutils.NewMockStreamingConnector(t)

	meterAdapter, err := meteradapter.New(nil)
	require.NoError(t, err)
	require.NotNil(t, meterAdapter)

	locker, err := lockr.NewLocker(&lockr.LockerConfig{
		Logger: slog.Default(),
	})
	require.NoError(t, err)

	// Entitlement
	entitlementRegistry := registrybuilder.GetEntitlementRegistry(registrybuilder.EntitlementOptions{
		DatabaseClient:     in.DBClient,
		StreamingConnector: mockStreamingConnector,
		Logger:             slog.Default(),
		MeterService:       meterAdapter,
		CustomerService:    in.CustomerService,
		Publisher:          eventbus.NewMock(t),
		EntitlementsConfiguration: config.EntitlementsConfiguration{
			GracePeriod: datetime.ISODurationString("P1D"),
		},
		Locker: locker,
	})

	// Feature
	featureService := entitlementRegistry.Feature

	// Billing
	billingAdapter, err := billingadapter.New(billingadapter.Config{
		Client: in.DBClient,
		Logger: slog.Default(),
	})
	require.NoError(t, err)

	return billingservice.New(billingservice.Config{
		Adapter:                      billingAdapter,
		CustomerService:              in.CustomerService,
		AppService:                   in.AppService,
		Logger:                       slog.Default(),
		FeatureService:               featureService,
		MeterService:                 meterAdapter,
		StreamingConnector:           mockStreamingConnector,
		Publisher:                    eventbus.NewMock(t),
		AdvancementStrategy:          billing.ForegroundAdvancementStrategy,
		MaxParallelQuantitySnapshots: 2,
	})
}
