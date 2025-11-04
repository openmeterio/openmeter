package customer

import (
	"context"
	"fmt"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/trace/noop"

	"github.com/openmeterio/openmeter/app/config"
	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/customer"
	customeradapter "github.com/openmeterio/openmeter/openmeter/customer/adapter"
	customerservice "github.com/openmeterio/openmeter/openmeter/customer/service"
	customerhooks "github.com/openmeterio/openmeter/openmeter/customer/service/hooks"
	"github.com/openmeterio/openmeter/openmeter/entitlement"
	entcustomervalidator "github.com/openmeterio/openmeter/openmeter/entitlement/validators/customer"
	"github.com/openmeterio/openmeter/openmeter/meter"
	meteradapter "github.com/openmeterio/openmeter/openmeter/meter/mockadapter"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/feature"
	registrybuilder "github.com/openmeterio/openmeter/openmeter/registry/builder"
	streamingtestutils "github.com/openmeterio/openmeter/openmeter/streaming/testutils"
	subject "github.com/openmeterio/openmeter/openmeter/subject"
	subjectadapter "github.com/openmeterio/openmeter/openmeter/subject/adapter"
	subjectservice "github.com/openmeterio/openmeter/openmeter/subject/service"
	subjecthooks "github.com/openmeterio/openmeter/openmeter/subject/service/hooks"
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
	Entitlement() entitlement.Service
	Feature() feature.FeatureConnector
	Subject() subject.Service

	Close() error
}

var _ TestEnv = (*testEnv)(nil)

type testEnv struct {
	customer     customer.Service
	subscription subscription.Service
	entitlement  entitlement.Service
	feature      feature.FeatureConnector
	subject      subject.Service

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

func (n testEnv) Entitlement() entitlement.Service {
	return n.entitlement
}

func (n testEnv) Feature() feature.FeatureConnector {
	return n.feature
}

func (n testEnv) Subject() subject.Service {
	return n.subject
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

	entValidator, err := entcustomervalidator.NewValidator(entitlementRegistry.EntitlementRepo)
	if err != nil {
		return nil, err
	}

	customerService.RegisterRequestValidator(entValidator)

	// Subject
	subjectAdapter, err := subjectadapter.New(dbDeps.DBClient)
	if err != nil {
		return nil, err
	}

	subjectService, err := subjectservice.New(subjectAdapter)
	if err != nil {
		return nil, err
	}

	subjectCustomerHook, err := customerhooks.NewSubjectCustomerHook(customerhooks.SubjectCustomerHookConfig{
		Customer:         customerService,
		CustomerOverride: noopCustomerOverrideService{},
		Logger:           logger,
		Tracer:           noop.NewTracerProvider().Tracer("test_env"),
	})
	if err != nil {
		return nil, err
	}

	subjectService.RegisterHooks(subjectCustomerHook)

	customerSubjectHook, err := subjecthooks.NewCustomerSubjectHook(subjecthooks.CustomerSubjectHookConfig{
		Subject: subjectService,
		Logger:  logger,
		Tracer:  noop.NewTracerProvider().Tracer("test_env"),
	})
	if err != nil {
		return nil, err
	}

	customerService.RegisterHooks(customerSubjectHook)

	// Subscription
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
		entitlement:  entitlementRegistry.Entitlement,
		feature:      entitlementRegistry.Feature,
		subscription: subsDeps.SubscriptionService,
		subject:      subjectService,
	}, nil
}

type noopCustomerOverrideService struct {
	billing.CustomerOverrideService
}
