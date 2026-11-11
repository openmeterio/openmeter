package adapter_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/currencies"
	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/openmeter/ent/db"
	demanddb "github.com/openmeterio/openmeter/openmeter/ent/db/rtedemand"
	"github.com/openmeterio/openmeter/openmeter/rte"
	"github.com/openmeterio/openmeter/openmeter/rte/adapter"
	"github.com/openmeterio/openmeter/openmeter/rte/creditbalance"
	"github.com/openmeterio/openmeter/openmeter/testutils"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/framework/entutils"
	"github.com/openmeterio/openmeter/pkg/framework/transaction"
	"github.com/openmeterio/openmeter/pkg/models"
)

type demandFixture struct {
	repository rte.DemandRepository
	client     *db.Client
	store      rte.Store[creditbalance.Key, creditbalance.Balance]
	key        creditbalance.Key
	target     rte.CalculationTarget
	input      rte.ProcessDemandInput
}

func newDemandFixture(t *testing.T) demandFixture {
	t.Helper()
	database := testutils.InitPostgresDB(t, testutils.PostgresDBStateAtlasMigrated)
	t.Cleanup(func() { require.NoError(t, database.EntDriver.Close()); require.NoError(t, database.PGDriver.Close()) })
	config := adapter.Config{Client: database.EntDriver.Client(), Logger: testutils.NewDiscardLogger(t)}
	repository, err := adapter.NewDemandRepository(config)
	require.NoError(t, err)
	store, err := adapter.NewCreditBalanceStore(config)
	require.NoError(t, err)
	key := creditbalance.Key{CustomerID: customer.CustomerID{Namespace: "rte-demand", ID: ulid.Make().String()}, Currency: currencies.NewCurrencyReference("USD")}
	event, err := rte.NewCalculationRequestedEvent(key)
	require.NoError(t, err)

	return demandFixture{repository: repository, client: config.Client, store: store, key: key, target: event.CalculationTarget, input: rte.ProcessDemandInput{CalculationTimeout: time.Second, RetryMinDelay: time.Minute, RetryMaxDelay: time.Hour}}
}

func TestDemandInvalidationCoalescesDistinctAndDuplicateFacts(t *testing.T) {
	// Given one active tracked key, its clean state and an unrelated inactive key.
	f := newDemandFixture(t)
	require.NoError(t, f.repository.Activate(t.Context(), rte.ActivateDemandInput{Target: f.target}))
	result, err := f.repository.Process(t.Context(), f.input, func(context.Context, rte.CalculationTarget) error { return nil })
	require.NoError(t, err)
	require.True(t, result.Processed)
	before, err := f.client.RTEDemand.Query().Only(t.Context())
	require.NoError(t, err)
	input := rte.InvalidateInput{Source: "ledger", EventID: "newer-fact", Selectors: []rte.DemandSelector{{Namespace: f.key.CustomerID.Namespace, Kind: f.target.Kind, OwnerIDs: []string{f.key.CustomerID.ID}}, {Namespace: f.key.CustomerID.Namespace, Kind: f.target.Kind, Targets: []rte.CalculationTarget{f.target}}}}

	// When a newer fact arrives twice, then an independent older fact arrives.
	marked, err := f.repository.Invalidate(t.Context(), input)
	require.NoError(t, err)
	require.Equal(t, 1, marked.Matched)
	require.False(t, marked.Duplicate)
	duplicate, err := f.repository.Invalidate(t.Context(), input)
	require.NoError(t, err)
	require.True(t, duplicate.Duplicate)
	require.Zero(t, duplicate.Matched)
	input.EventID = "older-fact"
	marked, err = f.repository.Invalidate(t.Context(), input)
	require.NoError(t, err)
	require.Equal(t, 1, marked.Matched)
	row, err := f.client.RTEDemand.Query().Only(t.Context())
	require.NoError(t, err)
	require.Equal(t, before.DirtyVersion+2, row.DirtyVersion)

	// Then one calculation consumes both facts, and deactivation stops scheduling.
	result, err = f.repository.Process(t.Context(), f.input, func(context.Context, rte.CalculationTarget) error { return nil })
	require.NoError(t, err)
	require.True(t, result.Processed)
	result, err = f.repository.Process(t.Context(), f.input, func(context.Context, rte.CalculationTarget) error { t.Fatal("clean key recalculated"); return nil })
	require.NoError(t, err)
	require.False(t, result.Processed)
	require.NoError(t, f.repository.Deactivate(t.Context(), f.target))
	input.EventID = "inactive-fact"
	marked, err = f.repository.Invalidate(t.Context(), input)
	require.NoError(t, err)
	require.Zero(t, marked.Matched)
	require.NoError(t, f.repository.Activate(t.Context(), rte.ActivateDemandInput{Target: f.target}))
	result, err = f.repository.Process(t.Context(), f.input, func(context.Context, rte.CalculationTarget) error { return nil })
	require.NoError(t, err)
	require.True(t, result.Processed)
}

