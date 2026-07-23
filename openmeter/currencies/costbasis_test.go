package currencies_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/currencies"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/models"
)

func TestCostBasisIsEffectiveAt(t *testing.T) {
	at := time.Date(2026, time.July, 27, 12, 0, 0, 0, time.UTC)
	futureDeletedAt := at.Add(time.Hour)

	testCases := []struct {
		name      string
		costBasis currencies.CostBasis
		expected  bool
	}{
		{
			name: "effective from is inclusive",
			costBasis: currencies.CostBasis{
				CostBasis: currencyx.CostBasis{EffectiveFrom: at},
			},
			expected: true,
		},
		{
			name: "effective to is exclusive",
			costBasis: currencies.CostBasis{
				CostBasis: currencyx.CostBasis{
					EffectiveFrom: at.Add(-time.Hour),
					EffectiveTo:   &at,
				},
			},
		},
		{
			name: "scheduled",
			costBasis: currencies.CostBasis{
				CostBasis: currencyx.CostBasis{EffectiveFrom: at.Add(time.Hour)},
			},
		},
		{
			name: "deleted",
			costBasis: currencies.CostBasis{
				ManagedModel: models.ManagedModel{DeletedAt: &at},
				CostBasis: currencyx.CostBasis{
					EffectiveFrom: at.Add(-time.Hour),
				},
			},
		},
		{
			name: "future deletion",
			costBasis: currencies.CostBasis{
				ManagedModel: models.ManagedModel{DeletedAt: &futureDeletedAt},
				CostBasis: currencyx.CostBasis{
					EffectiveFrom: at.Add(-time.Hour),
				},
			},
			expected: true,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			require.Equal(t, testCase.expected, testCase.costBasis.IsEffectiveAt(at))
		})
	}
}
