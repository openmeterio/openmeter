package adapter_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/require"

	creditadapter "github.com/openmeterio/openmeter/openmeter/credit/adapter"
	"github.com/openmeterio/openmeter/openmeter/credit/balance"
	"github.com/openmeterio/openmeter/openmeter/entitlement"
	entitlementadapter "github.com/openmeterio/openmeter/openmeter/entitlement/adapter"
	"github.com/openmeterio/openmeter/openmeter/streaming"
	"github.com/openmeterio/openmeter/openmeter/testutils"
	"github.com/openmeterio/openmeter/pkg/framework/transaction"
	"github.com/openmeterio/openmeter/pkg/models"
)

func TestBalanceSnapshotInvalidationVersionIsAtomicWithInvalidation(t *testing.T) {
	// given: an entitlement with a saved balance snapshot
	testDB := testutils.InitPostgresDB(t, testutils.PostgresDBStateEntMigrated)
	dbClient := testDB.EntDriver.Client()
	t.Cleanup(func() {
		_ = dbClient.Close()
		testDB.Close(t)
	})

	namespace := ulid.Make().String()
	customer, err := dbClient.Customer.Create().
		SetNamespace(namespace).
		SetName("Customer").
		SetKey("customer-" + ulid.Make().String()).
		Save(t.Context())
	require.NoError(t, err)

	feature, err := dbClient.Feature.Create().
		SetNamespace(namespace).
		SetName("Feature").
		SetKey("feature-" + ulid.Make().String()).
		Save(t.Context())
	require.NoError(t, err)

	entitlementRepo := entitlementadapter.NewPostgresEntitlementRepo(dbClient)
	ent, err := entitlementRepo.CreateEntitlement(t.Context(), entitlement.CreateEntitlementRepoInputs{
		Namespace:  namespace,
		FeatureID:  feature.ID,
		FeatureKey: feature.Key,
		UsageAttribution: streaming.CustomerUsageAttribution{
			ID:  customer.ID,
			Key: &customer.Key,
		},
		EntitlementType: entitlement.EntitlementTypeMetered,
	})
	require.NoError(t, err)

	owner := models.NamespacedID{Namespace: namespace, ID: ent.ID}
	snapshotAt := time.Now().UTC().Truncate(time.Minute)
	repo := creditadapter.NewPostgresBalanceSnapshotRepo(dbClient)
	require.NoError(t, repo.Save(t.Context(), owner, []balance.Snapshot{
		balance.NewStartingSnapshot(nil, snapshotAt),
	}))

	version, err := repo.GetInvalidationVersion(t.Context(), owner)
	require.NoError(t, err)
	require.Zero(t, version)

	// when: the outer transaction fails after invalidation
	rollbackErr := errors.New("rollback")
	err = transaction.RunWithNoValue(t.Context(), repo, func(ctx context.Context) error {
		if err := repo.InvalidateAfter(ctx, owner, snapshotAt.Add(-time.Minute)); err != nil {
			return err
		}

		version, err := repo.GetInvalidationVersion(ctx, owner)
		require.NoError(t, err)
		require.Equal(t, balance.SnapshotInvalidationVersion(1), version)

		return rollbackErr
	})
	require.ErrorIs(t, err, rollbackErr)

	// then: the version increment and snapshot invalidation both roll back
	version, err = repo.GetInvalidationVersion(t.Context(), owner)
	require.NoError(t, err)
	require.Zero(t, version)

	_, err = repo.GetLatestValidAt(t.Context(), owner, snapshotAt)
	require.NoError(t, err)

	// and: a committed invalidation advances the durable fence
	require.NoError(t, repo.InvalidateAfter(t.Context(), owner, snapshotAt.Add(-time.Minute)))
	version, err = repo.GetInvalidationVersion(t.Context(), owner)
	require.NoError(t, err)
	require.Equal(t, balance.SnapshotInvalidationVersion(1), version)

	_, err = repo.GetLatestValidAt(t.Context(), owner, snapshotAt)
	var noSnapshot *balance.NoSavedBalanceForOwnerError
	require.ErrorAs(t, err, &noSnapshot)
}
