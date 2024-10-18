package subscriptiontestutils

import (
	"context"
	"testing"

	"github.com/openmeterio/openmeter/openmeter/customer"
	customeradapter "github.com/openmeterio/openmeter/openmeter/customer/adapter"
	customerentity "github.com/openmeterio/openmeter/openmeter/customer/entity"
	"github.com/openmeterio/openmeter/openmeter/testutils"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/timezone"
	"github.com/samber/lo"
)

func NewCustomerAdapter(t *testing.T, dbDeps *DBDeps) testCustomerRepo {
	t.Helper()

	logger := testutils.NewLogger(t)

	repo, err := customeradapter.New(customeradapter.Config{
		Client: dbDeps.dbClient,
		Logger: logger,
	})
	if err != nil {
		t.Fatalf("failed to create customer repo: %v", err)
	}

	return testCustomerRepo{
		repo,
	}
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
}

var ExampleCreateCustomerInput customerentity.CreateCustomerInput = customerentity.CreateCustomerInput{
	Namespace: ExampleNamespace,
	Customer:  ExampleCustomerEntity,
}
