package billing_test

import (
	"context"
	"testing"
	"time"

	"github.com/samber/lo"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/openmeterio/openmeter/openmeter/billing"
	customerentity "github.com/openmeterio/openmeter/openmeter/customer/entity"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/timezone"
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
	require.ErrorAs(s.T(), err, &customerentity.NotFoundError{})
	require.Nil(s.T(), customerEntity)
}

func (s *CustomerOverrideTestSuite) TestDefaultProfileHandling() {
	ns := "test-ns-default-profile-handling"
	ctx := context.Background()

	// Given we have an existing customer
	customer, err := s.CustomerService.CreateCustomer(ctx, customerentity.CreateCustomerInput{
		Namespace: ns,
		Customer: customerentity.Customer{
			ManagedResource: models.NewManagedResource(
				models.ManagedResourceInput{
					Name: "Johny the Doe",
				}),
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
		profileInput := minimalCreateProfileInputTemplate
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
		defaultProfile.WorkflowConfig.CreatedAt = customerProfile.Profile.WorkflowConfig.CreatedAt
		defaultProfile.WorkflowConfig.UpdatedAt = customerProfile.Profile.WorkflowConfig.UpdatedAt

		require.Equal(t, *defaultProfile, customerProfile.Profile)
	})

	s.T().Run("customer with default profile, with override", func(t *testing.T) {
		// Given we have an override for the customer
		createdCustomerOverride, err := s.BillingService.CreateCustomerOverride(ctx, billing.CreateCustomerOverrideInput{
			Namespace:  ns,
			CustomerID: customerID,

			Collection: billing.CollectionOverrideConfig{
				ItemCollectionPeriod: lo.ToPtr(time.Hour),
			},
			Invoicing: billing.InvoicingOverrideConfig{
				AutoAdvance: lo.ToPtr(false),
				DraftPeriod: lo.ToPtr(2 * time.Hour),
				DueAfter:    lo.ToPtr(3 * time.Hour),

				ItemResolution: lo.ToPtr(billing.GranularityResolutionDay),
				ItemPerSubject: lo.ToPtr(true),
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

		require.Equal(t, wfConfig.Collection.ItemCollectionPeriod, time.Hour)
		require.Equal(t, wfConfig.Invoicing.AutoAdvance, false)
		require.Equal(t, wfConfig.Invoicing.DraftPeriod, 2*time.Hour)
		require.Equal(t, wfConfig.Invoicing.DueAfter, 3*time.Hour)
		require.Equal(t, wfConfig.Invoicing.ItemResolution, billing.GranularityResolutionDay)
		require.Equal(t, wfConfig.Invoicing.ItemPerSubject, true)
		require.Equal(t, wfConfig.Payment.CollectionMethod, billing.CollectionMethodSendInvoice)
	})
}

func (s *CustomerOverrideTestSuite) TestPinnedProfileHandling() {
	ns := "test-ns-pinned-profile-handling"
	ctx := context.Background()

	// Given we have an existing customer
	customer, err := s.CustomerService.CreateCustomer(ctx, customerentity.CreateCustomerInput{
		Namespace: ns,
		Customer: customerentity.Customer{
			ManagedResource: models.NewManagedResource(
				models.ManagedResourceInput{
					Name: "Johny the Doe",
				}),
		},
	})
	require.NoError(s.T(), err)
	require.NotNil(s.T(), customer)
	customerID := customer.ID

	// Given we have a non-default profile
	profileInput := minimalCreateProfileInputTemplate
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
				ItemCollectionPeriod: lo.ToPtr(time.Hour),
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

		require.Equal(t, wfConfig.Collection.ItemCollectionPeriod, time.Hour)
		require.Equal(t, wfConfig.Invoicing.AutoAdvance, false)
		require.Equal(t, wfConfig.Invoicing.DraftPeriod, 24*time.Hour)
		require.Equal(t, wfConfig.Invoicing.DueAfter, 30*24*time.Hour)
		require.Equal(t, wfConfig.Invoicing.ItemResolution, billing.GranularityResolutionPeriod)
		require.Equal(t, wfConfig.Invoicing.ItemPerSubject, false)
		require.Equal(t, wfConfig.Payment.CollectionMethod, billing.CollectionMethodChargeAutomatically)
	})
}

func (s *CustomerOverrideTestSuite) TestSanityOverrideOperations() {
	// Given we have an existing customer
	ns := "test-sanity-override-operations"
	ctx := context.Background()

	customer, err := s.CustomerService.CreateCustomer(ctx, customerentity.CreateCustomerInput{
		Namespace: ns,
		Customer: customerentity.Customer{
			ManagedResource: models.NewManagedResource(models.ManagedResourceInput{
				Name: "Johny the Doe",
			}),
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

	s.T().Run("create, delete, create override", func(t *testing.T) {
		// Given we have an override for the customer
		createdCustomerOverride, err := s.BillingService.CreateCustomerOverride(ctx, billing.CreateCustomerOverrideInput{
			Namespace:  ns,
			CustomerID: customer.ID,

			Collection: billing.CollectionOverrideConfig{
				ItemCollectionPeriod: lo.ToPtr(time.Hour),
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
		require.ErrorAs(t, err, &billing.NotFoundError{})
		require.Nil(t, customerProfile)

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
				ItemCollectionPeriod: lo.ToPtr(48 * time.Hour),
			},
		})

		// Then the override is created
		require.NoError(t, err)
		require.NotNil(t, createdCustomerOverride)
		require.Equal(t, *createdCustomerOverride.Collection.ItemCollectionPeriod, 48*time.Hour)
	})
}

func (s *CustomerOverrideTestSuite) TestCustomerIntegration() {
	// Given we have an existing customer and a default profile
	ns := "test-customer-integration"
	ctx := context.Background()

	customer, err := s.CustomerService.CreateCustomer(ctx, customerentity.CreateCustomerInput{
		Namespace: ns,

		Customer: customerentity.Customer{
			ManagedResource: models.NewManagedResource(models.ManagedResourceInput{
				Name: "Johny the Doe",
			}),
			Timezone: lo.ToPtr(timezone.Timezone("Europe/Berlin")),
			BillingAddress: &models.Address{
				City:    lo.ToPtr("Berlin"),
				Country: lo.ToPtr(models.CountryCode("DE")),
			},
		},
	})

	require.NoError(s.T(), err)
	require.NotNil(s.T(), customer)

	profileInput := minimalCreateProfileInputTemplate
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

	// Then we get the customer object, also the timezone gets overridden in the workflow configuration
	require.Equal(s.T(), customer.Name, customerProfile.Customer.Name)
	require.Equal(s.T(), customer.Timezone, customerProfile.Customer.Timezone)
	require.Equal(s.T(), customer.BillingAddress, customerProfile.Customer.BillingAddress)
	require.Equal(s.T(), customer.Timezone, customerProfile.Profile.WorkflowConfig.Timezone)
}

func (s *CustomerOverrideTestSuite) TestNullSetting() {
	// Given we have an existing customer and a default profile and an override
	ns := "test-null-setting"
	ctx := context.Background()

	customer, err := s.CustomerService.CreateCustomer(ctx, customerentity.CreateCustomerInput{
		Namespace: ns,

		Customer: customerentity.Customer{
			ManagedResource: models.NewManagedResource(models.ManagedResourceInput{
				Name: "Johny the Doe",
			}),
		},
	})

	require.NoError(s.T(), err)
	require.NotNil(s.T(), customer)

	profileInput := minimalCreateProfileInputTemplate
	profileInput.Namespace = ns

	defaultProfile, err := s.BillingService.CreateProfile(ctx, profileInput)
	require.NoError(s.T(), err)
	require.NotNil(s.T(), defaultProfile)

	createdCustomerOverride, err := s.BillingService.CreateCustomerOverride(ctx, billing.CreateCustomerOverrideInput{
		Namespace:  ns,
		CustomerID: customer.ID,

		Collection: billing.CollectionOverrideConfig{
			ItemCollectionPeriod: lo.ToPtr(time.Hour),
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
			ItemCollectionPeriod: nil,
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
	require.Nil(s.T(), customerProfile.Collection.ItemCollectionPeriod)
}
