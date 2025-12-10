package billing

import (
	"context"
	"testing"

	"github.com/samber/lo"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/pkg/datetime"
	"github.com/openmeterio/openmeter/pkg/models"
)

type CustomerOverrideTestSuite struct {
	BaseSuite
}

func TestCustomerOverride(t *testing.T) {
	suite.Run(t, new(CustomerOverrideTestSuite))
}

func (s *CustomerOverrideTestSuite) TestFetchNonExistingCustomer() {
	// Given we have a non-existing customer
	nonExistingCustomerID := "non-existing-customer-id"
	ns := "test-ns"
	ctx := context.Background()

	sandboxApp := s.InstallSandboxApp(s.T(), ns)
	s.ProvisionBillingProfile(ctx, ns, sandboxApp.GetID())

	// When querying the customer's billing profile overrides
	customerEntity, err := s.BillingService.GetCustomerOverride(ctx, billing.GetCustomerOverrideInput{
		Customer: customer.CustomerID{
			Namespace: ns,
			ID:        nonExistingCustomerID,
		},
	})

	// Then we get a customer not found error
	require.True(s.T(), models.IsGenericNotFoundError(err), "expect a customer not found error")
	require.Empty(s.T(), customerEntity)
}

func (s *CustomerOverrideTestSuite) TestDefaultProfileHandling() {
	ns := "test-ns-default-profile-handling"
	ctx := context.Background()

	sandboxApp := s.InstallSandboxApp(s.T(), ns)

	// Given we have an existing customer
	custKey := "johny-the-doe-1"
	cust, err := s.CustomerService.CreateCustomer(ctx, customer.CreateCustomerInput{
		Namespace: ns,
		CustomerMutate: customer.CustomerMutate{
			Name: "Johny the Doe",
			Key:  &custKey,
		},
	})
	require.NoError(s.T(), err)
	require.NotNil(s.T(), cust)
	customerID := cust.ID

	s.T().Run("customer without default profile, no override", func(t *testing.T) {
		// When not having a default profile
		// Then we get a NotFoundError
		profileWithOverride, err := s.BillingService.GetCustomerOverride(ctx, billing.GetCustomerOverrideInput{
			Customer: customer.CustomerID{
				Namespace: ns,
				ID:        customerID,
			},
		})
		require.ErrorIs(t, err, billing.ErrDefaultProfileNotFound)
		require.ErrorAs(t, err, &billing.NotFoundError{})
		require.Empty(t, profileWithOverride)
	})

	var defaultProfile *billing.Profile

	s.T().Run("customer with default profile, no override", func(t *testing.T) {
		// Given having a default profile
		profileInput := minimalCreateProfileInputTemplate(sandboxApp.GetID())
		profileInput.Namespace = ns

		defaultProfile, err = s.BillingService.CreateProfile(ctx, profileInput)
		require.NoError(t, err)
		require.NotNil(t, defaultProfile)

		// Let's fetch the default profile again to make sure we have truncated timestamps
		defaultProfile, err = s.BillingService.GetDefaultProfile(ctx, billing.GetDefaultProfileInput{
			Namespace: ns,
		})
		require.NoError(t, err)
		require.NotNil(t, defaultProfile)

		// When fetching the profile
		customerProfile, err := s.BillingService.GetCustomerOverride(ctx, billing.GetCustomerOverrideInput{
			Customer: customer.CustomerID{
				Namespace: ns,
				ID:        customerID,
			},
		})

		// Then we get the default profile as the customer profile
		require.NoError(t, err)
		require.NotNil(t, customerProfile)

		require.Equal(t, defaultProfile.BaseProfile, customerProfile.MergedProfile.BaseProfile)
	})

	s.T().Run("customer with default profile, with override", func(t *testing.T) {
		// Given we have an override for the customer
		createdCustomerOverride, err := s.BillingService.UpsertCustomerOverride(ctx, billing.UpsertCustomerOverrideInput{
			Namespace:  ns,
			CustomerID: customerID,

			Collection: billing.CollectionOverrideConfig{
				Interval: lo.ToPtr(datetime.MustParseDuration(s.T(), "PT1H")),
			},
			Invoicing: billing.InvoicingOverrideConfig{
				AutoAdvance: lo.ToPtr(false),
				DraftPeriod: lo.ToPtr(datetime.MustParseDuration(s.T(), "PT2H")),
				DueAfter:    lo.ToPtr(datetime.MustParseDuration(s.T(), "PT3H")),
			},
			Payment: billing.PaymentOverrideConfig{
				CollectionMethod: lo.ToPtr(billing.CollectionMethodSendInvoice),
			},
		})

		require.NoError(t, err)
		require.NotNil(t, createdCustomerOverride)
		// When fetching the override
		customerProfile, err := s.BillingService.GetCustomerOverride(ctx, billing.GetCustomerOverrideInput{
			Customer: customer.CustomerID{
				Namespace: ns,
				ID:        customerID,
			},
		})

		// Then we get the override as the customer profile
		require.NoError(t, err)
		require.NotNil(t, customerProfile)

		wfConfig := customerProfile.MergedProfile.WorkflowConfig

		require.Equal(t, wfConfig.Collection.Interval, datetime.MustParseDuration(t, "PT1H"))
		require.Equal(t, wfConfig.Invoicing.AutoAdvance, false)
		require.Equal(t, wfConfig.Invoicing.DraftPeriod, datetime.MustParseDuration(t, "PT2H"))
		require.Equal(t, wfConfig.Invoicing.DueAfter, datetime.MustParseDuration(t, "PT3H"))
		require.Equal(t, wfConfig.Payment.CollectionMethod, billing.CollectionMethodSendInvoice)
	})
}

