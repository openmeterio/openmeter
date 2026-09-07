package billing

import (
	"context"
	"testing"
	"time"

	"github.com/oklog/ulid/v2"
	"github.com/samber/lo"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/openmeterio/openmeter/openmeter/app"
	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/taxcode"
	"github.com/openmeterio/openmeter/pkg/datetime"
	"github.com/openmeterio/openmeter/pkg/filter"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/pagination"
	"github.com/openmeterio/openmeter/pkg/sortx"
)

type ProfileTestSuite struct {
	BaseSuite
}

func TestProfile(t *testing.T) {
	suite.Run(t, new(ProfileTestSuite))
}

func (s *ProfileTestSuite) TestGetRejectsCrossNamespaceWorkflowReferences() {
	// given:
	// - profiles whose workflow config or tax code is deliberately moved to another namespace
	// when:
	// - either profile is read
	// then:
	// - the foreign reference is not returned and is reported as missing
	ctx := s.T().Context()
	foreignNamespace := s.GetUniqueNamespace("profile-reference-foreign")

	workflowNamespace := s.GetUniqueNamespace("profile-workflow-target")
	workflowProfile := s.ProvisionBillingProfile(ctx, workflowNamespace, s.InstallSandboxApp(s.T(), workflowNamespace).GetID())
	workflowProfileDB, err := s.BillingAdapter.GetProfile(ctx, billing.GetProfileInput{Profile: workflowProfile.ProfileID()})
	require.NoError(s.T(), err)
	_, err = s.TestDB.PGDriver.DB().ExecContext(ctx,
		`UPDATE billing_workflow_configs SET namespace = $1 WHERE id = $2`,
		foreignNamespace,
		workflowProfileDB.WorkflowConfigID,
	)
	require.NoError(s.T(), err)

	_, err = s.BillingService.GetProfile(ctx, billing.GetProfileInput{Profile: workflowProfile.ProfileID()})
	require.True(s.T(), models.IsGenericNotFoundError(err))

	taxNamespace := s.GetUniqueNamespace("profile-tax-target")
	taxProfile := s.ProvisionBillingProfile(ctx, taxNamespace, s.InstallSandboxApp(s.T(), taxNamespace).GetID())
	taxProfileDB, err := s.BillingAdapter.GetProfile(ctx, billing.GetProfileInput{Profile: taxProfile.ProfileID()})
	require.NoError(s.T(), err)
	foreignTaxCode := s.ProvisionProviderDefaultTaxCode(ctx, foreignNamespace)
	_, err = s.DBClient.BillingWorkflowConfig.UpdateOneID(taxProfileDB.WorkflowConfigID).
		SetTaxCodeID(foreignTaxCode.ID).
		Save(ctx)
	require.NoError(s.T(), err)

	_, err = s.BillingService.GetProfile(ctx, billing.GetProfileInput{Profile: taxProfile.ProfileID()})
	require.True(s.T(), taxcode.IsTaxCodeNotFoundError(err))
}

