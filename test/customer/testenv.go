package customer

import (
	"context"
	"fmt"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/app/config"
	"github.com/openmeterio/openmeter/openmeter/customer"
	customeradapter "github.com/openmeterio/openmeter/openmeter/customer/adapter"
	customerservice "github.com/openmeterio/openmeter/openmeter/customer/service"
	entcustomervalidator "github.com/openmeterio/openmeter/openmeter/entitlement/validators/customer"
	"github.com/openmeterio/openmeter/openmeter/meter"
	meteradapter "github.com/openmeterio/openmeter/openmeter/meter/mockadapter"
	registrybuilder "github.com/openmeterio/openmeter/openmeter/registry/builder"
	streamingtestutils "github.com/openmeterio/openmeter/openmeter/streaming/testutils"
	"github.com/openmeterio/openmeter/openmeter/subscription"
	subscriptiontestutils "github.com/openmeterio/openmeter/openmeter/subscription/testutils"
	subscriptioncustomer "github.com/openmeterio/openmeter/openmeter/subscription/validators/customer"
	"github.com/openmeterio/openmeter/openmeter/watermill/eventbus"
	"github.com/openmeterio/openmeter/pkg/datetime"
	"github.com/openmeterio/openmeter/pkg/framework/lockr"
)

const (
	PostgresURLTemplate = "postgres://postgres:postgres@%s:5432/postgres?sslmode=disable"
)

type TestEnv interface {
	Customer() customer.Service
	Subscription() subscription.Service

	Close() error
}

var _ TestEnv = (*testEnv)(nil)

type testEnv struct {
	customer     customer.Service
	subscription subscription.Service

	closerFunc func() error
}

func (n testEnv) Close() error {
	return n.closerFunc()
}

func (n testEnv) Customer() customer.Service {
	return n.customer
}

func (n testEnv) Subscription() subscription.Service {
	return n.subscription
}

const (
	DefaultPostgresHost = "127.0.0.1"
)

func NewTestEnv(t *testing.T, ctx context.Context) (TestEnv, error) {
	logger := slog.Default().WithGroup("customer")

	// Initialize postgres driver
	dbDeps := subscriptiontestutils.SetupDBDeps(t)

	// Streaming
	streamingConnector := streamingtestutils.NewMockStreamingConnector(t)

	// Meter
	meterAdapter, err := meteradapter.New([]meter.Meter{})
	if err != nil {
		return nil, fmt.Errorf("failed to create meter adapter: %w", err)
	}

	locker, err := lockr.NewLocker(&lockr.LockerConfig{
		Logger: logger,
	})
	require.NoError(t, err)

	// Entitlement
	entitlementRegistry := registrybuilder.GetEntitlementRegistry(registrybuilder.EntitlementOptions{
		DatabaseClient:     dbDeps.DBClient,
		StreamingConnector: streamingConnector,
		Logger:             logger,
		MeterService:       meterAdapter,
		Publisher:          eventbus.NewMock(t),
		EntitlementsConfiguration: config.EntitlementsConfiguration{
			GracePeriod: datetime.ISODurationString("P1D"),
		},
		Locker: locker,
	})

	// Customer
	customerAdapter, err := customeradapter.New(customeradapter.Config{
		Client: dbDeps.DBClient,
		Logger: logger.WithGroup("postgres"),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create customer adapter: %w", err)
	}

	customerService, err := customerservice.New(customerservice.Config{
		Adapter:   customerAdapter,
		Publisher: eventbus.NewMock(t),
	})
	if err != nil {
		return nil, err
	}

	entValidator, err := entcustomervalidator.NewValidator(customerService, entitlementRegistry.EntitlementRepo)
	if err != nil {
		return nil, err
	}

	customerService.RegisterRequestValidator(entValidator)

	subsDeps := subscriptiontestutils.NewService(t, dbDeps)

	subsCustValidator, err := subscriptioncustomer.NewValidator(subsDeps.SubscriptionService, customerService)
	if err != nil {
		return nil, err
	}

	customerService.RegisterRequestValidator(subsCustValidator)

	closerFunc := func() error {
		dbDeps.Cleanup(t)
		return nil
	}

	return &testEnv{
		customer:     customerService,
		closerFunc:   closerFunc,
		subscription: subsDeps.SubscriptionService,
	}, nil
}