func TestDemandFailedAttemptRollsBackStateAndPersistsRetry(t *testing.T) {
	// Given a demanded balance whose calculator fails after writing its result.
	f := newDemandFixture(t)
	require.NoError(t, f.repository.Activate(t.Context(), rte.ActivateDemandInput{Target: f.target}))
	failure := errors.New("attempt failed")
	result, err := f.repository.Process(t.Context(), f.input, func(ctx context.Context, _ rte.CalculationTarget) error {
		_, err := f.store.Store(ctx, rte.StoreInput[creditbalance.Key, creditbalance.Balance]{Key: f.key, State: rte.State[creditbalance.Balance]{CalculatedAt: clock.Now(), Value: creditbalance.Balance{Live: alpacadecimal.NewFromInt(10)}}})
		if err != nil {
			return err
		}

		return failure
	})

	// When the failed attempt commits only retry bookkeeping.
	require.NoError(t, err)
	require.ErrorIs(t, result.CalculationError, failure)
	_, err = f.store.Get(t.Context(), f.key)
	require.True(t, models.IsGenericNotFoundError(err))
	row, err := f.client.RTEDemand.Query().Only(t.Context())
	require.NoError(t, err)
	require.Equal(t, 1, row.Attempts)
	require.Greater(t, row.DirtyVersion, row.CompletedVersion)
	require.True(t, row.NextAttemptAt.After(clock.Now()))
	result, err = f.repository.Process(t.Context(), f.input, func(context.Context, rte.CalculationTarget) error { t.Fatal("retry backoff ignored"); return nil })
	require.NoError(t, err)
	require.False(t, result.Processed)

	// Then an immediate pull bypasses backoff and successful work clears failure.
	input := f.input
	input.Target = &f.target
	result, err = f.repository.Process(t.Context(), input, func(context.Context, rte.CalculationTarget) error { return nil })
	require.NoError(t, err)
	require.True(t, result.Processed)
	require.NoError(t, result.CalculationError)
	row, err = f.client.RTEDemand.Query().Only(t.Context())
	require.NoError(t, err)
	require.Equal(t, row.DirtyVersion, row.CompletedVersion)
	require.Zero(t, row.Attempts)
}

