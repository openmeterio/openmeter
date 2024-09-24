package billing_test

import (
	"context"
	"testing"
	"time"

	"github.com/samber/lo"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/provider"
	"github.com/openmeterio/openmeter/pkg/models"
)

var minimalCreateProfileInputTemplate = billing.CreateProfileInput{
	Default: true,

	TaxConfiguration: provider.TaxConfiguration{
		Type: provider.TaxProviderOpenMeterSandbox,
	},

	InvoicingConfiguration: provider.InvoicingConfiguration{
		Type: provider.InvoicingProviderOpenMeterSandbox,
	},

	PaymentConfiguration: provider.PaymentConfiguration{
		Type: provider.PaymentProviderOpenMeterSandbox,
	},

	WorkflowConfig: billing.WorkflowConfig{
		Collection: billing.CollectionConfig{
			Alignment: billing.AlignmentKindSubscription,
		},
	},

	Supplier: billing.SupplierContact{
		Name: "Awesome Supplier",
		Address: models.Address{
			Country: lo.ToPtr(models.CountryCode("US")),
		},
	},
}

type ProfileTestSuite struct {
	BaseSuite
}

func TestProfile(t *testing.T) {
	suite.Run(t, new(ProfileTestSuite))
}

func (s *ProfileTestSuite) TestProfileLifecycle() {
	ctx := context.Background()
	ns := "test_create_billing_profile"

	s.T().Run("missing default profile", func(t *testing.T) {
		defaultProfile, err := s.BillingService.GetDefaultProfile(ctx, billing.GetDefaultProfileInput{
			Namespace: ns,
		})
		require.NoError(t, err)
		require.Nil(t, defaultProfile)
	})

	var profile *billing.Profile
	var err error

	minimalCreateProfileInput := minimalCreateProfileInputTemplate
	minimalCreateProfileInput.Namespace = ns

	s.T().Run("create default profile", func(t *testing.T) {
		profile, err = s.BillingService.CreateProfile(ctx, minimalCreateProfileInput)

		require.NoError(t, err)
		require.NotNil(t, profile)
	})

	profile.CreatedAt = profile.CreatedAt.Truncate(time.Microsecond)
	profile.UpdatedAt = profile.UpdatedAt.Truncate(time.Microsecond)
	profile.WorkflowConfig.CreatedAt = profile.WorkflowConfig.CreatedAt.Truncate(time.Microsecond)
	profile.WorkflowConfig.UpdatedAt = profile.WorkflowConfig.UpdatedAt.Truncate(time.Microsecond)

	s.T().Run("fetching the default profile is possible", func(t *testing.T) {
		defaultProfile, err := s.BillingService.GetDefaultProfile(ctx, billing.GetDefaultProfileInput{
			Namespace: ns,
		})
		require.NoError(t, err)
		require.NotNil(t, defaultProfile)
		require.Equal(t, profile, defaultProfile)
	})

	s.T().Run("creating a second default profile fails", func(t *testing.T) {
		_, err := s.BillingService.CreateProfile(ctx, minimalCreateProfileInput)
		require.Error(t, err)
		require.ErrorIs(t, err, billing.ErrDefaultProfileAlreadyExists)
	})

	s.T().Run("fetching the profile by id", func(t *testing.T) {
		fetchedProfile, err := s.BillingService.GetProfile(ctx, billing.GetProfileInput{
			Namespace: ns,
			ID:        profile.ID,
		})

		require.NoError(t, err)
		require.Equal(t, profile, fetchedProfile)
	})

	s.T().Run("deleted profile handling", func(t *testing.T) {
		require.NoError(t, s.BillingService.DeleteProfile(ctx, billing.DeleteProfileInput{
			Namespace: ns,
			ID:        profile.ID,
		}))

		t.Run("deleting a profile twice yields an error", func(t *testing.T) {
			require.ErrorIs(t, s.BillingService.DeleteProfile(ctx, billing.DeleteProfileInput{
				Namespace: ns,
				ID:        profile.ID,
			}), billing.ErrProfileAlreadyDeleted)
		})

		t.Run("fetching a deleted profile by id returns the profile", func(t *testing.T) {
			fetchedProfile, err := s.BillingService.GetProfile(ctx, billing.GetProfileInput{
				Namespace: ns,
				ID:        profile.ID,
			})

			require.NoError(t, err)
			require.Equal(t, profile.ID, fetchedProfile.ID)
		})

		t.Run("same profile can be created after the previous one was deleted", func(t *testing.T) {
			profile, err = s.BillingService.CreateProfile(ctx, minimalCreateProfileInput)

			require.NoError(t, err)
			require.NotNil(t, profile)
		})
	})
}

