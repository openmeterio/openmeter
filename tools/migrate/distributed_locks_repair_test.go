package migrate_test

import (
	"database/sql"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRepairDistributedLocksMigration(t *testing.T) {
	t.Run("repairs missing objects", func(t *testing.T) {
		// given:
		// - a database at the previous migration head without the raw-SQL lock objects
		// when:
		// - the distributed lock repair migration runs
		// then:
		// - the lock table and its record-version sequence are usable
		runner{
			stops: stops{
				{
					version:   20260818090123,
					direction: directionUp,
					action: func(t *testing.T, db *sql.DB) {
						_, err := db.ExecContext(t.Context(), `DROP TABLE distributed_locks`)
						require.NoError(t, err)
					},
				},
				{
					version:   20260824141945,
					direction: directionUp,
					action: func(t *testing.T, db *sql.DB) {
						_, err := db.ExecContext(t.Context(), `
							INSERT INTO distributed_locks (name, record_version_number)
							VALUES ('repair-test', nextval('distributed_locks_rvn'))
						`)
						require.NoError(t, err)
					},
				},
			},
		}.Test(t)
	})

	t.Run("preserves existing objects", func(t *testing.T) {
		// given:
		// - a normally migrated database with an existing distributed lock row
		// when:
		// - the idempotent repair migration runs
		// then:
		// - the existing lock row remains unchanged
		runner{
			stops: stops{
				{
					version:   20260818090123,
					direction: directionUp,
					action: func(t *testing.T, db *sql.DB) {
						_, err := db.ExecContext(t.Context(), `
							INSERT INTO distributed_locks (name, record_version_number, owner)
							VALUES ('existing-lock', nextval('distributed_locks_rvn'), 'existing-owner')
						`)
						require.NoError(t, err)
					},
				},
				{
					version:   20260824141945,
					direction: directionUp,
					action: func(t *testing.T, db *sql.DB) {
						var owner string
						err := db.QueryRowContext(t.Context(), `
							SELECT owner FROM distributed_locks WHERE name = 'existing-lock'
						`).Scan(&owner)
						require.NoError(t, err)
						require.Equal(t, "existing-owner", owner)
					},
				},
			},
		}.Test(t)
	})
}
