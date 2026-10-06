package adapter_test

import (
	"context"
	"testing"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/oapi-codegen/nullable"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/ent/db"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/adapter"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/feature"
	"github.com/openmeterio/openmeter/openmeter/testutils"
	"github.com/openmeterio/openmeter/pkg/framework/entutils"
	"github.com/openmeterio/openmeter/pkg/framework/transaction"
	"github.com/openmeterio/openmeter/pkg/pagination"
)

func TestFeatureRepositoryReusesTransaction(t *testing.T) {
	// given an uncommitted meter and a transaction holding the only connection
	testDB := testutils.InitPostgresDB(t, testutils.PostgresDBStateEntMigrated)
	t.Cleanup(func() { testDB.Close(t) })
	testDB.PGDriver.DB().SetMaxOpenConns(1)
	repo := adapter.NewPostgresFeatureRepo(testDB.EntDriver.Client(), testutils.NewDiscardLogger(t))
	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancel()
	txCtx, driver, err := repo.Tx(ctx)
	require.NoError(t, err)
	defer func() { require.NoError(t, driver.Rollback()) }()
	txCtx, err = transaction.SetDriverOnContext(txCtx, driver)
	require.NoError(t, err)
	txDriver, err := entutils.GetDriverFromContext(txCtx)
	require.NoError(t, err)
	client := db.NewTxClientFromRawConfig(txCtx, *txDriver.GetConfig()).Client()
	meter, err := client.Meter.Create().
		SetNamespace("transaction-features").
		SetName("Meter").
		SetKey("meter").
		SetEventType("event").
		SetAggregation("COUNT").
		Save(txCtx)
	require.NoError(t, err)

	// when the original repository receives the transaction context
	created, err := repo.CreateFeature(txCtx, feature.CreateFeatureInputs{
		Namespace: meter.Namespace,
		Name:      "Feature",
		Key:       "feature",
		MeterID:   &meter.ID,
	})
	require.NoError(t, err)

	// then every operation, including eager loading, counts, and follow-up reads,
	// completes on the caller's connection; CRUD semantics are covered in feature_test.go.
	for _, reference := range []string{created.ID, created.Key} {
		_, err := repo.GetByIdOrKey(txCtx, created.Namespace, reference, bool(feature.IncludeArchivedFeatureFalse))
		require.NoError(t, err)
	}
	for _, page := range []pagination.Page{{}, pagination.NewPage(1, 10)} {
		listed, err := repo.ListFeatures(txCtx, feature.ListFeaturesParams{Namespace: created.Namespace, Page: page})
		require.NoError(t, err)
		require.Len(t, listed.Items, 1, "the caller's uncommitted feature must be visible")
	}
	_, err = repo.UpdateFeature(txCtx, feature.UpdateFeatureInputs{
		Namespace: created.Namespace,
		ID:        created.ID,
		UnitCost: nullable.NewNullableWithValue(feature.UnitCost{
			Type:   feature.UnitCostTypeManual,
			Manual: &feature.ManualUnitCost{Amount: alpacadecimal.NewFromInt(2)},
		}),
	})
	require.NoError(t, err)
	active, err := repo.HasActiveFeatureForMeter(txCtx, created.Namespace, meter.ID)
	require.NoError(t, err)
	require.True(t, active, "the caller's uncommitted feature must be visible")
	require.NoError(t, repo.ArchiveFeature(txCtx, feature.ArchiveFeatureInput{Namespace: created.Namespace, ID: created.ID}))
}
