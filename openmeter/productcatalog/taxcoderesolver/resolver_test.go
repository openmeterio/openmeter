package taxcoderesolver_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/productcatalog/taxcoderesolver"
	"github.com/openmeterio/openmeter/openmeter/taxcode"
	taxcodetestutils "github.com/openmeterio/openmeter/openmeter/taxcode/testutils"
	"github.com/openmeterio/openmeter/openmeter/testutils"
	"github.com/openmeterio/openmeter/pkg/models"
)

func TestTaxCodeResolver(t *testing.T) {
	env := taxcodetestutils.NewTestEnv(t)

	namespace := testutils.NameGenerator.Generate().Key
	otherNamespace := testutils.NameGenerator.Generate().Key

	// Deleting a tax code checks it isn't an org default, so the namespace needs defaults.
	env.ProvisionDefaultTaxCodes(t, namespace)

	tc := env.CreateTaxCode(t, namespace)

	resolver, err := taxcoderesolver.New(env.Service)
	require.NoError(t, err)

	t.Run("Found", func(t *testing.T) {
		// given: a tax code in the namespace
		// when: resolving it by ID
		got, err := resolver.Resolve(t.Context(), namespace, tc.ID)

		// then: the tax code is returned
		require.NoError(t, err)
		require.NotNil(t, got)
		assert.Equal(t, tc.ID, got.ID)
		assert.Equal(t, tc.Key, got.Key)
	})

	t.Run("NotFound", func(t *testing.T) {
		_, err := resolver.Resolve(t.Context(), namespace, "01JBP3SGZ20Y7VRVC351TDFXYZ")

		require.Error(t, err)
		assert.True(t, taxcode.IsTaxCodeNotFoundError(err), "expected tax code not found error, got: %v", err)
	})

	t.Run("NotFoundInOtherNamespace", func(t *testing.T) {
		// given: the tax code ID belongs to a different namespace
		// when: resolving it in the other namespace
		_, err := resolver.Resolve(t.Context(), otherNamespace, tc.ID)

		// then: tax codes never resolve across namespaces
		require.Error(t, err)
		assert.True(t, taxcode.IsTaxCodeNotFoundError(err), "expected tax code not found error, got: %v", err)
	})

	t.Run("SoftDeletedIsNotFound", func(t *testing.T) {
		// given: a soft-deleted tax code
		deleted := env.CreateTaxCode(t, namespace)
		require.NoError(t, env.Service.DeleteTaxCode(t.Context(), taxcode.DeleteTaxCodeInput{
			NamespacedID: models.NamespacedID{Namespace: namespace, ID: deleted.ID},
		}))

		// when: resolving it by ID
		_, err := resolver.Resolve(t.Context(), namespace, deleted.ID)

		// then: single resolve only sees live tax codes
		require.Error(t, err)
		assert.True(t, taxcode.IsTaxCodeNotFoundError(err), "expected tax code not found error, got: %v", err)
	})

	t.Run("EmptyNamespace", func(t *testing.T) {
		_, err := resolver.Resolve(t.Context(), "", tc.ID)

		require.Error(t, err)
	})

	t.Run("BatchResolve", func(t *testing.T) {
		// given:
		// - a live and a soft-deleted tax code in the namespace
		// - a tax code in another namespace
		// - an ID that doesn't exist
		live := env.CreateTaxCode(t, namespace)
		deleted := env.CreateTaxCode(t, namespace)
		require.NoError(t, env.Service.DeleteTaxCode(t.Context(), taxcode.DeleteTaxCodeInput{
			NamespacedID: models.NamespacedID{Namespace: namespace, ID: deleted.ID},
		}))
		foreign := env.CreateTaxCode(t, otherNamespace)
		unknownID := "01JBP3SGZ20Y7VRVC351TDFXYZ"

		// when: batch resolving all of them
		got, err := resolver.BatchResolve(t.Context(), namespace, live.ID, deleted.ID, foreign.ID, unknownID)

		// then:
		// - live and soft-deleted codes come back keyed by ID; existing references must keep working
		// - other namespaces and unknown IDs are absent
		require.NoError(t, err)
		require.Len(t, got, 2)
		require.Contains(t, got, live.ID)
		assert.Equal(t, live.Key, got[live.ID].Key)
		assert.Nil(t, got[live.ID].DeletedAt)
		require.Contains(t, got, deleted.ID)
		assert.Equal(t, deleted.Key, got[deleted.ID].Key)
		assert.NotNil(t, got[deleted.ID].DeletedAt)
	})

	t.Run("BatchResolveNoIDs", func(t *testing.T) {
		got, err := resolver.BatchResolve(t.Context(), namespace)

		require.NoError(t, err)
		assert.Empty(t, got)
	})

	t.Run("BatchResolveEmptyNamespace", func(t *testing.T) {
		_, err := resolver.BatchResolve(t.Context(), "", tc.ID)

		require.Error(t, err)
	})

	t.Run("Namespaced", func(t *testing.T) {
		namespaced := resolver.WithNamespace(namespace)
		assert.Equal(t, namespace, namespaced.Namespace())

		got, err := namespaced.Resolve(t.Context(), tc.ID)
		require.NoError(t, err)
		require.NotNil(t, got)
		assert.Equal(t, tc.ID, got.ID)

		_, err = resolver.WithNamespace(otherNamespace).Resolve(t.Context(), tc.ID)
		require.Error(t, err)
		assert.True(t, taxcode.IsTaxCodeNotFoundError(err), "expected tax code not found error, got: %v", err)

		batch, err := namespaced.BatchResolve(t.Context(), tc.ID)
		require.NoError(t, err)
		require.Contains(t, batch, tc.ID)

		batch, err = resolver.WithNamespace(otherNamespace).BatchResolve(t.Context(), tc.ID)
		require.NoError(t, err)
		assert.Empty(t, batch)
	})
}

func TestNewRequiresService(t *testing.T) {
	_, err := taxcoderesolver.New(nil)
	require.Error(t, err)
}