func (s *CustomerOverrideTestSuite) TestPinnedProfileHandling() {
	ns := "test-ns-pinned-profile-handling"
	ctx := context.Background()

	sandboxApp := s.InstallSandboxApp(s.T(), ns)

	// Given we have an existing customer
	custKey := "johny-the-doe-2"
	cust, err := s.CustomerService.CreateCustomer(ctx, customer.CreateCustomerInput{
		Namespace: ns,
		CustomerMutate: customer.CustomerMutate{
			Name: "Johny the Doe",
			Key:  &custKey,
		},
	})
	require.NoError(s.T(), err)
	require.NotNil(s.T(), cust)
	customerID := cust.ID

	// Given we have a non-default profile
	profileInput := minimalCreateProfileInputTemplate(sandboxApp.GetID())
	profileInput.Namespace = ns
	profileInput.Default = false

	pinnedProfile, err := s.BillingService.CreateProfile(ctx, profileInput)
	require.NoError(s.T(), err)
	require.NotNil(s.T(), pinnedProfile)

	s.T().Run("customer without default profile, no profile override", func(t *testing.T) {
		// When not having a default profile
		// Then we get a NotFoundError
		profileWithOverride, err := s.BillingService.GetCustomerOverride(ctx, billing.GetCustomerOverrideInput{
			Customer: customer.CustomerID{
				Namespace: ns,
				ID:        customerID,
			},
		})
		require.ErrorIs(t, err, billing.ErrDefaultProfileNotFound)
		require.ErrorAs(t, err, &billing.NotFoundError{})
		require.Empty(t, profileWithOverride)
	})

	s.T().Run("customer without default profile, with override", func(t *testing.T) {
		// Given we have an override for the customer
		createdCustomerOverride, err := s.BillingService.UpsertCustomerOverride(ctx, billing.UpsertCustomerOverrideInput{
			Namespace:  ns,
			CustomerID: customerID,
			ProfileID:  pinnedProfile.ID,

			Collection: billing.CollectionOverrideConfig{
				Interval: lo.ToPtr(datetime.MustParseDuration(s.T(), "PT1H")),
			},
		})

		require.NoError(t, err)
		require.NotNil(t, createdCustomerOverride)
		// When fetching the override
		customerProfile, err := s.BillingService.GetCustomerOverride(ctx, billing.GetCustomerOverrideInput{
			Customer: customer.CustomerID{
				Namespace: ns,
				ID:        customerID,
			},
		})

		// Then we get the override as the customer profile
		require.NoError(t, err)
		require.NotNil(t, customerProfile)

		wfConfig := customerProfile.MergedProfile.WorkflowConfig

		require.Equal(t, wfConfig.Collection.Interval, datetime.MustParseDuration(s.T(), "PT1H"))
		require.Equal(t, wfConfig.Invoicing.AutoAdvance, true)
		require.Equal(t, wfConfig.Invoicing.DraftPeriod, lo.Must(datetime.ISODurationString("P1D").Parse()))
		require.Equal(t, wfConfig.Invoicing.DueAfter, lo.Must(datetime.ISODurationString("P1W").Parse()))
		require.Equal(t, wfConfig.Payment.CollectionMethod, billing.CollectionMethodChargeAutomatically)
	})
}