func TestDemandCompletionRollsBackWithOuterTransaction(t *testing.T) {
	// Given committed dirty work and an attempt whose transaction never commits.
	f := newDemandFixture(t)
	require.NoError(t, f.repository.Activate(t.Context(), rte.ActivateDemandInput{Target: f.target}))
	abort := errors.New("simulate interrupted transaction")
	err := transaction.RunWithNoValue(t.Context(), f.repository.(transaction.Creator), func(ctx context.Context) error {
		result, err := f.repository.Process(ctx, f.input, func(ctx context.Context, _ rte.CalculationTarget) error {
			_, err := f.store.Store(ctx, rte.StoreInput[creditbalance.Key, creditbalance.Balance]{Key: f.key, State: rte.State[creditbalance.Balance]{CalculatedAt: clock.Now()}})
			return err
		})
		require.NoError(t, err)
		require.True(t, result.Processed)

		return abort
	})

	// When its outer transaction rolls back, both result and completion disappear.
	require.ErrorIs(t, err, abort)
	_, err = f.store.Get(t.Context(), f.key)
	require.True(t, models.IsGenericNotFoundError(err))
	row, err := f.client.RTEDemand.Query().Only(t.Context())
	require.NoError(t, err)
	require.Greater(t, row.DirtyVersion, row.CompletedVersion)

	// Then a fresh worker can recover the same work without a lease timeout.
	result, err := f.repository.Process(t.Context(), f.input, func(context.Context, rte.CalculationTarget) error { return nil })
	require.NoError(t, err)
	require.True(t, result.Processed)
}

func TestDemandConcurrentWorkersAndInvalidationDuringCalculation(t *testing.T) {
	// Given a worker holding a demanded key inside its calculation transaction.
	f := newDemandFixture(t)
	require.NoError(t, f.repository.Activate(t.Context(), rte.ActivateDemandInput{Target: f.target}))
	ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
	defer cancel()
	started, release, finished := make(chan struct{}), make(chan struct{}), make(chan error, 1)
	defer func() {
		select {
		case <-release:
		default:
			close(release)
		}
	}()
	go func() {
		_, err := f.repository.Process(ctx, f.input, func(context.Context, rte.CalculationTarget) error {
			close(started)
			select {
			case <-release:
				return nil
			case <-ctx.Done():
				return ctx.Err()
			}
		})
		finished <- err
	}()
	select {
	case <-started:
	case <-ctx.Done():
		t.Fatal(ctx.Err())
	}

	// When another worker polls, it skips the lock; a fact waits for completion.
	result, err := f.repository.Process(ctx, f.input, func(context.Context, rte.CalculationTarget) error {
		t.Error("duplicate concurrent calculation")
		return nil
	})
	require.NoError(t, err)
	require.False(t, result.Processed)
	invalidated := make(chan error, 1)
	go func() {
		_, err := f.repository.Invalidate(ctx, rte.InvalidateInput{Source: "ledger", EventID: "arrived-in-flight", Selectors: []rte.DemandSelector{{Namespace: f.key.CustomerID.Namespace, Kind: f.target.Kind, Targets: []rte.CalculationTarget{f.target}}}})
		invalidated <- err
	}()
	close(release)
	require.NoError(t, <-finished)
	require.NoError(t, <-invalidated)

	// Then the in-flight invalidation survives completion and is calculated next.
	row, err := f.client.RTEDemand.Query().Only(ctx)
	require.NoError(t, err)
	require.Greater(t, row.DirtyVersion, row.CompletedVersion)
	result, err = f.repository.Process(ctx, f.input, func(context.Context, rte.CalculationTarget) error { return nil })
	require.NoError(t, err)
	require.True(t, result.Processed)
}

func TestDemandPollProcessesOneOfSeveralKeys(t *testing.T) {
	// Given multiple dirty keys, a single poll must lock exactly one of them.
	f := newDemandFixture(t)
	require.NoError(t, f.repository.Activate(t.Context(), rte.ActivateDemandInput{Target: f.target}))
	other := f.key
	other.CustomerID.ID = ulid.Make().String()
	event, err := rte.NewCalculationRequestedEvent(other)
	require.NoError(t, err)
	require.NoError(t, f.repository.Activate(t.Context(), rte.ActivateDemandInput{Target: event.CalculationTarget}))

	// When independent polls process the ready queue.
	seen := map[string]bool{}
	for range 2 {
		result, err := f.repository.Process(t.Context(), f.input, func(_ context.Context, target rte.CalculationTarget) error {
			identity, err := target.Identity()
			if err != nil {
				return err
			}

			require.False(t, seen[identity.KeyHash])
			seen[identity.KeyHash] = true

			return nil
		})
		require.NoError(t, err)
		require.True(t, result.Processed)
	}

	// Then both keys have completed once and no ready work remains.
	require.Len(t, seen, 2)
	result, err := f.repository.Process(t.Context(), f.input, func(context.Context, rte.CalculationTarget) error { t.Fatal("clean key processed"); return nil })
	require.NoError(t, err)
	require.False(t, result.Processed)
}

