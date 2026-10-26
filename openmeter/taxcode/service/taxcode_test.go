package service_test

import (
	"testing"

	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/app"
	"github.com/openmeterio/openmeter/openmeter/taxcode"
	taxcodetestutils "github.com/openmeterio/openmeter/openmeter/taxcode/testutils"
	"github.com/openmeterio/openmeter/openmeter/testutils"
	"github.com/openmeterio/openmeter/pkg/models"
)

func TestTaxCodeService(t *testing.T) {
	env := taxcodetestutils.NewTestEnv(t)
	t.Cleanup(func() { env.Close(t) })

	ns := testutils.NameGenerator.Generate().Key
	env.ProvisionDefaultTaxCodes(t, ns)

	t.Run("GetByKey", func(t *testing.T) {
		// given:
		// - namespace defaults have seeded the provider-default tax code
		// when:
		// - resolving by the stable provider default key
		// then:
		// - the seeded tax code is returned
		got, err := env.Service.GetTaxCodeByKey(t.Context(), taxcode.GetTaxCodeByKeyInput{
			Namespace: ns,
			Key:       taxcode.ProviderDefaultTaxCodeKey,
		})
		require.NoError(t, err)
		assert.Equal(t, taxcode.ProviderDefaultTaxCodeKey, got.Key)
	})

	t.Run("SystemManaged", func(t *testing.T) {
		// Create a system-managed tax code by explicitly setting the annotation.
		name := testutils.NameGenerator.Generate()
		tc, err := env.Service.CreateTaxCode(t.Context(), taxcode.CreateTaxCodeInput{
			Namespace: ns,
			Key:       name.Key,
			Name:      name.Name,
			Annotations: models.Annotations{
				taxcode.AnnotationKeyManagedBy: taxcode.AnnotationValueManagedBySystem,
			},
		})
		require.NoError(t, err)
		assert.True(t, tc.IsManagedBySystem())

		t.Run("UpdateIsBlocked", func(t *testing.T) {
			_, err := env.Service.UpdateTaxCode(t.Context(), taxcode.UpdateTaxCodeInput{
				NamespacedID: models.NamespacedID{Namespace: ns, ID: tc.ID},
				Name:         "updated name",
			})
			require.Error(t, err)

			var vi models.ValidationIssue
			require.ErrorAs(t, err, &vi)
			assert.Equal(t, taxcode.ErrCodeTaxCodeManagedBySystem, vi.Code())
		})

		t.Run("UpdateIsByPassed", func(t *testing.T) {
			input := taxcode.UpdateTaxCodeInput{
				NamespacedID: models.NamespacedID{Namespace: ns, ID: tc.ID},
				Name:         "updated name",
				Annotations: models.Annotations{
					taxcode.AnnotationKeyManagedBy: taxcode.AnnotationValueManagedBySystem,
				},
			}
			input.AllowAnnotations = true
			updated, err := env.Service.UpdateTaxCode(t.Context(), input)
			require.NoError(t, err)
			assert.Equal(t, "updated name", updated.Name)
		})

		t.Run("DeleteIsBlocked", func(t *testing.T) {
			err := env.Service.DeleteTaxCode(t.Context(), taxcode.DeleteTaxCodeInput{
				NamespacedID: models.NamespacedID{Namespace: ns, ID: tc.ID},
			})
			require.Error(t, err)

			var vi models.ValidationIssue
			require.ErrorAs(t, err, &vi)
			assert.Equal(t, taxcode.ErrCodeTaxCodeManagedBySystem, vi.Code())
		})

		t.Run("DeleteIsByPassed", func(t *testing.T) {
			input := taxcode.DeleteTaxCodeInput{
				NamespacedID: models.NamespacedID{Namespace: ns, ID: tc.ID},
			}
			input.AllowAnnotations = true
			err := env.Service.DeleteTaxCode(t.Context(), input)
			require.NoError(t, err)
		})
	})

	t.Run("OrganizationDefault", func(t *testing.T) {
		// given:
		// - two tax codes set as org defaults (invoicing + credit grant)
		// - one extra tax code that is not a default
		// when:
		// - caller tries to delete an org default tax code
		// then:
		// - deletion is blocked with ErrCodeTaxCodeIsOrganizationDefault (409)
		// - deletion of a non-default tax code succeeds
		defaultNs := testutils.NameGenerator.Generate().Key
		invoicing := env.CreateTaxCode(t, defaultNs)
		creditGrant := env.CreateTaxCode(t, defaultNs)
		other := env.CreateTaxCode(t, defaultNs)

		_, err := env.Service.UpsertOrganizationDefaultTaxCodes(t.Context(), taxcode.UpsertOrganizationDefaultTaxCodesInput{
			Namespace:            defaultNs,
			InvoicingTaxCodeID:   invoicing.ID,
			CreditGrantTaxCodeID: creditGrant.ID,
		})
		require.NoError(t, err)

		t.Run("DeleteInvoicingDefaultIsBlocked", func(t *testing.T) {
			err := env.Service.DeleteTaxCode(t.Context(), taxcode.DeleteTaxCodeInput{
				NamespacedID: models.NamespacedID{Namespace: defaultNs, ID: invoicing.ID},
			})
			require.Error(t, err)

			var vi models.ValidationIssue
			require.ErrorAs(t, err, &vi)
			assert.Equal(t, taxcode.ErrCodeTaxCodeIsOrganizationDefault, vi.Code())
		})

		t.Run("DeleteCreditGrantDefaultIsBlocked", func(t *testing.T) {
			err := env.Service.DeleteTaxCode(t.Context(), taxcode.DeleteTaxCodeInput{
				NamespacedID: models.NamespacedID{Namespace: defaultNs, ID: creditGrant.ID},
			})
			require.Error(t, err)

			var vi models.ValidationIssue
			require.ErrorAs(t, err, &vi)
			assert.Equal(t, taxcode.ErrCodeTaxCodeIsOrganizationDefault, vi.Code())
		})

		t.Run("DeleteNonDefaultSucceeds", func(t *testing.T) {
			err := env.Service.DeleteTaxCode(t.Context(), taxcode.DeleteTaxCodeInput{
				NamespacedID: models.NamespacedID{Namespace: defaultNs, ID: other.ID},
			})
			require.NoError(t, err)
		})
	})

	t.Run("UserManaged", func(t *testing.T) {
		name := testutils.NameGenerator.Generate()
		tc, err := env.Service.CreateTaxCode(t.Context(), taxcode.CreateTaxCodeInput{
			Namespace: ns,
			Key:       name.Key,
			Name:      name.Name,
		})
		require.NoError(t, err)
		assert.False(t, tc.IsManagedBySystem())

		t.Run("UpdateSucceeds", func(t *testing.T) {
			updated, err := env.Service.UpdateTaxCode(t.Context(), taxcode.UpdateTaxCodeInput{
				NamespacedID: models.NamespacedID{Namespace: ns, ID: tc.ID},
				Name:         "updated name",
			})
			require.NoError(t, err)
			assert.Equal(t, "updated name", updated.Name)
		})

		t.Run("UpdateAnnotationsSucceeds", func(t *testing.T) {
			updated, err := env.Service.UpdateTaxCode(t.Context(), taxcode.UpdateTaxCodeInput{
				NamespacedID: models.NamespacedID{Namespace: ns, ID: tc.ID},
				Name:         tc.Name,
				Annotations: models.Annotations{
					taxcode.AnnotationKeyManagedBy: taxcode.AnnotationValueManagedBySystem,
					"schema_version":               1,
				},
			})
			require.NoError(t, err)
			assert.True(t, updated.IsManagedBySystem())
			assert.Equal(t, float64(1), updated.Annotations["schema_version"])
		})

		t.Run("DeleteSucceeds", func(t *testing.T) {
			// Create a fresh one to delete (the one above was updated, still valid).
			name2 := testutils.NameGenerator.Generate()
			tc2, err := env.Service.CreateTaxCode(t.Context(), taxcode.CreateTaxCodeInput{
				Namespace: ns,
				Key:       name2.Key,
				Name:      name2.Name,
			})
			require.NoError(t, err)

			err = env.Service.DeleteTaxCode(t.Context(), taxcode.DeleteTaxCodeInput{
				NamespacedID: models.NamespacedID{Namespace: ns, ID: tc2.ID},
			})
			require.NoError(t, err)
		})
	})

	t.Run("ListByIDs", func(t *testing.T) {
		// given:
		// - a live and a soft-deleted tax code that are requested
		// - a live tax code that is not requested
		// - a tax code with a requested ID in another namespace
		listNs := testutils.NameGenerator.Generate().Key
		env.ProvisionDefaultTaxCodes(t, listNs)

		live := env.CreateTaxCode(t, listNs)
		deleted := env.CreateTaxCode(t, listNs)
		env.CreateTaxCode(t, listNs)
		require.NoError(t, env.Service.DeleteTaxCode(t.Context(), taxcode.DeleteTaxCodeInput{
			NamespacedID: models.NamespacedID{Namespace: listNs, ID: deleted.ID},
		}))
		foreign := env.CreateTaxCode(t, testutils.NameGenerator.Generate().Key)

		ids := []string{live.ID, deleted.ID, foreign.ID}

		t.Run("LiveOnly", func(t *testing.T) {
			// when: listing by IDs without deleted codes
			res, err := env.Service.ListTaxCodes(t.Context(), taxcode.ListTaxCodesInput{
				Namespace: listNs,
				IDs:       ids,
			})

			// then: only the live requested code in the namespace is returned
			require.NoError(t, err)
			assert.ElementsMatch(t, []string{live.ID}, lo.Map(res.Items, func(tc taxcode.TaxCode, _ int) string { return tc.ID }))
		})

		t.Run("IncludeDeleted", func(t *testing.T) {
			// when: listing by IDs including deleted codes
			res, err := env.Service.ListTaxCodes(t.Context(), taxcode.ListTaxCodesInput{
				Namespace:      listNs,
				IDs:            ids,
				IncludeDeleted: true,
			})

			// then: live and soft-deleted requested codes in the namespace are returned
			require.NoError(t, err)
			assert.ElementsMatch(t, []string{live.ID, deleted.ID}, lo.Map(res.Items, func(tc taxcode.TaxCode, _ int) string { return tc.ID }))
		})

		t.Run("EmptyIDIsInvalid", func(t *testing.T) {
			_, err := env.Service.ListTaxCodes(t.Context(), taxcode.ListTaxCodesInput{
				Namespace: listNs,
				IDs:       []string{live.ID, ""},
			})

			require.Error(t, err)
			assert.True(t, models.IsGenericValidationError(err), "expected validation error, got: %v", err)
		})
	})

	t.Run("GetByAppMappingPrefersSystemManagedDuplicate", func(t *testing.T) {
		// given:
		// - a user-created tax code and a system-managed seed tax code share the same Stripe mapping
		// when:
		// - resolving by app mapping
		// then:
		// - the system-managed code is preferred over the user-created duplicate
		duplicateNs := testutils.NameGenerator.Generate().Key
		stripeCode := "txcd_10103001"

		_, err := env.Service.CreateTaxCode(t.Context(), taxcode.CreateTaxCodeInput{
			Namespace: duplicateNs,
			Key:       "stripe_txcd_10103001",
			Name:      stripeCode,
			AppMappings: taxcode.TaxCodeAppMappings{
				{AppType: app.AppTypeStripe, TaxCode: stripeCode},
			},
		})
		require.NoError(t, err)

		systemManaged, err := env.Service.CreateTaxCode(t.Context(), taxcode.CreateTaxCodeInput{
			Namespace: duplicateNs,
			Key:       "saas_business",
			Name:      "Software as a Service (SaaS) - Business Use",
			AppMappings: taxcode.TaxCodeAppMappings{
				{AppType: app.AppTypeStripe, TaxCode: stripeCode},
			},
			Annotations: models.Annotations{
				taxcode.AnnotationKeyManagedBy: taxcode.AnnotationValueManagedBySystem,
				"schema_version":               1,
			},
		})
		require.NoError(t, err)

		got, err := env.Service.GetTaxCodeByAppMapping(t.Context(), taxcode.GetTaxCodeByAppMappingInput{
			Namespace: duplicateNs,
			AppType:   app.AppTypeStripe,
			TaxCode:   stripeCode,
		})
		require.NoError(t, err)
		assert.Equal(t, systemManaged.ID, got.ID)
		assert.True(t, got.IsManagedBySystem())
	})
}

