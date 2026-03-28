package llmcost

import (
	"testing"

	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	api "github.com/openmeterio/openmeter/api/v3"
)

func TestFilterSingleStringToDomain(t *testing.T) {
	t.Run("nil input returns nil", func(t *testing.T) {
		got, err := filterSingleStringToDomain(nil)
		require.NoError(t, err)
		assert.Nil(t, got)
	})

	t.Run("empty filter returns nil", func(t *testing.T) {
		got, err := filterSingleStringToDomain(&api.FilterSingleString{})
		require.NoError(t, err)
		assert.Nil(t, got)
	})

	t.Run("eq filter", func(t *testing.T) {
		got, err := filterSingleStringToDomain(&api.FilterSingleString{
			Eq: lo.ToPtr("system"),
		})
		require.NoError(t, err)
		require.NotNil(t, got)
		assert.Equal(t, lo.ToPtr("system"), got.Eq)
		assert.Nil(t, got.Neq)
		assert.Nil(t, got.Contains)
	})

	t.Run("neq filter", func(t *testing.T) {
		got, err := filterSingleStringToDomain(&api.FilterSingleString{
			Neq: lo.ToPtr("manual"),
		})
		require.NoError(t, err)
		require.NotNil(t, got)
		assert.Equal(t, lo.ToPtr("manual"), got.Neq)
		assert.Nil(t, got.Eq)
		assert.Nil(t, got.Contains)
	})

	t.Run("contains filter", func(t *testing.T) {
		got, err := filterSingleStringToDomain(&api.FilterSingleString{
			Contains: lo.ToPtr("sys"),
		})
		require.NoError(t, err)
		require.NotNil(t, got)
		assert.Equal(t, lo.ToPtr("sys"), got.Contains)
		assert.Nil(t, got.Eq)
		assert.Nil(t, got.Neq)
	})

	t.Run("mutually exclusive eq and neq returns error", func(t *testing.T) {
		_, err := filterSingleStringToDomain(&api.FilterSingleString{
			Eq:  lo.ToPtr("system"),
			Neq: lo.ToPtr("manual"),
		})
		require.Error(t, err)
	})

	t.Run("mutually exclusive eq and contains returns error", func(t *testing.T) {
		_, err := filterSingleStringToDomain(&api.FilterSingleString{
			Eq:       lo.ToPtr("system"),
			Contains: lo.ToPtr("sys"),
		})
		require.Error(t, err)
	})

	t.Run("mutually exclusive neq and contains returns error", func(t *testing.T) {
		_, err := filterSingleStringToDomain(&api.FilterSingleString{
			Neq:      lo.ToPtr("system"),
			Contains: lo.ToPtr("sys"),
		})
		require.Error(t, err)
	})
}

func TestFilterSourceUsedInListPricesParams(t *testing.T) {
	// Verify that the generated API types include the Source field in the filter struct.
	// This is a compile-time assertion that the TypeSpec → generated code pipeline
	// produced the expected field.
	filter := &api.ListLLMCostPricesParamsFilter{
		Source: &api.FilterSingleString{
			Eq: lo.ToPtr("system"),
		},
	}
	require.NotNil(t, filter.Source)
	assert.Equal(t, lo.ToPtr("system"), filter.Source.Eq)
}
