package customer

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/invopop/gobl/currency"
	"github.com/oklog/ulid/v2"
	"github.com/samber/lo"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/openmeter/entitlement"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/feature"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/plan"
	plansubscriptionservice "github.com/openmeterio/openmeter/openmeter/productcatalog/subscription/service"
	"github.com/openmeterio/openmeter/openmeter/streaming"
	"github.com/openmeterio/openmeter/openmeter/subscription"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/datetime"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/pagination"
	"github.com/openmeterio/openmeter/pkg/sortx"
)

var (
	TestKey                = "test-customer"
	TestName               = "Test Customer"
	TestPrimaryEmail       = "test@openmeter.io"
	TestCurrency           = currencyx.Code("USD")
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
	createdCustomer, err := service.CreateCustomer(ctx, customer.CreateCustomerInput{
		Namespace: s.namespace,
		CustomerMutate: customer.CustomerMutate{
			Key:            lo.ToPtr(TestKey),
			Name:           TestName,
			PrimaryEmail:   &TestPrimaryEmail,
			Currency:       &TestCurrency,
			BillingAddress: &TestAddress,
			UsageAttribution: &customer.CustomerUsageAttribution{
				SubjectKeys: TestSubjectKeys,
			},
			Metadata: &models.Metadata{
				"foo": "bar",
			},
		},
	})

	require.NoError(t, err, "Creating customer must not return error")

	require.NotNil(t, createdCustomer, "Customer must not be nil")
	require.Equal(t, s.namespace, createdCustomer.Namespace, "Customer namespace must match")
	require.NotNil(t, createdCustomer.ID, "Customer ID must not be nil")
	require.Equal(t, &TestKey, createdCustomer.Key, "Customer key must match")
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
	require.Equal(t, &models.Metadata{"foo": "bar"}, createdCustomer.Metadata, "Customer metadata must match")

	// Test subjects are created
	t.Run("Should create subjects alongside customer", func(t *testing.T) {
		for _, subjectKey := range TestSubjectKeys {
			subject, err := s.Env.Subject().GetByKey(ctx, models.NamespacedKey{
				Namespace: s.namespace,
				Key:       subjectKey,
			})

			require.NoError(t, err, "Getting subject must not return error")
			require.NotNil(t, subject, "Subject must not be nil")
			require.Equal(t, subjectKey, subject.Key, "Subject key must match")
		}
	})

	// Test key conflicts
	t.Run("Should return conflict error if subject keys conflict", func(t *testing.T) {
		_, err := service.CreateCustomer(ctx, customer.CreateCustomerInput{
			Namespace: s.namespace,
			CustomerMutate: customer.CustomerMutate{
				Name: TestName,
				UsageAttribution: &customer.CustomerUsageAttribution{
					SubjectKeys: TestSubjectKeys,
				},
			},
		})

		require.True(
			t,
			customer.IsSubjectKeyConflictError(err),
			"Creating a customer with same subject keys must return conflict error",
		)
	})

	// Test key overlaps with id
	t.Run("Should return conflict error if key overlaps with id", func(t *testing.T) {
		_, err := service.CreateCustomer(ctx, customer.CreateCustomerInput{
			Namespace: s.namespace,
			CustomerMutate: customer.CustomerMutate{
				Key:  lo.ToPtr(createdCustomer.ID), // Overlaps with id of existing customer
				Name: TestName,
				UsageAttribution: &customer.CustomerUsageAttribution{
					SubjectKeys: []string{"subject-1"},
				},
			},
		})

		require.True(
			t,
			models.IsGenericConflictError(err),
			"Creating a customer with a key that overlaps with id must return conflict error",
		)
	})

	// Test key overlaps with subject
	t.Run("Should return conflict error if key overlaps with subject", func(t *testing.T) {
		_, err := service.CreateCustomer(ctx, customer.CreateCustomerInput{
			Namespace: s.namespace,
			CustomerMutate: customer.CustomerMutate{
				Key:  lo.ToPtr(TestSubjectKeys[0]), // Overlaps with subject of existing customer
				Name: TestName,
				UsageAttribution: &customer.CustomerUsageAttribution{
					SubjectKeys: []string{"subject-1"},
				},
			},
		})

		require.True(
			t,
			models.IsGenericConflictError(err),
			"Creating a customer with a key that overlaps with subject must return conflict error",
		)
	})

	// Test creating a customer without subjects
	t.Run("Should allow creation without subject keys", func(t *testing.T) {
		customerWithoutSubjects, err := service.CreateCustomer(ctx, customer.CreateCustomerInput{
			Namespace: s.namespace,
			CustomerMutate: customer.CustomerMutate{
				Key:  lo.ToPtr("customer-no-subjects"),
				Name: "Customer Without Subjects",
				UsageAttribution: &customer.CustomerUsageAttribution{
					SubjectKeys: []string{},
				},
			},
		})

		require.NoError(t, err, "Creating customer without subject keys must not return error")
		require.NotNil(t, customerWithoutSubjects, "Customer without subjects must not be nil")
		// UsageAttribution is nil when there are no subject keys
		require.Nil(t, customerWithoutSubjects.UsageAttribution, "Customer usage attribution must be nil when no subject keys")
	})
}

