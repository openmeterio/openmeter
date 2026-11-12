package migrate_test

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestChargeSubscriptionLookupIndexes(t *testing.T) {
	ctx := t.Context()
	db, migrator := newMigrationTestEnv(t)

	// Given the schema before the independent concurrent index migrations.
	require.NoError(t, migrator.Migrate(20261110144149))
	indexes := []struct {
		version uint
		name    string
	}{
		{version: 20261112190449, name: "chargecreditpurchase_namespace_subscription_id"},
		{version: 20261112190450, name: "chargeflatfee_namespace_subscription_id"},
		{version: 20261112190451, name: "chargeusagebased_namespace_subscription_id"},
	}

	// When the migration runner applies each index outside a transaction block.
	for _, index := range indexes {
		require.NoError(t, migrator.Migrate(index.version))

		// Then the index is ready for reads and contains the intended lookup keys.
		var valid bool
		var definition string
		err := db.QueryRowContext(ctx, `
			SELECT i.indisvalid AND i.indisready, pg_get_indexdef(i.indexrelid)
			FROM pg_index i
			JOIN pg_class c ON c.oid = i.indexrelid
			WHERE c.relnamespace = 'public'::regnamespace AND c.relname = $1
		`, index.name).Scan(&valid, &definition)
		require.NoError(t, err)
		require.True(t, valid, index.name)
		require.Contains(t, definition, "USING btree (namespace, subscription_id)")
	}

	// Concurrent rollback must also work through the migration runner.
	for i := len(indexes) - 1; i >= 0; i-- {
		require.NoError(t, migrator.Steps(-1))

		var exists bool
		err := db.QueryRowContext(ctx, `
			SELECT EXISTS (
				SELECT 1 FROM pg_class
				WHERE relnamespace = 'public'::regnamespace AND relname = $1
			)
		`, indexes[i].name).Scan(&exists)
		require.NoError(t, err)
		require.False(t, exists, indexes[i].name)
	}

	require.NoError(t, migrator.Up())
}
