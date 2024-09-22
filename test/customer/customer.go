package customer

import (
	"context"
	"testing"

	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/pkg/models"
)

var (
	TestName               = "Test Customer"
	TestPrimaryEmail       = "test@openmeter.io"
	TestCurrency           = models.CurrencyCode("USD")
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
	TestSubjectKeys       = []string{"subject-0"}
	TestTaxProvider       = models.TaxProviderOpenMeterSandbox
	TestInvoicingProvider = models.InvoicingProviderOpenMeterSandbox
	TestPaymentProvider   = models.PaymentProviderOpenMeterSandbox
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
	createdCustomer, err := service.CreateCustomer(ctx, customer.CreateCustomerInput{
		Namespace: s.namespace,
		Customer: customer.Customer{
			Name:           TestName,
			PrimaryEmail:   &TestPrimaryEmail,
			Currency:       &TestCurrency,
			BillingAddress: &TestAddress,
			UsageAttribution: customer.CustomerUsageAttribution{
				SubjectKeys: TestSubjectKeys,
			},
			TaxProvider:       &TestTaxProvider,
			InvoicingProvider: &TestInvoicingProvider,
			PaymentProvider:   &TestPaymentProvider,
		},
	})

	require.NoError(t, err, "Creating customer must not return error")

	require.NotNil(t, createdCustomer, "Customer must not be nil")
	require.Equal(t, s.namespace, createdCustomer.Namespace, "Customer namespace must match")
	require.NotNil(t, createdCustomer.ID, "Customer ID must not be nil")
	require.Equal(t, TestName, createdCustomer.Name, "Customer name must match")
	require.Equal(t, &TestPrimaryEmail, createdCustomer.PrimaryEmail, "Customer primary email must match")
	require.Equal(t, &TestCurrency, createdCustomer.Currency, "Customer currency must match")
	require.Equal(t, &TestAddressCountry, createdCustomer.BillingAddress.Country, "Customer billing address country must match")
	require.Equal(t, &TestAddressCity, createdCustomer.BillingAddress.City, "Customer billing address city must match")
	require.Equal(t, &TestAddressLine1, createdCustomer.BillingAddress.Line1, "Customer billing address line1 must match")
	require.Equal(t, &TestAddressLine2, createdCustomer.BillingAddress.Line2, "Customer billing address line2 must match")
	require.Equal(t, &TestAddressPostalCode, createdCustomer.BillingAddress.PostalCode, "Customer billing address postal code must match")
	require.Equal(t, &TestAddressPhoneNumber, createdCustomer.BillingAddress.PhoneNumber, "Customer billing address phone number must match")
	require.Equal(t, TestSubjectKeys, createdCustomer.UsageAttribution.SubjectKeys, "Customer usage attribution subject keys must match")
	require.Equal(t, &TestTaxProvider, createdCustomer.TaxProvider, "Customer tax provider must match")
	require.Equal(t, &TestInvoicingProvider, createdCustomer.InvoicingProvider, "Customer invoicing provider must match")
	require.Equal(t, &TestPaymentProvider, createdCustomer.PaymentProvider, "Customer payment provider must match")

	// Test conflicts
	_, err = service.CreateCustomer(ctx, customer.CreateCustomerInput{
		Namespace: s.namespace,
		Customer: customer.Customer{
			Name: TestName,
			UsageAttribution: customer.CustomerUsageAttribution{
				SubjectKeys: TestSubjectKeys,
			},
		},
	})

	require.ErrorAs(
		t,
		err,
		&customer.SubjectKeyConflictError{Namespace: s.namespace, SubjectKeys: TestSubjectKeys},
		"Creating a customer with same subject keys must return conflict error",
	)
}

