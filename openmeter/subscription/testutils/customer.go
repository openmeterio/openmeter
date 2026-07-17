package subscriptiontestutils

import (
	"context"
	"testing"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/customer"
	customeradapter "github.com/openmeterio/openmeter/openmeter/customer/adapter"
	customerservice "github.com/openmeterio/openmeter/openmeter/customer/service"
	"github.com/openmeterio/openmeter/openmeter/subject"
	subjectadapter "github.com/openmeterio/openmeter/openmeter/subject/adapter"
	subjectservice "github.com/openmeterio/openmeter/openmeter/subject/service"
	"github.com/openmeterio/openmeter/openmeter/testutils"
	"github.com/openmeterio/openmeter/openmeter/watermill/eventbus"
	"github.com/openmeterio/openmeter/pkg/currencyx"
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

	subjectService := NewSubjectService(t, dbDeps)

	return &testCustomerRepo{
		Adapter:        repo,
		subjectService: subjectService,
	}
}

func NewSubjectService(t *testing.T, dbDeps *DBDeps) subject.Service {
	t.Helper()

	subjectAdapter, err := subjectadapter.New(dbDeps.DBClient)
	if err != nil {
		t.Fatalf("failed to create subject adapter: %v", err)
	}

	subjectService, err := subjectservice.New(subjectAdapter)
	if err != nil {
		t.Fatalf("failed to create subject service: %v", err)
	}

	return subjectService
}

func NewCustomerService(t *testing.T, dbDeps *DBDeps) customer.Service {
	t.Helper()

	customerAdapter := NewCustomerAdapter(t, dbDeps)

	customerService, err := customerservice.New(customerservice.Config{
		Adapter:   customerAdapter,
		Publisher: eventbus.NewMock(t),
	})
	if err != nil {
		t.Fatalf("failed to create customer service: %v", err)
	}

	return customerService
}

type testCustomerRepo struct {
	customer.Adapter
	subjectService subject.Service
}

func (a *testCustomerRepo) CreateExampleCustomer(t *testing.T) *customer.Customer {
	t.Helper()

	// Create the subjects first
	for _, subjectKey := range ExampleCreateCustomerInput.UsageAttribution.SubjectKeys {
		_, err := a.subjectService.Create(context.Background(), subject.CreateInput{
			Namespace: ExampleCreateCustomerInput.Namespace,
			Key:       subjectKey,
		})
		if err != nil {
			t.Fatalf("failed to create subject %s: %v", subjectKey, err)
		}
	}

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
	UsageAttribution: &customer.CustomerUsageAttribution{
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
