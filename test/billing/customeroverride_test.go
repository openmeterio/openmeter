package billing

import (
	"context"
	"testing"

	"github.com/samber/lo"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/pkg/isodate"
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

	// When querying the customer's billing profile overrides
	customerEntity, err := s.BillingService.GetProfileWithCustomerOverride(context.Background(), billing.GetProfileWithCustomerOverrideInput{
		Namespace:  ns,
		CustomerID: nonExistingCustomerID,
	})

	// Then we get a customer not found error
	require.True(s.T(), customer.IsNotFoundError(err), "expect a customer not found error")
	require.Nil(s.T(), customerEntity)
}

func (s *CustomerOverrideTestSuite) TestDefaultProfileHandling() {
	ns := "test-ns-default-profile-handling"
	ctx := context.Background()

	_ = s.InstallSandboxApp(s.T(), ns)

	// Given we have an existing customer
	customer, err := s.CustomerService.CreateCustomer(ctx, customer.CreateCustomerInput{
		Namespace: ns,
		CustomerMutate: customer.CustomerMutate{
			Name: "Johny the Doe",
		},
	})
	require.NoError(s.T(), err)
	require.NotNil(s.T(), customer)
	customerID := customer.ID

	s.T().Run("customer without default profile, no override", func(t *testing.T) {
		// When not having a default profile
		// Then we get a NotFoundError
		profileWithOverride, err := s.BillingService.GetProfileWithCustomerOverride(ctx, billing.GetProfileWithCustomerOverrideInput{
			Namespace:  ns,
			CustomerID: customerID,
		})
		require.ErrorIs(t, err, billing.ErrDefaultProfileNotFound)
		require.ErrorAs(t, err, &billing.NotFoundError{})
		require.Nil(t, profileWithOverride)
	})

	var defaultProfile *billing.Profile

	s.T().Run("customer with default profile, no override", func(t *testing.T) {
		// Given having a default profile
		profileInput := MinimalCreateProfileInputTemplate
		profileInput.Namespace = ns

		defaultProfile, err = s.BillingService.CreateProfile(ctx, profileInput)
		require.NoError(t, err)
		require.NotNil(t, defaultProfile)

		// When fetching the profile
		customerProfile, err := s.BillingService.GetProfileWithCustomerOverride(ctx, billing.GetProfileWithCustomerOverrideInput{
			Namespace:  ns,
			CustomerID: customerID,
		})

		// Then we get the default profile as the customer profile
		require.NoError(t, err)
		require.NotNil(t, customerProfile)

		defaultProfile.CreatedAt = customerProfile.Profile.CreatedAt
		defaultProfile.UpdatedAt = customerProfile.Profile.UpdatedAt

		require.Equal(t, *defaultProfile, customerProfile.Profile)
	})

	s.T().Run("customer with default profile, with override", func(t *testing.T) {
		// Given we have an override for the customer
		createdCustomerOverride, err := s.BillingService.CreateCustomerOverride(ctx, billing.CreateCustomerOverrideInput{
			Namespace:  ns,
			CustomerID: customerID,

			Collection: billing.CollectionOverrideConfig{
				Interval: lo.ToPtr(isodate.MustParse(s.T(), "PT1H")),
			},
			Invoicing: billing.InvoicingOverrideConfig{
				AutoAdvance: lo.ToPtr(false),
				DraftPeriod: lo.ToPtr(isodate.MustParse(s.T(), "PT2H")),
				DueAfter:    lo.ToPtr(isodate.MustParse(s.T(), "PT3H")),
			},
			Payment: billing.PaymentOverrideConfig{
				CollectionMethod: lo.ToPtr(billing.CollectionMethodSendInvoice),
			},
		})

		require.NoError(t, err)
		require.NotNil(t, createdCustomerOverride)
		// When fetching the override
		customerProfile, err := s.BillingService.GetProfileWithCustomerOverride(ctx, billing.GetProfileWithCustomerOverrideInput{
			Namespace:  ns,
			CustomerID: customerID,
		})

		// Then we get the override as the customer profile
		require.NoError(t, err)
		require.NotNil(t, customerProfile)

		wfConfig := customerProfile.Profile.WorkflowConfig

		require.Equal(t, wfConfig.Collection.Interval, isodate.MustParse(t, "PT1H"))
		require.Equal(t, wfConfig.Invoicing.AutoAdvance, false)
		require.Equal(t, wfConfig.Invoicing.DraftPeriod, isodate.MustParse(t, "PT2H"))
		require.Equal(t, wfConfig.Invoicing.DueAfter, isodate.MustParse(t, "PT3H"))
		require.Equal(t, wfConfig.Payment.CollectionMethod, billing.CollectionMethodSendInvoice)
	})
}

