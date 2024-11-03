package customer

import (
	"context"
	"testing"

	"github.com/oklog/ulid/v2"
	"github.com/samber/lo"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/api"
	customerentity "github.com/openmeterio/openmeter/openmeter/customer/entity"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/pagination"
	"github.com/openmeterio/openmeter/pkg/sortx"
	"github.com/openmeterio/openmeter/pkg/timezone"
)

var (
	TestName               = "Test Customer"
	TestPrimaryEmail       = "test@openmeter.io"
	TestCurrency           = currencyx.Code("USD")
	TestTimezone           = timezone.Timezone("America/Los_Angeles")
	TestAddressCountry     = models.CountryCode("US")
	TestAddressCity        = "San Francisco"
	TestAddressState       = "CA"
	TestAddressPostalCode  = "94105"
	TestAddressLine1       = "123 Main St"
	TestAddressLine2       = "Apt 1"
	TestAddressPhoneNumber = "123-456-7890"
	TestAddress            = models.Address{
		Country:     &TestAddressCountry,
		City:        &TestAddressCity,
		Line1:       &TestAddressLine1,
		Line2:       &TestAddressLine2,
		PostalCode:  &TestAddressPostalCode,
		PhoneNumber: &TestAddressPhoneNumber,
	}
	TestSubjectKeys = []string{"subject-0"}
)

type CustomerHandlerTestSuite struct {
	Env TestEnv

	namespace string
}

// setupNamespace can be used to set up an independent namespace for testing, it contains a single
// feature and rule with a channel. For more complex scenarios, additional setup might be required.
func (s *CustomerHandlerTestSuite) setupNamespace(t *testing.T) {
	t.Helper()

	s.namespace = ulid.Make().String()
}

// TestCreate tests the creation of a customer
func (s *CustomerHandlerTestSuite) TestCreate(ctx context.Context, t *testing.T) {
	s.setupNamespace(t)

	service := s.Env.Customer()

	// Create a createdCustomer
	createdCustomer, err := service.CreateCustomer(ctx, customerentity.CreateCustomerInput{
		Namespace: s.namespace,
		Customer: customerentity.Customer{
			ManagedResource: models.NewManagedResource(models.ManagedResourceInput{
				Name: TestName,
			}),
			PrimaryEmail:   &TestPrimaryEmail,
			Currency:       &TestCurrency,
			Timezone:       &TestTimezone,
			BillingAddress: &TestAddress,
			UsageAttribution: customerentity.CustomerUsageAttribution{
				SubjectKeys: TestSubjectKeys,
			},
		},
	})

	require.NoError(t, err, "Creating customer must not return error")

	require.NotNil(t, createdCustomer, "Customer must not be nil")
	require.Equal(t, s.namespace, createdCustomer.Namespace, "Customer namespace must match")
	require.NotNil(t, createdCustomer.ID, "Customer ID must not be nil")
	require.Equal(t, TestName, createdCustomer.Name, "Customer name must match")
	require.Equal(t, &TestPrimaryEmail, createdCustomer.PrimaryEmail, "Customer primary email must match")
	require.Equal(t, &TestCurrency, createdCustomer.Currency, "Customer currency must match")
	require.Equal(t, &TestTimezone, createdCustomer.Timezone, "Customer timezone must match")
	require.Equal(t, &TestAddressCountry, createdCustomer.BillingAddress.Country, "Customer billing address country must match")
	require.Equal(t, &TestAddressCity, createdCustomer.BillingAddress.City, "Customer billing address city must match")
	require.Equal(t, &TestAddressLine1, createdCustomer.BillingAddress.Line1, "Customer billing address line1 must match")
	require.Equal(t, &TestAddressLine2, createdCustomer.BillingAddress.Line2, "Customer billing address line2 must match")
	require.Equal(t, &TestAddressPostalCode, createdCustomer.BillingAddress.PostalCode, "Customer billing address postal code must match")
	require.Equal(t, &TestAddressPhoneNumber, createdCustomer.BillingAddress.PhoneNumber, "Customer billing address phone number must match")
	require.Equal(t, TestSubjectKeys, createdCustomer.UsageAttribution.SubjectKeys, "Customer usage attribution subject keys must match")

	// Test conflicts
	_, err = service.CreateCustomer(ctx, customerentity.CreateCustomerInput{
		Namespace: s.namespace,
		Customer: customerentity.Customer{
			ManagedResource: models.NewManagedResource(models.ManagedResourceInput{
				Name: TestName,
			}),
			UsageAttribution: customerentity.CustomerUsageAttribution{
				SubjectKeys: TestSubjectKeys,
			},
		},
	})

	require.ErrorAs(
		t,
		err,
		&customerentity.SubjectKeyConflictError{Namespace: s.namespace, SubjectKeys: TestSubjectKeys},
		"Creating a customer with same subject keys must return conflict error",
	)
}

