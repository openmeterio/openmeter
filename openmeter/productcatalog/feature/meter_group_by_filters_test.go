package feature

import (
	"testing"

	"github.com/samber/lo"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/meter"
	"github.com/openmeterio/openmeter/pkg/filter"
)

func TestMeterGroupByFiltersValidate(t *testing.T) {
	m := meter.Meter{
		Key: "tokens_total",
		GroupBy: map[string]string{
			"provider": "$.provider",
			"model":    "$.model",
		},
	}

	t.Run("valid single operator", func(t *testing.T) {
		f := MeterGroupByFilters{
			"provider": filter.FilterString{Eq: lo.ToPtr("openai")},
		}

		require.NoError(t, f.Validate(m))
	})

	t.Run("unknown dimension key", func(t *testing.T) {
		f := MeterGroupByFilters{
			"nonexistent": filter.FilterString{Eq: lo.ToPtr("value")},
		}

		err := f.Validate(m)
		require.Error(t, err)
	})

	t.Run("multiple operators on a single filter rejected", func(t *testing.T) {
		f := MeterGroupByFilters{
			"provider": filter.FilterString{
				Eq: lo.ToPtr("openai"),
				Ne: lo.ToPtr("anthropic"),
			},
		}

		err := f.Validate(m)
		require.Error(t, err)
	})
}
