package entutils_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/pkg/framework/entutils"
	"github.com/openmeterio/openmeter/pkg/models"
)

// mockEntity implements InIDOrderAccessor.
type mockEntity struct {
	ID        string
	Namespace string
}

func (m mockEntity) GetID() string        { return m.ID }
func (m mockEntity) GetNamespace() string  { return m.Namespace }

func nid(ns, id string) models.NamespacedID {
	return models.NamespacedID{Namespace: ns, ID: id}
}

func entity(ns, id string) mockEntity {
	return mockEntity{Namespace: ns, ID: id}
}

func TestInIDOrder(t *testing.T) {
	t.Run("both empty", func(t *testing.T) {
		out, err := entutils.InIDOrder[mockEntity](nil, nil)
		require.NoError(t, err)
		assert.Empty(t, out)
	})

	t.Run("single element", func(t *testing.T) {
		ids := []models.NamespacedID{nid("ns", "a")}
		results := []mockEntity{entity("ns", "a")}

		out, err := entutils.InIDOrder(ids, results)
		require.NoError(t, err)
		require.Len(t, out, 1)
		assert.Equal(t, "a", out[0].ID)
	})

	t.Run("reorders results to match target order", func(t *testing.T) {
		ids := []models.NamespacedID{
			nid("ns", "b"),
			nid("ns", "a"),
			nid("ns", "c"),
		}
		results := []mockEntity{
			entity("ns", "a"),
			entity("ns", "b"),
			entity("ns", "c"),
		}

		out, err := entutils.InIDOrder(ids, results)
		require.NoError(t, err)
		require.Len(t, out, 3)
		assert.Equal(t, "b", out[0].ID)
		assert.Equal(t, "a", out[1].ID)
		assert.Equal(t, "c", out[2].ID)
	})

	// -- Validation / security corner cases --

	t.Run("empty namespace in targetOrderIDs", func(t *testing.T) {
		ids := []models.NamespacedID{nid("", "a")}
		results := []mockEntity{entity("ns", "a")}

		_, err := entutils.InIDOrder(ids, results)
		require.Error(t, err)
	})

	t.Run("empty id in targetOrderIDs", func(t *testing.T) {
		ids := []models.NamespacedID{nid("ns", "")}
		results := []mockEntity{entity("ns", "a")}

		_, err := entutils.InIDOrder(ids, results)
		require.Error(t, err)
	})

	t.Run("empty namespace in results", func(t *testing.T) {
		ids := []models.NamespacedID{nid("ns", "a")}
		results := []mockEntity{entity("", "a")}

		_, err := entutils.InIDOrder(ids, results)
		require.Error(t, err)
	})

	t.Run("empty id in results", func(t *testing.T) {
		ids := []models.NamespacedID{nid("ns", "a")}
		results := []mockEntity{entity("ns", "")}

		_, err := entutils.InIDOrder(ids, results)
		require.Error(t, err)
	})

	// -- Not found --

	t.Run("target id missing from results returns not found error", func(t *testing.T) {
		ids := []models.NamespacedID{nid("ns", "a"), nid("ns", "missing")}
		results := []mockEntity{entity("ns", "a")}

		_, err := entutils.InIDOrder(ids, results)
		require.Error(t, err)
		assert.True(t, models.IsGenericNotFoundError(err))
	})

	t.Run("multiple missing ids are all reported", func(t *testing.T) {
		ids := []models.NamespacedID{nid("ns", "x"), nid("ns", "y")}
		results := []mockEntity{entity("ns", "a")}

		_, err := entutils.InIDOrder(ids, results)
		require.Error(t, err)
		assert.True(t, models.IsGenericNotFoundError(err))
	})

	// -- Namespace isolation (security) --

	t.Run("same id different namespace is not found", func(t *testing.T) {
		// Ensures cross-namespace access is not possible
		ids := []models.NamespacedID{nid("tenant-a", "id1")}
		results := []mockEntity{entity("tenant-b", "id1")}

		_, err := entutils.InIDOrder(ids, results)
		require.Error(t, err)
		assert.True(t, models.IsGenericNotFoundError(err))
	})

	t.Run("multi-namespace results with correct targeting", func(t *testing.T) {
		ids := []models.NamespacedID{nid("ns1", "a"), nid("ns2", "b")}
		results := []mockEntity{
			entity("ns1", "a"),
			entity("ns2", "b"),
			entity("ns3", "c"), // extra entity from another namespace, not targeted
		}

		out, err := entutils.InIDOrder(ids, results)
		require.NoError(t, err)
		require.Len(t, out, 2)
		assert.Equal(t, "ns1", out[0].Namespace)
		assert.Equal(t, "ns2", out[1].Namespace)
	})

	// -- Edge cases --

	t.Run("extra results are tolerated", func(t *testing.T) {
		ids := []models.NamespacedID{nid("ns", "a")}
		results := []mockEntity{
			entity("ns", "a"),
			entity("ns", "b"),
			entity("ns", "c"),
		}

		out, err := entutils.InIDOrder(ids, results)
		require.NoError(t, err)
		require.Len(t, out, 1)
		assert.Equal(t, "a", out[0].ID)
	})

	t.Run("duplicate target ids produce duplicate output", func(t *testing.T) {
		ids := []models.NamespacedID{nid("ns", "a"), nid("ns", "a")}
		results := []mockEntity{entity("ns", "a")}

		out, err := entutils.InIDOrder(ids, results)
		require.NoError(t, err)
		assert.Len(t, out, 2)
	})

	t.Run("empty targetOrderIDs with non-empty results", func(t *testing.T) {
		ids := []models.NamespacedID{}
		results := []mockEntity{entity("ns", "a")}

		out, err := entutils.InIDOrder(ids, results)
		require.NoError(t, err)
		assert.Empty(t, out)
	})

	t.Run("non-empty targetOrderIDs with empty results returns not found", func(t *testing.T) {
		ids := []models.NamespacedID{nid("ns", "a")}

		_, err := entutils.InIDOrder[mockEntity](ids, nil)
		require.Error(t, err)
		assert.True(t, models.IsGenericNotFoundError(err))
	})

	t.Run("duplicate results return error", func(t *testing.T) {
		ids := []models.NamespacedID{nid("ns", "a")}
		results := []mockEntity{entity("ns", "a"), entity("ns", "a")}

		_, err := entutils.InIDOrder(ids, results)
		require.Error(t, err)
	})
}
