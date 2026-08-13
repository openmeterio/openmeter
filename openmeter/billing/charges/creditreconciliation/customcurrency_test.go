package creditreconciliation

import (
	"testing"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/costbasis"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/creditrealization"
	"github.com/openmeterio/openmeter/openmeter/billing/models/totals"
	"github.com/openmeterio/openmeter/openmeter/currencies"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/models"
)

func TestReconcileCustomCurrencyWithFiatOverage(t *testing.T) {
	input := newCustomCurrencyWithFiatOverageInput(t)
	chargeCurrencyHandler := &testHandler{
		allocations: creditrealization.CreateAllocationInputs{
			{Amount: alpacadecimal.NewFromInt(12)},
		},
	}
	fiatOverageHandler := &testHandler{
		allocations: creditrealization.CreateAllocationInputs{
			{Amount: alpacadecimal.NewFromInt(5)},
		},
	}
	input.ChargeCurrencyHandler = chargeCurrencyHandler
	input.FiatOverageHandler = fiatOverageHandler

	// given: 26 TOKENS with 12 TOKENS and 5 USD available for allocation

	// when: charge currency and fiat overage credits are reconciled together
	result, err := ReconcileCustomCurrencyWithFiatOverage(t.Context(), input)

	// then: only the post-TOKEN-credit overage is converted before USD allocation
	require.NoError(t, err)
	require.Equal(t, float64(12), result.Totals.CreditsTotal.InexactFloat64())
	require.Equal(t, float64(14), result.Totals.Total.InexactFloat64())
	require.Equal(t, float64(7), result.ConvertedFiatOverage.Amount.InexactFloat64())
	require.Equal(t, float64(2), result.RemainingFiatOverage.InexactFloat64())
	require.Equal(t, float64(26), chargeCurrencyHandler.allocated.InexactFloat64())
	require.Equal(t, float64(7), fiatOverageHandler.allocated.InexactFloat64())
	require.Len(t, result.ChargeCurrency.Realizations, 1)
	require.Len(t, result.FiatOverageCredits.Realizations, 1)
}

func TestReconcileCustomCurrencyWithFiatOverageValidatesBothHandlersBeforeAllocation(t *testing.T) {
	input := newCustomCurrencyWithFiatOverageInput(t)
	chargeCurrencyHandler := &testHandler{
		allocations: creditrealization.CreateAllocationInputs{
			{Amount: alpacadecimal.NewFromInt(12)},
		},
	}
	input.ChargeCurrencyHandler = chargeCurrencyHandler

	// given: a two-stage reconciliation without a fiat-overage handler

	// when: reconciliation is requested
	_, err := ReconcileCustomCurrencyWithFiatOverage(t.Context(), input)

	// then: validation fails before charge-currency credits are allocated
	require.ErrorContains(t, err, "fiat overage handler is required")
	require.True(t, chargeCurrencyHandler.allocated.IsZero())
}

func newCustomCurrencyWithFiatOverageInput(t testing.TB) ReconcileCustomCurrencyWithFiatOverageInput {
	t.Helper()

	customCurrency, err := currencyx.NewCurrencyBuilder(currencyx.CurrencyTypeCustom).
		WithCode("TOKENS").
		WithName("Tokens").
		WithPrecision(2).
		Build()
	require.NoError(t, err)

	fiatCurrency, err := currencyx.NewFiatCurrency("USD")
	require.NoError(t, err)

	costBasisIntent := costbasis.NewIntent(costbasis.ManualIntent{
		FiatCurrency: fiatCurrency,
		Rate:         alpacadecimal.NewFromFloat(0.5),
	})
	resolvedCostBasis := costbasis.State{
		CostBasis:  alpacadecimal.NewFromFloat(0.5),
		ResolvedAt: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
	}

	return ReconcileCustomCurrencyWithFiatOverageInput{
		UnallocatedTotals: totals.Totals{
			Amount: alpacadecimal.NewFromInt(26),
			Total:  alpacadecimal.NewFromInt(26),
		},
		Currency: currencies.Currency{
			NamespacedID: models.NamespacedID{
				Namespace: "namespace",
				ID:        "currency-id",
			},
			Currency: customCurrency,
		},
		CostBasisIntent:   &costBasisIntent,
		ResolvedCostBasis: &resolvedCostBasis,
	}
}
