package subscriptiontestutils

import (
	"context"
	"testing"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/customer"
	customeradapter "github.com/openmeterio/openmeter/openmeter/customer/adapter"
	customerentity "github.com/openmeterio/openmeter/openmeter/customer/entity"
	customerservice "github.com/openmeterio/openmeter/openmeter/customer/service"
	"github.com/openmeterio/openmeter/openmeter/testutils"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/timezone"
)

func NewCustomerAdapter(t *testing.T, dbDeps *DBDeps) *testCustomerRepo {
	t.Helper()

	logger := testutils.NewLogger(t)

	repo, err := customeradapter.New(customeradapter.Config{
		Client: dbDeps.dbClient,
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

	customerAdapter := NewCustomerAdapter(t, dbDeps)

	customerService, err := customerservice.New(customerservice.Config{
		Adapter: customerAdapter,
	})
	if err != nil {
		t.Fatalf("failed to create customer service: %v", err)
	}

	return customerService
}

type testCustomerRepo struct {
	customer.Adapter
}

func (a *testCustomerRepo) CreateExampleCustomer(t *testing.T) *customerentity.Customer {
	t.Helper()

	c, err := a.CreateCustomer(context.Background(), ExampleCreateCustomerInput)
	if err != nil {
		t.Fatalf("failed to create example customer: %v", err)
	}
	return c
}

var ExampleCustomerEntity customerentity.Customer = customerentity.Customer{
	Name:         "John Doe",
	PrimaryEmail: lo.ToPtr("mail@me.uk"),
	Currency:     lo.ToPtr(currencyx.Code("USD")),
	Timezone:     lo.ToPtr(timezone.Timezone("America/Los_Angeles")),
	UsageAttribution: customerentity.CustomerUsageAttribution{
		SubjectKeys: []string{"john-doe"},
	},
}

var ExampleCreateCustomerInput customerentity.CreateCustomerInput = customerentity.CreateCustomerInput{
	Namespace: ExampleNamespace,
	Customer:  ExampleCustomerEntity,
}
