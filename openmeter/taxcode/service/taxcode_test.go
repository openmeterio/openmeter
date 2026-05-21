package service_test

import (
	"testing"

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
	env.DBSchemaMigrate(t)

	ns := testutils.NameGenerator.Generate().Key
	env.SetupNamespaceDefaults(t.Context(), t, ns)

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
		invoicing := env.CreateTaxCode(t.Context(), t, defaultNs)
		creditGrant := env.CreateTaxCode(t.Context(), t, defaultNs)
		other := env.CreateTaxCode(t.Context(), t, defaultNs)

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
