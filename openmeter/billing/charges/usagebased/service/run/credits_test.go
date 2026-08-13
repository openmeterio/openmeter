package run

import (
	"testing"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/costbasis"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/creditrealization"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/ledgertransaction"
	currenciestestutils "github.com/openmeterio/openmeter/openmeter/currencies/testutils"
	"github.com/openmeterio/openmeter/pkg/currencyx"
)

func TestChargeCurrencyCreditReconciliationHandlerOwnsCurrencyCalculator(t *testing.T) {
	service := Service{}
	handler := service.NewChargeCurrencyCreditReconciliationHandler(CreditReconciliationHandlerInput{
		Charge:     newUsageBasedCharge(t),
		Run:        newUsageBasedRun("line-1"),
		AllocateAt: time.Now().UTC(),
	})

	require.NoError(t, handler.Validate())
	require.Equal(t, "USD", handler.CurrencyCalculator().Details().Code.String())
}

func TestAllocateFiatOverageCreditsInputRejectsExistingRealizations(t *testing.T) {
	// given:
	// - a custom-currency credit-then-invoice run already has a fiat allocation
	charge := newUsageBasedCharge(t)
	baseIntent := charge.Intent.GetBaseIntent()
	baseIntent.Currency = currenciestestutils.NewCustomCurrency(t, "TOKENS", 2)
	fiatCurrency, err := currencyx.NewFiatCurrency("USD")
	require.NoError(t, err)
	costBasisIntent := costbasis.NewIntent(costbasis.ManualIntent{
		FiatCurrency: fiatCurrency,
		Rate:         alpacadecimal.NewFromFloat(0.5),
	})
	baseIntent.CostBasis = &costBasisIntent
	charge.Intent = baseIntent.AsOverridableIntent()
	costBasisID := "cost-basis-1"
	charge.State.CostBasisID = &costBasisID

	run := newUsageBasedRun("line-1")
	currentRunID := run.ID.ID
	charge.State.CurrentRealizationRunID = &currentRunID
	run.FiatOverageCreditRealizations = creditrealization.Realizations{
		{
			CreateInput: creditrealization.CreateInput{
				ID:            "fiat-allocation-1",
				ServicePeriod: charge.Intent.GetEffectiveServicePeriod(),
				LedgerTransaction: ledgertransaction.GroupReference{
					TransactionGroupID: "fiat-allocation-transaction-1",
				},
				Amount: alpacadecimal.NewFromInt(5),
				Type:   creditrealization.TypeAllocation,
			},
		},
	}

	// when:
	// - invoice finalization attempts the one-shot fiat allocation again
	err = (AllocateFiatOverageCreditsInput{
		Charge: charge,
		Run:    run,
	}).Validate()

	// then:
	// - validation rejects the repeated allocation before any handler effect
	require.ErrorContains(t, err, "run already has fiat overage credit realizations")
}