func TestDemandPullExpiryAgeAndPeriodicRefresh(t *testing.T) {
	// Given pull-only demand with a bounded lifetime and a completed version.
	f := newDemandFixture(t)
	now := time.Date(2026, time.November, 11, 12, 0, 0, 0, time.UTC)
	clock.FreezeTime(now)
	defer clock.UnFreeze()
	pull := rte.EnsureFreshInput{Target: f.target, PullTTL: time.Minute}
	require.NoError(t, f.repository.EnsurePull(t.Context(), pull))
	targeted := f.input
	targeted.Target = &f.target
	result, err := f.repository.Process(t.Context(), targeted, func(context.Context, rte.CalculationTarget) error { return nil })
	require.NoError(t, err)
	require.True(t, result.Processed)

	// When its age bound expires, a pull dirties it; expiry stops background work.
	clock.FreezeTime(now.Add(30 * time.Second))
	pull.MaxAge = 10 * time.Second
	require.NoError(t, f.repository.EnsurePull(t.Context(), pull))
	row, err := f.client.RTEDemand.Query().Only(t.Context())
	require.NoError(t, err)
	require.Greater(t, row.DirtyVersion, row.CompletedVersion)
	clock.FreezeTime(now.Add(2 * time.Minute))
	result, err = f.repository.Process(t.Context(), f.input, func(context.Context, rte.CalculationTarget) error { t.Fatal("expired pull processed"); return nil })
	require.NoError(t, err)
	require.False(t, result.Processed)

	// Then push activation resumes work and its timer cannot be postponed by registration.
	activate := rte.ActivateDemandInput{Target: f.target, RefreshInterval: time.Minute}
	require.NoError(t, f.repository.Activate(t.Context(), activate))
	result, err = f.repository.Process(t.Context(), f.input, func(context.Context, rte.CalculationTarget) error { return nil })
	require.NoError(t, err)
	require.True(t, result.Processed)
	clock.FreezeTime(now.Add(2*time.Minute + 30*time.Second))
	require.NoError(t, f.repository.Activate(t.Context(), activate))
	clock.FreezeTime(now.Add(3 * time.Minute))
	result, err = f.repository.Process(t.Context(), f.input, func(context.Context, rte.CalculationTarget) error { return nil })
	require.NoError(t, err)
	require.True(t, result.Processed)
	row, err = f.client.RTEDemand.Query().Where(demanddb.KindEQ(string(f.target.Kind))).Only(t.Context())
	require.NoError(t, err)
	require.Equal(t, row.DirtyVersion, row.CompletedVersion)
	clock.FreezeTime(now.Add(3*time.Minute + time.Second))
	_, err = f.repository.Invalidate(t.Context(), rte.InvalidateInput{Source: "ledger", EventID: "before-refresh-deadline", Selectors: []rte.DemandSelector{{Namespace: f.key.CustomerID.Namespace, Kind: f.target.Kind, Targets: []rte.CalculationTarget{f.target}}}})
	require.NoError(t, err)
	result, err = f.repository.Process(t.Context(), f.input, func(context.Context, rte.CalculationTarget) error { return nil })
	require.NoError(t, err)
	require.True(t, result.Processed, "source invalidation must not wait for the refresh deadline")
}

