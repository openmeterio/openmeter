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

	t.Run("eq maps correctly", func(t *testing.T) {
		got, err := filterSingleStringToDomain(&api.FilterSingleString{
			Eq: lo.ToPtr("system"),
		})
		require.NoError(t, err)
		require.NotNil(t, got)
		assert.Equal(t, lo.ToPtr("system"), got.Eq)
		assert.Nil(t, got.Neq)
		assert.Nil(t, got.Contains)
	})

	t.Run("neq maps correctly", func(t *testing.T) {
		got, err := filterSingleStringToDomain(&api.FilterSingleString{
			Neq: lo.ToPtr("manual"),
		})
		require.NoError(t, err)
		require.NotNil(t, got)
		assert.Equal(t, lo.ToPtr("manual"), got.Neq)
	})

	t.Run("contains maps correctly", func(t *testing.T) {
		got, err := filterSingleStringToDomain(&api.FilterSingleString{
			Contains: lo.ToPtr("sys"),
		})
		require.NoError(t, err)
		require.NotNil(t, got)
		assert.Equal(t, lo.ToPtr("sys"), got.Contains)
	})
}

func TestFilterSourceInListPricesParams(t *testing.T) {
	// Compile-time assertion that the generated API type includes the Source field.
	filter := &api.ListLLMCostPricesParamsFilter{
		Source: &api.FilterSingleString{
			Eq: lo.ToPtr("system"),
		},
	}
	require.NotNil(t, filter.Source)
	assert.Equal(t, lo.ToPtr("system"), filter.Source.Eq)
}
