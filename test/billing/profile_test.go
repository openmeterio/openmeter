package billing

import (
	"context"
	"testing"
	"time"

	"github.com/samber/lo"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	appentitybase "github.com/openmeterio/openmeter/openmeter/app/entity/base"
	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/pkg/datex"
	"github.com/openmeterio/openmeter/pkg/models"
)

type ProfileTestSuite struct {
	BaseSuite
}

func TestProfile(t *testing.T) {
	suite.Run(t, new(ProfileTestSuite))
}

// createProfileFixture creates a profile with the given default flag.
// Note: only non default profiles can be deleted.
func (s *ProfileTestSuite) createProfileFixture(isDefault bool) *billing.Profile {
	t := s.T()
	ctx := context.Background()
	ns := s.GetUniqueNamespace("test_billing_profile")

	// Create a profile input
	input := MinimalCreateProfileInputTemplate
	input.Namespace = ns
	input.Default = isDefault

	// Create a sandbox app
	app := s.InstallSandboxApp(s.T(), ns)
	require.NotNil(t, app)

	// Create a default profile
	profile, err := s.BillingService.CreateProfile(ctx, input)
	require.NoError(t, err)
	require.NotNil(t, profile)

	profile.CreatedAt = profile.CreatedAt.Truncate(time.Microsecond)
	profile.UpdatedAt = profile.UpdatedAt.Truncate(time.Microsecond)
	profile.WorkflowConfig.CreatedAt = profile.WorkflowConfig.CreatedAt.Truncate(time.Microsecond)
	profile.WorkflowConfig.UpdatedAt = profile.WorkflowConfig.UpdatedAt.Truncate(time.Microsecond)

	return profile
}

func (s *ProfileTestSuite) TestProfileLifecycle() {
	ctx := context.Background()

	s.T().Run("missing default profile", func(t *testing.T) {
		profile := s.createProfileFixture(false)

		defaultProfile, err := s.BillingService.GetDefaultProfile(ctx, billing.GetDefaultProfileInput{
			Namespace: profile.Namespace,
		})
		require.NoError(t, err)
		require.Nil(t, defaultProfile)
	})

	s.T().Run("create default profile", func(t *testing.T) {
		profile := s.createProfileFixture(true)

		require.NotNil(t, profile)
		require.True(t, profile.Default)
	})

	s.T().Run("fetching the default profile is possible", func(t *testing.T) {
		profile := s.createProfileFixture(true)

		defaultProfile, err := s.BillingService.GetDefaultProfile(ctx, billing.GetDefaultProfileInput{
			Namespace: profile.Namespace,
		})

		require.NoError(t, err)
		require.NotNil(t, defaultProfile)
		require.Equal(t, profile, defaultProfile)
	})

	s.T().Run("fetching the profile by id", func(t *testing.T) {
		profile := s.createProfileFixture(false)

		fetchedProfile, err := s.BillingService.GetProfile(ctx, billing.GetProfileInput{
			Profile: models.NamespacedID{
				Namespace: profile.Namespace,
				ID:        profile.ID,
			},
			Expand: billing.ProfileExpand{
				Apps: true,
			},
		})

		require.NoError(t, err)
		require.Equal(t, profile, fetchedProfile)
	})

	s.T().Run("creating a second default profile fails", func(t *testing.T) {
		profile := s.createProfileFixture(true)

		// Try to create a second default profile in the same namespace
		input := MinimalCreateProfileInputTemplate
		input.Namespace = profile.Namespace

		_, err := s.BillingService.CreateProfile(ctx, input)
		require.Error(t, err)
		require.ErrorIs(t, err, billing.ErrDefaultProfileAlreadyExists)
	})

	s.T().Run("creating a second default profile succeeds with override", func(t *testing.T) {
		profile1 := s.createProfileFixture(true)

		// Create a second default profile with override
		input := MinimalCreateProfileInputTemplate
		input.Namespace = profile1.Namespace
		input.DefaultOverride = true

		profile2, err := s.BillingService.CreateProfile(ctx, input)
		require.NoError(t, err)
		require.NotNil(t, profile2)

		require.NotEqual(t, profile1.ID, profile2.ID)
		require.True(t, profile2.Default)
	})

	s.T().Run("deleted profile handling", func(t *testing.T) {
		t.Run("deleting a profile succeeds", func(t *testing.T) {
			profile := s.createProfileFixture(false)

			require.NoError(t, s.BillingService.DeleteProfile(ctx, billing.DeleteProfileInput{
				Namespace: profile.Namespace,
				ID:        profile.ID,
			}))
		})

		t.Run("deleting a profile twice yields an error", func(t *testing.T) {
			profile := s.createProfileFixture(false)

			// Delete the profile
			require.NoError(t, s.BillingService.DeleteProfile(ctx, billing.DeleteProfileInput{
				Namespace: profile.Namespace,
				ID:        profile.ID,
			}))

			// Try to delete the profile again
			require.ErrorIs(t, s.BillingService.DeleteProfile(ctx, billing.DeleteProfileInput{
				Namespace: profile.Namespace,
				ID:        profile.ID,
			}), billing.ErrProfileAlreadyDeleted)
		})

		t.Run("deleting the default profile yields an error", func(t *testing.T) {
			profile := s.createProfileFixture(true)

			// Try to delete the default profile
			require.ErrorIs(t, s.BillingService.DeleteProfile(ctx, billing.DeleteProfileInput{
				Namespace: profile.Namespace,
				ID:        profile.ID,
			}), billing.ErrDefaultProfileCannotBeDeleted)
		})

		t.Run("fetching a deleted profile by id returns the profile", func(t *testing.T) {
			profile := s.createProfileFixture(false)

			// Delete the profile
			require.NoError(t, s.BillingService.DeleteProfile(ctx, billing.DeleteProfileInput{
				Namespace: profile.Namespace,
				ID:        profile.ID,
			}))

			// Fetch the profile
			fetchedProfile, err := s.BillingService.GetProfile(ctx, billing.GetProfileInput{
				Profile: models.NamespacedID{
					Namespace: profile.Namespace,
					ID:        profile.ID,
				},
			})

			require.NoError(t, err)
			require.Equal(t, profile.ID, fetchedProfile.ID)
		})
	})
}

