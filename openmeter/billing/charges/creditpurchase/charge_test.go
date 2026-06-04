package creditpurchase

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestFeatureFiltersStrings(t *testing.T) {
	require.Equal(t, []string{"api-calls", "storage"}, FeatureFilters([]string{"storage", "api-calls", "storage"}).Strings())
}

func TestFeatureFiltersValidate(t *testing.T) {
	t.Run("valid", func(t *testing.T) {
		require.NoError(t, FeatureFilters([]string{"api-calls", "storage"}).Validate())
	})

	t.Run("empty key", func(t *testing.T) {
		require.Error(t, FeatureFilters([]string{""}).Validate())
	})

	t.Run("duplicate key", func(t *testing.T) {
		require.Error(t, FeatureFilters([]string{"api-calls", "api-calls"}).Validate())
	})
}
