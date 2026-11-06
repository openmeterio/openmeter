package calculator_test

import (
	"context"
	"testing"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/currencies"
	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/openmeter/ledger/customerbalance"
	"github.com/openmeterio/openmeter/openmeter/rte"
	"github.com/openmeterio/openmeter/openmeter/rte/creditbalance"
	"github.com/openmeterio/openmeter/openmeter/rte/creditbalance/calculator"
	"github.com/openmeterio/openmeter/openmeter/testutils"
)

type sourceFunc func(context.Context, customerbalance.GetBalanceServiceInput) (customerbalance.Balance, error)

func (f sourceFunc) GetBalance(ctx context.Context, input customerbalance.GetBalanceServiceInput) (customerbalance.Balance, error) {
	return f(ctx, input)
}

type sourceBalance struct{ booked, spendable, live, pending int64 }

func (b sourceBalance) Booked() alpacadecimal.Decimal {
	return alpacadecimal.NewFromInt(b.booked)
}

func (b sourceBalance) Spendable() alpacadecimal.Decimal {
	return alpacadecimal.NewFromInt(b.spendable)
}

func (b sourceBalance) Live() alpacadecimal.Decimal {
	return alpacadecimal.NewFromInt(b.live)
}

func (b sourceBalance) Pending() alpacadecimal.Decimal {
	return alpacadecimal.NewFromInt(b.pending)
}

func TestCalculateUsesCurrentCustomerBalance(t *testing.T) {
	// Given a scoped credit calculation and the existing customer balance source.
	ctx := t.Context()
	key := creditbalance.Key{
		CustomerID:    customer.CustomerID{Namespace: "rte-credit", ID: "customer"},
		Currency:      currencies.NewCurrencyReference("USD"),
		FeatureFilter: customerbalance.NewFeatureFilter([]string{"feature"}),
	}
	calc, err := calculator.New(calculator.Config{
		Logger: testutils.NewDiscardLogger(t),
		Source: sourceFunc(func(gotCtx context.Context, input customerbalance.GetBalanceServiceInput) (customerbalance.Balance, error) {
			require.Equal(t, ctx, gotCtx)
			require.Equal(t, key.CustomerID, input.CustomerID)
			require.Equal(t, key.Currency, input.Currency)
			require.Equal(t, key.FeatureFilter, input.FeatureFilter)
			require.Nil(t, input.BalanceQuery.AsOf, "historical queries omit live usage")

			return sourceBalance{100, 90, 80, 10}, nil
		}),
	})
	require.NoError(t, err)

	// When RTE supplies its invocation time, it must not become a historical AsOf.
	result, err := calc.Calculate(ctx, rte.CalculateInput[creditbalance.Key]{
		Key: key, At: time.Date(2026, time.January, 1, 0, 0, 15, 0, time.UTC),
	})
	require.NoError(t, err)

	// Then the bridge retains all four source balance values.
	require.Equal(t, float64(100), result.Booked.InexactFloat64())
	require.Equal(t, float64(90), result.Spendable.InexactFloat64())
	require.Equal(t, float64(80), result.Live.InexactFloat64())
	require.Equal(t, float64(10), result.Pending.InexactFloat64())
}
