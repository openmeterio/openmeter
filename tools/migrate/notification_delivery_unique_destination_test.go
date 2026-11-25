package migrate_test

import (
	"io/fs"
	"os"
	"path"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/lib/pq"
	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/testutils"
	"github.com/openmeterio/openmeter/tools/migrate"
)

func TestNotificationDeliveryUniqueDestinationMigration(t *testing.T) {
	testDB := testutils.InitPostgresDB(t, testutils.PostgresDBStateEntMigrated)
	t.Cleanup(func() { testDB.Close(t) })

	const migration = "20261125171748_notification_delivery_unique_destination"
	up, err := fs.ReadFile(migrate.OMMigrationsConfig.FS, path.Join(migrate.OMMigrationsConfig.FSPath, migration+".up.sql"))
	require.NoError(t, err)

	down, err := fs.ReadFile(migrate.OMMigrationsConfig.FS, path.Join(migrate.OMMigrationsConfig.FSPath, migration+".down.sql"))
	require.NoError(t, err)

	preflight, err := os.ReadFile("notification_delivery_duplicate_preflight.sql")
	require.NoError(t, err)

	ns := ulid.Make().String()
	event := ulid.Make().String()
	channel := ulid.Make().String()
	tests := []struct {
		name      string
		namespace string
		eventID   string
		channelID string
		duplicate bool
	}{
		{
			name:      "duplicate histories block the migration",
			namespace: ns,
			eventID:   event,
			channelID: channel,
			duplicate: true,
		},
		{
			name:      "another namespace",
			namespace: ulid.Make().String(),
			eventID:   event,
			channelID: channel,
		},
		{
			name:      "another event",
			namespace: ns,
			eventID:   ulid.Make().String(),
			channelID: channel,
		},
		{
			name:      "another channel",
			namespace: ns,
			eventID:   event,
			channelID: ulid.Make().String(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := t.Context()
			tx, err := testDB.PGDriver.DB().BeginTx(ctx, nil)
			require.NoError(t, err)

			defer func() { require.NoError(t, tx.Rollback()) }()

			// Given two deliveries stored under the original nonunique index.
			_, err = tx.ExecContext(ctx, string(down))
			require.NoError(t, err)

			const insert = `INSERT INTO notification_event_delivery_status
				(id, namespace, event_id, channel_id, created_at, updated_at)
				VALUES ($1, $2, $3, $4, now(), now())`
			firstID, secondID := ulid.Make().String(), ulid.Make().String()
			_, err = tx.ExecContext(ctx, insert, firstID, ns, event, channel)
			require.NoError(t, err)

			_, err = tx.ExecContext(ctx, insert, secondID, tt.namespace, tt.eventID, tt.channelID)
			require.NoError(t, err)

			// When an operator checks which destinations block the upgrade.
			rows, err := tx.QueryContext(ctx, string(preflight))
			require.NoError(t, err)

			if tt.duplicate {
				require.True(t, rows.Next())

				var namespace, eventID, channelID string
				var count int
				var ids pq.StringArray
				err := rows.Scan(&namespace, &eventID, &channelID, &count, &ids)
				require.NoError(t, err)

				require.Equal(t, ns, namespace)
				require.Equal(t, event, eventID)
				require.Equal(t, channel, channelID)
				require.Equal(t, 2, count)
				require.ElementsMatch(t, []string{firstID, secondID}, []string(ids))
			}

			require.False(t, rows.Next())
			require.NoError(t, rows.Err())
			require.NoError(t, rows.Close())

			// Then a collision blocks the upgrade, or a clean upgrade prevents future duplicates.
			_, err = tx.ExecContext(ctx, string(up))
			if !tt.duplicate {
				require.NoError(t, err)

				_, err = tx.ExecContext(ctx, insert, ulid.Make().String(), ns, event, channel)
			}

			var pgErr *pgconn.PgError
			require.ErrorAs(t, err, &pgErr)
			require.Equal(t, "23505", pgErr.Code)
		})
	}
}
