package subscriptiontestutils

import (
	"context"
	"testing"
	"time"

	"github.com/oklog/ulid/v2"
	"github.com/samber/lo"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/app/config"
	"github.com/openmeterio/openmeter/openmeter/customer"
	customeradapter "github.com/openmeterio/openmeter/openmeter/customer/adapter"
	customerservice "github.com/openmeterio/openmeter/openmeter/customer/service"
	"github.com/openmeterio/openmeter/openmeter/meter"
	meteradapter "github.com/openmeterio/openmeter/openmeter/meter/mockadapter"
	registrybuilder "github.com/openmeterio/openmeter/openmeter/registry/builder"
	streamingtestutils "github.com/openmeterio/openmeter/openmeter/streaming/testutils"
	"github.com/openmeterio/openmeter/openmeter/testutils"
	"github.com/openmeterio/openmeter/openmeter/watermill/eventbus"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/datetime"
	"github.com/openmeterio/openmeter/pkg/framework/lockr"
	"github.com/openmeterio/openmeter/pkg/models"
)

func NewCustomerAdapter(t *testing.T, dbDeps *DBDeps) *testCustomerRepo {
	t.Helper()

	logger := testutils.NewLogger(t)

	repo, err := customeradapter.New(customeradapter.Config{
		Client: dbDeps.DBClient,
		Logger: logger,
	})
	if err != nil {
		t.Fatalf("failed to create customer repo: %v", err)
	}

	return &testCustomerRepo{
		repo,
	}
}

func NewCustomerService(t *testing.T, dbDeps *DBDeps) customer.Service {
	t.Helper()

	meterAdapter, err := meteradapter.New([]meter.Meter{{
		ManagedResource: models.ManagedResource{
			ID: ulid.Make().String(),
			NamespacedModel: models.NamespacedModel{
				Namespace: ExampleNamespace,
			},
			ManagedModel: models.ManagedModel{
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			},
			Name: "Meter 1",
		},
		Key:           ExampleFeatureMeterSlug,
		Aggregation:   meter.MeterAggregationSum,
		EventType:     "test",
		ValueProperty: lo.ToPtr("$.value"),
	}})
	if err != nil {
		t.Fatalf("failed to create meter adapter: %v", err)
	}

	locker, err := lockr.NewLocker(&lockr.LockerConfig{
		Logger: testutils.NewLogger(t),
	})
	require.NoError(t, err)

	entitlementRegistry := registrybuilder.GetEntitlementRegistry(registrybuilder.EntitlementOptions{
		DatabaseClient:     dbDeps.DBClient,
		StreamingConnector: streamingtestutils.NewMockStreamingConnector(t),
		Logger:             testutils.NewLogger(t),
		MeterService:       meterAdapter,
		Publisher:          eventbus.NewMock(t),
		EntitlementsConfiguration: config.EntitlementsConfiguration{
			GracePeriod: datetime.ISODurationString("P1D"),
		},
		Locker: locker,
	})

	customerAdapter := NewCustomerAdapter(t, dbDeps)

	customerService, err := customerservice.New(customerservice.Config{
		Adapter:              customerAdapter,
		EntitlementConnector: entitlementRegistry.Entitlement,
		Publisher:            eventbus.NewMock(t),
	})
	if err != nil {
		t.Fatalf("failed to create customer service: %v", err)
	}

	return customerService
}

type testCustomerRepo struct {
	customer.Adapter
}

func (a *testCustomerRepo) CreateExampleCustomer(t *testing.T) *customer.Customer {
	t.Helper()

	c, err := a.CreateCustomer(context.Background(), ExampleCreateCustomerInput)
	if err != nil {
		t.Fatalf("failed to create example customer: %v", err)
	}
	return c
}

var ExampleCustomerEntity customer.Customer = customer.Customer{
	ManagedResource: models.ManagedResource{
		Name: "John Doe",
	},
	PrimaryEmail: lo.ToPtr("mail@me.uk"),
	Currency:     lo.ToPtr(currencyx.Code("USD")),
	UsageAttribution: customer.CustomerUsageAttribution{
		SubjectKeys: []string{"john-doe"},
	},
}

var ExampleCreateCustomerInput customer.CreateCustomerInput = customer.CreateCustomerInput{
	Namespace: ExampleNamespace,
	CustomerMutate: customer.CustomerMutate{
		Name:             ExampleCustomerEntity.Name,
		PrimaryEmail:     ExampleCustomerEntity.PrimaryEmail,
		Currency:         ExampleCustomerEntity.Currency,
		UsageAttribution: ExampleCustomerEntity.UsageAttribution,
	},
}
