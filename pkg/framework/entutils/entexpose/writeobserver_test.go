package entexpose_test

import (
	"testing"

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
