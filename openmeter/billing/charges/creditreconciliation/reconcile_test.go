package creditreconciliation

import (
	"context"
	"testing"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/creditrealization"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/ledgertransaction"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/timeutil"
)

func TestReconcile(t *testing.T) {
	currency, err := currencyx.NewFiatCurrency("USD")
	require.NoError(t, err)

	t.Run("requires a handler", func(t *testing.T) {
		// given: otherwise valid reconciliation input without a monetary-domain handler

		// when: reconciliation is requested
		_, err := Reconcile(t.Context(), ReconcileInput{
			TargetAmount: alpacadecimal.NewFromInt(5),
		})

		// then: reconciliation fails before attempting an economic effect
		require.ErrorContains(t, err, "credit reconciliation handler is required")
	})

	t.Run("requires the handler to provide its currency calculator", func(t *testing.T) {
		// given: a monetary-domain handler without a currency calculator
		handler := &testHandler{}

		// when: reconciliation is requested
		_, err := Reconcile(t.Context(), ReconcileInput{
			TargetAmount: alpacadecimal.NewFromInt(5),
			Handler:      handler,
		})

		// then: reconciliation fails before attempting an economic effect
		require.ErrorContains(t, err, "credit reconciliation handler currency calculator is required")
	})

	t.Run("allocates positive delta and persists realizations", func(t *testing.T) {
		// given: a monetary domain with two credits realized toward a target of five
		handler := &testHandler{
			currency: currency,
			current:  creditrealization.Realizations{newTestAllocation("allocation-1", 2)},
			allocations: creditrealization.CreateAllocationInputs{
				{Amount: alpacadecimal.NewFromInt(3)},
			},
		}

		// when: the domain is reconciled to the target
		result, err := Reconcile(t.Context(), ReconcileInput{
			TargetAmount: alpacadecimal.NewFromInt(5),
			Handler:      handler,
		})

		// then: only the positive delta is allocated and persisted
		require.NoError(t, err)
		require.Equal(t, float64(3), result.Delta.InexactFloat64())
		require.Equal(t, float64(3), handler.allocated.InexactFloat64())
		require.Len(t, handler.created, 1)
		require.Len(t, result.Realizations, 1)
	})

	t.Run("corrects negative delta and persists realizations", func(t *testing.T) {
		// given: a monetary domain with five credits realized toward a target of two
		handler := &testHandler{
			currency: currency,
			current:  creditrealization.Realizations{newTestAllocation("allocation-1", 5)},
			corrections: creditrealization.CreateCorrectionInputs{
				{
					Amount:                alpacadecimal.NewFromInt(-3),
					CorrectsRealizationID: "allocation-1",
					LedgerTransaction: ledgertransaction.GroupReference{
						TransactionGroupID: "correction-transaction-1",
					},
				},
			},
		}

		// when: the domain is reconciled to the lower target
		result, err := Reconcile(t.Context(), ReconcileInput{
			TargetAmount: alpacadecimal.NewFromInt(2),
			Handler:      handler,
		})

		// then: the negative delta is corrected against the current realizations
		require.NoError(t, err)
		require.Equal(t, float64(-3), result.Delta.InexactFloat64())
		require.Len(t, handler.corrected, 1)
		require.Equal(t, float64(-3), handler.corrected[0].Amount.InexactFloat64())
		require.Len(t, handler.created, 1)
		require.Len(t, result.Realizations, 1)
	})

	t.Run("zero delta is a no-op", func(t *testing.T) {
		// given: a monetary domain already reconciled to its target
		handler := &testHandler{
			currency: currency,
			current:  creditrealization.Realizations{newTestAllocation("allocation-1", 2)},
		}

		// when: the domain is reconciled
		result, err := Reconcile(t.Context(), ReconcileInput{
			TargetAmount: alpacadecimal.NewFromInt(2),
			Handler:      handler,
		})

		// then: no handler effect or persistence is requested
		require.NoError(t, err)
		require.True(t, result.Delta.IsZero())
		require.True(t, handler.allocated.IsZero())
		require.Empty(t, handler.corrected)
		require.Empty(t, handler.created)
	})

	t.Run("exact allocation rejects a partial result", func(t *testing.T) {
		// given: a handler that can allocate only part of the requested amount
		handler := &testHandler{
			currency: currency,
			allocations: creditrealization.CreateAllocationInputs{
				{Amount: alpacadecimal.NewFromInt(2)},
			},
		}

		// when: exact reconciliation requests five credits
		_, err := Reconcile(t.Context(), ReconcileInput{
			TargetAmount:    alpacadecimal.NewFromInt(5),
			ExactAllocation: true,
			Handler:         handler,
		})

		// then: the mismatched allocation is rejected before persistence
		require.Error(t, err)
		require.Empty(t, handler.created)
	})

	t.Run("over-allocation is always rejected", func(t *testing.T) {
		// given: a handler returning more than the requested allocation
		handler := &testHandler{
			currency: currency,
			allocations: creditrealization.CreateAllocationInputs{
				{Amount: alpacadecimal.NewFromInt(6)},
			},
		}

		// when: reconciliation requests five credits
		_, err := Reconcile(t.Context(), ReconcileInput{
			TargetAmount: alpacadecimal.NewFromInt(5),
			Handler:      handler,
		})

		// then: over-allocation is rejected before persistence
		require.Error(t, err)
		require.Empty(t, handler.created)
	})
}

