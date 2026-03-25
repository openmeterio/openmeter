package entutils_test

import (
	"errors"
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

func (m mockEntity) GetID() string       { return m.ID }
func (m mockEntity) GetNamespace() string { return m.Namespace }

func entity(ns, id string) mockEntity {
	return mockEntity{Namespace: ns, ID: id}
}

func TestInIDOrder(t *testing.T) {
	t.Run("both empty", func(t *testing.T) {
		out, err := entutils.InIDOrder[mockEntity]("ns", nil, nil)
		require.NoError(t, err)
		assert.Empty(t, out)
	})

	t.Run("single element", func(t *testing.T) {
		out, err := entutils.InIDOrder("ns", []string{"a"}, []mockEntity{entity("ns", "a")})
		require.NoError(t, err)
		require.Len(t, out, 1)
		assert.Equal(t, "a", out[0].ID)
	})

	t.Run("reorders results to match target order", func(t *testing.T) {
		results := []mockEntity{
			entity("ns", "a"),
			entity("ns", "b"),
			entity("ns", "c"),
		}

		out, err := entutils.InIDOrder("ns", []string{"b", "a", "c"}, results)
		require.NoError(t, err)
		require.Len(t, out, 3)
		assert.Equal(t, "b", out[0].ID)
		assert.Equal(t, "a", out[1].ID)
		assert.Equal(t, "c", out[2].ID)
	})

	// -- Validation corner cases --

	t.Run("empty namespace", func(t *testing.T) {
		_, err := entutils.InIDOrder("", []string{"a"}, []mockEntity{entity("ns", "a")})
		require.ErrorIs(t, err, entutils.ErrNamespaceRequired)
	})

	t.Run("empty id in targetOrderIDs", func(t *testing.T) {
		_, err := entutils.InIDOrder("ns", []string{""}, []mockEntity{entity("ns", "a")})
		require.ErrorIs(t, err, entutils.ErrIDRequired)
	})

	t.Run("empty namespace in results", func(t *testing.T) {
		_, err := entutils.InIDOrder("ns", []string{"a"}, []mockEntity{entity("", "a")})
		require.Error(t, err)
	})

	t.Run("empty id in results", func(t *testing.T) {
		_, err := entutils.InIDOrder("ns", []string{"a"}, []mockEntity{entity("ns", "")})
		require.Error(t, err)
	})

	// -- Not found --

	t.Run("target id missing from results returns not found error", func(t *testing.T) {
		_, err := entutils.InIDOrder("ns", []string{"a", "missing"}, []mockEntity{entity("ns", "a")})
		require.Error(t, err)
		assert.True(t, models.IsGenericNotFoundError(err))
		assert.ErrorIs(t, err, entutils.ErrNotFound)
	})

	t.Run("multiple missing ids are all reported", func(t *testing.T) {
		_, err := entutils.InIDOrder("ns", []string{"x", "y"}, []mockEntity{entity("ns", "a")})
		require.Error(t, err)
		assert.True(t, models.IsGenericNotFoundError(err))
	})

	// -- Namespace isolation (security) --

	t.Run("same id different namespace is not found", func(t *testing.T) {
		// Ensures cross-namespace access is not possible
		_, err := entutils.InIDOrder("tenant-a", []string{"id1"}, []mockEntity{entity("tenant-b", "id1")})
		require.Error(t, err)
		assert.True(t, models.IsGenericNotFoundError(err))
	})

	// -- Edge cases --

	t.Run("extra results are tolerated", func(t *testing.T) {
		results := []mockEntity{
			entity("ns", "a"),
			entity("ns", "b"),
			entity("ns", "c"),
		}

		out, err := entutils.InIDOrder("ns", []string{"a"}, results)
		require.NoError(t, err)
		require.Len(t, out, 1)
		assert.Equal(t, "a", out[0].ID)
	})

	t.Run("duplicate target ids produce duplicate output", func(t *testing.T) {
		out, err := entutils.InIDOrder("ns", []string{"a", "a"}, []mockEntity{entity("ns", "a")})
		require.NoError(t, err)
		assert.Len(t, out, 2)
	})

	t.Run("empty targetOrderIDs with non-empty results", func(t *testing.T) {
		out, err := entutils.InIDOrder("ns", []string{}, []mockEntity{entity("ns", "a")})
		require.NoError(t, err)
		assert.Empty(t, out)
	})

	t.Run("non-empty targetOrderIDs with empty results returns not found", func(t *testing.T) {
		_, err := entutils.InIDOrder[mockEntity]("ns", []string{"a"}, nil)
		require.Error(t, err)
		assert.True(t, models.IsGenericNotFoundError(err))
	})

	t.Run("duplicate results return error", func(t *testing.T) {
		_, err := entutils.InIDOrder("ns", []string{"a"}, []mockEntity{entity("ns", "a"), entity("ns", "a")})
		require.Error(t, err)
		assert.True(t, errors.Is(err, entutils.ErrDuplicateID))
	})
}