func TestDemandPullOnlyCoalescesWithoutBackgroundCalculations(t *testing.T) {
	// Given a pull-only key with a completed version and continued interest.
	f := newDemandFixture(t)
	require.NoError(t, f.repository.EnsurePull(t.Context(), rte.EnsureFreshInput{Target: f.target, PullTTL: time.Hour}))
	targeted := f.input
	targeted.Target = &f.target
	result, err := f.repository.Process(t.Context(), targeted, func(context.Context, rte.CalculationTarget) error { return nil })
	require.NoError(t, err)
	require.True(t, result.Processed)

	// When several facts arrive and background workers poll after every fact.
	for _, id := range []string{"first", "second", "third"} {
		marked, err := f.repository.Invalidate(t.Context(), rte.InvalidateInput{Source: "ledger", EventID: id, Selectors: []rte.DemandSelector{{Namespace: f.key.CustomerID.Namespace, Kind: f.target.Kind, Targets: []rte.CalculationTarget{f.target}}}})
		require.NoError(t, err)
		require.Equal(t, 1, marked.Matched)
		result, err := f.repository.Process(t.Context(), f.input, func(context.Context, rte.CalculationTarget) error {
			t.Fatal("pull-only key processed in background")
			return nil
		})
		require.NoError(t, err)
		require.False(t, result.Processed)
	}

	// Then a single explicit pull consumes all dirtiness and the next pull is clean.
	result, err = f.repository.Process(t.Context(), targeted, func(context.Context, rte.CalculationTarget) error { return nil })
	require.NoError(t, err)
	require.True(t, result.Processed)
	result, err = f.repository.Process(t.Context(), targeted, func(context.Context, rte.CalculationTarget) error { t.Fatal("clean pull recalculated"); return nil })
	require.NoError(t, err)
	require.False(t, result.Processed)
}

func TestDemandReceiptRetentionBoundsDeletionAndAllowsSafeReplay(t *testing.T) {
	// Given two old receipts and a recent receipt for an active calculation.
	f := newDemandFixture(t)
	now := time.Date(2026, time.November, 11, 12, 0, 0, 0, time.UTC)
	clock.FreezeTime(now.Add(-8 * 24 * time.Hour))
	defer clock.UnFreeze()
	require.NoError(t, f.repository.Activate(t.Context(), rte.ActivateDemandInput{Target: f.target}))
	input := rte.InvalidateInput{Source: "ledger", Selectors: []rte.DemandSelector{{Namespace: f.key.CustomerID.Namespace, Kind: f.target.Kind, Targets: []rte.CalculationTarget{f.target}}}}
	for _, id := range []string{"old-first", "old-second"} {
		input.EventID = id
		_, err := f.repository.Invalidate(t.Context(), input)
		require.NoError(t, err)
	}

	clock.FreezeTime(now)
	input.EventID = "recent"
	_, err := f.repository.Invalidate(t.Context(), input)
	require.NoError(t, err)

	// When maintenance prunes only one expired receipt.
	pruned, err := f.repository.PruneInvalidations(t.Context(), rte.PruneInvalidationsInput{Before: now.Add(-7 * 24 * time.Hour), Limit: 1})
	require.NoError(t, err)
	require.Equal(t, 1, pruned)
	count, err := f.client.RTEInvalidation.Query().Count(t.Context())
	require.NoError(t, err)
	require.Equal(t, 2, count)
	input.EventID = "recent"
	duplicate, err := f.repository.Invalidate(t.Context(), input)
	require.NoError(t, err)
	require.True(t, duplicate.Duplicate)
	input.EventID = "old-second"
	duplicate, err = f.repository.Invalidate(t.Context(), input)
	require.NoError(t, err)
	require.True(t, duplicate.Duplicate)

	// Then replaying the deleted receipt schedules extra work without losing state.
	input.EventID = "old-first"
	marked, err := f.repository.Invalidate(t.Context(), input)
	require.NoError(t, err)
	require.False(t, marked.Duplicate)
	require.Equal(t, 1, marked.Matched)
}

