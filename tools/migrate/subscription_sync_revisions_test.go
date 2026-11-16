package migrate_test

import (
	"database/sql"
	"testing"
	"time"

	"github.com/oklog/ulid/v2"
	"github.com/samber/lo"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/testutils"
)

func TestBackfillSubscriptionSyncRevisions(t *testing.T) {
	// Given all combinations of missing revisions, using the real SQL ULID function.
	testDB := testutils.InitPostgresDB(t, testutils.PostgresDBStateAtlasMigrated)
	defer testDB.PGDriver.Close()

	ctx := t.Context()
	conn, err := testDB.PGDriver.DB().Conn(ctx)
	require.NoError(t, err)
	defer conn.Close()

	// Shadow the production table to test the data migration without unrelated fixtures.
	_, err = conn.ExecContext(ctx, `
		CREATE TEMP TABLE subscription_sync_states_v2 (
			id text PRIMARY KEY,
			last_synced_subs_revision character(26) NULL,
			current_subs_revision character(26) NULL,
			last_success_state text NOT NULL DEFAULT 'subscription_deleted',
			last_success_at timestamptz NOT NULL DEFAULT '2026-11-01T00:00:00Z',
			updated_at timestamptz NOT NULL DEFAULT '2026-11-01T00:00:00Z'
		)
	`)
	require.NoError(t, err)

	lastSyncedRevision := ulid.Make().String()
	currentRevision := ulid.Make().String()
	_, err = conn.ExecContext(ctx, `
		INSERT INTO subscription_sync_states_v2 (id, last_synced_subs_revision, current_subs_revision)
		VALUES
			('missing-both', NULL, NULL),
			('another-missing-both', NULL, NULL),
			('missing-last-synced', NULL, $2),
			('missing-current', $1, NULL),
			('complete', $1, $2),
			('equal', $2, $2)
	`, lastSyncedRevision, currentRevision)
	require.NoError(t, err)
	before := readSubscriptionSyncRevisions(t, conn)

	// When the backfill runs, it fills NULLs without recording another cleanup.
	up := readMigration(t, "20261116101155_backfill_subscription_sync_revisions.up.sql")
	_, err = conn.ExecContext(ctx, up)
	require.NoError(t, err)
	after := readSubscriptionSyncRevisions(t, conn)

	// Then every row has revisions and its successful cleanup metadata is unchanged.
	require.Len(t, after, len(before))
	for id, state := range after {
		require.NotNil(t, state.lastSynced, id)
		require.NotNil(t, state.current, id)
		require.Len(t, *state.lastSynced, 26, id)
		require.Len(t, *state.current, 26, id)
		require.Equal(t, before[id].lastSuccessState, state.lastSuccessState, id)
		require.Equal(t, before[id].lastSuccessAt, state.lastSuccessAt, id)
		require.Equal(t, before[id].updatedAt, state.updatedAt, id)
	}

	// Empty rows share one generated revision within each row, not across rows.
	require.Equal(t, after["missing-both"].lastSynced, after["missing-both"].current)
	require.Equal(t, after["another-missing-both"].lastSynced, after["another-missing-both"].current)
	require.NotEqual(t, after["missing-both"].current, after["another-missing-both"].current)

	// Filling one missing revision must not acknowledge or replace the existing revision.
	require.Equal(t, lo.ToPtr(currentRevision), after["missing-last-synced"].current)
	require.NotEqual(t, after["missing-last-synced"].current, after["missing-last-synced"].lastSynced)
	require.Equal(t, lo.ToPtr(lastSyncedRevision), after["missing-current"].lastSynced)
	require.NotEqual(t, after["missing-current"].lastSynced, after["missing-current"].current)
	require.Equal(t, before["complete"], after["complete"])
	require.Equal(t, before["equal"], after["equal"])

	// Reapplying the backfill must not generate new revisions for completed rows.
	_, err = conn.ExecContext(ctx, up)
	require.NoError(t, err)
	require.Equal(t, after, readSubscriptionSyncRevisions(t, conn))

	// Once existing rows are backfilled, the next migration forbids future NULL revisions.
	_, err = conn.ExecContext(ctx, readMigration(t, "20261116104138_require_subscription_sync_revisions.up.sql"))
	require.NoError(t, err)

	t.Run("last synced revision is required without a default", func(t *testing.T) {
		_, err := conn.ExecContext(t.Context(), `
			INSERT INTO subscription_sync_states_v2 (id, current_subs_revision)
			VALUES ('new-without-last', $1)
		`, currentRevision)
		require.ErrorContains(t, err, `null value in column "last_synced_subs_revision"`)
	})

	t.Run("current revision is required without a default", func(t *testing.T) {
		_, err := conn.ExecContext(t.Context(), `
			INSERT INTO subscription_sync_states_v2 (id, last_synced_subs_revision)
			VALUES ('new-without-current', $1)
		`, lastSyncedRevision)
		require.ErrorContains(t, err, `null value in column "current_subs_revision"`)
	})

	// Downgrading restores nullable columns without erasing any revisions.
	_, err = conn.ExecContext(ctx, readMigration(t, "20261116104138_require_subscription_sync_revisions.down.sql"))
	require.NoError(t, err)
	require.Equal(t, after, readSubscriptionSyncRevisions(t, conn))
	_, err = conn.ExecContext(ctx, `INSERT INTO subscription_sync_states_v2 (id) VALUES ('nullable-again')`)
	require.NoError(t, err)
}

type subscriptionSyncRevisionRow struct {
	lastSynced       *string
	current          *string
	lastSuccessState string
	lastSuccessAt    time.Time
	updatedAt        time.Time
}

func readSubscriptionSyncRevisions(t *testing.T, conn *sql.Conn) map[string]subscriptionSyncRevisionRow {
	t.Helper()

	rows, err := conn.QueryContext(t.Context(), `
		SELECT id, last_synced_subs_revision, current_subs_revision,
		       last_success_state, last_success_at, updated_at
		FROM subscription_sync_states_v2
	`)
	require.NoError(t, err)
	defer rows.Close()

	states := make(map[string]subscriptionSyncRevisionRow)
	for rows.Next() {
		var id string
		var state subscriptionSyncRevisionRow
		require.NoError(t, rows.Scan(&id, &state.lastSynced, &state.current,
			&state.lastSuccessState, &state.lastSuccessAt, &state.updatedAt))
		states[id] = state
	}

	require.NoError(t, rows.Err())

	return states
}