// TestUpdate tests the updating of a customer
func (s *CustomerHandlerTestSuite) TestUpdate(ctx context.Context, t *testing.T) {
	s.setupNamespace(t)

	service := s.Env.Customer()

	// Create a customer with mandatory fields
	originalCustomer, err := service.CreateCustomer(ctx, customer.CreateCustomerInput{
		Namespace: s.namespace,
		Customer: customer.Customer{
			Name: TestName,
			UsageAttribution: customer.CustomerUsageAttribution{
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
	updatedCustomer, err := service.UpdateCustomer(ctx, customer.UpdateCustomerInput{
		Namespace: s.namespace,
		ID:        originalCustomer.ID,
		Customer: customer.Customer{
			Name:           newName,
			PrimaryEmail:   &TestPrimaryEmail,
			Currency:       &TestCurrency,
			BillingAddress: &TestAddress,
			UsageAttribution: customer.CustomerUsageAttribution{
				SubjectKeys: newSubjectKeys,
			},
			TaxProvider:       &TestTaxProvider,
			InvoicingProvider: &TestInvoicingProvider,
			PaymentProvider:   &TestPaymentProvider,
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
	require.Equal(t, &TestAddressCountry, updatedCustomer.BillingAddress.Country, "Customer billing address country must match")
	require.Equal(t, &TestAddressCity, updatedCustomer.BillingAddress.City, "Customer billing address city must match")
	require.Equal(t, &TestAddressLine1, updatedCustomer.BillingAddress.Line1, "Customer billing address line1 must match")
	require.Equal(t, &TestAddressLine2, updatedCustomer.BillingAddress.Line2, "Customer billing address line2 must match")
	require.Equal(t, &TestAddressPostalCode, updatedCustomer.BillingAddress.PostalCode, "Customer billing address postal code must match")
	require.Equal(t, &TestAddressPhoneNumber, updatedCustomer.BillingAddress.PhoneNumber, "Customer billing address phone number must match")
	require.Equal(t, &TestTaxProvider, updatedCustomer.TaxProvider, "Customer tax provider must match")
	require.Equal(t, &TestInvoicingProvider, updatedCustomer.InvoicingProvider, "Customer invoicing provider must match")
	require.Equal(t, &TestPaymentProvider, updatedCustomer.PaymentProvider, "Customer payment provider must match")
}

// TestList tests the listing of customers
func (s *CustomerHandlerTestSuite) TestList(ctx context.Context, t *testing.T) {
	s.setupNamespace(t)

	service := s.Env.Customer()

	// Create a customer 1
	_, err := service.CreateCustomer(ctx, customer.CreateCustomerInput{
		Namespace: s.namespace,
		Customer: customer.Customer{
			Name: "Customer 1",
			UsageAttribution: customer.CustomerUsageAttribution{
				SubjectKeys: []string{"subject-1"},
			},
		},
	})

	require.NoError(t, err, "Creating customer must not return error")

	// Create a customer 2
	_, err = service.CreateCustomer(ctx, customer.CreateCustomerInput{
		Namespace: s.namespace,
		Customer: customer.Customer{
			Name: "Customer 2",
			UsageAttribution: customer.CustomerUsageAttribution{
				SubjectKeys: []string{"subject-2"},
			},
		},
	})

	require.NoError(t, err, "Creating customer must not return error")

	// Create a customer 3 in a different namespace
	differentNamespace := ulid.Make().String()

	_, err = service.CreateCustomer(ctx, customer.CreateCustomerInput{
		Namespace: differentNamespace,
		Customer: customer.Customer{
			Name: "Customer 3",
			UsageAttribution: customer.CustomerUsageAttribution{
				SubjectKeys: []string{"subject-3"},
			},
		},
	})

	require.NoError(t, err, "Creating customer must not return error")

	// List customers
	list, err := service.ListCustomers(ctx, customer.ListCustomersInput{
		Namespace: s.namespace,
	})

	require.NoError(t, err, "Listing customers must not return error")
	require.NotNil(t, list, "Customers must not be nil")
	require.Equal(t, 2, list.TotalCount, "Customers total count must be 1")
	require.Equal(t, 1, list.Page, "Customers page must be 1")
	require.Len(t, list.Items, 2, "Customers must have a single item")
	require.Equal(t, s.namespace, list.Items[0].Namespace, "Customer namespace must match")
	require.NotNil(t, list.Items[0].ID, "Customer ID must not be nil")
	require.Equal(t, "Customer 1", list.Items[0].Name, "Customer name must match")
	require.Equal(t, []string{"subject-1"}, list.Items[0].UsageAttribution.SubjectKeys, "Customer usage attribution subject keys must match")
	require.Equal(t, s.namespace, list.Items[1].Namespace, "Customer namespace must match")
	require.NotNil(t, list.Items[1].ID, "Customer ID must not be nil")
	require.Equal(t, "Customer 2", list.Items[1].Name, "Customer name must match")
	require.Equal(t, []string{"subject-2"}, list.Items[1].UsageAttribution.SubjectKeys, "Customer usage attribution subject keys must match")
}

// TestGet tests the getting of a customer by ID
func (s *CustomerHandlerTestSuite) TestGet(ctx context.Context, t *testing.T) {
	s.setupNamespace(t)

	service := s.Env.Customer()

	// Create a customer
	originalCustomer, err := service.CreateCustomer(ctx, customer.CreateCustomerInput{
		Namespace: s.namespace,
		Customer: customer.Customer{
			Name: TestName,
			UsageAttribution: customer.CustomerUsageAttribution{
				SubjectKeys: TestSubjectKeys,
			},
		},
	})

	require.NoError(t, err, "Creating customer must not return error")
	require.NotNil(t, originalCustomer, "Customer must not be nil")

	// Get the customer
	customer, err := service.GetCustomer(ctx, customer.GetCustomerInput{
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
	originalCustomer, err := service.CreateCustomer(ctx, customer.CreateCustomerInput{
		Namespace: s.namespace,
		Customer: customer.Customer{
			Name: TestName,
			UsageAttribution: customer.CustomerUsageAttribution{
				SubjectKeys: TestSubjectKeys,
			},
		},
	})

	require.NoError(t, err, "Creating customer must not return error")
	require.NotNil(t, originalCustomer, "Customer must not be nil")

	customerId := customer.CustomerID{
		Namespace: s.namespace,
		ID:        originalCustomer.ID,
	}

	// Delete the customer
	err = service.DeleteCustomer(ctx, customer.DeleteCustomerInput(customerId))

	require.NoError(t, err, "Deleting customer must not return error")

	// Get the customer
	_, err = service.GetCustomer(ctx, customer.GetCustomerInput(customerId))

	require.ErrorAs(t, err, customer.NotFoundError{CustomerID: customerId}, "Fetching customer must return not found error")

	// Delete the customer again should return not found error
	err = service.DeleteCustomer(ctx, customer.DeleteCustomerInput(customerId))

	require.ErrorAs(t, err, customer.NotFoundError{CustomerID: customerId}, "Deleting customer again must return not found error")
}