func TestDemandLargeInvalidationUsesArrayBindings(t *testing.T) {
	// Given one active key selected by more identities than PostgreSQL's bind limit.
	f := newDemandFixture(t)
	require.NoError(t, f.repository.Activate(t.Context(), rte.ActivateDemandInput{Target: f.target}))
	owners := make([]string, 65536)
	targets := make([]rte.CalculationTarget, 65536)
	for n := range owners {
		owners[n] = f.key.CustomerID.ID
		targets[n] = f.target
	}

	// When the large selector reaches the real PostgreSQL adapter.
	result, err := f.repository.Invalidate(t.Context(), rte.InvalidateInput{Source: "usage", EventID: "large-fanout", Selectors: []rte.DemandSelector{{Namespace: f.key.CustomerID.Namespace, Kind: f.target.Kind, OwnerIDs: owners, Targets: targets}}})

	// Then the key is marked once, without one bind parameter per identity.
	require.NoError(t, err)
	require.Equal(t, 1, result.Matched)
}

func TestDemandAttemptTimeoutPreservesRetryAndOtherReadyKeys(t *testing.T) {
	// Given a demanded key whose calculation stalls, plus another ready key.
	f := newDemandFixture(t)
	require.NoError(t, f.repository.Activate(t.Context(), rte.ActivateDemandInput{Target: f.target}))
	other := f.key
	other.CustomerID.ID = ulid.Make().String()
	event, err := rte.NewCalculationRequestedEvent(other)
	require.NoError(t, err)
	require.NoError(t, f.repository.Activate(t.Context(), rte.ActivateDemandInput{Target: event.CalculationTarget}))
	input := f.input
	input.CalculationTimeout = 50 * time.Millisecond

	// When the calculation writes state and exceeds only its attempt deadline.
	result, err := f.repository.Process(t.Context(), input, func(ctx context.Context, _ rte.CalculationTarget) error {
		_, err := f.store.Store(ctx, rte.StoreInput[creditbalance.Key, creditbalance.Balance]{Key: f.key, State: rte.State[creditbalance.Balance]{CalculatedAt: clock.Now()}})
		if err != nil {
			return err
		}

		<-ctx.Done()

		return ctx.Err()
	})
	require.NoError(t, err)
	require.True(t, result.Processed)
	require.ErrorIs(t, result.CalculationError, context.DeadlineExceeded)
	_, err = f.store.Get(t.Context(), f.key)
	require.True(t, models.IsGenericNotFoundError(err))
	row, err := f.client.RTEDemand.Query().Where(demanddb.OwnerIDEQ(f.key.CustomerID.ID)).Only(t.Context())
	require.NoError(t, err)
	require.Equal(t, 1, row.Attempts)
	require.True(t, row.NextAttemptAt.After(clock.Now()))

	// Then the retry has durable backoff and another ready key can make progress.
	result, err = f.repository.Process(t.Context(), f.input, func(_ context.Context, target rte.CalculationTarget) error {
		identity, err := target.Identity()
		if err != nil {
			return err
		}

		require.Equal(t, other.CustomerID.ID, identity.OwnerID)

		return nil
	})
	require.NoError(t, err)
	require.True(t, result.Processed)
	require.NoError(t, result.CalculationError)
}

