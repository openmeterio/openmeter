package creditpurchase

import (
	"encoding/json"
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
				Currency:  currencyx.FiatCode("USD"),
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

func TestGenericSettlementRejectsCustomCurrencyCode(t *testing.T) {
	settlement := GenericSettlement{
		Currency:  currencyx.FiatCode("TOKENS"),
		CostBasis: alpacadecimal.NewFromFloat(0.5),
	}

	err := settlement.Validate()

	require.Error(t, err)
	require.ErrorContains(t, err, "invalid fiat currency code: TOKENS")
	require.True(t, models.IsGenericValidationError(err))
}

func TestSettlementJSONRoundTripPreservesFiatCurrencyCode(t *testing.T) {
	settlement := NewSettlement(InvoiceSettlement{
		GenericSettlement: GenericSettlement{
			Currency:  currencyx.FiatCode("USD"),
			CostBasis: alpacadecimal.NewFromFloat(0.5),
		},
	})

	data, err := json.Marshal(settlement)
	require.NoError(t, err)
	require.JSONEq(t, `{"type":"invoice","currency":"USD","costBasis":"0.5"}`, string(data))

	var decoded Settlement
	require.NoError(t, json.Unmarshal(data, &decoded))

	invoiceSettlement, err := decoded.AsInvoiceSettlement()
	require.NoError(t, err)
	require.Equal(t, currencyx.FiatCode("USD"), invoiceSettlement.Currency)
	require.Equal(t, float64(0.5), invoiceSettlement.CostBasis.InexactFloat64())
}
