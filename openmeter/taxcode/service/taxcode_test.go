package service_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

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
}
