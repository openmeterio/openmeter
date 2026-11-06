package adapter_test

import (
	"context"
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
	"github.com/openmeterio/openmeter/openmeter/rte/creditbalance"
	"github.com/openmeterio/openmeter/openmeter/testutils"
	"github.com/openmeterio/openmeter/pkg/models"
)

func TestCreditBalanceStore(t *testing.T) {
	// Given independent stores backed by the same migrated PostgreSQL database.
	database := testutils.InitPostgresDB(t, testutils.PostgresDBStateAtlasMigrated)
	t.Cleanup(func() {
		require.NoError(t, database.EntDriver.Close())
		require.NoError(t, database.PGDriver.Close())
	})
	logger := testutils.NewDiscardLogger(t)
	newStore := func() rte.Store[creditbalance.Key, creditbalance.Balance] {
		store, err := adapter.NewCreditBalanceStore(adapter.Config{Client: database.EntDriver.Client(), Logger: logger})
		require.NoError(t, err)

		return store
	}
	reader := newStore()
	key := creditbalance.Key{CustomerID: customer.CustomerID{Namespace: "rte-credit-store", ID: ulid.Make().String()}, Currency: currencies.NewCurrencyReference("USD")}
	startedAt := time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC)

	// When a scope has never been stored, reading it cannot create placeholder state.
	_, err := reader.Get(t.Context(), key)
	require.True(t, models.IsGenericNotFoundError(err))
	rows, err := database.EntDriver.Client().RTECreditBalance.Query().Count(t.Context())
	require.NoError(t, err)
	require.Zero(t, rows)

	// Then all four decimal values round-trip exactly through a fresh store.
	expected := creditbalance.Balance{
		Booked:    alpacadecimal.RequireFromString("100.1234567890123456789"),
		Spendable: alpacadecimal.RequireFromString("90.1234567890123456789"),
		Live:      alpacadecimal.RequireFromString("80.1234567890123456789"),
		Pending:   alpacadecimal.RequireFromString("10.1234567890123456789"),
	}
	_, err = reader.Store(t.Context(), rte.StoreInput[creditbalance.Key, creditbalance.Balance]{
		Key: key, State: rte.State[creditbalance.Balance]{CalculatedAt: startedAt, Value: expected},
	})
	require.NoError(t, err)
	stored, err := newStore().Get(t.Context(), key)
	require.NoError(t, err)
	require.True(t, startedAt.Equal(stored.CalculatedAt))
	require.True(t, expected.Booked.Equal(stored.Value.Booked))
	require.True(t, expected.Spendable.Equal(stored.Value.Spendable))
	require.True(t, expected.Live.Equal(stored.Value.Live))
	require.True(t, expected.Pending.Equal(stored.Value.Pending))

	t.Run("older and equal completions retain newer state", func(t *testing.T) {
		// Given explicit start times from two independent replicas. No clock
		// mutation is needed while the delayed completion is in flight.
		freshKey := key
		freshKey.CustomerID.ID = ulid.Make().String()
		startedAt := time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC)
		older := rte.StoreInput[creditbalance.Key, creditbalance.Balance]{
			Key:   freshKey,
			State: rte.State[creditbalance.Balance]{CalculatedAt: startedAt, Value: creditbalance.Balance{Live: alpacadecimal.NewFromInt(80)}},
		}
		newer := rte.StoreInput[creditbalance.Key, creditbalance.Balance]{
			Key:   freshKey,
			State: rte.State[creditbalance.Balance]{CalculatedAt: startedAt.Add(time.Second), Value: creditbalance.Balance{Live: alpacadecimal.NewFromInt(180)}},
		}
		first, second := newStore(), newStore()
		release := make(chan struct{})
		defer close(release)
		type outcome struct {
			balance rte.State[creditbalance.Balance]
			err     error
		}
		done := make(chan outcome, 1)
		ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
		defer cancel()

		go func() {
			select {
			case <-release:
				result, err := first.Store(ctx, older)
				done <- outcome{result, err}
			case <-ctx.Done():
				done <- outcome{err: ctx.Err()}
			}
		}()

		// When the newer calculation is stored first, the older completion
		// returns that winner instead of replacing it.
		fresh, err := second.Store(ctx, newer)
		require.NoError(t, err)
		release <- struct{}{}
		late := <-done
		require.NoError(t, late.err)
		require.True(t, late.balance.CalculatedAt.Equal(fresh.CalculatedAt))
		require.Equal(t, float64(180), late.balance.Value.Live.InexactFloat64())

		// Equal timestamps also retain the existing success.
		older.State.CalculatedAt = newer.State.CalculatedAt
		equal, err := first.Store(ctx, older)
		require.NoError(t, err)
		require.Equal(t, float64(180), equal.Value.Live.InexactFloat64())

		stored, err := reader.Get(ctx, freshKey)
		require.NoError(t, err)
		require.True(t, stored.CalculatedAt.Equal(fresh.CalculatedAt))
		require.Equal(t, float64(180), stored.Value.Live.InexactFloat64())
	})

	t.Run("simultaneous successful calculations", func(t *testing.T) {
		// Given a previously uncalculated scope and distinct calculation starts.
		freshKey := key
		freshKey.CustomerID.ID = ulid.Make().String()
		startedAt := time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC)
		const attempts = 12
		replicas := make([]rte.Store[creditbalance.Key, creditbalance.Balance], attempts)
		for i := range replicas {
			replicas[i] = newStore()
		}

		failures := make(chan error, attempts)
		start := make(chan struct{})
		ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
		defer cancel()
		for i, replica := range replicas {
			go func() {
				<-start
				_, err := replica.Store(ctx, rte.StoreInput[creditbalance.Key, creditbalance.Balance]{
					Key: freshKey,
					State: rte.State[creditbalance.Balance]{
						CalculatedAt: startedAt.Add(time.Duration(i) * time.Second),
						Value: creditbalance.Balance{
							Live: alpacadecimal.NewFromInt(int64(i)),
						},
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
		result, err := reader.Get(ctx, freshKey)
		require.NoError(t, err)
		require.True(t, result.CalculatedAt.Equal(startedAt.Add((attempts-1)*time.Second)))
		require.Equal(t, float64(attempts-1), result.Value.Live.InexactFloat64())
	})

	t.Run("scope isolation", func(t *testing.T) {
		// Given keys differing in each supported dimension, including same-code custom currencies.
		keys := []creditbalance.Key{}
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
			_, err := reader.Get(t.Context(), k)
			require.True(t, models.IsGenericNotFoundError(err))
			_, err = reader.Store(t.Context(), rte.StoreInput[creditbalance.Key, creditbalance.Balance]{
				Key: k,
				State: rte.State[creditbalance.Balance]{
					CalculatedAt: startedAt,
					Value:        creditbalance.Balance{Booked: alpacadecimal.NewFromInt(int64(i + 1))},
				},
			})
			require.NoError(t, err)
		}

		// Then every stored result remains independently addressable.
		for i, k := range keys {
			result, err := reader.Get(t.Context(), k)
			require.NoError(t, err)
			require.Equal(t, float64(i+1), result.Value.Booked.InexactFloat64())
		}
	})
}