func (s *CustomerOverrideTestSuite) TestPinnedProfileHandling() {
	ns := "test-ns-pinned-profile-handling"
	ctx := context.Background()

	_ = s.InstallSandboxApp(s.T(), ns)

	// Given we have an existing customer
	customer, err := s.CustomerService.CreateCustomer(ctx, customer.CreateCustomerInput{
		Namespace: ns,
		CustomerMutate: customer.CustomerMutate{
			Name: "Johny the Doe",
		},
	})
	require.NoError(s.T(), err)
	require.NotNil(s.T(), customer)
	customerID := customer.ID

	// Given we have a non-default profile
	profileInput := MinimalCreateProfileInputTemplate
	profileInput.Namespace = ns
	profileInput.Default = false

	pinnedProfile, err := s.BillingService.CreateProfile(ctx, profileInput)
	require.NoError(s.T(), err)
	require.NotNil(s.T(), pinnedProfile)

	s.T().Run("customer without default profile, no profile override", func(t *testing.T) {
		// When not having a default profile
		// Then we get a NotFoundError
		profileWithOverride, err := s.BillingService.GetProfileWithCustomerOverride(ctx, billing.GetProfileWithCustomerOverrideInput{
			Namespace:  ns,
			CustomerID: customerID,
		})
		require.ErrorIs(t, err, billing.ErrDefaultProfileNotFound)
		require.ErrorAs(t, err, &billing.NotFoundError{})
		require.Nil(t, profileWithOverride)
	})

	s.T().Run("customer without default profile, with override", func(t *testing.T) {
		// Given we have an override for the customer
		createdCustomerOverride, err := s.BillingService.CreateCustomerOverride(ctx, billing.CreateCustomerOverrideInput{
			Namespace:  ns,
			CustomerID: customerID,
			ProfileID:  pinnedProfile.ID,

			Collection: billing.CollectionOverrideConfig{
				Interval: lo.ToPtr(isodate.MustParse(s.T(), "PT1H")),
			},
		})

		require.NoError(t, err)
		require.NotNil(t, createdCustomerOverride)
		// When fetching the override
		customerProfile, err := s.BillingService.GetProfileWithCustomerOverride(ctx, billing.GetProfileWithCustomerOverrideInput{
			Namespace:  ns,
			CustomerID: customerID,
		})

		// Then we get the override as the customer profile
		require.NoError(t, err)
		require.NotNil(t, customerProfile)

		wfConfig := customerProfile.Profile.WorkflowConfig

		require.Equal(t, wfConfig.Collection.Interval, isodate.MustParse(s.T(), "PT1H"))
		require.Equal(t, wfConfig.Invoicing.AutoAdvance, true)
		require.Equal(t, wfConfig.Invoicing.DraftPeriod, lo.Must(isodate.String("P1D").Parse()))
		require.Equal(t, wfConfig.Invoicing.DueAfter, lo.Must(isodate.String("P1W").Parse()))
		require.Equal(t, wfConfig.Payment.CollectionMethod, billing.CollectionMethodChargeAutomatically)
	})
}

func (s *CustomerOverrideTestSuite) TestSanityOverrideOperations() {
	// Given we have an existing customer
	ns := "test-sanity-override-operations"
	ctx := context.Background()

	s.InstallSandboxApp(s.T(), ns)

	customer, err := s.CustomerService.CreateCustomer(ctx, customer.CreateCustomerInput{
		Namespace: ns,
		CustomerMutate: customer.CustomerMutate{
			Name: "Johny the Doe",
		},
	})
	require.NoError(s.T(), err)
	require.NotNil(s.T(), customer)

	s.T().Run("delete non-existingoverride", func(t *testing.T) {
		// When deleting a non-existing override
		err := s.BillingService.DeleteCustomerOverride(ctx, billing.DeleteCustomerOverrideInput{
			Namespace:  ns,
			CustomerID: customer.ID,
		})

		// Then we get a NotFoundError
		require.ErrorAs(t, err, &billing.NotFoundError{})
	})

	profileInput := MinimalCreateProfileInputTemplate
	profileInput.Namespace = ns

	defaultProfile, err := s.BillingService.CreateProfile(ctx, profileInput)
	require.NoError(s.T(), err)
	require.NotNil(s.T(), defaultProfile)

	s.T().Run("create, delete, create override", func(t *testing.T) {
		// Given we have an override for the customer
		createdCustomerOverride, err := s.BillingService.CreateCustomerOverride(ctx, billing.CreateCustomerOverrideInput{
			Namespace:  ns,
			CustomerID: customer.ID,

			Collection: billing.CollectionOverrideConfig{
				Interval: lo.ToPtr(isodate.MustParse(s.T(), "PT1234H")),
			},
		})

		require.NoError(t, err)
		require.NotNil(t, createdCustomerOverride)

		// When deleting the override
		err = s.BillingService.DeleteCustomerOverride(ctx, billing.DeleteCustomerOverrideInput{
			Namespace:  ns,
			CustomerID: customer.ID,
		})

		// Then the override is deleted
		require.NoError(t, err)

		// When fetching the customer profile
		customerProfile, err := s.BillingService.GetProfileWithCustomerOverride(ctx, billing.GetProfileWithCustomerOverrideInput{
			Namespace:  ns,
			CustomerID: customer.ID,
		})

		// Then we get a NotFoundError
		require.NoError(t, err)
		require.NotNil(t, customerProfile)
		require.Equal(t, defaultProfile.WorkflowConfig.Collection.Interval, customerProfile.Profile.WorkflowConfig.Collection.Interval)

		// When fetching the override
		_, err = s.BillingService.GetCustomerOverride(ctx, billing.GetCustomerOverrideInput{
			Namespace:  ns,
			CustomerID: customer.ID,
		})

		// Then we get a NotFoundError
		require.ErrorAs(t, err, &billing.NotFoundError{})

		// When creating the override again
		// Note: this is an implicit update test
		createdCustomerOverride, err = s.BillingService.CreateCustomerOverride(ctx, billing.CreateCustomerOverrideInput{
			Namespace:  ns,
			CustomerID: customer.ID,

			Collection: billing.CollectionOverrideConfig{
				Interval: lo.ToPtr(isodate.MustParse(s.T(), "PT48H")),
			},
		})

		// Then the override is created
		require.NoError(t, err)
		require.NotNil(t, createdCustomerOverride)
		require.Equal(t, *createdCustomerOverride.Collection.Interval, isodate.MustParse(s.T(), "PT48H"))
	})
}