// TestUpdate tests the updating of a customer
func (s *CustomerHandlerTestSuite) TestUpdate(ctx context.Context, t *testing.T) {
	s.setupNamespace(t)

	service := s.Env.Customer()

	// Create a customer with mandatory fields
	originalCustomer, err := service.CreateCustomer(ctx, customerentity.CreateCustomerInput{
		Namespace: s.namespace,
		Customer: customerentity.Customer{
			ManagedResource: models.NewManagedResource(models.ManagedResourceInput{
				Name: TestName,
			}),
			UsageAttribution: customerentity.CustomerUsageAttribution{
				SubjectKeys: TestSubjectKeys,
			},
		},
	})

	require.NoError(t, err, "Creating customer must not return error")
	require.NotNil(t, originalCustomer, "Customer must not be nil")
	require.Equal(t, TestName, originalCustomer.Name, "Customer name must match")
	require.Equal(t, TestSubjectKeys, originalCustomer.UsageAttribution.SubjectKeys, "Customer usage attribution subject keys must match")

	newName := "New Name"
	newSubjectKeys := []string{"subject-1"}

	// Update the customer with new fields
	updatedCustomer, err := service.UpdateCustomer(ctx, customerentity.UpdateCustomerInput{
		Namespace: s.namespace,
		ID:        originalCustomer.ID,
		Customer: customerentity.Customer{
			ManagedResource: models.NewManagedResource(models.ManagedResourceInput{
				Name: newName,
			}),
			PrimaryEmail:   &TestPrimaryEmail,
			Currency:       &TestCurrency,
			Timezone:       &TestTimezone,
			BillingAddress: &TestAddress,
			UsageAttribution: customerentity.CustomerUsageAttribution{
				SubjectKeys: newSubjectKeys,
			},
		},
	})

	require.NoError(t, err, "Updating customer must not return error")
	require.NotNil(t, updatedCustomer, "Customer must not be nil")
	require.Equal(t, s.namespace, updatedCustomer.Namespace, "Customer namespace must match")
	require.Equal(t, originalCustomer.ID, updatedCustomer.ID, "Customer ID must match")
	require.Equal(t, newName, updatedCustomer.Name, "Customer name must match")
	require.Equal(t, newSubjectKeys, updatedCustomer.UsageAttribution.SubjectKeys, "Customer usage attribution subject keys must match")
	require.Equal(t, &TestPrimaryEmail, updatedCustomer.PrimaryEmail, "Customer primary email must match")
	require.Equal(t, &TestCurrency, updatedCustomer.Currency, "Customer currency must match")
	require.Equal(t, &TestTimezone, updatedCustomer.Timezone, "Customer timezone must match")
	require.Equal(t, &TestAddressCountry, updatedCustomer.BillingAddress.Country, "Customer billing address country must match")
	require.Equal(t, &TestAddressCity, updatedCustomer.BillingAddress.City, "Customer billing address city must match")
	require.Equal(t, &TestAddressLine1, updatedCustomer.BillingAddress.Line1, "Customer billing address line1 must match")
	require.Equal(t, &TestAddressLine2, updatedCustomer.BillingAddress.Line2, "Customer billing address line2 must match")
	require.Equal(t, &TestAddressPostalCode, updatedCustomer.BillingAddress.PostalCode, "Customer billing address postal code must match")
	require.Equal(t, &TestAddressPhoneNumber, updatedCustomer.BillingAddress.PhoneNumber, "Customer billing address phone number must match")
}