func TestDemandDatabaseTimeoutPersistsRetry(t *testing.T) {
	// Given an attempt that times out inside a PostgreSQL read on its transaction.
	f := newDemandFixture(t)
	require.NoError(t, f.repository.Activate(t.Context(), rte.ActivateDemandInput{Target: f.target}))
	other := f.key
	other.CustomerID.ID = ulid.Make().String()
	event, err := rte.NewCalculationRequestedEvent(other)
	require.NoError(t, err)
	require.NoError(t, f.repository.Activate(t.Context(), rte.ActivateDemandInput{Target: event.CalculationTarget}))
	input := f.input
	input.CalculationTimeout = 100 * time.Millisecond
	result, err := f.repository.Process(t.Context(), input, func(ctx context.Context, _ rte.CalculationTarget) error {
		driver, err := entutils.GetDriverFromContext(ctx)
		if err != nil {
			return err
		}

		client := db.NewTxClientFromRawConfig(ctx, *driver.GetConfig()).Client()
		_, err = client.ExecContext(ctx, "SELECT pg_sleep(1)")

		return errors.Join(ctx.Err(), err)
	})

	// Then cancellation cannot leave the failed key hot at the front of the queue.
	require.NoError(t, err)
	require.ErrorIs(t, result.CalculationError, context.DeadlineExceeded)
	row, err := f.client.RTEDemand.Query().Where(demanddb.OwnerIDEQ(f.key.CustomerID.ID)).Only(t.Context())
	require.NoError(t, err)
	require.Equal(t, 1, row.Attempts)
	require.True(t, row.NextAttemptAt.After(clock.Now()))
	result, err = f.repository.Process(t.Context(), f.input, func(_ context.Context, target rte.CalculationTarget) error {
		identity, err := target.Identity()
		if err != nil {
			return err
		}

		require.Equal(t, other.CustomerID.ID, identity.OwnerID)

		return nil
	})
	require.NoError(t, err)
	require.True(t, result.Processed)
}

func TestDemandAttemptRequiresPositiveTimeout(t *testing.T) {
	input := rte.ProcessDemandInput{RetryMinDelay: time.Second, RetryMaxDelay: time.Minute}
	require.ErrorContains(t, input.Validate(), "calculation timeout must be positive")
	input.CalculationTimeout = time.Second
	require.NoError(t, input.Validate())
}

func TestDemandTimeoutRecoveryRetainsConcurrentPeriodicSuccess(t *testing.T) {
	// Given clean periodic demand whose next timer has become due.
	f := newDemandFixture(t)
	now := time.Date(2026, time.November, 11, 12, 0, 0, 0, time.UTC)
	clock.FreezeTime(now)
	defer clock.UnFreeze()
	require.NoError(t, f.repository.Activate(t.Context(), rte.ActivateDemandInput{Target: f.target, RefreshInterval: time.Minute}))
	_, err := f.repository.Process(t.Context(), f.input, func(context.Context, rte.CalculationTarget) error { return nil })
	require.NoError(t, err)
	before, err := f.client.RTEDemand.Query().Only(t.Context())
	require.NoError(t, err)
	clock.FreezeTime(now.Add(time.Minute))
	input := f.input
	input.CalculationTimeout = 100 * time.Millisecond
	input.RetryMinDelay = 10 * time.Minute

	// When a database timeout releases the lock and another worker finishes the
	// same timer before the failed worker can record retry bookkeeping.
	result, err := f.repository.Process(t.Context(), input, func(ctx context.Context, _ rte.CalculationTarget) error {
		driver, err := entutils.GetDriverFromContext(ctx)
		if err != nil {
			return err
		}
		client := db.NewTxClientFromRawConfig(ctx, *driver.GetConfig()).Client()
		_, timeoutErr := client.ExecContext(ctx, "SELECT pg_sleep(1)")
		otherCtx, cancel := context.WithTimeout(t.Context(), 3*time.Second)
		defer cancel()
		targeted := f.input
		targeted.Target = &f.target
		other, err := f.repository.Process(otherCtx, targeted, func(context.Context, rte.CalculationTarget) error { return nil })
		if err != nil {
			return err
		}
		require.True(t, other.Processed)
		return errors.Join(ctx.Err(), timeoutErr)
	})

	// Then failure recovery preserves the successful version and its next timer.
	require.NoError(t, err)
	require.ErrorIs(t, result.CalculationError, context.DeadlineExceeded)
	row, err := f.client.RTEDemand.Query().Only(t.Context())
	require.NoError(t, err)
	require.Equal(t, before.DirtyVersion+1, row.DirtyVersion)
	require.Equal(t, row.DirtyVersion, row.CompletedVersion)
	require.Zero(t, row.Attempts)
	require.True(t, row.NextAttemptAt.Equal(now.Add(2*time.Minute)))
}