func (s *CustomerOverrideTestSuite) TestSanityOverrideOperations() {
	// Given we have an existing customer
	ns := "test-sanity-override-operations"
	ctx := context.Background()

	sandboxApp := s.InstallSandboxApp(s.T(), ns)

	custKey := "johny-the-doe-3"
	cust, err := s.CustomerService.CreateCustomer(ctx, customer.CreateCustomerInput{
		Namespace: ns,
		CustomerMutate: customer.CustomerMutate{
			Name: "Johny the Doe",
			Key:  &custKey,
		},
	})
	require.NoError(s.T(), err)
	require.NotNil(s.T(), cust)

	s.T().Run("delete non-existingoverride", func(t *testing.T) {
		// When deleting a non-existing override
		err := s.BillingService.DeleteCustomerOverride(ctx, billing.DeleteCustomerOverrideInput{
			Customer: customer.CustomerID{
				Namespace: ns,
				ID:        cust.ID,
			},
		})

		// Then we get a NotFoundError
		require.ErrorAs(t, err, &billing.NotFoundError{})
	})

	profileInput := minimalCreateProfileInputTemplate(sandboxApp.GetID())
	profileInput.Namespace = ns

	defaultProfile, err := s.BillingService.CreateProfile(ctx, profileInput)
	require.NoError(s.T(), err)
	require.NotNil(s.T(), defaultProfile)

	s.T().Run("create, delete, create override", func(t *testing.T) {
		// Given we have an override for the customer
		createdCustomerOverride, err := s.BillingService.UpsertCustomerOverride(ctx, billing.UpsertCustomerOverrideInput{
			Namespace:  ns,
			CustomerID: cust.ID,

			Collection: billing.CollectionOverrideConfig{
				Interval: lo.ToPtr(datetime.MustParseDuration(s.T(), "PT1234H")),
			},
		})

		require.NoError(t, err)
		require.NotNil(t, createdCustomerOverride)

		// When deleting the override
		err = s.BillingService.DeleteCustomerOverride(ctx, billing.DeleteCustomerOverrideInput{
			Customer: customer.CustomerID{
				Namespace: ns,
				ID:        cust.ID,
			},
		})

		// Then the override is deleted
		require.NoError(t, err)

		// When fetching the customer profile
		customerProfile, err := s.BillingService.GetCustomerOverride(ctx, billing.GetCustomerOverrideInput{
			Customer: customer.CustomerID{
				Namespace: ns,
				ID:        cust.ID,
			},
		})

		// Then we get a NotFoundError
		require.NoError(t, err)
		require.NotNil(t, customerProfile)
		require.Equal(t, defaultProfile.WorkflowConfig.Collection.Interval, customerProfile.MergedProfile.WorkflowConfig.Collection.Interval)
		require.Nil(t, customerProfile.CustomerOverride, "expect the customer override to be nil, as it has been deleted")

		// When creating the override again
		// Note: this is an implicit update test
		createdCustomerOverride, err = s.BillingService.UpsertCustomerOverride(ctx, billing.UpsertCustomerOverrideInput{
			Namespace:  ns,
			CustomerID: cust.ID,

			Collection: billing.CollectionOverrideConfig{
				Interval: lo.ToPtr(datetime.MustParseDuration(s.T(), "PT48H")),
			},
		})

		// Then the override is created
		require.NoError(t, err)
		require.NotNil(t, createdCustomerOverride)
		require.Equal(t, *createdCustomerOverride.CustomerOverride.Collection.Interval, datetime.MustParseDuration(s.T(), "PT48H"))
	})
}