func (s *ProfileTestSuite) TestProfileFieldSetting() {
	ctx := context.Background()
	t := s.T()
	ns := "test_profile_field_setting"

	app := s.InstallSandboxApp(s.T(), ns)

	input := billing.CreateProfileInput{
		Namespace: ns,
		Default:   true,
		Name:      "Awesome Default Profile",

		Metadata: map[string]string{
			"key": "value",
		},

		WorkflowConfig: billing.WorkflowConfig{
			Collection: billing.CollectionConfig{
				Alignment: billing.AlignmentKindSubscription,
				Interval:  datex.MustParse(t, "PT30M"),
			},
			Invoicing: billing.InvoicingConfig{
				AutoAdvance: true,
				DraftPeriod: datex.MustParse(t, "PT1H"),
				DueAfter:    datex.MustParse(t, "PT24H"),
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

		Apps: billing.CreateProfileAppsInput{
			Invoicing: billing.AppReference{
				Type: appentitybase.AppTypeSandbox,
			},
			Payment: billing.AppReference{
				Type: appentitybase.AppTypeSandbox,
			},
			Tax: billing.AppReference{
				Type: appentitybase.AppTypeSandbox,
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
		Profile: models.NamespacedID{
			Namespace: ns,
			ID:        profile.ID,
		},
		Expand: billing.ProfileExpandAll,
	})

	// Sanity check db conversion & fetching
	require.NoError(s.T(), err)
	require.Equal(s.T(), profile, fetchedProfile)

	// let's add the db derived fields to the input
	expectedProfile := billing.Profile{
		BaseProfile: billing.BaseProfile{
			ID: profile.ID,

			Namespace:   input.Namespace,
			Name:        input.Name,
			Description: input.Description,
			Default:     input.Default,

			CreatedAt: fetchedProfile.CreatedAt,
			UpdatedAt: fetchedProfile.UpdatedAt,
			DeletedAt: fetchedProfile.DeletedAt,

			WorkflowConfig: input.WorkflowConfig,
			Supplier:       input.Supplier,

			Metadata:      input.Metadata,
			AppReferences: fetchedProfile.AppReferences,
		},
		Apps: fetchedProfile.Apps,
	}

	expectedProfile.WorkflowConfig.ID = fetchedProfile.WorkflowConfig.ID
	expectedProfile.WorkflowConfig.CreatedAt = fetchedProfile.WorkflowConfig.CreatedAt
	expectedProfile.WorkflowConfig.UpdatedAt = fetchedProfile.WorkflowConfig.UpdatedAt
	expectedProfile.WorkflowConfig.DeletedAt = fetchedProfile.WorkflowConfig.DeletedAt

	// Let's check if the fields are set correctly
	require.Equal(s.T(), expectedProfile, *fetchedProfile)
	require.Equal(s.T(), app.GetID(), fetchedProfile.Apps.Tax.GetID())
	require.Equal(s.T(), app.GetID(), fetchedProfile.Apps.Invoicing.GetID())
	require.Equal(s.T(), app.GetID(), fetchedProfile.Apps.Payment.GetID())
}

func (s *ProfileTestSuite) TestProfileUpdates() {
	// Given a profile
	ctx := context.Background()
	ns := "test_profile_updates"

	_ = s.InstallSandboxApp(s.T(), ns)

	input := billing.CreateProfileInput{
		Namespace: ns,
		Default:   true,

		Name: "Awesome Default Profile",

		Apps: MinimalCreateProfileInputTemplate.Apps,

		WorkflowConfig: billing.WorkflowConfig{
			Collection: billing.CollectionConfig{
				Alignment: billing.AlignmentKindSubscription,
				Interval:  datex.MustParse(s.T(), "PT30M"),
			},
			Invoicing: billing.InvoicingConfig{
				AutoAdvance: true,
				DraftPeriod: datex.MustParse(s.T(), "PT1H"),
				DueAfter:    datex.MustParse(s.T(), "PT24H"),
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
		Profile: models.NamespacedID{
			Namespace: ns,
			ID:        profile.ID,
		},
		Expand: billing.ProfileExpandAll,
	})

	// Sanity check db conversion & fetching
	require.NoError(s.T(), err)
	require.Equal(s.T(), profile, fetchedProfile)

	s.T().Run("update profile", func(t *testing.T) {
		// When updating the profile
		updateInput := billing.UpdateProfileInput{
			ID:          profile.ID,
			Namespace:   ns,
			Default:     true,
			Name:        "Awesome Default Profile [update]",
			Description: lo.ToPtr("Updated description"),
			CreatedAt:   profile.CreatedAt,

			UpdatedAt: profile.UpdatedAt,

			WorkflowConfig: billing.WorkflowConfig{
				CreatedAt: profile.WorkflowConfig.CreatedAt,
				Collection: billing.CollectionConfig{
					Alignment: billing.AlignmentKindSubscription,
					Interval:  datex.MustParse(s.T(), "PT30M"),
				},
				Invoicing: billing.InvoicingConfig{
					AutoAdvance: true,
					DraftPeriod: datex.MustParse(s.T(), "PT2H"),
					DueAfter:    datex.MustParse(s.T(), "PT48H"),
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
		expectedOutput := billing.Profile{
			BaseProfile: billing.BaseProfile{
				ID:          updateInput.ID,
				Namespace:   updateInput.Namespace,
				CreatedAt:   fetchedProfile.CreatedAt,
				Name:        updateInput.Name,
				Description: updateInput.Description,
				Metadata:    updateInput.Metadata,
				Supplier:    updateInput.Supplier,
				Default:     updateInput.Default,

				WorkflowConfig: updateInput.WorkflowConfig,
				AppReferences:  fetchedProfile.AppReferences,
			},
		}
		expectedOutput.WorkflowConfig.ID = fetchedProfile.WorkflowConfig.ID
		expectedOutput.UpdatedAt = updatedProfile.UpdatedAt                               // This is checked by the previous assertion
		expectedOutput.WorkflowConfig.UpdatedAt = updatedProfile.WorkflowConfig.UpdatedAt // This is checked by the previous assertion
		expectedOutput.Apps = fetchedProfile.Apps

		require.Equal(t, expectedOutput, *updatedProfile)
	})
}
