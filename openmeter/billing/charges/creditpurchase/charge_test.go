package creditpurchase

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/billing/charges/meta"
	"github.com/openmeterio/openmeter/pkg/timeutil"
)

func TestIntentNormalizedPinsServicePeriodsToEffectiveAt(t *testing.T) {
	effectiveAt := time.Date(2026, 4, 17, 11, 23, 0, 0, time.UTC)
	originalPeriod := timeutil.ClosedPeriod{
		From: effectiveAt.Add(-time.Hour),
		To:   effectiveAt.Add(time.Hour),
	}

	intent := Intent{
		IntentMutableFields: IntentMutableFields{
			IntentMutableFields: meta.IntentMutableFields{
				ServicePeriod:     originalPeriod,
				FullServicePeriod: originalPeriod,
				BillingPeriod:     originalPeriod,
			},
			EffectiveAt: &effectiveAt,
		},
	}

	got := intent.Normalized()

	expectedPeriod := timeutil.ClosedPeriod{From: effectiveAt, To: effectiveAt}
	require.Equal(t, expectedPeriod, got.ServicePeriod)
	require.Equal(t, expectedPeriod, got.FullServicePeriod)
	require.Equal(t, expectedPeriod, got.BillingPeriod)
}

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