func (s *CustomerOverrideTestSuite) TestCustomerIntegration() {
	// Given we have an existing customer and a default profile
	ns := "test-customer-integration"
	ctx := context.Background()

	sandboxApp := s.InstallSandboxApp(s.T(), ns)

	custKey := "johny-the-doe-4"
	cust, err := s.CustomerService.CreateCustomer(ctx, customer.CreateCustomerInput{
		Namespace: ns,

		CustomerMutate: customer.CustomerMutate{
			Name: "Johny the Doe",
			Key:  &custKey,
			BillingAddress: &models.Address{
				City:    lo.ToPtr("Berlin"),
				Country: lo.ToPtr(models.CountryCode("DE")),
			},
		},
	})
	require.NoError(s.T(), err)
	require.NotNil(s.T(), cust)

	// Let's fetch the customer again to make sure we have truncated timestamps
	cust, err = s.CustomerService.GetCustomer(ctx, customer.GetCustomerInput{
		CustomerID: &customer.CustomerID{
			Namespace: ns,
			ID:        cust.ID,
		},
	})
	require.NoError(s.T(), err)
	require.NotNil(s.T(), cust)

	profileInput := minimalCreateProfileInputTemplate(sandboxApp.GetID())
	profileInput.Namespace = ns

	defaultProfile, err := s.BillingService.CreateProfile(ctx, profileInput)
	require.NoError(s.T(), err)
	require.NotNil(s.T(), defaultProfile)

	// When querying the customer's billing profile overrides
	customerProfile, err := s.BillingService.GetCustomerOverride(ctx, billing.GetCustomerOverrideInput{
		Customer: customer.CustomerID{
			Namespace: ns,
			ID:        cust.ID,
		},
		Expand: billing.CustomerOverrideExpand{
			Customer: true,
		},
	})

	require.NoError(s.T(), err)
	require.NotNil(s.T(), customerProfile)

	// Then we get the customer object, name and billing address gets overridden
	require.Equal(s.T(), *cust, *customerProfile.Customer)
}

func (s *CustomerOverrideTestSuite) TestNullSetting() {
	// Given we have an existing customer and a default profile and an override
	ns := "test-null-setting"
	ctx := context.Background()

	sandboxApp := s.InstallSandboxApp(s.T(), ns)

	custKey := "johny-the-doe-5"
	cust, err := s.CustomerService.CreateCustomer(ctx, customer.CreateCustomerInput{
		Namespace: ns,

		CustomerMutate: customer.CustomerMutate{
			Name: "Johny the Doe",
			Key:  &custKey,
		},
	})

	require.NoError(s.T(), err)
	require.NotNil(s.T(), cust)

	profileInput := minimalCreateProfileInputTemplate(sandboxApp.GetID())
	profileInput.Namespace = ns

	defaultProfile, err := s.BillingService.CreateProfile(ctx, profileInput)
	require.NoError(s.T(), err)
	require.NotNil(s.T(), defaultProfile)

	createdCustomerOverride, err := s.BillingService.UpsertCustomerOverride(ctx, billing.UpsertCustomerOverrideInput{
		Namespace:  ns,
		CustomerID: cust.ID,

		Collection: billing.CollectionOverrideConfig{
			Interval: lo.ToPtr(datetime.MustParseDuration(s.T(), "PT1H")),
		},
	})
	require.NoError(s.T(), err)
	require.NotNil(s.T(), createdCustomerOverride)

	// When updating the override with null values
	updatedCustomerOverride, err := s.BillingService.UpsertCustomerOverride(ctx, billing.UpsertCustomerOverrideInput{
		Namespace:  ns,
		CustomerID: cust.ID,

		Collection: billing.CollectionOverrideConfig{
			Interval: nil,
		},
	})

	require.NoError(s.T(), err)
	require.NotNil(s.T(), updatedCustomerOverride)

	// Then the override is updated with the null values
	customerProfile, err := s.BillingService.GetCustomerOverride(ctx, billing.GetCustomerOverrideInput{
		Customer: customer.CustomerID{
			Namespace: ns,
			ID:        cust.ID,
		},
	})

	require.NoError(s.T(), err)
	require.NotNil(s.T(), customerProfile)
	require.Nil(s.T(), customerProfile.CustomerOverride.Collection.Interval)
}

