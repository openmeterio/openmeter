package service_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/oklog/ulid/v2"
	"github.com/samber/lo"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/currencies"
	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/openmeter/ledger/customerbalance"
	"github.com/openmeterio/openmeter/openmeter/rte"
	"github.com/openmeterio/openmeter/openmeter/rte/adapter"
	"github.com/openmeterio/openmeter/openmeter/rte/service"
	"github.com/openmeterio/openmeter/openmeter/testutils"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/models"
)

type calculatorFunc func(context.Context, customerbalance.GetBalanceServiceInput) (customerbalance.Balance, error)

func (f calculatorFunc) GetBalance(ctx context.Context, input customerbalance.GetBalanceServiceInput) (customerbalance.Balance, error) {
	return f(ctx, input)
}

type balanceValues struct{ booked, spendable, live, pending int64 }

func (b balanceValues) Booked() alpacadecimal.Decimal {
	return alpacadecimal.NewFromInt(b.booked)
}

func (b balanceValues) Spendable() alpacadecimal.Decimal {
	return alpacadecimal.NewFromInt(b.spendable)
}

func (b balanceValues) Live() alpacadecimal.Decimal {
	return alpacadecimal.NewFromInt(b.live)
}

func (b balanceValues) Pending() alpacadecimal.Decimal {
	return alpacadecimal.NewFromInt(b.pending)
}

