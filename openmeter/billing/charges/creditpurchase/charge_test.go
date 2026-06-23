package creditpurchase

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestFeatureFiltersNormalize(t *testing.T) {
	require.Equal(t, FeatureFilters{"api-calls", "storage"}, FeatureFilters([]string{"storage", "api-calls", "storage"}).Normalize())
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

func TestFeatureFiltersValidateAsFeatureFilter(t *testing.T) {
	t.Run("valid", func(t *testing.T) {
		require.NoError(t, FeatureFilters([]string{"api-calls"}).ValidateAsFeatureFilter())
	})

	t.Run("empty", func(t *testing.T) {
		require.Error(t, FeatureFilters(nil).ValidateAsFeatureFilter())
	})

	t.Run("multiple", func(t *testing.T) {
		require.Error(t, FeatureFilters([]string{"api-calls", "storage"}).ValidateAsFeatureFilter())
	})

	t.Run("invalid feature", func(t *testing.T) {
		require.Error(t, FeatureFilters([]string{""}).ValidateAsFeatureFilter())
	})
}