func (s *ProfileTestSuite) TestProfileFieldSetting() {
	ctx := context.Background()
	ns := "test_profile_field_setting"

	input := billing.CreateProfileInput{
		Namespace: ns,
		Default:   true,

		TaxConfiguration: provider.TaxConfiguration{
			Type: provider.TaxProviderOpenMeterSandbox,
		},

		InvoicingConfiguration: provider.InvoicingConfiguration{
			Type: provider.InvoicingProviderOpenMeterSandbox,
		},

		PaymentConfiguration: provider.PaymentConfiguration{
			Type: provider.PaymentProviderOpenMeterSandbox,
		},

		WorkflowConfig: billing.WorkflowConfig{
			Collection: billing.CollectionConfig{
				Alignment:            billing.AlignmentKindSubscription,
				ItemCollectionPeriod: 30 * time.Minute,
			},
			Invoicing: billing.InvoicingConfig{
				AutoAdvance: true,
				DraftPeriod: 1 * time.Hour,
				DueAfter:    24 * time.Hour,

				ItemResolution: billing.GranularityResolutionDay,
				ItemPerSubject: true,
			},
			Payment: billing.PaymentConfig{
				CollectionMethod: billing.CollectionMethodSendInvoice,
			},
		},

		Supplier: billing.SupplierContact{
			Name: "Awesome Supplier",
			Address: models.Address{
				Country:     lo.ToPtr(models.CountryCode("US")),
				PostalCode:  lo.ToPtr("12345"),
				City:        lo.ToPtr("City"),
				State:       lo.ToPtr("State"),
				Line1:       lo.ToPtr("Line 1"),
				Line2:       lo.ToPtr("Line 2"),
				PhoneNumber: lo.ToPtr("1234567890"),
			},
		},
	}

	profile, err := s.BillingService.CreateProfile(ctx, input)
	require.NoError(s.T(), err)
	require.NotNil(s.T(), profile)

	profile.CreatedAt = profile.CreatedAt.Truncate(time.Microsecond)
	profile.UpdatedAt = profile.UpdatedAt.Truncate(time.Microsecond)
	profile.WorkflowConfig.CreatedAt = profile.WorkflowConfig.CreatedAt.Truncate(time.Microsecond)
	profile.WorkflowConfig.UpdatedAt = profile.WorkflowConfig.UpdatedAt.Truncate(time.Microsecond)

	// Let's fetch the profile again
	fetchedProfile, err := s.BillingService.GetProfile(ctx, billing.GetProfileInput{
		Namespace: ns,
		ID:        profile.ID,
	})

	// Sanity check db conversion & fetching
	require.NoError(s.T(), err)
	require.Equal(s.T(), profile, fetchedProfile)

	// let's add the db derived fields to the input
	input.ID = profile.ID
	input.CreatedAt = fetchedProfile.CreatedAt
	input.UpdatedAt = fetchedProfile.UpdatedAt
	input.DeletedAt = fetchedProfile.DeletedAt
	input.WorkflowConfig.ID = fetchedProfile.WorkflowConfig.ID
	input.WorkflowConfig.CreatedAt = fetchedProfile.WorkflowConfig.CreatedAt
	input.WorkflowConfig.UpdatedAt = fetchedProfile.WorkflowConfig.UpdatedAt
	input.WorkflowConfig.DeletedAt = fetchedProfile.WorkflowConfig.DeletedAt

	// Let's check if the fields are set correctly
	require.Equal(s.T(), billing.Profile(input), *fetchedProfile)
}

