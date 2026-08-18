package run

import (
	"testing"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/costbasis"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/invoicedusage"
	"github.com/openmeterio/openmeter/openmeter/billing/models/totals"
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

func TestAllocateFiatOverageCreditsInputRejectsCompletedAllocation(t *testing.T) {
	// given:
	// - a custom-currency credit-then-invoice run already completed fiat allocation
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
	run.InvoiceUsage = &invoicedusage.AccruedUsage{
		ServicePeriod: charge.Intent.GetEffectiveServicePeriod(),
		Totals: totals.Totals{
			Amount: alpacadecimal.NewFromInt(5),
			Total:  alpacadecimal.NewFromInt(5),
		},
	}
	run.FiatOverageCreditAllocationCompleted = true

	// when:
	// - invoice finalization attempts the completed allocation again
	err = (AllocateFiatOverageCreditsInput{
		Charge: charge,
		Run:    run,
	}).Validate()

	// then:
	// - validation rejects the repeated allocation before any handler effect
	require.ErrorContains(t, err, "fiat overage credit allocation already completed")
}