func TestTaxCodeServiceAppMappings(t *testing.T) {
	env := taxcodetestutils.NewTestEnv(t)
	t.Cleanup(func() { env.Close(t) })

	ns := testutils.NameGenerator.Generate().Key
	stripeCode := "txcd_10103001"

	// given:
	// - a user-managed tax code with a single Stripe app mapping
	tc := createTaxCodeWithStripeMapping(t, env, ns, stripeCode)
	assert.False(t, tc.IsManagedBySystem())

	t.Run("UpdateSucceedsWithUnchangedMapping", func(t *testing.T) {
		// given:
		// - the tax code has a Stripe app mapping
		// when:
		// - updating name and metadata while passing the mapping unchanged
		// then:
		// - the update succeeds and the mapping is kept
		updated, err := env.Service.UpdateTaxCode(t.Context(), taxcode.UpdateTaxCodeInput{
			NamespacedID: models.NamespacedID{Namespace: ns, ID: tc.ID},
			Name:         "updated name",
			AppMappings:  tc.AppMappings,
			Metadata:     models.Metadata{"tier": "enterprise"},
		})
		require.NoError(t, err)
		assert.Equal(t, "updated name", updated.Name)
		assert.Equal(t, tc.AppMappings, updated.AppMappings)
		assert.Equal(t, models.Metadata{"tier": "enterprise"}, updated.Metadata)
	})

	t.Run("UpdateSucceedsWithNewAppType", func(t *testing.T) {
		// given:
		// - the tax code has a Stripe app mapping
		// when:
		// - keeping the Stripe mapping and adding a mapping for a new app type
		// then:
		// - the update succeeds and carries both mappings
		updated, err := env.Service.UpdateTaxCode(t.Context(), taxcode.UpdateTaxCodeInput{
			NamespacedID: models.NamespacedID{Namespace: ns, ID: tc.ID},
			Name:         tc.Name,
			AppMappings: taxcode.TaxCodeAppMappings{
				{AppType: app.AppTypeStripe, TaxCode: stripeCode},
				{AppType: app.AppTypeCustomInvoicing, TaxCode: "custom_standard"},
			},
		})
		require.NoError(t, err)
		assert.Equal(t, taxcode.TaxCodeAppMappings{
			{AppType: app.AppTypeStripe, TaxCode: stripeCode},
			{AppType: app.AppTypeCustomInvoicing, TaxCode: "custom_standard"},
		}, updated.AppMappings)
	})

	t.Run("UpdateRejectsChangedTaxCode", func(t *testing.T) {
		// given:
		// - a tax code with a Stripe app mapping
		// when:
		// - updating with the same app type pointing to a different Stripe code
		// then:
		// - the update is rejected with ErrCodeTaxCodeAppMappingImmutable
		changed := createTaxCodeWithStripeMapping(t, env, ns, stripeCode)

		_, err := env.Service.UpdateTaxCode(t.Context(), taxcode.UpdateTaxCodeInput{
			NamespacedID: models.NamespacedID{Namespace: ns, ID: changed.ID},
			Name:         changed.Name,
			AppMappings: taxcode.TaxCodeAppMappings{
				{AppType: app.AppTypeStripe, TaxCode: "txcd_10002001"},
			},
		})
		require.Error(t, err)

		var vi models.ValidationIssue
		require.ErrorAs(t, err, &vi)
		assert.Equal(t, taxcode.ErrCodeTaxCodeAppMappingImmutable, vi.Code())
	})

	t.Run("UpdateRejectsRemovedMapping", func(t *testing.T) {
		// given:
		// - a tax code with a Stripe app mapping
		// when:
		// - updating without any app mappings
		// then:
		// - the update is rejected with ErrCodeTaxCodeAppMappingImmutable
		removed := createTaxCodeWithStripeMapping(t, env, ns, stripeCode)

		_, err := env.Service.UpdateTaxCode(t.Context(), taxcode.UpdateTaxCodeInput{
			NamespacedID: models.NamespacedID{Namespace: ns, ID: removed.ID},
			Name:         removed.Name,
		})
		require.Error(t, err)

		var vi models.ValidationIssue
		require.ErrorAs(t, err, &vi)
		assert.Equal(t, taxcode.ErrCodeTaxCodeAppMappingImmutable, vi.Code())
	})

	t.Run("UpdateOverrideSucceeds", func(t *testing.T) {
		// given:
		// - a tax code with a Stripe app mapping
		// when:
		// - changing the mapped Stripe code with AllowAppMappingOverride set
		// then:
		// - the update succeeds and the mapping points to the new code
		overridden := createTaxCodeWithStripeMapping(t, env, ns, stripeCode)

		input := taxcode.UpdateTaxCodeInput{
			NamespacedID: models.NamespacedID{Namespace: ns, ID: overridden.ID},
			Name:         overridden.Name,
			AppMappings: taxcode.TaxCodeAppMappings{
				{AppType: app.AppTypeStripe, TaxCode: "txcd_10002001"},
			},
		}
		input.AllowAppMappingOverride = true

		updated, err := env.Service.UpdateTaxCode(t.Context(), input)
		require.NoError(t, err)
		assert.Equal(t, taxcode.TaxCodeAppMappings{
			{AppType: app.AppTypeStripe, TaxCode: "txcd_10002001"},
		}, updated.AppMappings)
	})
}

// createTaxCodeWithStripeMapping creates a user-managed tax code whose only app
// mapping points to the given Stripe code.
func createTaxCodeWithStripeMapping(t *testing.T, env *taxcodetestutils.TestEnv, ns, stripeCode string) taxcode.TaxCode {
	t.Helper()

	name := testutils.NameGenerator.Generate()
	tc, err := env.Service.CreateTaxCode(t.Context(), taxcode.CreateTaxCodeInput{
		Namespace: ns,
		Key:       name.Key,
		Name:      name.Name,
		AppMappings: taxcode.TaxCodeAppMappings{
			{AppType: app.AppTypeStripe, TaxCode: stripeCode},
		},
	})
	require.NoError(t, err)

	return tc
}
