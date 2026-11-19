package migrate_test

import (
	"database/sql"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRTEDemandCalculationLocksMigration(t *testing.T) {
	// This is the registration shape used before calculation locks existed.
	insertDemand := func(t *testing.T, db *sql.DB, key string) int {
		t.Helper()
		var id int
		err := db.QueryRowContext(t.Context(), `
			INSERT INTO rte_demands (namespace, created_at, updated_at, kind, owner_id, key_hash, key, next_attempt_at)
			VALUES ('test', now(), now(), 'credit_balance', 'customer', $1, '{}', now())
			RETURNING id`, key).Scan(&id)
		require.NoError(t, err)

		return id
	}

	var existingID int
	runner{stops: stops{
		{
			version: 20261119143226, direction: directionUp,
			action: func(t *testing.T, db *sql.DB) {
				// Given a demand created before the migration.
				existingID = insertDemand(t, db, "existing")
			},
		},
		{
			version: 20261119143321, direction: directionUp,
			action: func(t *testing.T, db *sql.DB) {
				// When an older binary also registers a key after migration.
				newID := insertDemand(t, db, "new")

				// Then both keys have a lock, and deleting one demand removes only its lock.
				var count int
				err := db.QueryRowContext(t.Context(), `SELECT count(*) FROM rte_demand_locks WHERE id IN ($1, $2)`, existingID, newID).Scan(&count)
				require.NoError(t, err)
				require.Equal(t, 2, count)

				_, err = db.ExecContext(t.Context(), `DELETE FROM rte_demands WHERE id = $1`, newID)
				require.NoError(t, err)
				err = db.QueryRowContext(t.Context(), `SELECT count(*) FROM rte_demand_locks WHERE id = $1`, newID).Scan(&count)
				require.NoError(t, err)
				require.Zero(t, count)
			},
		},
		{
			version: 20261119143226, direction: directionDown,
			action: func(t *testing.T, db *sql.DB) {
				// A rolled-back writer can still register a demand after the lock table is dropped.
				insertDemand(t, db, "after-rollback")
			},
		},
	}}.Test(t)
}
