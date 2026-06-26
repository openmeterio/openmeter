package currencies

import (
	"testing"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/currencies"
	"github.com/openmeterio/openmeter/pkg/models"
)

func TestToAPIBillingCostBasisExposesIDAndEffectiveTo(t *testing.T) {
	effectiveFrom := time.Date(2026, 6, 26, 12, 0, 0, 0, time.UTC)
	effectiveTo := effectiveFrom.Add(24 * time.Hour)
	createdAt := effectiveFrom.Add(-time.Hour)

	tests := []struct {
		name        string
		effectiveTo *time.Time
	}{
		{
			name:        "bounded",
			effectiveTo: &effectiveTo,
		},
		{
			name:        "open ended",
			effectiveTo: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ToAPIBillingCostBasis(currencies.CostBasis{
				NamespacedID: models.NamespacedID{
					ID:        "01K00000000000000000000000",
					Namespace: "test",
				},
				ManagedModel: models.ManagedModel{
					CreatedAt: createdAt,
				},
				CurrencyID:    "01J00000000000000000000000",
				FiatCode:      "USD",
				Rate:          alpacadecimal.RequireFromString("0.5"),
				EffectiveFrom: effectiveFrom,
				EffectiveTo:   tt.effectiveTo,
			})

			require.Equal(t, "01K00000000000000000000000", got.Id)
			require.Equal(t, "USD", got.FiatCode)
			require.Equal(t, "0.5", got.Rate)
			require.NotNil(t, got.EffectiveFrom)
			require.Equal(t, effectiveFrom, *got.EffectiveFrom)
			if tt.effectiveTo == nil {
				require.Nil(t, got.EffectiveTo)
			} else {
				require.NotNil(t, got.EffectiveTo)
				require.Equal(t, *tt.effectiveTo, *got.EffectiveTo)
			}
			require.Equal(t, createdAt, got.CreatedAt)
		})
	}
}
