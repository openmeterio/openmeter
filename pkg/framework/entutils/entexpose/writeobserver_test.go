package entexpose_test

import (
	"context"
	"testing"

	"entgo.io/ent"
	"github.com/stretchr/testify/require"

	entdb "github.com/openmeterio/openmeter/openmeter/ent/db"
	dbfeature "github.com/openmeterio/openmeter/openmeter/ent/db/feature"
	"github.com/openmeterio/openmeter/openmeter/testutils"
	"github.com/openmeterio/openmeter/pkg/framework/transaction"
	transactiontestutils "github.com/openmeterio/openmeter/pkg/framework/transaction/testutils"
)

// Downstream binaries build the OpenMeter client with entdb.NewClient on their own
// driver, so the reporting must not depend on how or where the client is constructed.
func TestGeneratedClientReportsWrites(t *testing.T) {
	testDB := testutils.InitPostgresDB(t, testutils.PostgresDBStateEntMigrated)
	t.Cleanup(func() { testDB.Close(t) })

	// given a client built directly from the generated constructor, and a context with a write observer
	client := entdb.NewClient(entdb.Driver(testDB.EntDriver.Driver()))

	recorder := &transactiontestutils.WriteRecorder{}
	ctx, err := transaction.WithWriteObserver(t.Context(), recorder)
	require.NoError(t, err)

	// when it mutates and locks rows outside a transaction
	_, err = client.Feature.Delete().Where(dbfeature.Namespace("write-observer-test")).Exec(ctx)
	require.NoError(t, err)

	_, err = client.Feature.Query().Where(dbfeature.Namespace("write-observer-test")).ForUpdate().All(ctx)
	require.NoError(t, err)

	// then both are reported
	require.Equal(t, []transaction.WriteAttempt{
		{Kind: transaction.WriteKindMutation, Target: "Feature", Operation: "OpDelete"},
		{Kind: transaction.WriteKindRowLock, Target: "Feature", Operation: "All"},
	}, recorder.Attempts())
}

// An eager-loaded query runs with the query context of the query that loads it,
// so the reported entity must come from the locked query itself.
func TestEagerLoadedRowLockReportsTheLockedEntity(t *testing.T) {
	testDB := testutils.InitPostgresDB(t, testutils.PostgresDBStateEntMigrated)
	t.Cleanup(func() { testDB.Close(t) })

	client := entdb.NewClient(entdb.Driver(testDB.EntDriver.Driver()))

	// given a feature, so the query has a parent row to load entitlements for
	_, err := client.Feature.Create().
		SetNamespace("write-observer-test").
		SetName("feature").
		SetKey("feature").
		Save(t.Context())
	require.NoError(t, err)

	recorder := &transactiontestutils.WriteRecorder{}
	ctx, err := transaction.WithWriteObserver(t.Context(), recorder)
	require.NoError(t, err)

	// when only the eager-loaded entitlements are locked
	_, err = client.Feature.Query().
		Where(dbfeature.Namespace("write-observer-test")).
		WithEntitlement(func(q *entdb.EntitlementQuery) {
			q.ForUpdate()
		}).
		All(ctx)
	require.NoError(t, err)

	// then the lock is reported on the entitlements, not on the features that load them
	require.Equal(t, []transaction.WriteAttempt{
		{Kind: transaction.WriteKindRowLock, Target: "Entitlement", Operation: "All"},
	}, recorder.Attempts())
}

// The write observers are shared by every client of the package. Hooks and
// interceptors that clients register on top must stay with the client they were
// registered on, and must not replace the ones of another client.
func TestRegisteredHooksStayWithTheirClient(t *testing.T) {
	testDB := testutils.InitPostgresDB(t, testutils.PostgresDBStateEntMigrated)
	t.Cleanup(func() { testDB.Close(t) })

	// given two clients, each with its own hook and interceptor
	type calls struct{ hook, interceptor int }

	register := func(client *entdb.Client, counted *calls) {
		client.Feature.Use(func(next ent.Mutator) ent.Mutator {
			return ent.MutateFunc(func(ctx context.Context, m ent.Mutation) (ent.Value, error) {
				counted.hook++

				return next.Mutate(ctx, m)
			})
		})

		client.Feature.Intercept(ent.InterceptFunc(func(next ent.Querier) ent.Querier {
			return ent.QuerierFunc(func(ctx context.Context, q ent.Query) (ent.Value, error) {
				counted.interceptor++

				return next.Query(ctx, q)
			})
		}))
	}

	first := entdb.NewClient(entdb.Driver(testDB.EntDriver.Driver()))
	second := entdb.NewClient(entdb.Driver(testDB.EntDriver.Driver()))

	var firstCalls, secondCalls calls
	register(first, &firstCalls)
	register(second, &secondCalls)

	recorder := &transactiontestutils.WriteRecorder{}
	ctx, err := transaction.WithWriteObserver(t.Context(), recorder)
	require.NoError(t, err)

	// when only the first client mutates and locks rows
	_, err = first.Feature.Delete().Where(dbfeature.Namespace("write-observer-test")).Exec(ctx)
	require.NoError(t, err)

	_, err = first.Feature.Query().Where(dbfeature.Namespace("write-observer-test")).ForUpdate().All(ctx)
	require.NoError(t, err)

	// then only its own hook and interceptor ran, and the writes are still reported
	require.Equal(t, calls{hook: 1, interceptor: 1}, firstCalls)
	require.Equal(t, calls{}, secondCalls)
	require.Len(t, recorder.Attempts(), 2)
}
