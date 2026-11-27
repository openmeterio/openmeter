package migrate_test

import (
	"context"
	"database/sql"
	"io/fs"
	"os"
	"path"
	"sync"
	"testing"
	"time"

	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/testutils"
	"github.com/openmeterio/openmeter/tools/migrate"
)

func TestNotificationChannelEventTypeFiltersMigration(t *testing.T) {
	up, err := fs.ReadFile(migrate.OMMigrationsConfig.FS, path.Join(migrate.OMMigrationsConfig.FSPath, "20261126090000_notification_channel_event_type_filters.up.sql"))
	require.NoError(t, err)

	preflight, err := os.ReadFile("notification_channel_event_type_filters_preflight.sql")
	require.NoError(t, err)

	const (
		insertChannel = `INSERT INTO notification_channels
			(id, namespace, type, name, config, disabled, deleted_at, event_type_filter, created_at, updated_at)
			VALUES ($1, $2, 'webhook', 'test', '{}', $3, $4, $5, NOW(), NOW()) RETURNING updated_at`
		insertRule = `INSERT INTO notification_rules
			(id, namespace, type, name, config, disabled, deleted_at, created_at, updated_at)
			VALUES ($1, $2, $3, 'test', '{}', $4, $5, NOW(), NOW())`
		assign = `INSERT INTO notification_channel_rules (notification_channel_id, notification_rule_id) VALUES ($1, $2)`
	)

	tests := []struct {
		name             string
		disabled         bool
		nullDisabled     bool
		deleted          bool
		mixedAssignments bool
		existing         *string
		expected         string
		drift            bool
		catchAll         bool
	}{
		{
			name:             "distinct sorted types include disabled rules and omit deleted and foreign rules",
			mixedAssignments: true,
			existing:         new(`["stale"]`),
			expected:         `["v1.billing.invoice.created","v1.billing.invoice.updated"]`,
			drift:            true,
		},
		{
			name:     "unassigned enabled channel stays enabled and requires cutover review",
			expected: "[]",
			drift:    true,
			catchAll: true,
		},
		{
			name:         "legacy NULL disabled flag also needs cutover review",
			nullDisabled: true,
			expected:     "[]",
			drift:        true,
			catchAll:     true,
		},
		{
			name:     "unassigned disabled channel only needs its filter backfilled",
			disabled: true,
			expected: "[]",
			drift:    true,
		},
		{
			name:     "enabled channel created by the new writer still requires cutover review",
			existing: new("[]"),
			expected: "[]",
			catchAll: true,
		},
		{
			name:     "deleted channel is excluded",
			deleted:  true,
			existing: new(`["historical"]`),
			expected: `["historical"]`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given historical assignments and a Channel in a known lifecycle state.
			testDB := testutils.InitPostgresDB(t, testutils.PostgresDBStateEntMigrated)
			t.Cleanup(func() { testDB.Close(t) })
			db := testDB.PGDriver.DB()
			ctx := t.Context()
			id, ns := ulid.Make().String(), ulid.Make().String()
			var deletedAt *time.Time
			if tt.deleted {
				deletedAt = new(time.Now())
			}

			disabledFlag := new(tt.disabled)
			if tt.nullDisabled {
				disabledFlag = nil
			}

			var originalUpdatedAt time.Time
			err := db.QueryRowContext(ctx, insertChannel, id, ns, disabledFlag, deletedAt, tt.existing).Scan(&originalUpdatedAt)
			require.NoError(t, err)

			if tt.mixedAssignments {
				for _, rule := range []struct {
					namespace string
					eventType string
					disabled  bool
					deletedAt *time.Time
				}{
					{
						namespace: ns,
						eventType: "v1.billing.invoice.updated",
						disabled:  true,
					},
					{
						namespace: ns,
						eventType: "v1.billing.invoice.created",
					},
					{
						namespace: ns,
						eventType: "v1.billing.invoice.created",
					},
					{
						namespace: ns,
						eventType: "v1.entitlement.usage_period.reset",
						deletedAt: new(time.Now()),
					},
					{
						namespace: ulid.Make().String(),
						eventType: "v1.entitlement.usage_period.reset",
					},
				} {
					ruleID := ulid.Make().String()
					_, err := db.ExecContext(ctx, insertRule, ruleID, rule.namespace, rule.eventType, rule.disabled, rule.deletedAt)
					require.NoError(t, err)

					_, err = db.ExecContext(ctx, assign, id, ruleID)
					require.NoError(t, err)
				}
			}

			checkPreflight := func(drift bool) {
				t.Helper()
				var gotID, gotNS, expected string
				var stored sql.NullString
				var needsBackfill, catchAll bool
				err := db.QueryRowContext(ctx, string(preflight)).Scan(&gotNS, &gotID, &stored, &expected, &needsBackfill, &catchAll)
				if !drift && !tt.catchAll {
					require.ErrorIs(t, err, sql.ErrNoRows)
					return
				}

				require.NoError(t, err)
				require.Equal(t, ns, gotNS)
				require.Equal(t, id, gotID)
				require.JSONEq(t, tt.expected, expected)
				require.Equal(t, drift, needsBackfill)
				require.Equal(t, tt.catchAll, catchAll)
			}

			// When previewed and applied, only the filter changes.
			checkPreflight(tt.drift)
			_, err = db.ExecContext(ctx, string(up))
			require.NoError(t, err)

			var stored, version string
			var disabled *bool
			var updatedAt time.Time
			err = db.QueryRowContext(ctx, "SELECT event_type_filter, disabled, updated_at, xmin::text FROM notification_channels WHERE id = $1", id).
				Scan(&stored, &disabled, &updatedAt, &version)
			require.NoError(t, err)

			require.JSONEq(t, tt.expected, stored)
			require.Equal(t, disabledFlag, disabled)
			require.Equal(t, originalUpdatedAt, updatedAt)

			// Then a second run leaves the row untouched and the catch-all warning remains actionable.
			_, err = db.ExecContext(ctx, string(up))
			require.NoError(t, err)

			var nextVersion string
			err = db.QueryRowContext(ctx, "SELECT xmin::text FROM notification_channels WHERE id = $1", id).Scan(&nextVersion)
			require.NoError(t, err)
			require.Equal(t, version, nextVersion)

			checkPreflight(false)
		})
	}

	t.Run("sees assignments committed while waiting for the writer", func(t *testing.T) {
		testDB := testutils.InitPostgresDB(t, testutils.PostgresDBStateEntMigrated)
		t.Cleanup(func() { testDB.Close(t) })
		db := testDB.PGDriver.DB()
		ctx, cancel := context.WithTimeout(t.Context(), 15*time.Second)
		defer cancel()

		id, ns := ulid.Make().String(), ulid.Make().String()
		_, err := db.ExecContext(ctx, insertChannel, id, ns, false, nil, nil)
		require.NoError(t, err)

		// Given a V1 writer holding the Channel lock with a new uncommitted assignment.
		tx, err := db.BeginTx(ctx, nil)
		require.NoError(t, err)
		var workers sync.WaitGroup
		defer func() {
			_ = tx.Rollback()
			workers.Wait()
		}()

		var pid int
		err = tx.QueryRowContext(ctx, "SELECT pg_backend_pid() FROM notification_channels WHERE id = $1 FOR NO KEY UPDATE", id).Scan(&pid)
		require.NoError(t, err)

		ruleID := ulid.Make().String()
		_, err = tx.ExecContext(ctx, insertRule, ruleID, ns, "v1.billing.invoice.created", false, nil)
		require.NoError(t, err)

		_, err = tx.ExecContext(ctx, assign, id, ruleID)
		require.NoError(t, err)

		// When the backfill waits on that Channel, let the assignment commit.
		var backfillErr error
		workers.Go(func() { _, backfillErr = db.ExecContext(ctx, string(up)) })
		require.EventuallyWithT(t, func(c *assert.CollectT) {
			var waiting bool
			err := db.QueryRowContext(ctx, `SELECT EXISTS (SELECT 1 FROM pg_stat_activity
				WHERE datname = current_database() AND $1 = ANY(pg_blocking_pids(pid)))`, pid).Scan(&waiting)
			require.NoError(c, err)
			require.True(c, waiting)
		}, 5*time.Second, 20*time.Millisecond)

		require.NoError(t, tx.Commit())
		workers.Wait()
		require.NoError(t, backfillErr)

		// Then the backfill uses the committed Rule rather than a stale empty set.
		var result string
		err = db.QueryRowContext(ctx, "SELECT event_type_filter FROM notification_channels WHERE id = $1", id).Scan(&result)
		require.NoError(t, err)
		require.JSONEq(t, `["v1.billing.invoice.created"]`, result)
	})
}