func (s *CustomerOverrideTestSuite) TestCustomerIntegration() {
	// Given we have an existing customer and a default profile
	ns := "test-customer-integration"
	ctx := context.Background()

	_ = s.InstallSandboxApp(s.T(), ns)

	customer, err := s.CustomerService.CreateCustomer(ctx, customer.CreateCustomerInput{
		Namespace: ns,

		CustomerMutate: customer.CustomerMutate{
			Name: "Johny the Doe",
			BillingAddress: &models.Address{
				City:    lo.ToPtr("Berlin"),
				Country: lo.ToPtr(models.CountryCode("DE")),
			},
		},
	})

	require.NoError(s.T(), err)
	require.NotNil(s.T(), customer)

	profileInput := MinimalCreateProfileInputTemplate
	profileInput.Namespace = ns

	defaultProfile, err := s.BillingService.CreateProfile(ctx, profileInput)
	require.NoError(s.T(), err)
	require.NotNil(s.T(), defaultProfile)

	// When querying the customer's billing profile overrides
	customerProfile, err := s.BillingService.GetProfileWithCustomerOverride(ctx, billing.GetProfileWithCustomerOverrideInput{
		Namespace:  ns,
		CustomerID: customer.ID,
	})

	require.NoError(s.T(), err)
	require.NotNil(s.T(), customerProfile)

	// Then we get the customer object, name and billing address gets overridden
	require.Equal(s.T(), customer.Name, customerProfile.Customer.Name)
	require.Equal(s.T(), customer.BillingAddress, customerProfile.Customer.BillingAddress)
}

func (s *CustomerOverrideTestSuite) TestNullSetting() {
	// Given we have an existing customer and a default profile and an override
	ns := "test-null-setting"
	ctx := context.Background()

	_ = s.InstallSandboxApp(s.T(), ns)

	customer, err := s.CustomerService.CreateCustomer(ctx, customer.CreateCustomerInput{
		Namespace: ns,

		CustomerMutate: customer.CustomerMutate{
			Name: "Johny the Doe",
		},
	})

	require.NoError(s.T(), err)
	require.NotNil(s.T(), customer)

	profileInput := MinimalCreateProfileInputTemplate
	profileInput.Namespace = ns

	defaultProfile, err := s.BillingService.CreateProfile(ctx, profileInput)
	require.NoError(s.T(), err)
	require.NotNil(s.T(), defaultProfile)

	createdCustomerOverride, err := s.BillingService.CreateCustomerOverride(ctx, billing.CreateCustomerOverrideInput{
		Namespace:  ns,
		CustomerID: customer.ID,

		Collection: billing.CollectionOverrideConfig{
			Interval: lo.ToPtr(isodate.MustParse(s.T(), "PT1H")),
		},
	})
	require.NoError(s.T(), err)
	require.NotNil(s.T(), createdCustomerOverride)

	// When updating the override with null values
	updatedCustomerOverride, err := s.BillingService.UpdateCustomerOverride(ctx, billing.UpdateCustomerOverrideInput{
		Namespace:  ns,
		CustomerID: customer.ID,
		UpdatedAt:  createdCustomerOverride.UpdatedAt,

		Collection: billing.CollectionOverrideConfig{
			Interval: nil,
		},
	})

	require.NoError(s.T(), err)
	require.NotNil(s.T(), updatedCustomerOverride)

	// Then the override is updated with the null values
	customerProfile, err := s.BillingService.GetCustomerOverride(ctx, billing.GetCustomerOverrideInput{
		Namespace:  ns,
		CustomerID: customer.ID,
	})

	require.NoError(s.T(), err)
	require.NotNil(s.T(), customerProfile)
	require.Nil(s.T(), customerProfile.Collection.Interval)
}
