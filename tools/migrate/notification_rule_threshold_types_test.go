package migrate_test

import (
	"io/fs"
	"path"
	"testing"
	"time"

	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/testutils"
	"github.com/openmeterio/openmeter/tools/migrate"
)

func TestNotificationRuleThresholdTypesMigration(t *testing.T) {
	up, err := fs.ReadFile(migrate.OMMigrationsConfig.FS, path.Join(migrate.OMMigrationsConfig.FSPath, "20261126124038_notification_rule_threshold_types.up.sql"))
	require.NoError(t, err)

	const (
		mixed     = `{"features":["feature-id"],"thresholds":[{"type":"PERCENT","value":90,"extra":true},{"type":"balance_value","value":0},{"type":"NUMBER","value":7},{"type":"NUMBER","value":7}]}`
		canonical = `{"features":["feature-id"],"thresholds":[{"type":"usage_percentage","value":90,"extra":true},{"type":"balance_value","value":0},{"type":"usage_value","value":7},{"type":"usage_value","value":7}]}`
	)

	tests := []struct {
		name      string
		eventType string
		stored    string
		want      string
		deleted   bool
		changed   int64
	}{
		{
			name:      "active Rule with mixed and repeated thresholds",
			eventType: "v1.entitlement.balance_threshold.triggered",
			stored:    mixed,
			want:      canonical,
			changed:   1,
		},
		{
			name:      "deleted Rule with historical Event type",
			eventType: "entitlements.balance.threshold",
			stored:    mixed,
			want:      canonical,
			deleted:   true,
			changed:   1,
		},
		{
			name:      "already canonical Rule",
			eventType: "v1.entitlement.balance_threshold.triggered",
			stored:    canonical,
			want:      canonical,
		},
		{
			name:      "different Event type",
			eventType: "v1.billing.invoice.created",
			stored:    mixed,
			want:      mixed,
		},
	}

	testDB := testutils.InitPostgresDB(t, testutils.PostgresDBStateEntMigrated)
	t.Cleanup(func() { testDB.Close(t) })
	db := testDB.PGDriver.DB()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given a stored Rule with a known revision and modification time.
			ctx := t.Context()
			id := ulid.Make().String()
			var deletedAt *time.Time
			if tt.deleted {
				deletedAt = new(time.Now())
			}

			var updatedAt time.Time
			err := db.QueryRowContext(ctx, `INSERT INTO notification_rules
				(id, namespace, type, name, config, revision, created_at, updated_at, deleted_at)
				VALUES ($1, $2, $3, 'test', $4, 7, NOW(), NOW(), $5) RETURNING updated_at`,
				id, ulid.Make().String(), tt.eventType, tt.stored, deletedAt).Scan(&updatedAt)
			require.NoError(t, err)

			// When the migration runs again, the previously converted rows are skipped.
			for _, expectedRows := range []int64{tt.changed, 0} {
				result, err := db.ExecContext(ctx, string(up))
				require.NoError(t, err)

				rows, err := result.RowsAffected()
				require.NoError(t, err)
				require.Equal(t, expectedRows, rows)
			}

			// Then only the threshold spellings changed.
			var config string
			var revision int
			var storedUpdatedAt time.Time
			err = db.QueryRowContext(ctx, "SELECT config, revision, updated_at FROM notification_rules WHERE id = $1", id).
				Scan(&config, &revision, &storedUpdatedAt)
			require.NoError(t, err)

			require.JSONEq(t, tt.want, config)
			require.Equal(t, 7, revision)
			require.Equal(t, updatedAt, storedUpdatedAt)
		})
	}
}
