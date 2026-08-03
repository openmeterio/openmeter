package testutils

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestInitPostgresDBStates(t *testing.T) {
	tests := []struct {
		name              string
		state             PostgresDBState
		hasCustomerTable  bool
		hasMigrationTable bool
	}{
		{
			name:  "empty",
			state: PostgresDBStateEmpty,
		},
		{
			name:             "ent migrated",
			state:            PostgresDBStateEntMigrated,
			hasCustomerTable: true,
		},
		{
			name:              "atlas migrated",
			state:             PostgresDBStateAtlasMigrated,
			hasCustomerTable:  true,
			hasMigrationTable: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := InitPostgresDB(t, tt.state)
			t.Cleanup(func() { db.Close(t) })

			require.Equal(t, tt.hasCustomerTable, tableExists(t, db, "customers"))
			require.Equal(t, tt.hasMigrationTable, tableExists(t, db, migrateTableName))
		})
	}
}

const migrateTableName = "schema_om"

func tableExists(t *testing.T, db *TestDB, tableName string) bool {
	t.Helper()

	var exists bool
	err := db.PGDriver.DB().QueryRowContext(t.Context(), `
		SELECT EXISTS (
			SELECT 1
			FROM information_schema.tables
			WHERE table_schema = 'public' AND table_name = $1
		)
	`, tableName).Scan(&exists)
	require.NoError(t, err)

	return exists
}
