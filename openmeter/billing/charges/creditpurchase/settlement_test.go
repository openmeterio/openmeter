package creditpurchase

import (
	"testing"

	"github.com/alpacahq/alpacadecimal"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/models"
)

func TestGenericSettlementValidateRequiresPositiveCostBasis(t *testing.T) {
	for _, tc := range []struct {
		name      string
		costBasis alpacadecimal.Decimal
		wantErr   bool
	}{
		{
			name:      "positive",
			costBasis: alpacadecimal.NewFromFloat(0.5),
		},
		{
			name:      "zero",
			costBasis: alpacadecimal.Zero,
			wantErr:   true,
		},
		{
			name:      "negative",
			costBasis: alpacadecimal.NewFromFloat(-0.5),
			wantErr:   true,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			settlement := GenericSettlement{
				Currency:  currencyx.Code("USD"),
				CostBasis: tc.costBasis,
			}

			err := settlement.Validate()

			if tc.wantErr {
				require.Error(t, err)
				require.ErrorContains(t, err, "cost basis must be positive")
				require.True(t, models.IsGenericValidationError(err))
				return
			}

			require.NoError(t, err)
		})
	}
}