func TestCorrectAll(t *testing.T) {
	currency, err := currencyx.NewFiatCurrency("USD")
	require.NoError(t, err)

	// given: a monetary domain with five active credits
	handler := &testHandler{
		currency: currency,
		current:  creditrealization.Realizations{newTestAllocation("allocation-1", 5)},
		corrections: creditrealization.CreateCorrectionInputs{
			{
				Amount:                alpacadecimal.NewFromInt(-5),
				CorrectsRealizationID: "allocation-1",
				LedgerTransaction: ledgertransaction.GroupReference{
					TransactionGroupID: "correction-transaction-1",
				},
			},
		},
	}

	// when: all credits are corrected
	result, err := CorrectAll(t.Context(), CorrectAllInput{
		Handler: handler,
	})

	// then: correction delegates to reconciliation with a zero target
	require.NoError(t, err)
	require.Equal(t, float64(-5), result.Delta.InexactFloat64())
	require.Len(t, handler.corrected, 1)
	require.Equal(t, float64(-5), handler.corrected[0].Amount.InexactFloat64())
	require.Len(t, result.Realizations, 1)
}

type testHandler struct {
	currency    currencyx.Currency
	current     creditrealization.Realizations
	allocations creditrealization.CreateAllocationInputs
	corrections creditrealization.CreateCorrectionInputs
	allocated   alpacadecimal.Decimal
	corrected   creditrealization.CorrectionRequest
	created     creditrealization.CreateInputs
}

func (h *testHandler) Validate() error {
	return nil
}

func (h *testHandler) CurrencyCalculator() currencyx.Currency {
	return h.currency
}

func (h *testHandler) Realizations() creditrealization.Realizations {
	return h.current
}

func (h *testHandler) Allocate(_ context.Context, amount alpacadecimal.Decimal) (creditrealization.CreateAllocationInputs, error) {
	h.allocated = amount

	return h.allocations, nil
}

func (h *testHandler) Correct(
	_ context.Context,
	request creditrealization.CorrectionRequest,
) (creditrealization.CreateCorrectionInputs, error) {
	h.corrected = request

	return h.corrections, nil
}

func (h *testHandler) Create(_ context.Context, inputs creditrealization.CreateInputs) (creditrealization.Realizations, error) {
	h.created = inputs

	realizations := make(creditrealization.Realizations, 0, len(inputs))
	for _, input := range inputs {
		realizations = append(realizations, creditrealization.Realization{CreateInput: input})
	}

	return realizations, nil
}

func newTestAllocation(id string, amount int64) creditrealization.Realization {
	servicePeriod := timeutil.ClosedPeriod{
		From: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		To:   time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC),
	}

	return creditrealization.Realization{
		CreateInput: creditrealization.CreateInput{
			ID:            id,
			ServicePeriod: servicePeriod,
			LedgerTransaction: ledgertransaction.GroupReference{
				TransactionGroupID: id + "-transaction",
			},
			Amount: alpacadecimal.NewFromInt(amount),
			Type:   creditrealization.TypeAllocation,
		},
	}
}
