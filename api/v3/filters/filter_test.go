package filters

import (
	"testing"

	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStringFilter_IsEmpty(t *testing.T) {
	assert.True(t, StringFilter{}.IsEmpty())
	assert.False(t, StringFilter{Eq: lo.ToPtr("x")}.IsEmpty())
	assert.False(t, StringFilter{Neq: lo.ToPtr("x")}.IsEmpty())
	assert.False(t, StringFilter{Contains: lo.ToPtr("x")}.IsEmpty())
}

func TestStringFilter_Validate(t *testing.T) {
	t.Run("empty is valid", func(t *testing.T) {
		assert.NoError(t, StringFilter{}.Validate())
	})

	t.Run("single eq is valid", func(t *testing.T) {
		assert.NoError(t, StringFilter{Eq: lo.ToPtr("system")}.Validate())
	})

	t.Run("single neq is valid", func(t *testing.T) {
		assert.NoError(t, StringFilter{Neq: lo.ToPtr("manual")}.Validate())
	})

	t.Run("single contains is valid", func(t *testing.T) {
		assert.NoError(t, StringFilter{Contains: lo.ToPtr("sys")}.Validate())
	})

	t.Run("eq and neq are mutually exclusive", func(t *testing.T) {
		require.Error(t, StringFilter{
			Eq:  lo.ToPtr("system"),
			Neq: lo.ToPtr("manual"),
		}.Validate())
	})

	t.Run("eq and contains are mutually exclusive", func(t *testing.T) {
		require.Error(t, StringFilter{
			Eq:       lo.ToPtr("system"),
			Contains: lo.ToPtr("sys"),
		}.Validate())
	})

	t.Run("neq and contains are mutually exclusive", func(t *testing.T) {
		require.Error(t, StringFilter{
			Neq:      lo.ToPtr("system"),
			Contains: lo.ToPtr("sys"),
		}.Validate())
	})
}