func (s *ProfileTestSuite) TestProfileUpdates() {
	// Given a profile
	ctx := context.Background()
	ns := "test_profile_updates"

	input := billing.CreateProfileInput{
		Namespace: ns,
		Default:   true,

		TaxConfiguration: provider.TaxConfiguration{
			Type: provider.TaxProviderOpenMeterSandbox,
		},

		InvoicingConfiguration: provider.InvoicingConfiguration{
			Type: provider.InvoicingProviderOpenMeterSandbox,
		},

		PaymentConfiguration: provider.PaymentConfiguration{
			Type: provider.PaymentProviderOpenMeterSandbox,
		},

		WorkflowConfig: billing.WorkflowConfig{
			Collection: billing.CollectionConfig{
				Alignment:            billing.AlignmentKindSubscription,
				ItemCollectionPeriod: 30 * time.Minute,
			},
			Invoicing: billing.InvoicingConfig{
				AutoAdvance: true,
				DraftPeriod: 1 * time.Hour,
				DueAfter:    24 * time.Hour,

				ItemResolution: billing.GranularityResolutionDay,
				ItemPerSubject: true,
			},
			Payment: billing.PaymentConfig{
				CollectionMethod: billing.CollectionMethodSendInvoice,
			},
		},

		Supplier: billing.SupplierContact{
			Name: "Awesome Supplier",
			Address: models.Address{
				Country:     lo.ToPtr(models.CountryCode("US")),
				PostalCode:  lo.ToPtr("12345"),
				City:        lo.ToPtr("City"),
				State:       lo.ToPtr("State"),
				Line1:       lo.ToPtr("Line 1"),
				Line2:       lo.ToPtr("Line 2"),
				PhoneNumber: lo.ToPtr("1234567890"),
			},
		},
	}

	profile, err := s.BillingService.CreateProfile(ctx, input)
	require.NoError(s.T(), err)
	require.NotNil(s.T(), profile)

	profile.CreatedAt = profile.CreatedAt.Truncate(time.Microsecond)
	profile.UpdatedAt = profile.UpdatedAt.Truncate(time.Microsecond)
	profile.WorkflowConfig.CreatedAt = profile.WorkflowConfig.CreatedAt.Truncate(time.Microsecond)
	profile.WorkflowConfig.UpdatedAt = profile.WorkflowConfig.UpdatedAt.Truncate(time.Microsecond)

	// Let's fetch the profile again
	fetchedProfile, err := s.BillingService.GetProfile(ctx, billing.GetProfileInput{
		Namespace: ns,
		ID:        profile.ID,
	})

	// Sanity check db conversion & fetching
	require.NoError(s.T(), err)
	require.Equal(s.T(), profile, fetchedProfile)

	s.T().Run("update profile", func(t *testing.T) {
		// When updating the profile
		updateInput := billing.UpdateProfileInput{
			ID:        profile.ID,
			Namespace: ns,
			Default:   true,
			CreatedAt: profile.CreatedAt,

			UpdatedAt: profile.UpdatedAt,

			TaxConfiguration: provider.TaxConfiguration{
				Type: provider.TaxProviderOpenMeterSandbox,
			},

			InvoicingConfiguration: provider.InvoicingConfiguration{
				Type: provider.InvoicingProviderOpenMeterSandbox,
			},

			PaymentConfiguration: provider.PaymentConfiguration{
				Type: provider.PaymentProviderOpenMeterSandbox,
			},

			WorkflowConfig: billing.WorkflowConfig{
				CreatedAt: profile.WorkflowConfig.CreatedAt,
				Collection: billing.CollectionConfig{
					Alignment:            billing.AlignmentKindSubscription,
					ItemCollectionPeriod: 60 * time.Minute,
				},
				Invoicing: billing.InvoicingConfig{
					AutoAdvance: false,
					DraftPeriod: 2 * time.Hour,
					DueAfter:    48 * time.Hour,

					ItemResolution: billing.GranularityResolutionPeriod,
					ItemPerSubject: false,
				},
				Payment: billing.PaymentConfig{
					CollectionMethod: billing.CollectionMethodChargeAutomatically,
				},
			},

			Supplier: billing.SupplierContact{
				Name: "Awesome Supplier [update]",
				Address: models.Address{
					Country:     lo.ToPtr(models.CountryCode("HU")),
					PostalCode:  lo.ToPtr("12345 [update]"),
					City:        lo.ToPtr("City [update]"),
					State:       lo.ToPtr("State [update]"),
					Line1:       lo.ToPtr("Line 1 [update]"),
					Line2:       lo.ToPtr("Line 2 [update]"),
					PhoneNumber: lo.ToPtr("1234567890 [update]"),
				},
			},
		}

		updatedProfile, err := s.BillingService.UpdateProfile(ctx, updateInput)

		// Then the profile is updated

		require.NoError(t, err)
		require.NotNil(t, updatedProfile)

		require.NotEqual(t, fetchedProfile.UpdatedAt, updatedProfile.UpdatedAt, "the new updated at is returned")

		// Set up DB only fields
		expecedOutput := billing.Profile(updateInput)
		expecedOutput.WorkflowConfig.ID = fetchedProfile.WorkflowConfig.ID
		expecedOutput.UpdatedAt = updatedProfile.UpdatedAt                               // This is checked by the previous assertion
		expecedOutput.WorkflowConfig.UpdatedAt = updatedProfile.WorkflowConfig.UpdatedAt // This is checked by the previous assertion

		require.Equal(t, expecedOutput, *updatedProfile)
	})

	s.T().Run("update profile, out of date input", func(t *testing.T) {
		// When updating the profile with and old updated at
		updateInput := billing.UpdateProfileInput{
			ID:        profile.ID,
			Namespace: ns,
			Default:   true,
			CreatedAt: profile.CreatedAt,

			UpdatedAt: profile.UpdatedAt,

			TaxConfiguration: provider.TaxConfiguration{
				Type: provider.TaxProviderStripeTax,
			},

			InvoicingConfiguration: provider.InvoicingConfiguration{
				Type: provider.InvoicingProviderStripeInvoicing,
			},

			PaymentConfiguration: provider.PaymentConfiguration{
				Type: provider.PaymentProviderStripePayments,
			},

			WorkflowConfig: billing.WorkflowConfig{
				CreatedAt: profile.WorkflowConfig.CreatedAt,
				Collection: billing.CollectionConfig{
					Alignment:            billing.AlignmentKindSubscription,
					ItemCollectionPeriod: 60 * time.Minute,
				},
				Invoicing: billing.InvoicingConfig{
					AutoAdvance: false,
					DraftPeriod: 2 * time.Hour,
					DueAfter:    48 * time.Hour,

					ItemResolution: billing.GranularityResolutionPeriod,
					ItemPerSubject: false,
				},
				Payment: billing.PaymentConfig{
					CollectionMethod: billing.CollectionMethodChargeAutomatically,
				},
			},

			Supplier: billing.SupplierContact{
				Name: "Awesome Supplier [update]",
				Address: models.Address{
					Country:     lo.ToPtr(models.CountryCode("HU")),
					PostalCode:  lo.ToPtr("12345 [update]"),
					City:        lo.ToPtr("City [update]"),
					State:       lo.ToPtr("State [update]"),
					Line1:       lo.ToPtr("Line 1 [update]"),
					Line2:       lo.ToPtr("Line 2 [update]"),
					PhoneNumber: lo.ToPtr("1234567890 [update]"),
				},
			},
		}

		_, err := s.BillingService.UpdateProfile(ctx, updateInput)

		// Then the profile is updated
		require.ErrorIs(t, err, billing.ErrProfileConflict)
		require.ErrorAs(t, err, &billing.UpdateAfterDeleteError{})
	})

	s.T().Run("update profile, provider change => fails", func(t *testing.T) {
		// When updating the profile with new providers (which is not allowed)
		profile, err = s.BillingService.GetProfile(ctx, billing.GetProfileInput{
			Namespace: ns,
			ID:        profile.ID,
		})

		require.NoError(t, err)

		updateInput := billing.UpdateProfileInput{
			ID:        profile.ID,
			Namespace: ns,
			Default:   true,
			CreatedAt: profile.CreatedAt,

			UpdatedAt: profile.UpdatedAt,

			TaxConfiguration: provider.TaxConfiguration{
				Type: provider.TaxProviderStripeTax,
			},

			InvoicingConfiguration: provider.InvoicingConfiguration{
				Type: provider.InvoicingProviderStripeInvoicing,
			},

			PaymentConfiguration: provider.PaymentConfiguration{
				Type: provider.PaymentProviderStripePayments,
			},

			WorkflowConfig: billing.WorkflowConfig{
				CreatedAt: profile.WorkflowConfig.CreatedAt,
				Collection: billing.CollectionConfig{
					Alignment:            billing.AlignmentKindSubscription,
					ItemCollectionPeriod: 60 * time.Minute,
				},
				Invoicing: billing.InvoicingConfig{
					AutoAdvance: false,
					DraftPeriod: 2 * time.Hour,
					DueAfter:    48 * time.Hour,

					ItemResolution: billing.GranularityResolutionPeriod,
					ItemPerSubject: false,
				},
				Payment: billing.PaymentConfig{
					CollectionMethod: billing.CollectionMethodChargeAutomatically,
				},
			},

			Supplier: billing.SupplierContact{
				Name: "Awesome Supplier [update]",
				Address: models.Address{
					Country:     lo.ToPtr(models.CountryCode("HU")),
					PostalCode:  lo.ToPtr("12345 [update]"),
					City:        lo.ToPtr("City [update]"),
					State:       lo.ToPtr("State [update]"),
					Line1:       lo.ToPtr("Line 1 [update]"),
					Line2:       lo.ToPtr("Line 2 [update]"),
					PhoneNumber: lo.ToPtr("1234567890 [update]"),
				},
			},
		}

		_, err := s.BillingService.UpdateProfile(ctx, updateInput)

		// Then the profile update is rejected
		require.ErrorIs(t, err, billing.ErrProfileTaxTypeChange)
		require.ErrorAs(t, err, &billing.ValidationError{})
	})
}
