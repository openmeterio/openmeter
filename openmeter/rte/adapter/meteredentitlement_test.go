package adapter_test

import (
	"context"
	"testing"
	"time"

	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/rte"
	"github.com/openmeterio/openmeter/openmeter/rte/adapter"
	"github.com/openmeterio/openmeter/openmeter/rte/meteredentitlement"
	"github.com/openmeterio/openmeter/openmeter/testutils"
	"github.com/openmeterio/openmeter/pkg/models"
)

func TestMeteredEntitlementBalanceStore(t *testing.T) {
	// Given independent stores backed by the same migrated PostgreSQL database.
	database := testutils.InitPostgresDB(t, testutils.PostgresDBStateAtlasMigrated)
	t.Cleanup(func() {
		require.NoError(t, database.EntDriver.Close())
		require.NoError(t, database.PGDriver.Close())
	})
	logger := testutils.NewDiscardLogger(t)
	newStore := func() rte.Store[meteredentitlement.Key, meteredentitlement.Balance] {
		store, err := adapter.NewMeteredEntitlementBalanceStore(adapter.Config{Client: database.EntDriver.Client(), Logger: logger})
		require.NoError(t, err)

		return store
	}
	repo := newStore()
	key := meteredentitlement.Key{EntitlementID: models.NamespacedID{Namespace: "rte-metered-store", ID: ulid.Make().String()}}
	startedAt := time.Date(2026, time.January, 1, 1, 0, 15, 0, time.UTC)

	// When a scope has never been stored, reading it cannot create placeholder state.
	_, err := repo.Get(t.Context(), key)
	require.True(t, models.IsGenericNotFoundError(err))
	rows, err := database.EntDriver.Client().RTEMeteredEntitlementBalance.Query().Count(t.Context())
	require.NoError(t, err)
	require.Zero(t, rows)

	// Then the complete calculation result survives a round-trip through a fresh store.
	expected := meteredentitlement.Balance{
		EntitlementID: key.EntitlementID.ID,
		Balance:       75, UsageInPeriod: 25, TotalAvailableGrantAmount: 100,
		GrantBalances: map[string]float64{"grant": 75}, StartOfPeriod: startedAt.Add(-time.Hour),
	}
	calculated, err := repo.Store(t.Context(), rte.StoreInput[meteredentitlement.Key, meteredentitlement.Balance]{
		Key: key, State: rte.State[meteredentitlement.Balance]{CalculatedAt: startedAt, Value: expected},
	})
	require.NoError(t, err)
	stored, err := newStore().Get(t.Context(), key)
	require.NoError(t, err)
	require.True(t, startedAt.Equal(stored.CalculatedAt))
	require.Equal(t, expected, stored.Value)

	// Mutating the caller's result cannot mutate persisted JSON state.
	expected.GrantBalances["grant"] = 0
	stored, err = repo.Get(t.Context(), key)
	require.NoError(t, err)
	require.Equal(t, calculated, stored)

	t.Run("older and equal results retain stored success", func(t *testing.T) {
		// Given a newer success and a delayed result from another service replica.
		second, err := adapter.NewMeteredEntitlementBalanceStore(adapter.Config{Client: database.EntDriver.Client(), Logger: logger})
		require.NoError(t, err)
		fresh := calculated
		fresh.CalculatedAt = startedAt.Add(time.Second)
		fresh.Value.Overage = 7
		fresh.Value.Balance = 0
		fresh, err = second.Store(t.Context(), rte.StoreInput[meteredentitlement.Key, meteredentitlement.Balance]{Key: key, State: fresh})
		require.NoError(t, err)

		// When the delayed result arrives, the whole newer result survives.
		older, err := repo.Store(t.Context(), rte.StoreInput[meteredentitlement.Key, meteredentitlement.Balance]{Key: key, State: calculated})
		require.NoError(t, err)
		require.Equal(t, fresh, older)

		calculated.CalculatedAt = fresh.CalculatedAt
		equal, err := repo.Store(t.Context(), rte.StoreInput[meteredentitlement.Key, meteredentitlement.Balance]{Key: key, State: calculated})
		require.NoError(t, err)
		require.Equal(t, fresh, equal)

		// Then reads also return the retained state.
		stored, err := repo.Get(t.Context(), key)
		require.NoError(t, err)
		require.Equal(t, fresh, stored)
	})

	t.Run("simultaneous successful calculations", func(t *testing.T) {
		// Given a previously uncalculated scope and distinct calculation starts.
		freshKey := key
		freshKey.EntitlementID.ID = ulid.Make().String()
		const attempts = 12
		failures := make(chan error, attempts)
		start := make(chan struct{})
		ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
		defer cancel()

		// When independent stores write those results concurrently.
		for i := range attempts {
			store := newStore()
			go func() {
				<-start
				_, err := store.Store(ctx, rte.StoreInput[meteredentitlement.Key, meteredentitlement.Balance]{
					Key: freshKey,
					State: rte.State[meteredentitlement.Balance]{
						CalculatedAt: startedAt.Add(time.Duration(i) * time.Second),
						Value:        meteredentitlement.Balance{EntitlementID: freshKey.EntitlementID.ID, Balance: float64(i)},
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
		result, err := repo.Get(ctx, freshKey)
		require.NoError(t, err)
		require.True(t, result.CalculatedAt.Equal(startedAt.Add((attempts-1)*time.Second)))
		require.Equal(t, float64(attempts-1), result.Value.Balance)
	})

	t.Run("namespace and entitlement isolation", func(t *testing.T) {
		// Given identical IDs in different namespaces and different IDs in one namespace.
		keys := []meteredentitlement.Key{
			{EntitlementID: models.NamespacedID{Namespace: "other", ID: key.EntitlementID.ID}},
			{EntitlementID: models.NamespacedID{Namespace: key.EntitlementID.Namespace, ID: ulid.Make().String()}},
		}
		for i, other := range keys {
			_, err := repo.Get(t.Context(), other)
			require.True(t, models.IsGenericNotFoundError(err))

			// When each scope receives a distinct result.
			balance := rte.State[meteredentitlement.Balance]{
				CalculatedAt: startedAt,
				Value:        meteredentitlement.Balance{EntitlementID: other.EntitlementID.ID, Balance: float64(i + 1)},
			}
			_, err = repo.Store(t.Context(), rte.StoreInput[meteredentitlement.Key, meteredentitlement.Balance]{Key: other, State: balance})
			require.NoError(t, err)
		}

		// Then both results are independently addressable.
		for i, other := range keys {
			result, err := repo.Get(t.Context(), other)
			require.NoError(t, err)
			require.Equal(t, float64(i+1), result.Value.Balance)
		}
	})
}
