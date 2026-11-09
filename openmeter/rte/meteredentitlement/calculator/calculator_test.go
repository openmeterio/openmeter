package calculator_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	meteredentitlement "github.com/openmeterio/openmeter/openmeter/entitlement/metered"
	"github.com/openmeterio/openmeter/openmeter/rte"
	rtemetered "github.com/openmeterio/openmeter/openmeter/rte/meteredentitlement"
	"github.com/openmeterio/openmeter/openmeter/rte/meteredentitlement/calculator"
	"github.com/openmeterio/openmeter/openmeter/testutils"
	"github.com/openmeterio/openmeter/pkg/models"
)

type sourceFunc func(context.Context, models.NamespacedID, time.Time) (*meteredentitlement.EntitlementBalance, error)

func (f sourceFunc) GetEntitlementBalance(ctx context.Context, id models.NamespacedID, at time.Time) (*meteredentitlement.EntitlementBalance, error) {
	return f(ctx, id, at)
}

func TestCalculateUsesExistingEntitlementBalance(t *testing.T) {
	// Given an entitlement and an invocation time within a minute.
	ctx := t.Context()
	key := rtemetered.Key{EntitlementID: models.NamespacedID{Namespace: "rte-metered", ID: "entitlement"}}
	at := time.Date(2026, time.January, 1, 0, 0, 15, 0, time.UTC)
	expected := meteredentitlement.EntitlementBalance{
		EntitlementID: key.EntitlementID.ID,
		Balance:       75, UsageInPeriod: 25, Overage: 0, TotalAvailableGrantAmount: 100,
		GrantBalances: map[string]float64{"grant": 75}, StartOfPeriod: at.Add(-time.Hour),
	}
	calc, err := calculator.New(calculator.Config{
		Logger: testutils.NewDiscardLogger(t),
		Source: sourceFunc(func(gotCtx context.Context, id models.NamespacedID, gotAt time.Time) (*meteredentitlement.EntitlementBalance, error) {
			require.Equal(t, ctx, gotCtx)
			require.Equal(t, key.EntitlementID, id)
			require.Equal(t, at, gotAt, "minute rounding belongs to the existing calculator")

			return &expected, nil
		}),
	})
	require.NoError(t, err)

	// When RTE requests the calculation, it forwards identity and time unchanged.
	result, err := calc.Calculate(ctx, rte.CalculateInput[rtemetered.Key]{Key: key, At: at})
	require.NoError(t, err)

	// Then the complete existing balance result is preserved.
	require.Equal(t, expected, result)
}
