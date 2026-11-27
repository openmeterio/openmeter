package migrate_test

import (
	"context"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/lib/pq"
	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	previousDeliveryIndexVersion = 20261125160000
	buildDeliveryIndexVersion    = 20261125171748
	dropOldDeliveryIndexVersion  = 20261125171749
)

func TestNotificationDeliveryUniqueDestinationMigration(t *testing.T) {
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
			// Given two deliveries stored under the original nonunique index.
			sqlDB, migrator := newMigrationTestEnv(t)
			require.NoError(t, migrator.Migrate(previousDeliveryIndexVersion))

			const insert = `INSERT INTO notification_event_delivery_status
				(id, namespace, event_id, channel_id, created_at, updated_at)
				VALUES ($1, $2, $3, $4, now(), now())`
			firstID, secondID := ulid.Make().String(), ulid.Make().String()
			_, err = sqlDB.ExecContext(ctx, insert, firstID, ns, event, channel)
			require.NoError(t, err)

			_, err = sqlDB.ExecContext(ctx, insert, secondID, tt.namespace, tt.eventID, tt.channelID)
			require.NoError(t, err)

			// When an operator checks which destinations block the upgrade.
			rows, err := sqlDB.QueryContext(ctx, string(preflight))
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

			// Then a failed build keeps the old index usable and can be retried after repair.
			if tt.duplicate {
				err = migrator.Migrate(buildDeliveryIndexVersion)
				require.ErrorContains(t, err, "could not create unique index")

				version, dirty, err := migrator.Version()
				require.NoError(t, err)
				require.EqualValues(t, buildDeliveryIndexVersion, version)
				require.True(t, dirty)

				var oldValid, newValid bool
				err = sqlDB.QueryRowContext(ctx, `SELECT old.indisvalid, replacement.indisvalid
					FROM pg_index old, pg_index replacement
					WHERE old.indexrelid = 'notificationeventdeliverystatus_namespace_event_id_channel_id'::regclass
					  AND replacement.indexrelid = 'notification_delivery_destination_unique'::regclass`).Scan(&oldValid, &newValid)
				require.NoError(t, err)
				require.True(t, oldValid)
				require.False(t, newValid)

				_, err = sqlDB.ExecContext(ctx, "DROP INDEX CONCURRENTLY notification_delivery_destination_unique")
				require.NoError(t, err)

				_, err = sqlDB.ExecContext(ctx, "DELETE FROM notification_event_delivery_status WHERE id = $1", secondID)
				require.NoError(t, err)

				require.NoError(t, migrator.Force(previousDeliveryIndexVersion))
			}

			require.NoError(t, migrator.Migrate(dropOldDeliveryIndexVersion))

			_, err = sqlDB.ExecContext(ctx, insert, ulid.Make().String(), ns, event, channel)
			var pgErr *pgconn.PgError
			require.ErrorAs(t, err, &pgErr)
			require.Equal(t, "23505", pgErr.Code)
			require.Equal(t, "notification_delivery_destination_unique", pgErr.ConstraintName)

			// Downgrading restores the original lookup index and allows duplicate destinations.
			require.NoError(t, migrator.Migrate(previousDeliveryIndexVersion))

			_, err = sqlDB.ExecContext(ctx, insert, ulid.Make().String(), ns, event, channel)
			require.NoError(t, err)
		})
	}
}

func TestNotificationDeliveryWritesDuringIndexBuild(t *testing.T) {
	sqlDB, migrator := newMigrationTestEnv(t)
	require.NoError(t, migrator.Migrate(previousDeliveryIndexVersion))

	// Given an open delivery writer that holds the concurrent build before its first scan.
	ctx := t.Context()
	writer, err := sqlDB.BeginTx(ctx, nil)
	require.NoError(t, err)

	var build sync.WaitGroup
	defer func() {
		_ = writer.Rollback()
		build.Wait()
	}()

	const insert = `INSERT INTO notification_event_delivery_status
		(id, namespace, event_id, channel_id, created_at, updated_at)
		VALUES ($1, $2, $3, $4, now(), now())`
	ns, event := ulid.Make().String(), ulid.Make().String()
	_, err = writer.ExecContext(ctx, insert, ulid.Make().String(), ns, event, ulid.Make().String())
	require.NoError(t, err)

	// When the migration waits for the first writer, a second delivery can still commit.
	var buildErr error
	build.Go(func() { buildErr = migrator.Migrate(buildDeliveryIndexVersion) })

	require.EventuallyWithT(t, func(c *assert.CollectT) {
		var waiting bool
		err := sqlDB.QueryRowContext(ctx, `SELECT EXISTS (
			SELECT 1 FROM pg_stat_progress_create_index
			WHERE relid = 'notification_event_delivery_status'::regclass
			  AND phase = 'waiting for writers before build'
		)`).Scan(&waiting)
		require.NoError(c, err)
		require.True(c, waiting)
	}, 10*time.Second, 25*time.Millisecond)

	writeCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	_, err = sqlDB.ExecContext(writeCtx, insert, ulid.Make().String(), ns, event, ulid.Make().String())
	require.NoError(t, err)

	// Then the build completes once the first writer commits.
	require.NoError(t, writer.Commit())
	build.Wait()
	require.NoError(t, buildErr)
}