// TestUpdate tests the updating of a customer
func (s *CustomerHandlerTestSuite) TestUpdate(ctx context.Context, t *testing.T) {
	s.setupNamespace(t)

	service := s.Env.Customer()

	// Create a customer with mandatory fields
	originalCustomer, err := service.CreateCustomer(ctx, customer.CreateCustomerInput{
		Namespace: s.namespace,
		CustomerMutate: customer.CustomerMutate{
			Name: TestName,
			UsageAttribution: &customer.CustomerUsageAttribution{
				SubjectKeys: TestSubjectKeys,
			},
		},
	})

	require.NoError(t, err, "Creating customer must not return error")
	require.NotNil(t, originalCustomer, "Customer must not be nil")
	require.Equal(t, TestName, originalCustomer.Name, "Customer name must match")
	require.Equal(t, TestSubjectKeys, originalCustomer.UsageAttribution.SubjectKeys, "Customer usage attribution subject keys must match")

	newName := "New Name"
	newSubjectKeys := []string{"subject-new"}

	// Update the customer with new fields
	updatedCustomer, err := service.UpdateCustomer(ctx, customer.UpdateCustomerInput{
		CustomerID: customer.CustomerID{
			Namespace: s.namespace,
			ID:        originalCustomer.ID,
		},
		CustomerMutate: customer.CustomerMutate{
			Name:           newName,
			PrimaryEmail:   &TestPrimaryEmail,
			Currency:       &TestCurrency,
			BillingAddress: &TestAddress,
			UsageAttribution: &customer.CustomerUsageAttribution{
				SubjectKeys: newSubjectKeys,
			},
			Metadata: &models.Metadata{
				"foo": "bar",
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
	require.Equal(t, &TestAddressCountry, updatedCustomer.BillingAddress.Country, "Customer billing address country must match")
	require.Equal(t, &TestAddressCity, updatedCustomer.BillingAddress.City, "Customer billing address city must match")
	require.Equal(t, &TestAddressLine1, updatedCustomer.BillingAddress.Line1, "Customer billing address line1 must match")
	require.Equal(t, &TestAddressLine2, updatedCustomer.BillingAddress.Line2, "Customer billing address line2 must match")
	require.Equal(t, &TestAddressPostalCode, updatedCustomer.BillingAddress.PostalCode, "Customer billing address postal code must match")
	require.Equal(t, &TestAddressPhoneNumber, updatedCustomer.BillingAddress.PhoneNumber, "Customer billing address phone number must match")
	require.Equal(t, &models.Metadata{"foo": "bar"}, updatedCustomer.Metadata, "Customer metadata must match")

	// Test subjects are created appropriately
	t.Run("Should create subjects alongside customer", func(t *testing.T) {
		for _, subjectKey := range newSubjectKeys {
			subject, err := s.Env.Subject().GetByKey(ctx, models.NamespacedKey{
				Namespace: s.namespace,
				Key:       subjectKey,
			})

			require.NoError(t, err, "Getting subject must not return error")
			require.NotNil(t, subject, "Subject must not be nil")
			require.Equal(t, subjectKey, subject.Key, "Subject key must match")
		}
	})

	// Test that old subjects are not deleted just left dangling
	t.Run("Should not delete old subjects", func(t *testing.T) {
		for _, subjectKey := range originalCustomer.UsageAttribution.SubjectKeys {
			subject, err := s.Env.Subject().GetByKey(ctx, models.NamespacedKey{
				Namespace: s.namespace,
				Key:       subjectKey,
			})

			require.NoError(t, err, "Getting subject must not return error")
			require.NotNil(t, subject, "Subject must not be nil")
			require.Equal(t, subjectKey, subject.Key, "Subject key must match")
		}
	})

	// Create another customer with a different key and subject key to test conflicts
	otherCustomerKey := "other-customer-key"
	otherCustomerSubjectKey := "other-subject-key"

	_, err = service.CreateCustomer(ctx, customer.CreateCustomerInput{
		Namespace: s.namespace,
		CustomerMutate: customer.CustomerMutate{
			Key:  lo.ToPtr(otherCustomerKey),
			Name: TestName,
			UsageAttribution: &customer.CustomerUsageAttribution{
				SubjectKeys: []string{otherCustomerSubjectKey},
			},
		},
	})

	require.NoError(t, err, "Creating customer must not return error")

	// Test key overlaps with existing customer's key
	_, err = service.UpdateCustomer(ctx, customer.UpdateCustomerInput{
		CustomerID: customer.CustomerID{
			Namespace: s.namespace,
			ID:        originalCustomer.ID,
		},
		CustomerMutate: customer.CustomerMutate{
			Key:  lo.ToPtr(otherCustomerKey), // Overlaps with key of existing customer
			Name: TestName,
			UsageAttribution: &customer.CustomerUsageAttribution{
				SubjectKeys: TestSubjectKeys,
			},
		},
	})

	require.True(
		t,
		models.IsGenericConflictError(err),
		"Updating a customer with a key that overlaps with key must return conflict error",
	)

	// Test key overlaps with existing customer's subject
	_, err = service.UpdateCustomer(ctx, customer.UpdateCustomerInput{
		CustomerID: customer.CustomerID{
			Namespace: s.namespace,
			ID:        originalCustomer.ID,
		},
		CustomerMutate: customer.CustomerMutate{
			Key:  lo.ToPtr(otherCustomerSubjectKey), // Overlaps with subject of existing customer
			Name: TestName,
			UsageAttribution: &customer.CustomerUsageAttribution{
				SubjectKeys: TestSubjectKeys,
			},
		},
	})

	require.True(
		t,
		models.IsGenericConflictError(err),
		"Updating a customer with a key that overlaps with subject must return conflict error",
	)
}

// If a customer has a subscription, UsageAttributions cannot be updated
func (s *CustomerHandlerTestSuite) TestUpdateWithSubscriptionPresent(ctx context.Context, t *testing.T) {
	s.setupNamespace(t)

	cService := s.Env.Customer()

	// Create a customer with mandatory fields
	originalCustomer, err := cService.CreateCustomer(ctx, customer.CreateCustomerInput{
		Namespace: s.namespace,
		CustomerMutate: customer.CustomerMutate{
			Name: TestName,
			UsageAttribution: &customer.CustomerUsageAttribution{
				SubjectKeys: TestSubjectKeys,
			},
		},
	})

	require.NoError(t, err, "Creating customer must not return error")
	require.NotNil(t, originalCustomer, "Customer must not be nil")
	require.Equal(t, TestName, originalCustomer.Name, "Customer name must match")
	require.Equal(t, TestSubjectKeys, originalCustomer.UsageAttribution.SubjectKeys, "Customer usage attribution subject keys must match")

	_, _ = s.startMockSubscription(ctx, t, originalCustomer.GetID())

	// Update the customer with new UsageAttribution
	newName := "New Name"
	newSubjectKeys := []string{"subject-1"}

	_, err = cService.UpdateCustomer(ctx, customer.UpdateCustomerInput{
		CustomerID: customer.CustomerID{
			Namespace: s.namespace,
			ID:        originalCustomer.ID,
		},
		CustomerMutate: customer.CustomerMutate{
			Name:           newName,
			PrimaryEmail:   &TestPrimaryEmail,
			Currency:       &TestCurrency,
			BillingAddress: &TestAddress,
			UsageAttribution: &customer.CustomerUsageAttribution{
				SubjectKeys: newSubjectKeys,
			},
		},
	})

	require.True(t, models.IsGenericValidationError(err), "Updating customer UsageAttribution with subscription must return validation error, got %T", err)

	// Update the customer but not the UsageAttribution
	updatedCustomer, err := cService.UpdateCustomer(ctx, customer.UpdateCustomerInput{
		CustomerID: customer.CustomerID{
			Namespace: s.namespace,
			ID:        originalCustomer.ID,
		},
		CustomerMutate: customer.CustomerMutate{
			Name:             newName,
			PrimaryEmail:     &TestPrimaryEmail,
			Currency:         &TestCurrency,
			BillingAddress:   &TestAddress,
			UsageAttribution: originalCustomer.UsageAttribution,
		},
	})

	require.NoError(t, err, "Updating customer must not return error")
	require.NotNil(t, updatedCustomer, "Customer must not be nil")
	require.Equal(t, s.namespace, updatedCustomer.Namespace, "Customer namespace must match")
	require.Equal(t, originalCustomer.ID, updatedCustomer.ID, "Customer ID must match")
	require.Equal(t, newName, updatedCustomer.Name, "Customer name must match")
	require.Equal(t, originalCustomer.UsageAttribution.SubjectKeys, updatedCustomer.UsageAttribution.SubjectKeys, "Customer usage attribution subject keys must match")
	require.Equal(t, &TestPrimaryEmail, updatedCustomer.PrimaryEmail, "Customer primary email must match")
	require.Equal(t, &TestCurrency, updatedCustomer.Currency, "Customer currency must match")
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
	createCustomer1, err := service.CreateCustomer(ctx, customer.CreateCustomerInput{
		Namespace: s.namespace,
		CustomerMutate: customer.CustomerMutate{
			Key:  lo.ToPtr("customer-1"),
			Name: "Customer 1",
			UsageAttribution: &customer.CustomerUsageAttribution{
				SubjectKeys: []string{"subject-1"},
			},
			PrimaryEmail: lo.ToPtr("customer-1@test.com"),
		},
	})

	require.NoError(t, err, "Creating customer must not return error")

	// Create a customer 2
	createCustomer2, err := service.CreateCustomer(ctx, customer.CreateCustomerInput{
		Namespace: s.namespace,
		CustomerMutate: customer.CustomerMutate{
			Name: "Customer 2",
			UsageAttribution: &customer.CustomerUsageAttribution{
				SubjectKeys: []string{"subject-2"},
			},
			PrimaryEmail: lo.ToPtr("customer-2@test.com"),
		},
	})

	require.NoError(t, err, "Creating customer must not return error")

	// Create a customer 3 in a different namespace
	differentNamespace := ulid.Make().String()

	_, err = service.CreateCustomer(ctx, customer.CreateCustomerInput{
		Namespace: differentNamespace,
		CustomerMutate: customer.CustomerMutate{
			Name: "Customer 3",
			UsageAttribution: &customer.CustomerUsageAttribution{
				SubjectKeys: []string{"subject-3"},
			},
		},
	})

	require.NoError(t, err, "Creating customer must not return error")

	page := pagination.Page{PageNumber: 1, PageSize: 10}

	// List customers
	list, err := service.ListCustomers(ctx, customer.ListCustomersInput{
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

	// List customers with key filter
	list, err = service.ListCustomers(ctx, customer.ListCustomersInput{
		Namespace: s.namespace,
		Page:      page,
		Key:       lo.ToPtr("customer-1"),
	})

	require.NoError(t, err, "Listing customers with key filter must not return error")
	require.Equal(t, 1, list.TotalCount, "Customers total count must be 1")
	require.Equal(t, createCustomer1.ID, list.Items[0].ID, "Customer ID must match")

	// List customers with name filter
	list, err = service.ListCustomers(ctx, customer.ListCustomersInput{
		Namespace: s.namespace,
		Page:      page,
		Name:      &createCustomer2.Name,
	})

	require.NoError(t, err, "Listing customers with name filter must not return error")
	require.Equal(t, 1, list.TotalCount, "Customers total count must be 1")
	require.Equal(t, createCustomer2.ID, list.Items[0].ID, "Customer ID must match")

	// List customers with partial name filter
	list, err = service.ListCustomers(ctx, customer.ListCustomersInput{
		Namespace: s.namespace,
		Page:      page,
		Name:      lo.ToPtr("2"),
	})

	require.NoError(t, err, "Listing customers with partial name filter must not return error")
	require.Equal(t, 1, list.TotalCount, "Customers total count must be 1")
	require.Equal(t, createCustomer2.ID, list.Items[0].ID, "Customer ID must match")

	// List customers with primary email filter
	list, err = service.ListCustomers(ctx, customer.ListCustomersInput{
		Namespace:    s.namespace,
		Page:         page,
		PrimaryEmail: createCustomer2.PrimaryEmail,
	})

	require.NoError(t, err, "Listing customers with primary email filter must not return error")
	require.Equal(t, 1, list.TotalCount, "Customers total count must be 1")
	require.Equal(t, createCustomer2.ID, list.Items[0].ID, "Customer ID must match")

	// Order by name descending
	list, err = service.ListCustomers(ctx, customer.ListCustomersInput{
		Namespace: s.namespace,
		Page:      page,
		OrderBy:   "name",
		Order:     sortx.OrderDesc,
	})

	require.NoError(t, err, "Listing customers with order by name must not return error")
	require.Equal(t, 2, list.TotalCount, "Customers total count must be 1")
	require.Equal(t, 1, list.Page.PageNumber, "Customers page must be 0")
	require.Equal(t, createCustomer2.ID, list.Items[0].ID, "Customer 2 must be first in order")
	require.Equal(t, createCustomer1.ID, list.Items[1].ID, "Customer 1 must be second in order")
}

// TestListWithSubscription tests the listing of customers with subscription
func (s *CustomerHandlerTestSuite) TestListWithSubscription(ctx context.Context, t *testing.T) {
	s.setupNamespace(t)

	service := s.Env.Customer()
	cService := s.Env.Customer()

	// Create a customer with mandatory fields
	testCutomer, err := cService.CreateCustomer(ctx, customer.CreateCustomerInput{
		Namespace: s.namespace,
		CustomerMutate: customer.CustomerMutate{
			Name: TestName,
			UsageAttribution: &customer.CustomerUsageAttribution{
				SubjectKeys: TestSubjectKeys,
			},
		},
	})

	require.NoError(t, err, "Creating customer must not return error")
	require.NotNil(t, testCutomer, "Customer must not be nil")
	require.Equal(t, TestName, testCutomer.Name, "Customer name must match")
	require.Equal(t, TestSubjectKeys, testCutomer.UsageAttribution.SubjectKeys, "Customer usage attribution subject keys must match")

	plan, _ := s.startMockSubscription(ctx, t, testCutomer.GetID())
	page := pagination.Page{PageNumber: 1, PageSize: 10}

	// List customers with plan ID filter
	list, err := service.ListCustomers(ctx, customer.ListCustomersInput{
		Namespace: s.namespace,
		Page:      page,
		Plan:      lo.ToPtr(plan.ID),
	})

	require.NoError(t, err, "Listing customers with key filter must not return error")
	require.Equal(t, 1, list.TotalCount, "Customers total count must be 1")
	require.Equal(t, testCutomer.ID, list.Items[0].ID, "Customer ID must match")

	// List customers with plan key filter
	list, err = service.ListCustomers(ctx, customer.ListCustomersInput{
		Namespace: s.namespace,
		Page:      page,
		PlanKey:   lo.ToPtr(plan.Key),
	})

	require.NoError(t, err, "Listing customers with plan key filter must not return error")
	require.Equal(t, 1, list.TotalCount, "Customers total count must be 1")
	require.Equal(t, testCutomer.ID, list.Items[0].ID, "Customer ID must match")

	// List customers with unknown plan key
	list, err = service.ListCustomers(ctx, customer.ListCustomersInput{
		Namespace: s.namespace,
		Page:      page,
		PlanKey:   lo.ToPtr("unknown-plan-key"),
	})

	require.NoError(t, err, "Listing customers with unknown plan key must not return error")
	require.Equal(t, 0, list.TotalCount, "Customers total count must be 0")

	// List customers with both plan ID and key filter
	list, err = service.ListCustomers(ctx, customer.ListCustomersInput{
		Namespace: s.namespace,
		Page:      page,
		Plan:      lo.ToPtr(plan.ID),
		PlanKey:   lo.ToPtr(plan.Key),
	})

	require.True(t, models.IsGenericValidationError(err), "Listing customers with both plan ID and key filter must return validation error")
}

// TestListCustomerUsageAttributions tests the listing of customer usage attributions
func (s *CustomerHandlerTestSuite) TestListCustomerUsageAttributions(ctx context.Context, t *testing.T) {
	s.setupNamespace(t)

	service := s.Env.Customer()

	// Create a customer 1
	createCustomer1, err := service.CreateCustomer(ctx, customer.CreateCustomerInput{
		Namespace: s.namespace,
		CustomerMutate: customer.CustomerMutate{
			Key:  lo.ToPtr("customer-1"),
			Name: "Customer 1",
			UsageAttribution: &customer.CustomerUsageAttribution{
				SubjectKeys: []string{"customer-1-subject-1", "customer-1-subject-2"},
			},
		},
	})

	require.NoError(t, err, "Creating customer must not return error")

	// Create a customer 2
	createCustomer2, err := service.CreateCustomer(ctx, customer.CreateCustomerInput{
		Namespace: s.namespace,
		CustomerMutate: customer.CustomerMutate{
			Name: "Customer 2",
			UsageAttribution: &customer.CustomerUsageAttribution{
				SubjectKeys: []string{"customer-2-subject-1"},
			},
		},
	})

	require.NoError(t, err, "Creating customer must not return error")

	page := pagination.Page{PageNumber: 1, PageSize: 10}

	list, err := service.ListCustomerUsageAttributions(ctx, customer.ListCustomerUsageAttributionsInput{
		Namespace: s.namespace,
		Page:      page,
	})

	require.NoError(t, err, "Listing customer usage attributions must not return error")
	require.Equal(t, 2, list.TotalCount, "Customer usage attributions total count must be 2")
	require.Equal(t, 1, list.Page.PageNumber, "Customer usage attributions page must be 0")
	require.Equal(t, createCustomer1.ID, list.Items[0].ID, "Customer 1 must be first in order")
	require.Equal(t, createCustomer2.ID, list.Items[1].ID, "Customer 2 must be second in order")

	expectedItems := []streaming.CustomerUsageAttribution{
		{
			ID:          createCustomer1.ID,
			Key:         createCustomer1.Key,
			SubjectKeys: []string{"customer-1-subject-1", "customer-1-subject-2"},
		},
		{
			ID:          createCustomer2.ID,
			SubjectKeys: []string{"customer-2-subject-1"},
		},
	}

	require.Equal(t, expectedItems, list.Items, "Customer usage attributions must match")
}

// TestGet tests the getting of a customer by ID
func (s *CustomerHandlerTestSuite) TestGet(ctx context.Context, t *testing.T) {
	s.setupNamespace(t)

	service := s.Env.Customer()

	// Create a customer
	createdCustomer, err := service.CreateCustomer(ctx, customer.CreateCustomerInput{
		Namespace: s.namespace,
		CustomerMutate: customer.CustomerMutate{
			Name: TestName,
			Key:  lo.ToPtr(TestKey),
			UsageAttribution: &customer.CustomerUsageAttribution{
				SubjectKeys: TestSubjectKeys,
			},
		},
	})

	require.NoError(t, err, "Creating customer must not return error")
	require.NotNil(t, createdCustomer, "Customer must not be nil")

	// Get the customer by ID
	cus, err := service.GetCustomer(ctx, customer.GetCustomerInput{
		CustomerID: &customer.CustomerID{
			Namespace: s.namespace,
			ID:        createdCustomer.ID,
		},
	})

	require.NoError(t, err, "Fetching customer must not return error")
	require.NotNil(t, cus, "Customer must not be nil")
	require.Equal(t, s.namespace, cus.Namespace, "Customer namespace must match")
	require.NotNil(t, cus.ID, "Customer ID must not be nil")
	require.Equal(t, createdCustomer.ID, cus.ID, "Customer ID must match")
	require.Equal(t, TestName, cus.Name, "Customer name must match")
	require.Equal(t, TestSubjectKeys, cus.UsageAttribution.SubjectKeys, "Customer usage attribution subject keys must match")

	// Get the customer by key
	cus, err = service.GetCustomer(ctx, customer.GetCustomerInput{
		CustomerKey: &customer.CustomerKey{
			Namespace: s.namespace,
			Key:       TestKey,
		},
	})

	require.NoError(t, err, "Fetching customer must not return error")
	require.NotNil(t, cus, "Customer must not be nil")
	require.Equal(t, createdCustomer.ID, cus.ID, "Customer ID must match")
	require.Equal(t, s.namespace, cus.Namespace, "Customer namespace must match")

	// Get the customer by idOrKey
	cus, err = service.GetCustomer(ctx, customer.GetCustomerInput{
		CustomerIDOrKey: &customer.CustomerIDOrKey{
			IDOrKey:   createdCustomer.ID,
			Namespace: s.namespace,
		},
	})

	require.NoError(t, err, "Fetching customer must not return error")
	require.NotNil(t, cus, "Customer must not be nil")
	require.Equal(t, createdCustomer.ID, cus.ID, "Customer ID must match")
	require.Equal(t, s.namespace, cus.Namespace, "Customer namespace must match")

	// Test not found
	_, err = service.GetCustomer(ctx, customer.GetCustomerInput{
		CustomerKey: &customer.CustomerKey{
			Namespace: s.namespace,
			Key:       "non-existent-key",
		},
	})

	require.True(t, models.IsGenericNotFoundError(err), "Fetching non-existent customer must return not found error")
}

// TestGetByUsageAttribution tests the getting of a customer by usage attribution
func (s *CustomerHandlerTestSuite) TestGetByUsageAttribution(ctx context.Context, t *testing.T) {
	s.setupNamespace(t)

	service := s.Env.Customer()

	// Create a customer
	createdCustomer, err := service.CreateCustomer(ctx, customer.CreateCustomerInput{
		Namespace: s.namespace,
		CustomerMutate: customer.CustomerMutate{
			Name: TestName,
			Key:  lo.ToPtr(TestKey),
			UsageAttribution: &customer.CustomerUsageAttribution{
				SubjectKeys: TestSubjectKeys,
			},
		},
	})

	require.NoError(t, err, "Creating customer must not return error")
	require.NotNil(t, createdCustomer, "Customer must not be nil")

	// Get the customer by usage attribution
	cus, err := service.GetCustomerByUsageAttribution(ctx, customer.GetCustomerByUsageAttributionInput{
		Namespace: s.namespace,
		Key:       TestSubjectKeys[0],
	})

	require.NoError(t, err, "Fetching customer must not return error")
	require.NotNil(t, cus, "Customer must not be nil")
	require.Equal(t, s.namespace, cus.Namespace, "Customer namespace must match")
	require.Equal(t, createdCustomer.ID, cus.ID, "Customer ID must match")

	// Get the customer by key
	cus, err = service.GetCustomerByUsageAttribution(ctx, customer.GetCustomerByUsageAttributionInput{
		Namespace: s.namespace,
		Key:       TestKey,
	})

	require.NoError(t, err, "Fetching customer must not return error")
	require.NotNil(t, cus, "Customer must not be nil")
	require.Equal(t, s.namespace, cus.Namespace, "Customer namespace must match")
	require.Equal(t, createdCustomer.ID, cus.ID, "Customer ID must match")

	// Get the customer by key
	cus, err = service.GetCustomerByUsageAttribution(ctx, customer.GetCustomerByUsageAttributionInput{
		Namespace: s.namespace,
		Key:       TestKey,
	})

	require.NoError(t, err, "Fetching customer must not return error")
	require.NotNil(t, cus, "Customer must not be nil")
	require.Equal(t, s.namespace, cus.Namespace, "Customer namespace must match")
	require.Equal(t, createdCustomer.ID, cus.ID, "Customer ID must match")

	// Get the customer by usage attribution with a non-existent subject key
	_, err = service.GetCustomerByUsageAttribution(ctx, customer.GetCustomerByUsageAttributionInput{
		Namespace: s.namespace,
		Key:       "non-existent-subject-key",
	})

	require.True(t, models.IsGenericNotFoundError(err), "Fetching customer with non-existent subject key must return not found error")
}

// TestDelete tests the deletion of a customer
func (s *CustomerHandlerTestSuite) TestDelete(ctx context.Context, t *testing.T) {
	s.setupNamespace(t)

	custService := s.Env.Customer()
	subService := s.Env.Subscription()

	// Create a customer
	input := customer.CreateCustomerInput{
		Namespace: s.namespace,
		CustomerMutate: customer.CustomerMutate{
			Name: TestName,
			UsageAttribution: &customer.CustomerUsageAttribution{
				SubjectKeys: TestSubjectKeys,
			},
		},
	}
	originalCustomer, err := custService.CreateCustomer(ctx, input)

	require.NoError(t, err, "Creating customer must not return error")
	require.NotNil(t, originalCustomer, "Customer must not be nil")

	customerId := customer.CustomerID{
		Namespace: s.namespace,
		ID:        originalCustomer.ID,
	}

	// Let's create a subscription for the customer
	emptyExamplePlan := plan.CreatePlanInput{
		NamespacedModel: models.NamespacedModel{
			Namespace: s.namespace,
		},
		Plan: productcatalog.Plan{
			PlanMeta: productcatalog.PlanMeta{
				Name:           "Empty Plan",
				Currency:       currency.Code("USD"),
				BillingCadence: datetime.MustParseDuration(t, "P1M"),
				ProRatingConfig: productcatalog.ProRatingConfig{
					Enabled: true,
					Mode:    productcatalog.ProRatingModeProratePrices,
				},
			},
			Phases: []productcatalog.Phase{
				{
					PhaseMeta: productcatalog.PhaseMeta{
						Key:  "empty-phase",
						Name: "Empty Phase",
					},
					RateCards: []productcatalog.RateCard{
						&productcatalog.FlatFeeRateCard{
							RateCardMeta: productcatalog.RateCardMeta{
								Key:  "empty-rate-card",
								Name: "Empty Rate Card",
							},
						},
					},
				},
			},
		},
	}

	p, err := plansubscriptionservice.PlanFromPlanInput(emptyExamplePlan)
	require.Nil(t, err)

	now := clock.Now()

	spec, err := subscription.NewSpecFromPlan(p, subscription.CreateSubscriptionCustomerInput{
		CustomerId:    originalCustomer.ID,
		Name:          "Test Subscription",
		Currency:      currencyx.Code("USD"),
		ActiveFrom:    now,
		BillingAnchor: now,
	})
	require.Nil(t, err)

	clock.SetTime(clock.Now().Add(1 * time.Minute))

	sub, err := subService.Create(ctx, s.namespace, spec)
	require.Nil(t, err)

	// Delete the customer with active subscription should return validation error
	require.Equal(t, sub.CustomerId, customerId.ID, "Subscription customer ID must match")
	err = custService.DeleteCustomer(ctx, customerId)

	require.ErrorAs(t, err, lo.ToPtr(&models.GenericValidationError{}), "Deleting customer with active subscription must return validation error, got %T", err)
	require.EqualError(t, err, fmt.Sprintf("validation error: customer %s still have active subscriptions, please cancel them before deleting the customer", customerId.ID), "Deleting customer with active subscription must return error")

	// Now let's delete the subscription
	_, err = subService.Cancel(ctx, sub.NamespacedID, subscription.Timing{
		Enum: lo.ToPtr(subscription.TimingImmediate),
	})
	require.NoError(t, err, "Canceling subscription must not return error")

	clock.SetTime(clock.Now().Add(1 * time.Minute))

	// Delete the customer again
	err = custService.DeleteCustomer(ctx, customerId)

	require.NoError(t, err, "Deleting customer must not return error")

	// Get the customer
	getCustomer, err := custService.GetCustomer(ctx, customer.GetCustomerInput{
		CustomerID: &customerId,
	})

	require.NoError(t, err, "Getting a deleted customer must not return error")
	require.NotNil(t, getCustomer.DeletedAt, "DeletedAt must not be nil")

	// Delete the customer by id again should not return an error
	err = custService.DeleteCustomer(ctx, customerId)
	require.NoError(t, err, "Deleting customer by id must not return an error")

	// Should allow to create a customer with the same subject keys
	createdCustomer, err := custService.CreateCustomer(ctx, input)
	require.NoError(t, err, "Creating a customer with the same subject keys must not return error")
	require.NotNil(t, createdCustomer, "Created customer must not be nil")

	// Delete the customer with active entitlement should return validation error
	entitlementService := s.Env.Entitlement()
	featureService := s.Env.Feature()

	feature, err := featureService.CreateFeature(ctx, feature.CreateFeatureInputs{
		Namespace: s.namespace,
		Key:       "test-feature",
		Name:      "Test Feature",
	})
	require.NoError(t, err, "Creating feature must not return error")
	require.NotNil(t, feature, "Feature must not be nil")

	entitlement, err := entitlementService.CreateEntitlement(ctx, entitlement.CreateEntitlementInputs{
		Namespace:        s.namespace,
		FeatureID:        lo.ToPtr(feature.ID),
		EntitlementType:  entitlement.EntitlementTypeBoolean,
		UsageAttribution: createdCustomer.GetUsageAttribution(),
	}, nil)
	require.NoError(t, err, "Creating entitlement must not return error")
	require.NotNil(t, entitlement, "Entitlement must not be nil")

	err = custService.DeleteCustomer(ctx, createdCustomer.GetID())
	require.ErrorAs(t, err, lo.ToPtr(&models.GenericValidationError{}), "Deleting customer with active entitlement must return validation error, got %T", err)
	require.EqualError(
		t,
		err,
		fmt.Sprintf("validation error: conflict error: customer %s still has active entitlements, please remove them before deleting the customer", createdCustomer.ID),
		"Deleting customer with active entitlement must return error",
	)

	// Delete the entitlement
	err = entitlementService.DeleteEntitlement(ctx, s.namespace, entitlement.ID, clock.Now())
	require.NoError(t, err, "Deleting entitlement must not return error")

	// Deleting the customer is now allowed
	err = custService.DeleteCustomer(ctx, createdCustomer.GetID())
	require.NoError(t, err, "Deleting customer must not return error")
}

// startMockSubscription starts a mock subscription for a customer
func (s *CustomerHandlerTestSuite) startMockSubscription(
	ctx context.Context,
	t *testing.T,
	customerId customer.CustomerID,
) (plan.Plan, subscription.Subscription) {
	plan := s.createMockPlan(ctx, t)
	sub := s.startSubscriptionFromPlan(ctx, t, plan, customerId)
	return plan, sub
}

// createMockPlan creates a mock plan for a customer
func (s *CustomerHandlerTestSuite) createMockPlan(ctx context.Context, t *testing.T) plan.Plan {
	// Create the plan
	testPlan, err := s.Env.Plan().CreatePlan(ctx, plan.CreatePlanInput{
		NamespacedModel: models.NamespacedModel{
			Namespace: s.namespace,
		},
		Plan: productcatalog.Plan{
			PlanMeta: productcatalog.PlanMeta{
				Name:           "Empty Plan",
				Key:            "empty-plan",
				Version:        1,
				Currency:       currency.Code("USD"),
				BillingCadence: datetime.MustParseDuration(t, "P1M"),
				ProRatingConfig: productcatalog.ProRatingConfig{
					Enabled: true,
					Mode:    productcatalog.ProRatingModeProratePrices,
				},
			},
			Phases: []productcatalog.Phase{
				{
					PhaseMeta: productcatalog.PhaseMeta{
						Key:  "empty-phase",
						Name: "Empty Phase",
					},
					RateCards: []productcatalog.RateCard{
						&productcatalog.FlatFeeRateCard{
							RateCardMeta: productcatalog.RateCardMeta{
								Key:  "empty-rate-card",
								Name: "Empty Rate Card",
							},
						},
					},
				},
			},
		},
	})
	require.NoError(t, err, "Creating plan must not return error")
	require.NotNil(t, testPlan, "Plan must not be nil")

	// Publish the plan
	s.Env.Plan().PublishPlan(ctx, plan.PublishPlanInput{
		NamespacedID: testPlan.NamespacedID,
		EffectivePeriod: productcatalog.EffectivePeriod{
			EffectiveFrom: lo.ToPtr(clock.Now()),
		},
	})

	return *testPlan
}

// startSubscriptionFromPlan starts a subscription from a plan
func (s *CustomerHandlerTestSuite) startSubscriptionFromPlan(
	ctx context.Context,
	t *testing.T,
	plan plan.Plan,
	customerId customer.CustomerID,
) subscription.Subscription {
	now := clock.Now()
	subscriptionPlan := plansubscriptionservice.PlanFromPlan(plan)

	spec, err := subscription.NewSpecFromPlan(subscriptionPlan, subscription.CreateSubscriptionCustomerInput{
		CustomerId:    customerId.ID,
		Name:          "Test Subscription",
		Currency:      currencyx.Code("USD"),
		ActiveFrom:    now,
		BillingAnchor: now,
	})
	require.Nil(t, err)

	clock.SetTime(clock.Now().Add(1 * time.Minute))

	sub, err := s.Env.Subscription().Create(ctx, s.namespace, spec)
	require.Nil(t, err)

	return sub
}