// TestList tests the listing of customers
func (s *CustomerHandlerTestSuite) TestList(ctx context.Context, t *testing.T) {
	s.setupNamespace(t)

	service := s.Env.Customer()

	// Create a customer 1
	createCustomer1, err := service.CreateCustomer(ctx, customerentity.CreateCustomerInput{
		Namespace: s.namespace,
		Customer: customerentity.Customer{
			ManagedResource: models.NewManagedResource(models.ManagedResourceInput{
				Name: "Customer 1",
			}),
			UsageAttribution: customerentity.CustomerUsageAttribution{
				SubjectKeys: []string{"subject-1"},
			},
			PrimaryEmail: lo.ToPtr("customer-1@test.com"),
		},
	})

	require.NoError(t, err, "Creating customer must not return error")

	// Create a customer 2
	createCustomer2, err := service.CreateCustomer(ctx, customerentity.CreateCustomerInput{
		Namespace: s.namespace,
		Customer: customerentity.Customer{
			ManagedResource: models.NewManagedResource(models.ManagedResourceInput{
				Name: "Customer 2",
			}),
			UsageAttribution: customerentity.CustomerUsageAttribution{
				SubjectKeys: []string{"subject-2"},
			},
			PrimaryEmail: lo.ToPtr("customer-2@test.com"),
		},
	})

	require.NoError(t, err, "Creating customer must not return error")

	// Create a customer 3 in a different namespace
	differentNamespace := ulid.Make().String()

	_, err = service.CreateCustomer(ctx, customerentity.CreateCustomerInput{
		Namespace: differentNamespace,
		Customer: customerentity.Customer{
			ManagedResource: models.NewManagedResource(models.ManagedResourceInput{
				Name: "Customer 3",
			}),
			UsageAttribution: customerentity.CustomerUsageAttribution{
				SubjectKeys: []string{"subject-3"},
			},
		},
	})

	require.NoError(t, err, "Creating customer must not return error")

	page := pagination.Page{PageNumber: 1, PageSize: 10}

	// List customers
	list, err := service.ListCustomers(ctx, customerentity.ListCustomersInput{
		Namespace: s.namespace,
		Page:      page,
	})

	require.NoError(t, err, "Listing customers must not return error")
	require.Equal(t, 2, list.TotalCount, "Customers total count must be 1")
	require.Equal(t, 1, list.Page.PageNumber, "Customers page must be 0")
	require.Len(t, list.Items, 2, "Customers must have a single item")
	require.Equal(t, s.namespace, list.Items[0].Namespace, "Customer namespace must match")
	require.Equal(t, createCustomer1.ID, list.Items[0].ID, "Customer ID must match")
	require.Equal(t, "Customer 1", list.Items[0].Name, "Customer name must match")
	require.Equal(t, []string{"subject-1"}, list.Items[0].UsageAttribution.SubjectKeys, "Customer usage attribution subject keys must match")
	require.Equal(t, s.namespace, list.Items[1].Namespace, "Customer namespace must match")
	require.Equal(t, createCustomer2.ID, list.Items[1].ID, "Customer ID must match")
	require.Equal(t, "Customer 2", list.Items[1].Name, "Customer name must match")
	require.Equal(t, []string{"subject-2"}, list.Items[1].UsageAttribution.SubjectKeys, "Customer usage attribution subject keys must match")

	// List customers with name filter
	list, err = service.ListCustomers(ctx, customerentity.ListCustomersInput{
		Namespace: s.namespace,
		Page:      page,
		Name:      &createCustomer2.Name,
	})

	require.NoError(t, err, "Listing customers with name filter must not return error")
	require.Equal(t, 1, list.TotalCount, "Customers total count must be 1")
	require.Equal(t, createCustomer2.ID, list.Items[0].ID, "Customer ID must match")

	// List customers with partial name filter
	list, err = service.ListCustomers(ctx, customerentity.ListCustomersInput{
		Namespace: s.namespace,
		Page:      page,
		Name:      lo.ToPtr("2"),
	})

	require.NoError(t, err, "Listing customers with partial name filter must not return error")
	require.Equal(t, 1, list.TotalCount, "Customers total count must be 1")
	require.Equal(t, createCustomer2.ID, list.Items[0].ID, "Customer ID must match")

	// List customers with primary email filter
	list, err = service.ListCustomers(ctx, customerentity.ListCustomersInput{
		Namespace:    s.namespace,
		Page:         page,
		PrimaryEmail: createCustomer2.PrimaryEmail,
	})

	require.NoError(t, err, "Listing customers with primary email filter must not return error")
	require.Equal(t, 1, list.TotalCount, "Customers total count must be 1")
	require.Equal(t, createCustomer2.ID, list.Items[0].ID, "Customer ID must match")

	// Order by name descending
	list, err = service.ListCustomers(ctx, customerentity.ListCustomersInput{
		Namespace: s.namespace,
		Page:      page,
		OrderBy:   api.CustomerOrderByName,
		Order:     sortx.Order(api.SortOrderDESC),
	})

	require.NoError(t, err, "Listing customers with order by name must not return error")
	require.Equal(t, 2, list.TotalCount, "Customers total count must be 1")
	require.Equal(t, 1, list.Page.PageNumber, "Customers page must be 0")
	require.Equal(t, createCustomer2.ID, list.Items[0].ID, "Customer 2 must be first in order")
	require.Equal(t, createCustomer1.ID, list.Items[1].ID, "Customer 1 must be second in order")
}