// createProfileFixture creates a profile with the given default flag.
// Note: only non default profiles can be deleted.
func (s *ProfileTestSuite) createProfileFixture(isDefault bool) *billing.Profile {
	t := s.T()
	ctx := context.Background()
	ns := s.GetUniqueNamespace("test_billing_profile")
	sandboxApp := s.InstallSandboxApp(s.T(), ns)

	// Create a profile input
	input := minimalCreateProfileInputTemplate(sandboxApp.GetID())
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
			Profile: profile.ProfileID(),
			Expand: billing.ProfileExpand{
				Apps: true,
			},
		})

		require.NoError(t, err)
		require.Equal(t, profile, fetchedProfile)
	})

	s.T().Run("creating a second default profile", func(t *testing.T) {
		profile1 := s.createProfileFixture(true)

		// Create a second default profile in the same namespace
		input := minimalCreateProfileInputTemplate(profile1.AppReferences.Invoicing)
		input.Namespace = profile1.Namespace
		input.Default = true

		profile2, err := s.BillingService.CreateProfile(ctx, input)
		require.NoError(t, err)
		require.NotNil(t, profile2)

		// Fetch the default profile
		defaultProfile, err := s.BillingService.GetDefaultProfile(ctx, billing.GetDefaultProfileInput{
			Namespace: profile1.Namespace,
		})
		require.NoError(t, err)
		require.NotNil(t, defaultProfile)
		require.Equal(t, profile2.ID, defaultProfile.ID)
	})

	s.T().Run("app used", func(t *testing.T) {
		s.T().Run("app should be used", func(t *testing.T) {
			profile := s.createProfileFixture(true)
			require.NotNil(t, profile)

			err := s.BillingService.IsAppUsed(ctx, profile.Apps.Invoicing.GetID())

			var conflictErr *models.GenericConflictError
			require.ErrorAs(t, err, &conflictErr)
		})

		s.T().Run("app should not be used", func(t *testing.T) {
			profile := s.createProfileFixture(true)
			require.NotNil(t, profile)

			anotherAppID := app.AppID{
				Namespace: profile.Namespace,
				ID:        ulid.Make().String(),
			}

			err := s.BillingService.IsAppUsed(ctx, anotherAppID)
			require.NoError(t, err)
		})
	})

	s.T().Run("deleted profile handling", func(t *testing.T) {
		t.Run("deleting a profile succeeds", func(t *testing.T) {
			profile := s.createProfileFixture(false)

			require.NoError(t, s.BillingService.DeleteProfile(ctx, billing.DeleteProfileInput{
				Namespace: profile.Namespace,
				ID:        profile.ID,
			}))
		})

		t.Run("deleting a profile twice yields no error", func(t *testing.T) {
			profile := s.createProfileFixture(false)

			// Delete the profile
			require.NoError(t, s.BillingService.DeleteProfile(ctx, billing.DeleteProfileInput{
				Namespace: profile.Namespace,
				ID:        profile.ID,
			}))

			// Try to delete the profile again
			require.NoError(t, s.BillingService.DeleteProfile(ctx, billing.DeleteProfileInput{
				Namespace: profile.Namespace,
				ID:        profile.ID,
			}))
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
				Profile: profile.ProfileID(),
			})

			require.NoError(t, err)
			require.Equal(t, profile.ID, fetchedProfile.ID)
		})

		t.Run("updating a deleted profile yields an error", func(t *testing.T) {
			profile := s.createProfileFixture(false)

			// Delete the profile
			require.NoError(t, s.BillingService.DeleteProfile(ctx, billing.DeleteProfileInput{
				Namespace: profile.Namespace,
				ID:        profile.ID,
			}))

			// Update the profile
			profile.BaseProfile.AppReferences = nil
			_, err := s.BillingService.UpdateProfile(ctx, billing.UpdateProfileInput(profile.BaseProfile))

			require.ErrorAs(t, err, &billing.ValidationIssue{})
			require.ErrorIs(t, err, billing.ErrProfileAlreadyDeleted)
		})
	})

	s.T().Run("update profile handling", func(t *testing.T) {
		t.Run("updating the default profile", func(t *testing.T) {
			var err error

			profile := s.createProfileFixture(true)
			profile.Name = "Updated Name"

			profile, err = s.BillingService.UpdateProfile(ctx, toUpdateProfileInput(*profile))
			require.NoError(t, err)
			require.NotNil(t, profile)

			require.True(t, profile.Default)
			require.Equal(t, "Updated Name", profile.Name)
		})

		t.Run("unsetting the default profile returns error", func(t *testing.T) {
			profile := s.createProfileFixture(true)
			profile.Default = false

			_, err := s.BillingService.UpdateProfile(ctx, toUpdateProfileInput(*profile))
			require.ErrorIs(t, err, billing.ErrDefaultProfileCannotBeUnset)
		})
	})
}

