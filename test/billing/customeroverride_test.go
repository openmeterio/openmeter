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
	ctx := context.Background()

	s.InstallSandboxApp(s.T(), ns)
	profileInput := MinimalCreateProfileInputTemplate
	profileInput.Namespace = ns

	_, err := s.BillingService.CreateProfile(ctx, profileInput)
	require.NoError(s.T(), err)

	// When querying the customer's billing profile overrides
	customerEntity, err := s.BillingService.GetCustomerOverride(ctx, billing.GetCustomerOverrideInput{
		Customer: customer.CustomerID{
			Namespace: ns,
			ID:        nonExistingCustomerID,
		},
	})

	// Then we get a customer not found error
	require.True(s.T(), customer.IsNotFoundError(err), "expect a customer not found error")
	require.Empty(s.T(), customerEntity)
}

func (s *CustomerOverrideTestSuite) TestDefaultProfileHandling() {
	ns := "test-ns-default-profile-handling"
	ctx := context.Background()

	_ = s.InstallSandboxApp(s.T(), ns)

	// Given we have an existing customer
	cust, err := s.CustomerService.CreateCustomer(ctx, customer.CreateCustomerInput{
		Namespace: ns,
		CustomerMutate: customer.CustomerMutate{
			Name: "Johny the Doe",
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
		profileInput := MinimalCreateProfileInputTemplate
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
	cust, err := s.CustomerService.CreateCustomer(ctx, customer.CreateCustomerInput{
		Namespace: ns,
		CustomerMutate: customer.CustomerMutate{
			Name: "Johny the Doe",
		},
	})
	require.NoError(s.T(), err)
	require.NotNil(s.T(), cust)
	customerID := cust.ID

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
				Interval: lo.ToPtr(isodate.MustParse(s.T(), "PT1H")),
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

	cust, err := s.CustomerService.CreateCustomer(ctx, customer.CreateCustomerInput{
		Namespace: ns,
		CustomerMutate: customer.CustomerMutate{
			Name: "Johny the Doe",
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

	profileInput := MinimalCreateProfileInputTemplate
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
				Interval: lo.ToPtr(isodate.MustParse(s.T(), "PT1234H")),
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
				Interval: lo.ToPtr(isodate.MustParse(s.T(), "PT48H")),
			},
		})

		// Then the override is created
		require.NoError(t, err)
		require.NotNil(t, createdCustomerOverride)
		require.Equal(t, *createdCustomerOverride.CustomerOverride.Collection.Interval, isodate.MustParse(s.T(), "PT48H"))
	})
}

func (s *CustomerOverrideTestSuite) TestCustomerIntegration() {
	// Given we have an existing customer and a default profile
	ns := "test-customer-integration"
	ctx := context.Background()

	_ = s.InstallSandboxApp(s.T(), ns)

	cust, err := s.CustomerService.CreateCustomer(ctx, customer.CreateCustomerInput{
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
	require.NotNil(s.T(), cust)

	// Let's fetch the customer again to make sure we have truncated timestamps
	cust, err = s.CustomerService.GetCustomer(ctx, customer.GetCustomerInput{
		Namespace: ns,
		ID:        cust.ID,
	})
	require.NoError(s.T(), err)
	require.NotNil(s.T(), cust)

	profileInput := MinimalCreateProfileInputTemplate
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

	_ = s.InstallSandboxApp(s.T(), ns)

	cust, err := s.CustomerService.CreateCustomer(ctx, customer.CreateCustomerInput{
		Namespace: ns,

		CustomerMutate: customer.CustomerMutate{
			Name: "Johny the Doe",
		},
	})

	require.NoError(s.T(), err)
	require.NotNil(s.T(), cust)

	profileInput := MinimalCreateProfileInputTemplate
	profileInput.Namespace = ns

	defaultProfile, err := s.BillingService.CreateProfile(ctx, profileInput)
	require.NoError(s.T(), err)
	require.NotNil(s.T(), defaultProfile)

	createdCustomerOverride, err := s.BillingService.UpsertCustomerOverride(ctx, billing.UpsertCustomerOverrideInput{
		Namespace:  ns,
		CustomerID: cust.ID,

		Collection: billing.CollectionOverrideConfig{
			Interval: lo.ToPtr(isodate.MustParse(s.T(), "PT1H")),
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
