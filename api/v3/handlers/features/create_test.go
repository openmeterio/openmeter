package features

import (
	"testing"

	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	api "github.com/openmeterio/openmeter/api/v3"
	"github.com/openmeterio/openmeter/openmeter/meter"
)

func TestValidateMeterFilters(t *testing.T) {
	testMeter := meter.Meter{
		Key: "tokens_total",
		GroupBy: map[string]string{
			"provider": "$.provider",
			"model":    "$.model",
			"type":     "$.type",
		},
	}

	t.Run("valid filters", func(t *testing.T) {
		filters := map[string]api.QueryFilterStringMapItem{
			"provider": {Eq: lo.ToPtr("openai")},
			"model":    {In: &[]string{"gpt-4", "gpt-4o"}},
		}

		err := validateMeterFilters(filters, testMeter)
		require.NoError(t, err)
	})

	t.Run("invalid dimension key", func(t *testing.T) {
		filters := map[string]api.QueryFilterStringMapItem{
			"nonexistent": {Eq: lo.ToPtr("value")},
		}

		err := validateMeterFilters(filters, testMeter)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "nonexistent")
		assert.Contains(t, err.Error(), "not a valid dimension")
	})

	t.Run("empty filters", func(t *testing.T) {
		err := validateMeterFilters(map[string]api.QueryFilterStringMapItem{}, testMeter)
		require.NoError(t, err)
	})

	t.Run("mix of valid and invalid keys", func(t *testing.T) {
		filters := map[string]api.QueryFilterStringMapItem{
			"provider": {Eq: lo.ToPtr("openai")},
			"bad_key":  {Eq: lo.ToPtr("value")},
		}

		err := validateMeterFilters(filters, testMeter)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "bad_key")
	})
}
