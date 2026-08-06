package service

import (
	"testing"

	"github.com/alpacahq/alpacadecimal"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/billing/charges/creditpurchase"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/ledgertransaction"
	currenciestestutils "github.com/openmeterio/openmeter/openmeter/currencies/testutils"
)

func TestRealizationsAuthorizeExternalPaymentRoundsFiatAmount(t *testing.T) {
	tests := []struct {
		name           string
		creditAmount   string
		costBasis      string
		expectedAmount string
		fiatCurrency   bool
	}{
		{
			name:           "below midpoint",
			creditAmount:   "1",
			costBasis:      "1.004",
			expectedAmount: "1.00",
		},
		{
			name:           "midpoint rounds up",
			creditAmount:   "1",
			costBasis:      "1.005",
			expectedAmount: "1.01",
		},
		{
			name:           "above midpoint",
			creditAmount:   "1",
			costBasis:      "1.006",
			expectedAmount: "1.01",
		},
		{
			name:           "midpoint rounds to even higher digit",
			creditAmount:   "1",
			costBasis:      "2.675",
			expectedAmount: "2.68",
		},
		{
			name:           "midpoint after multiplication",
			creditAmount:   "2",
			costBasis:      "1.0025",
			expectedAmount: "2.01",
		},
		{
			name:           "fiat credit currency uses cost basis",
			creditAmount:   "1",
			costBasis:      "0.005",
			expectedAmount: "0.01",
			fiatCurrency:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			charge := newExternalStateMachineTestChargeWithInput(t, externalStateMachineTestChargeInput{
				status:        creditpurchase.StatusActivePaymentPending,
				creditAmount:  alpacadecimal.RequireFromString(tt.creditAmount),
				costBasis:     alpacadecimal.RequireFromString(tt.costBasis),
				initialStatus: creditpurchase.CreatedInitialPaymentSettlementStatus,
			})
			if !tt.fiatCurrency {
				charge.Intent.Currency = currenciestestutils.NewCustomCurrency(t, "CREDITS", 2)
			}
			handler := &externalStateMachineHandler{}
			handler.On("OnCreditPurchasePaymentAuthorized", mock.Anything, mock.Anything).
				Run(func(args mock.Arguments) {
					input := args.Get(1).(creditpurchase.PaymentEventInput)
					require.True(t, input.FiatAmount.Equal(alpacadecimal.RequireFromString(tt.expectedAmount)))
				}).
				Return(ledgertransaction.GroupReference{TransactionGroupID: "authorized-ledger-tx"}, nil).
				Once()
			adapter := &externalStateMachineAdapter{}
			realizationsService := newExternalStateMachineRealizations(t, adapter, handler, &externalStateMachineLineage{})

			_, err := realizationsService.AuthorizeExternalPayment(t.Context(), charge)

			require.NoError(t, err)
			require.True(t, adapter.createdExternalPayment.FiatAmount.Equal(alpacadecimal.RequireFromString(tt.expectedAmount)))
			handler.AssertExpectations(t)
		})
	}
}