func TestCreditBalanceState(t *testing.T) {
	// Given independent service instances backed by the same migrated PostgreSQL database.
	database := testutils.InitPostgresDB(t, testutils.PostgresDBStateAtlasMigrated)
	t.Cleanup(func() {
		require.NoError(t, database.EntDriver.Close())
		require.NoError(t, database.PGDriver.Close())
	})
	logger := testutils.NewDiscardLogger(t)
	newAdapter := func() rte.Adapter {
		repo, err := adapter.New(adapter.Config{Client: database.EntDriver.Client(), Logger: logger})
		require.NoError(t, err)

		return repo
	}
	newService := func(calculator calculatorFunc) rte.Service {
		svc, err := service.New(service.Config{Adapter: newAdapter(), Calculator: calculator, Logger: logger})
		require.NoError(t, err)

		return svc
	}

	key := rte.CreditBalanceKey{CustomerID: customer.CustomerID{Namespace: "rte-test", ID: ulid.Make().String()}, Currency: currencies.NewCurrencyReference("USD")}
	calls := 0
	source := balanceValues{100, 90, 80, 10}
	var sourceErr error
	svc := newService(func(_ context.Context, input customerbalance.GetBalanceServiceInput) (customerbalance.Balance, error) {
		calls++
		require.Equal(t, key.CustomerID, input.CustomerID)
		require.Equal(t, key.Currency, input.Currency)
		require.Equal(t, key.FeatureFilter, input.FeatureFilter)
		require.Nil(t, input.BalanceQuery.AsOf, "current calculation must include live usage")

		return source, sourceErr
	})

	// When reading before the first calculation, no source calculation runs.
	_, err := svc.GetCreditBalance(t.Context(), key)
	require.True(t, models.IsGenericNotFoundError(err))
	require.Zero(t, calls)
	sourceErr = errors.New("source unavailable")
	_, err = svc.CalculateCreditBalance(t.Context(), key)
	require.ErrorContains(t, err, "source unavailable")
	_, err = svc.GetCreditBalance(t.Context(), key)
	require.True(t, models.IsGenericNotFoundError(err), "a failed calculation is not a calculated zero")

	rows, err := database.EntDriver.Client().RTECreditBalance.Query().Count(t.Context())
	require.NoError(t, err)
	require.Zero(t, rows, "failed calculations must not create placeholder state")

	// Then an explicit successful calculation persists all four existing balance values.
	sourceErr = nil
	calculated, err := svc.CalculateCreditBalance(t.Context(), key)
	require.NoError(t, err)
	require.False(t, calculated.CalculatedAt.IsZero())
	require.Equal(t, float64(100), calculated.Booked.InexactFloat64())
	require.Equal(t, float64(90), calculated.Spendable.InexactFloat64())
	require.Equal(t, float64(80), calculated.Live.InexactFloat64())
	require.Equal(t, float64(10), calculated.Pending.InexactFloat64())
	source = balanceValues{500, 500, 500, 0}
	reader := newService(func(context.Context, customerbalance.GetBalanceServiceInput) (customerbalance.Balance, error) {
		t.Error("read refreshed source")

		return nil, errors.New("unexpected calculation")
	})
	stored, err := reader.GetCreditBalance(t.Context(), key)
	require.NoError(t, err)
	require.True(t, calculated.CalculatedAt.Equal(stored.CalculatedAt))
	require.Equal(t, float64(80), stored.Live.InexactFloat64())
	sourceErr = errors.New("refresh failed")
	_, err = svc.CalculateCreditBalance(t.Context(), key)
	require.Error(t, err)
	stored, err = reader.GetCreditBalance(t.Context(), key)
	require.NoError(t, err)
	require.True(t, calculated.CalculatedAt.Equal(stored.CalculatedAt))

	t.Run("timestamp is captured before calculation", func(t *testing.T) {
		// Given a calculation whose source work finishes at a later time.
		startedAt := time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC)
		clock.FreezeTime(startedAt)
		defer clock.UnFreeze()

		freshKey := key
		freshKey.CustomerID.ID = ulid.Make().String()
		engine := newService(func(context.Context, customerbalance.GetBalanceServiceInput) (customerbalance.Balance, error) {
			clock.FreezeTime(startedAt.Add(time.Hour))
			defer clock.UnFreeze()

			return balanceValues{100, 90, 80, 10}, nil
		})

		// When calculation completes, state is ordered by its start time.
		result, err := engine.CalculateCreditBalance(t.Context(), freshKey)
		require.NoError(t, err)
		require.True(t, result.CalculatedAt.Equal(startedAt))
	})

	t.Run("older and equal completions retain newer state", func(t *testing.T) {
		// Given explicit start times from two independent replicas. No clock
		// mutation is needed while the delayed completion is in flight.
		freshKey := key
		freshKey.CustomerID.ID = ulid.Make().String()
		startedAt := time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC)
		older := rte.StoreCreditBalanceInput{
			Key:     freshKey,
			Balance: rte.CreditBalance{CalculatedAt: startedAt, Live: alpacadecimal.NewFromInt(80)},
		}
		newer := rte.StoreCreditBalanceInput{
			Key:     freshKey,
			Balance: rte.CreditBalance{CalculatedAt: startedAt.Add(time.Second), Live: alpacadecimal.NewFromInt(180)},
		}
		first, second := newAdapter(), newAdapter()
		release := make(chan struct{})
		defer close(release)
		type outcome struct {
			balance rte.CreditBalance
			err     error
		}
		done := make(chan outcome, 1)
		ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
		defer cancel()

		go func() {
			select {
			case <-release:
				result, err := first.StoreCreditBalance(ctx, older)
				done <- outcome{result, err}
			case <-ctx.Done():
				done <- outcome{err: ctx.Err()}
			}
		}()

		// When the newer calculation is stored first, the older completion
		// returns that winner instead of replacing it.
		fresh, err := second.StoreCreditBalance(ctx, newer)
		require.NoError(t, err)
		release <- struct{}{}
		late := <-done
		require.NoError(t, late.err)
		require.True(t, late.balance.CalculatedAt.Equal(fresh.CalculatedAt))
		require.Equal(t, float64(180), late.balance.Live.InexactFloat64())

		// Equal timestamps also retain the existing success.
		older.Balance.CalculatedAt = newer.Balance.CalculatedAt
		equal, err := first.StoreCreditBalance(ctx, older)
		require.NoError(t, err)
		require.Equal(t, float64(180), equal.Live.InexactFloat64())

		stored, err := reader.GetCreditBalance(ctx, freshKey)
		require.NoError(t, err)
		require.True(t, stored.CalculatedAt.Equal(fresh.CalculatedAt))
		require.Equal(t, float64(180), stored.Live.InexactFloat64())
	})

	t.Run("simultaneous successful calculations", func(t *testing.T) {
		// Given a previously uncalculated scope and distinct calculation starts.
		freshKey := key
		freshKey.CustomerID.ID = ulid.Make().String()
		startedAt := time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC)
		const attempts = 12
		replicas := make([]rte.Adapter, attempts)
		for i := range replicas {
			replicas[i] = newAdapter()
		}

		failures := make(chan error, attempts)
		start := make(chan struct{})
		ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
		defer cancel()
		for i, replica := range replicas {
			go func() {
				<-start
				_, err := replica.StoreCreditBalance(ctx, rte.StoreCreditBalanceInput{
					Key: freshKey,
					Balance: rte.CreditBalance{
						CalculatedAt: startedAt.Add(time.Duration(i) * time.Second),
						Live:         alpacadecimal.NewFromInt(int64(i)),
					},
				})
				failures <- err
			}()
		}

		close(start)
		for range attempts {
			require.NoError(t, <-failures)
		}

		// Then the latest start wins regardless of concurrent arrival order.
		result, err := reader.GetCreditBalance(ctx, freshKey)
		require.NoError(t, err)
		require.True(t, result.CalculatedAt.Equal(startedAt.Add((attempts-1)*time.Second)))
		require.Equal(t, float64(attempts-1), result.Live.InexactFloat64())
	})

	t.Run("scope isolation", func(t *testing.T) {
		// Given keys differing in each supported dimension, including same-code custom currencies.
		keys := []rte.CreditBalanceKey{}
		other := key
		other.CustomerID.Namespace = "other"
		keys = append(keys, other)

		other = key
		other.CustomerID.ID = ulid.Make().String()
		keys = append(keys, other)

		other = key
		other.Currency = currencies.NewCurrencyReference("EUR")
		keys = append(keys, other)

		other = key
		other.Currency = currencies.CurrencyReference{Code: "POINTS", CustomCurrencyID: lo.ToPtr(ulid.Make().String())}
		keys = append(keys, other)

		other.Currency.CustomCurrencyID = lo.ToPtr(ulid.Make().String())
		keys = append(keys, other)

		other = key
		other.FeatureFilter = customerbalance.NewUnrestrictedFeatureFilter()
		keys = append(keys, other)

		other = key
		other.FeatureFilter = customerbalance.NewFeatureFilter([]string{"feature-a"})
		keys = append(keys, other)

		other.FeatureFilter = customerbalance.NewFeatureFilter([]string{"feature-b"})
		keys = append(keys, other)

		for i, k := range keys {
			_, err := reader.GetCreditBalance(t.Context(), k)
			require.True(t, models.IsGenericNotFoundError(err))
			writer := newService(func(context.Context, customerbalance.GetBalanceServiceInput) (customerbalance.Balance, error) {
				return balanceValues{int64(i + 1), 0, 0, 0}, nil
			})
			_, err = writer.CalculateCreditBalance(t.Context(), k)
			require.NoError(t, err)
		}

		// Then every stored result remains independently addressable.
		for i, k := range keys {
			result, err := reader.GetCreditBalance(t.Context(), k)
			require.NoError(t, err)
			require.Equal(t, float64(i+1), result.Booked.InexactFloat64())
		}
	})
}