func (s *CustomerOverrideTestSuite) TestGetCustomerApp() {
	ns := "test-get-customer-app"
	ctx := context.Background()

	sandboxApp := s.InstallSandboxApp(s.T(), ns)

	// Create a default profile
	defaultProfileCreateInput := minimalCreateProfileInputTemplate(sandboxApp.GetID())
	defaultProfileCreateInput.Namespace = ns

	defaultProfile, err := s.BillingService.CreateProfile(ctx, defaultProfileCreateInput)
	require.NoError(s.T(), err)
	require.NotNil(s.T(), defaultProfile)

	// Create a customer
	custKey := "johny-the-doe-6"
	cust, err := s.CustomerService.CreateCustomer(ctx, customer.CreateCustomerInput{
		Namespace: ns,
		CustomerMutate: customer.CustomerMutate{
			Name: "Johny the Doe",
			Key:  &custKey,
		},
	})
	require.NoError(s.T(), err)
	require.NotNil(s.T(), cust)

	// Get the customer app
	customerApp, err := s.BillingService.GetCustomerApp(ctx, billing.GetCustomerAppInput{
		CustomerID: cust.GetID(),
		AppType:    sandboxApp.GetType(),
	})
	require.NoError(s.T(), err)
	require.NotNil(s.T(), customerApp)

	// Then we get the customer app
	require.Equal(s.T(), sandboxApp.GetID(), customerApp.GetID())
}