func (s *ProfileTestSuite) TestDeletedProfileAppUsageFollowsNonFinalInvoices() {
	t := s.T()
	ctx := t.Context()
	namespace := s.GetUniqueNamespace("deleted-profile-app-usage")

	// Given a draft invoice that snapshots the apps from the current default profile.
	referencedApp := s.InstallSandboxApp(t, namespace)
	referencedProfile := s.ProvisionBillingProfile(ctx, namespace, referencedApp.GetID())
	customerEntity := s.CreateTestCustomer(namespace, ulid.Make().String())
	invoice := s.CreateDraftInvoice(t, ctx, DraftInvoiceInput{
		Namespace: namespace,
		Customer:  customerEntity,
	})

	// And the profile is superseded and deleted, leaving the invoice as the only reference.
	replacementApp := s.InstallSandboxApp(t, namespace)
	s.ProvisionBillingProfile(ctx, namespace, replacementApp.GetID())
	require.NoError(t, s.BillingService.DeleteProfile(ctx, billing.DeleteProfileInput{
		Namespace: namespace,
		ID:        referencedProfile.ID,
	}))

	// Then the non-final invoice keeps the referenced app in use.
	err := s.BillingService.IsAppUsed(ctx, referencedApp.GetID())
	var conflictErr *models.GenericConflictError
	require.ErrorAs(t, err, &conflictErr)
	require.ErrorContains(t, err, invoice.ID)

	// When the invoice reaches a final state, it no longer prevents uninstall.
	invoice, err = s.BillingService.DeleteInvoice(ctx, billing.DeleteInvoiceInput{
		Invoice:        invoice.GetInvoiceID(),
		DeletionSource: billing.ChangeSourceAPIRequest,
	})
	require.NoError(t, err)
	require.Equal(t, billing.StandardInvoiceStatusDeleted, invoice.Status)
	require.NoError(t, s.BillingService.IsAppUsed(ctx, referencedApp.GetID()))
}

