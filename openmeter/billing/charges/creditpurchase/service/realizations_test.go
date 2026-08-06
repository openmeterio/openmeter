package service

import (
	"testing"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/samber/lo"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/billing/charges/creditpurchase"
	chargecostbasis "github.com/openmeterio/openmeter/openmeter/billing/charges/models/costbasis"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/ledgertransaction"
	currenciestestutils "github.com/openmeterio/openmeter/openmeter/currencies/testutils"
	"github.com/openmeterio/openmeter/pkg/currencyx"
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
			expectedAmount := alpacadecimal.RequireFromString(tt.expectedAmount).InexactFloat64()
			charge := newExternalStateMachineTestChargeWithInput(t, externalStateMachineTestChargeInput{
				status:        creditpurchase.StatusActivePaymentPending,
				creditAmount:  alpacadecimal.RequireFromString(tt.creditAmount),
				costBasis:     alpacadecimal.RequireFromString(tt.costBasis),
				initialStatus: creditpurchase.CreatedInitialPaymentSettlementStatus,
			})
			if !tt.fiatCurrency {
				charge.Intent.Currency = currenciestestutils.NewCustomCurrency(t, "CREDITS", 2)
				fiatCurrency, err := currencyx.NewFiatCurrency("USD")
				require.NoError(t, err)
				charge.Intent.CostBasis = lo.ToPtr(creditpurchase.NewCostBasis(chargecostbasis.NewIntent(chargecostbasis.ManualIntent{
					FiatCurrency: fiatCurrency,
					Rate:         alpacadecimal.RequireFromString(tt.costBasis),
				})))
				charge.State.ResolvedCostBasis = &chargecostbasis.State{
					CostBasis:  alpacadecimal.RequireFromString(tt.costBasis),
					ResolvedAt: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
				}
			}
			handler := &externalStateMachineHandler{}
			handler.On("OnCreditPurchasePaymentAuthorized", mock.Anything, mock.Anything).
				Run(func(args mock.Arguments) {
					input := args.Get(1).(creditpurchase.PaymentEventInput)
					require.Equal(t, expectedAmount, input.FiatAmount.InexactFloat64())
				}).
				Return(ledgertransaction.GroupReference{TransactionGroupID: "authorized-ledger-tx"}, nil).
				Once()
			adapter := &externalStateMachineAdapter{}
			realizationsService := newExternalStateMachineRealizations(t, adapter, handler, &externalStateMachineLineage{})

			_, err := realizationsService.AuthorizeExternalPayment(t.Context(), charge)

			require.NoError(t, err)
			require.Equal(t, expectedAmount, adapter.createdExternalPayment.FiatAmount.InexactFloat64())
			handler.AssertExpectations(t)
		})
	}
}
