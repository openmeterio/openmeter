package chargeadapter

import (
	"testing"

	"github.com/alpacahq/alpacadecimal"
	"github.com/stretchr/testify/require"

	chargecreditpurchase "github.com/openmeterio/openmeter/openmeter/billing/charges/creditpurchase"
	currenciestestutils "github.com/openmeterio/openmeter/openmeter/currencies/testutils/currency"
	"github.com/openmeterio/openmeter/pkg/currencyx"
)

func TestSettlementPaymentAmount_CustomCurrency_UsesBankRounding(t *testing.T) {
	tests := []struct {
		name           string
		settlement     currencyx.FiatCode
		creditAmount   string
		costBasis      string
		expectedAmount string
	}{
		{
			name:           "below midpoint",
			settlement:     "USD",
			creditAmount:   "1",
			costBasis:      "1.004",
			expectedAmount: "1.00",
		},
		{
			name:           "midpoint rounds to even lower digit",
			settlement:     "USD",
			creditAmount:   "1",
			costBasis:      "1.005",
			expectedAmount: "1.00",
		},
		{
			name:           "above midpoint",
			settlement:     "USD",
			creditAmount:   "1",
			costBasis:      "1.006",
			expectedAmount: "1.01",
		},
		{
			name:           "midpoint rounds to even higher digit",
			settlement:     "USD",
			creditAmount:   "1",
			costBasis:      "2.675",
			expectedAmount: "2.68",
		},
		{
			name:           "midpoint after multiplication",
			settlement:     "EUR",
			creditAmount:   "2",
			costBasis:      "1.0025",
			expectedAmount: "2.00",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			customCurrency := currenciestestutils.NewCustomCurrency(t, "ACME", 2)
			costBasis := alpacadecimal.RequireFromString(tt.costBasis)
			settlement := chargecreditpurchase.NewSettlement(chargecreditpurchase.ExternalSettlement{
				InitialStatus: chargecreditpurchase.CreatedInitialPaymentSettlementStatus,
				GenericSettlement: chargecreditpurchase.GenericSettlement{
					Currency:  tt.settlement,
					CostBasis: costBasis,
				},
			})

			amount, currency, err := settlementPaymentAmount(
				customCurrency,
				settlement,
				alpacadecimal.RequireFromString(tt.creditAmount),
				costBasis,
			)

			require.NoError(t, err)
			require.Equal(t, currencyx.Code(tt.settlement), currency)
			require.True(t, amount.Equal(alpacadecimal.RequireFromString(tt.expectedAmount)))
		})
	}
}