// TestGet tests the getting of a customer by ID
func (s *CustomerHandlerTestSuite) TestGet(ctx context.Context, t *testing.T) {
	s.setupNamespace(t)

	service := s.Env.Customer()

	// Create a customer
	originalCustomer, err := service.CreateCustomer(ctx, customerentity.CreateCustomerInput{
		Namespace: s.namespace,
		Customer: customerentity.Customer{
			ManagedResource: models.NewManagedResource(models.ManagedResourceInput{
				Name: TestName,
			}),
			UsageAttribution: customerentity.CustomerUsageAttribution{
				SubjectKeys: TestSubjectKeys,
			},
		},
	})

	require.NoError(t, err, "Creating customer must not return error")
	require.NotNil(t, originalCustomer, "Customer must not be nil")

	// Get the customer
	customer, err := service.GetCustomer(ctx, customerentity.GetCustomerInput{
		Namespace: s.namespace,
		ID:        originalCustomer.ID,
	})

	require.NoError(t, err, "Fetching customer must not return error")
	require.NotNil(t, customer, "Customer must not be nil")
	require.Equal(t, s.namespace, customer.Namespace, "Customer namespace must match")
	require.NotNil(t, customer.ID, "Customer ID must not be nil")
	require.Equal(t, TestName, customer.Name, "Customer name must match")
	require.Equal(t, TestSubjectKeys, customer.UsageAttribution.SubjectKeys, "Customer usage attribution subject keys must match")
}

// TestDelete tests the deletion of a customer
func (s *CustomerHandlerTestSuite) TestDelete(ctx context.Context, t *testing.T) {
	s.setupNamespace(t)

	service := s.Env.Customer()

	// Create a customer
	originalCustomer, err := service.CreateCustomer(ctx, customerentity.CreateCustomerInput{
		Namespace: s.namespace,
		Customer: customerentity.Customer{
			ManagedResource: models.NewManagedResource(models.ManagedResourceInput{
				Name: TestName,
			}),
			UsageAttribution: customerentity.CustomerUsageAttribution{
				SubjectKeys: TestSubjectKeys,
			},
		},
	})

	require.NoError(t, err, "Creating customer must not return error")
	require.NotNil(t, originalCustomer, "Customer must not be nil")

	customerId := customerentity.CustomerID{
		Namespace: s.namespace,
		ID:        originalCustomer.ID,
	}

	// Delete the customer
	err = service.DeleteCustomer(ctx, customerentity.DeleteCustomerInput(customerId))

	require.NoError(t, err, "Deleting customer must not return error")

	// Get the customer
	getCustomer, err := service.GetCustomer(ctx, customerentity.GetCustomerInput(customerId))

	require.NoError(t, err, "Getting a deleted customer must not return error")
	require.NotNil(t, getCustomer.DeletedAt, "DeletedAt must not be nil")

	// Delete the customer again should return not found error
	err = service.DeleteCustomer(ctx, customerentity.DeleteCustomerInput(customerId))

	// TODO: it is a wrapped error, we need to unwrap it, instead we are checking the error message for now
	// require.ErrorAs(t, err, customer.NotFoundError{CustomerID: customerId}, "Deleting customer again must return not found error")
	require.ErrorContains(t, err, "not found", "Deleting customer again must return not found error")
}
