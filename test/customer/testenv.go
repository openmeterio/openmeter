package customer

import (
	"context"
	"fmt"
	"log/slog"
	"testing"

	"github.com/openmeterio/openmeter/openmeter/customer"
	customeradapter "github.com/openmeterio/openmeter/openmeter/customer/adapter"
	customerservice "github.com/openmeterio/openmeter/openmeter/customer/service"
	"github.com/openmeterio/openmeter/openmeter/meter"
	meteradapter "github.com/openmeterio/openmeter/openmeter/meter/adapter"
	registrybuilder "github.com/openmeterio/openmeter/openmeter/registry/builder"
	streamingtestutils "github.com/openmeterio/openmeter/openmeter/streaming/testutils"
	"github.com/openmeterio/openmeter/openmeter/subscription"
	subscriptiontestutils "github.com/openmeterio/openmeter/openmeter/subscription/testutils"
	subscriptioncustomer "github.com/openmeterio/openmeter/openmeter/subscription/validators/customer"
	"github.com/openmeterio/openmeter/openmeter/watermill/eventbus"
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

	// Initialize customer adapter
	customerAdapter, err := customeradapter.New(customeradapter.Config{
		Client: dbDeps.DBClient,
		Logger: logger.WithGroup("postgres"),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create customer adapter: %w", err)
	}

	streamingConnector := streamingtestutils.NewMockStreamingConnector(t)

	meterAdapter, err := meteradapter.New([]meter.Meter{})
	if err != nil {
		return nil, fmt.Errorf("failed to create meter adapter: %w", err)
	}

	entitlementRegistry := registrybuilder.GetEntitlementRegistry(registrybuilder.EntitlementOptions{
		DatabaseClient:     dbDeps.DBClient,
		StreamingConnector: streamingConnector,
		Logger:             logger,
		MeterService:       meterAdapter,
		Publisher:          eventbus.NewMock(t),
	})

	customerService, err := customerservice.New(customerservice.Config{
		Adapter:              customerAdapter,
		EntitlementConnector: entitlementRegistry.Entitlement,
	})
	if err != nil {
		return nil, err
	}

	subsServices, _ := subscriptiontestutils.NewService(t, dbDeps)

	subsCustValidator, err := subscriptioncustomer.NewValidator(subsServices.Service)
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
		subscription: subsServices.Service,
	}, nil
}