func (s *ProfileTestSuite) TestProfileFieldSetting() {
	ctx := context.Background()
	t := s.T()
	ns := "test_profile_field_setting"

	sandboxApp := s.InstallSandboxApp(s.T(), ns)

	input := billing.CreateProfileInput{
		Namespace: ns,
		Default:   true,
		Name:      "Awesome Default Profile",

		Metadata: map[string]string{
			"key": "value",
		},

		WorkflowConfig: billing.WorkflowConfig{
			Collection: billing.CollectionConfig{
				Alignment: billing.AlignmentKindAnchored,
				AnchoredAlignmentDetail: lo.ToPtr(billing.AnchoredAlignmentDetail{
					Interval: datetime.MustParseDuration(t, "PT30M"),
					Anchor:   time.Now(),
				}),
				Interval: datetime.MustParseDuration(t, "PT30M"),
			},
			Invoicing: billing.InvoicingConfig{
				AutoAdvance:                  true,
				DraftPeriod:                  datetime.MustParseDuration(t, "PT1H"),
				DueAfter:                     datetime.MustParseDuration(t, "PT24H"),
				SubscriptionEndProrationMode: billing.SubscriptionEndProrationModeBillActualPeriod,
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
			Invoicing: sandboxApp.GetID(),
			Payment:   sandboxApp.GetID(),
			Tax:       sandboxApp.GetID(),
		},
	}

	profile, err := s.BillingService.CreateProfile(ctx, input)
	require.NoError(s.T(), err)
	require.NotNil(s.T(), profile)

	profile.CreatedAt = profile.CreatedAt.Truncate(time.Microsecond)
	profile.UpdatedAt = profile.UpdatedAt.Truncate(time.Microsecond)

	// Let's fetch the profile again
	fetchedProfile, err := s.BillingService.GetProfile(ctx, billing.GetProfileInput{
		Profile: profile.ProfileID(),
		Expand:  billing.ProfileExpandAll,
	})

	// We should strip monotonic time for comparison on CollectionConfig.
	profile.WorkflowConfig.Collection.AnchoredAlignmentDetail.Anchor = profile.WorkflowConfig.Collection.AnchoredAlignmentDetail.Anchor.Truncate(time.Nanosecond).UTC()
	fetchedProfile.WorkflowConfig.Collection.AnchoredAlignmentDetail.Anchor = fetchedProfile.WorkflowConfig.Collection.AnchoredAlignmentDetail.Anchor.Truncate(time.Nanosecond).UTC()

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

	// Let's check if the fields are set correctly
	require.Equal(s.T(), expectedProfile, *fetchedProfile)
	require.Equal(s.T(), sandboxApp.GetID(), fetchedProfile.Apps.Tax.GetID())
	require.Equal(s.T(), sandboxApp.GetID(), fetchedProfile.Apps.Invoicing.GetID())
	require.Equal(s.T(), sandboxApp.GetID(), fetchedProfile.Apps.Payment.GetID())
}

func (s *ProfileTestSuite) TestProfileUpdates() {
	// Given a profile
	ctx := context.Background()
	ns := "test_profile_updates"

	sandboxApp := s.InstallSandboxApp(s.T(), ns)

	input := billing.CreateProfileInput{
		Namespace: ns,
		Default:   true,

		Name: "Awesome Default Profile",

		Apps: minimalCreateProfileInputTemplate(sandboxApp.GetID()).Apps,

		WorkflowConfig: billing.WorkflowConfig{
			Collection: billing.CollectionConfig{
				Alignment: billing.AlignmentKindSubscription,
				Interval:  datetime.MustParseDuration(s.T(), "PT30M"),
			},
			Invoicing: billing.InvoicingConfig{
				AutoAdvance:                  true,
				DraftPeriod:                  datetime.MustParseDuration(s.T(), "PT1H"),
				DueAfter:                     datetime.MustParseDuration(s.T(), "PT24H"),
				SubscriptionEndProrationMode: billing.SubscriptionEndProrationModeBillActualPeriod,
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

	// Let's fetch the profile again
	fetchedProfile, err := s.BillingService.GetProfile(ctx, billing.GetProfileInput{
		Profile: profile.ProfileID(),
		Expand:  billing.ProfileExpandAll,
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

			WorkflowConfig: billing.WorkflowConfig{
				Collection: billing.CollectionConfig{
					Alignment: billing.AlignmentKindSubscription,
					Interval:  datetime.MustParseDuration(s.T(), "PT30M"),
				},
				Invoicing: billing.InvoicingConfig{
					AutoAdvance:                  true,
					DraftPeriod:                  datetime.MustParseDuration(s.T(), "PT2H"),
					DueAfter:                     datetime.MustParseDuration(s.T(), "PT48H"),
					SubscriptionEndProrationMode: billing.SubscriptionEndProrationModeBillActualPeriod,
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
		expectedOutput.UpdatedAt = updatedProfile.UpdatedAt // This is checked by the previous assertion
		expectedOutput.Apps = fetchedProfile.Apps

		require.Equal(t, expectedOutput, *updatedProfile)
	})
}

func (s *ProfileTestSuite) TestProfileSubscriptionAlignmentPersists() {
	ctx := context.Background()
	ns := "test_profile_subscription_alignment_persists"

	sandboxApp := s.InstallSandboxApp(s.T(), ns)

	input := minimalCreateProfileInputTemplate(sandboxApp.GetID())
	input.Namespace = ns
	input.WorkflowConfig.Collection.Alignment = billing.AlignmentKindSubscription

	prof, err := s.BillingService.CreateProfile(ctx, input)
	require.NoError(s.T(), err)
	require.NotNil(s.T(), prof)

	fetched, err := s.BillingService.GetProfile(ctx, billing.GetProfileInput{
		Profile: prof.ProfileID(),
		Expand:  billing.ProfileExpandAll,
	})
	require.NoError(s.T(), err)
	require.NotNil(s.T(), fetched)

	require.Equal(s.T(), billing.AlignmentKindSubscription, fetched.WorkflowConfig.Collection.Alignment)
	require.Nil(s.T(), fetched.WorkflowConfig.Collection.AnchoredAlignmentDetail)
}

func toUpdateProfileInput(profile billing.Profile) billing.UpdateProfileInput {
	return billing.UpdateProfileInput{
		ID:             profile.ID,
		Namespace:      profile.Namespace,
		Name:           profile.Name,
		Description:    profile.Description,
		Metadata:       profile.Metadata,
		Default:        profile.Default,
		WorkflowConfig: profile.WorkflowConfig,
		Supplier:       profile.Supplier,
		CreatedAt:      profile.CreatedAt,
		UpdatedAt:      profile.UpdatedAt,
	}
}

func (s *ProfileTestSuite) TestListProfilesFilterAndSort() {
	// given:
	// - three profiles with distinct names created in a known order
	// when:
	// - profiles are listed with each supported filter and sort
	// then:
	// - only matching profiles are returned in the requested order
	ctx := s.T().Context()
	ns := s.GetUniqueNamespace("profile-list")
	appID := s.InstallSandboxApp(s.T(), ns).GetID()

	newProfile := func(name string, isDefault bool) *billing.Profile {
		return s.ProvisionBillingProfile(ctx, ns, appID, WithBillingProfileEditFn(func(p *billing.CreateProfileInput) {
			p.Name = name
			p.Default = isDefault
		}))
	}

	charlie := newProfile("Charlie", true)
	alpha := newProfile("Alpha", false)
	bravo := newProfile("Bravo Team", false)

	testCases := []struct {
		name     string
		input    billing.ListProfilesInput
		expected []string
	}{
		{
			name:     "default sort is created_at asc",
			expected: []string{charlie.ID, alpha.ID, bravo.ID},
		},
		{
			name:     "sort by created_at desc",
			input:    billing.ListProfilesInput{OrderBy: billing.ProfileOrderByCreatedAt, Order: sortx.OrderDesc},
			expected: []string{bravo.ID, alpha.ID, charlie.ID},
		},
		{
			name:     "sort by name asc",
			input:    billing.ListProfilesInput{OrderBy: billing.ProfileOrderByName},
			expected: []string{alpha.ID, bravo.ID, charlie.ID},
		},
		{
			name:     "sort by name desc",
			input:    billing.ListProfilesInput{OrderBy: billing.ProfileOrderByName, Order: sortx.OrderDesc},
			expected: []string{charlie.ID, bravo.ID, alpha.ID},
		},
		{
			name:     "filter by id eq",
			input:    billing.ListProfilesInput{ID: &filter.FilterULID{FilterString: filter.FilterString{Eq: lo.ToPtr(alpha.ID)}}},
			expected: []string{alpha.ID},
		},
		{
			name:     "filter by id in",
			input:    billing.ListProfilesInput{ID: &filter.FilterULID{FilterString: filter.FilterString{In: &[]string{alpha.ID, bravo.ID}}}},
			expected: []string{alpha.ID, bravo.ID},
		},
		{
			name:     "filter by name eq",
			input:    billing.ListProfilesInput{Name: &filter.FilterString{Eq: lo.ToPtr("Charlie")}},
			expected: []string{charlie.ID},
		},
		{
			name:     "filter by name contains is case insensitive",
			input:    billing.ListProfilesInput{Name: &filter.FilterString{Contains: lo.ToPtr("bravo")}},
			expected: []string{bravo.ID},
		},
		{
			name:     "filter by name with no match",
			input:    billing.ListProfilesInput{Name: &filter.FilterString{Eq: lo.ToPtr("Nobody")}},
			expected: []string{},
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			input := tc.input
			input.Namespace = ns
			input.Page = pagination.NewPage(1, 20)

			result, err := s.BillingService.ListProfiles(ctx, input)
			s.NoError(err)
			s.Equal(len(tc.expected), result.TotalCount)
			s.Equal(tc.expected, lo.Map(result.Items, func(p billing.Profile, _ int) string { return p.ID }))
		})
	}
}

func (s *ProfileTestSuite) TestListProfilesRejectsInvalidInput() {
	ctx := s.T().Context()
	ns := s.GetUniqueNamespace("profile-list-invalid")

	_, err := s.BillingService.ListProfiles(ctx, billing.ListProfilesInput{
		Namespace: ns,
		Page:      pagination.NewPage(1, 20),
		OrderBy:   billing.ProfileOrderBy("nope"),
	})
	s.ErrorContains(err, "invalid order by")

	_, err = s.BillingService.ListProfiles(ctx, billing.ListProfilesInput{
		Namespace: ns,
		Page:      pagination.NewPage(1, 20),
		Name:      &filter.FilterString{Eq: lo.ToPtr("a"), Ne: lo.ToPtr("b")},
	})
	s.ErrorContains(err, "invalid name filter")
}
