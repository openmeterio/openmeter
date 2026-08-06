package creditpurchase

import (
	"encoding/json"
	"testing"

	"github.com/alpacahq/alpacadecimal"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/models"
)

func TestPersistedSettlementValidateRequiresPositiveCostBasis(t *testing.T) {
	currency := currencyx.FiatCode("USD")

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
			settlement := PersistedSettlement{
				Type:      SettlementTypeInvoice,
				Currency:  &currency,
				CostBasis: &tc.costBasis,
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

func TestPersistedSettlementRejectsCustomCurrencyCode(t *testing.T) {
	currency := currencyx.FiatCode("TOKENS")
	costBasis := alpacadecimal.NewFromFloat(0.5)
	settlement := PersistedSettlement{
		Type:      SettlementTypeInvoice,
		Currency:  &currency,
		CostBasis: &costBasis,
	}

	err := settlement.Validate()

	require.Error(t, err)
	require.ErrorContains(t, err, "invalid fiat currency code: TOKENS")
	require.True(t, models.IsGenericValidationError(err))
}

func TestSettlementJSONRoundTripPreservesFiatCurrencyCode(t *testing.T) {
	currency := currencyx.FiatCode("USD")
	costBasis := alpacadecimal.NewFromFloat(0.5)
	settlement := PersistedSettlement{
		Type:      SettlementTypeInvoice,
		Currency:  &currency,
		CostBasis: &costBasis,
	}

	data, err := json.Marshal(settlement)
	require.NoError(t, err)
	require.JSONEq(t, `{"type":"invoice","currency":"USD","costBasis":"0.5"}`, string(data))

	var decoded PersistedSettlement
	require.NoError(t, json.Unmarshal(data, &decoded))

	invoiceSettlement, err := decoded.AsSettlement()
	require.NoError(t, err)
	require.Equal(t, SettlementTypeInvoice, invoiceSettlement.Type())
	require.Equal(t, currencyx.FiatCode("USD"), *decoded.Currency)
	require.Equal(t, float64(0.5), decoded.CostBasis.InexactFloat64())
}