func (s *CustomerOverrideTestSuite) TestListCustomerOverrides() {
	ns := "test-list-customer-overrides"
	ctx := context.Background()

	sandboxApp := s.InstallSandboxApp(s.T(), ns)

	// Given we have a default profile and an override profile

	defaultProfileCreateInput := minimalCreateProfileInputTemplate(sandboxApp.GetID())
	defaultProfileCreateInput.Namespace = ns

	defaultProfile, err := s.BillingService.CreateProfile(ctx, defaultProfileCreateInput)
	require.NoError(s.T(), err)

	overrideProfileCreateInput := minimalCreateProfileInputTemplate(sandboxApp.GetID())
	overrideProfileCreateInput.Namespace = ns
	overrideProfileCreateInput.Default = false

	overrideProfile, err := s.BillingService.CreateProfile(ctx, overrideProfileCreateInput)
	require.NoError(s.T(), err)

	customers := []*customer.Customer{}
	for _, name := range []string{"custNoOverride", "custOverride", "custPinnedToDefault"} {
		custKey := name
		cust, err := s.CustomerService.CreateCustomer(ctx, customer.CreateCustomerInput{
			Namespace: ns,
			CustomerMutate: customer.CustomerMutate{
				Name: name,
				Key:  &custKey,
			},
		})
		require.NoError(s.T(), err)
		customers = append(customers, cust)
	}

	// Given we have a customer with no override (uses default profile)

	custNoOverride := customers[0]

	// Given we have a customer with an override (uses override profile)

	custOverrideProfile := customers[1]
	_, err = s.BillingService.UpsertCustomerOverride(ctx, billing.UpsertCustomerOverrideInput{
		Namespace:  ns,
		CustomerID: custOverrideProfile.ID,
		ProfileID:  overrideProfile.ID,
	})
	require.NoError(s.T(), err)

	// Given we have a customer with an override (uses *default* profile)

	custPinnedToDefaultProfile := customers[2]
	_, err = s.BillingService.UpsertCustomerOverride(ctx, billing.UpsertCustomerOverrideInput{
		Namespace:  ns,
		CustomerID: custPinnedToDefaultProfile.ID,
		ProfileID:  defaultProfile.ID,
	})
	require.NoError(s.T(), err)

	// When listing the customer overrides

	tcs := []struct {
		name              string
		listInput         billing.ListCustomerOverridesInput
		expectedCustomers []*customer.Customer
	}{
		{
			// When there's no filter only customers with overrides are returned
			name:      "no filter",
			listInput: billing.ListCustomerOverridesInput{},
			expectedCustomers: []*customer.Customer{
				custOverrideProfile,
				custPinnedToDefaultProfile,
			},
		},
		{
			// When we require all customers, all customers are returned
			name: "all customers",
			listInput: billing.ListCustomerOverridesInput{
				IncludeAllCustomers: true,
			},
			expectedCustomers: customers,
		},
		{
			// When we filter by profile ID, only customers with explicit overrides for that profile are returned
			name: "filter by profile ID",
			listInput: billing.ListCustomerOverridesInput{
				BillingProfiles: []string{defaultProfile.ID},
			},
			expectedCustomers: []*customer.Customer{
				custPinnedToDefaultProfile,
			},
		},
		{
			// When we filter by profile ID, but enable IncludeAllCustomers, for default profile we get all customers that either explicitly use the default profile or not set any profile
			name: "filter by profile ID, include all customers",
			listInput: billing.ListCustomerOverridesInput{
				BillingProfiles:     []string{defaultProfile.ID},
				IncludeAllCustomers: true,
			},
			expectedCustomers: []*customer.Customer{
				custNoOverride,
				custPinnedToDefaultProfile,
			},
		},
		{
			// When we filter by customer name, we get the customers that match the name
			name: "filter by customer name",
			listInput: billing.ListCustomerOverridesInput{
				CustomerName: "override",
			},
			expectedCustomers: []*customer.Customer{
				custOverrideProfile,
			},
		},
		{
			// When we filter by customer name, but don't have any customers that match, we get an empty list
			name: "filter by customer name, no customers match",
			listInput: billing.ListCustomerOverridesInput{
				CustomerName: "noOverride",
			},
			expectedCustomers: []*customer.Customer{},
		},
		{
			// When we filter by customer name and enable IncludeAllCustomers we get a match even if the customer has no override
			name: "filter by customer name, include all customers",
			listInput: billing.ListCustomerOverridesInput{
				CustomerName:        "noOverride",
				IncludeAllCustomers: true,
			},
			expectedCustomers: []*customer.Customer{
				custNoOverride,
			},
		},
		{
			// When we filter by explicit pinnings we only get customers that don't have an explicit override
			name: "filter by explicit pinnings",
			listInput: billing.ListCustomerOverridesInput{
				CustomersWithoutPinnedProfile: true,
			},
			expectedCustomers: []*customer.Customer{
				custNoOverride,
			},
		},
	}

	for _, tc := range tcs {
		s.T().Run(tc.name, func(t *testing.T) {
			tc.listInput.Expand = billing.CustomerOverrideExpand{
				Customer: true,
			}

			tc.listInput.Namespace = ns

			customerOverrides, err := s.BillingService.ListCustomerOverrides(ctx, tc.listInput)
			require.NoError(t, err)

			expectedCustomerNames := lo.Map(tc.expectedCustomers, func(c *customer.Customer, _ int) string {
				return c.Name
			})

			actualCustomerNames := lo.Map(customerOverrides.Items, func(co billing.CustomerOverrideWithDetails, _ int) string {
				return co.Customer.Name
			})

			require.ElementsMatch(t, expectedCustomerNames, actualCustomerNames)
		})
	}
}
