package migrate_test

import (
	"database/sql"
	"encoding/json"
	"io/fs"
	"os"
	"path"
	"strings"
	"testing"

	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/require"

	apievents "github.com/openmeterio/openmeter/api/v3/handlers/notification/events"
	"github.com/openmeterio/openmeter/openmeter/notification"
	notificationadapter "github.com/openmeterio/openmeter/openmeter/notification/adapter"
	"github.com/openmeterio/openmeter/openmeter/notification/httpdriver"
	"github.com/openmeterio/openmeter/openmeter/testutils"
	"github.com/openmeterio/openmeter/tools/migrate"
)

func TestNotificationEventPayloadSnapshotsMigration(t *testing.T) {
	up, err := fs.ReadFile(migrate.OMMigrationsConfig.FS, path.Join(migrate.OMMigrationsConfig.FSPath, "20261126153057_notification_event_payload_snapshots.up.sql"))
	require.NoError(t, err)

	preflight, err := os.ReadFile("notification_event_payload_snapshots_preflight.sql")
	require.NoError(t, err)

	fixturesJSON, err := os.ReadFile("../../api/v3/handlers/notification/events/testdata/legacy_payloads.json")
	require.NoError(t, err)

	var fixtures map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(fixturesJSON, &fixtures))

	testDB := testutils.InitPostgresDB(t, testutils.PostgresDBStateEntMigrated)
	t.Cleanup(func() { testDB.Close(t) })
	db := testDB.PGDriver.DB()
	adapter, err := notificationadapter.New(notificationadapter.Config{
		Client: testDB.EntDriver.Client(), Logger: testutils.NewDiscardLogger(t),
	})
	require.NoError(t, err)

	cases := []struct {
		legacyType    notification.EventType
		canonicalType string
		prefix        string
		wrapper       string
	}{
		{legacyType: notification.EventTypeInvoiceCreated, canonicalType: "v1.billing.invoice.created", prefix: "invoice_", wrapper: "invoice"},
		{legacyType: notification.EventTypeInvoiceUpdated, canonicalType: "v1.billing.invoice.updated", prefix: "invoice_", wrapper: "invoice"},
		{legacyType: notification.EventTypeBalanceThreshold, canonicalType: "v1.entitlement.balance_threshold.triggered", prefix: "balance_threshold_", wrapper: "balanceThreshold"},
		{legacyType: notification.EventTypeEntitlementReset, canonicalType: "v1.entitlement.usage_period.reset", prefix: "entitlement_reset_", wrapper: "entitlementReset"},
	}

	for _, tc := range cases {
		for name, raw := range fixtures {
			if !strings.HasPrefix(name, tc.prefix) {
				continue
			}

			for _, eventType := range []string{string(tc.legacyType), tc.canonicalType} {
				t.Run(eventType+"/"+name, func(t *testing.T) {
					// Given a historical snapshot, including one whose Rule is now deleted.
					namespace, ruleID, eventID := ulid.Make().String(), ulid.Make().String(), ulid.Make().String()
					_, err := db.ExecContext(t.Context(), `INSERT INTO notification_rules
      (id, namespace, type, name, config, created_at, updated_at, deleted_at)
      VALUES ($1, $2, $3, 'historical', '{}', NOW(), NOW(), NOW())`, ruleID, namespace, eventType)
					require.NoError(t, err)

					envelope, err := json.Marshal(map[string]any{"type": tc.legacyType, "version": 1, tc.wrapper: raw})
					require.NoError(t, err)

					// Historical writers serialized the typed V1 model, including union normalization.
					var legacy notification.EventPayload
					require.NoError(t, json.Unmarshal(envelope, &legacy))

					envelope, err = json.Marshal(legacy)
					require.NoError(t, err)

					_, err = db.ExecContext(t.Context(), `INSERT INTO notification_events
      (id, namespace, type, rule_id, payload, created_at, annotations, rule_snapshot, provider_data)
      VALUES ($1, $2, $3, $4, $5, NOW(), '{"preserve":true}', '{"preserve":true}', '{"svix.message_id":"message"}')`, eventID, namespace, eventType, ruleID, string(envelope))
					require.NoError(t, err)

					before, err := adapter.GetEvent(t.Context(), notification.GetEventInput{Namespace: namespace, ID: eventID})
					require.NoError(t, err)

					beforeV1, err := httpdriver.FromEvent(*before)
					require.NoError(t, err)

					beforeV3, err := apievents.ToAPIBillingNotificationEvent(*before)
					require.NoError(t, err)

					wireJSON, err := json.Marshal(beforeV3.Payload)
					require.NoError(t, err)

					var expected struct {
						Data json.RawMessage `json:"data"`
					}
					require.NoError(t, json.Unmarshal(wireJSON, &expected))

					var attributesBefore string
					require.NoError(t, db.QueryRowContext(t.Context(), `SELECT to_jsonb(e) - 'payload' FROM notification_events e WHERE id=$1`, eventID).Scan(&attributesBefore))

					// When the backfill converts the persisted snapshot, it never consults live domain data.
					_, err = db.ExecContext(t.Context(), string(up))
					require.NoError(t, err)

					var stored, attributesAfter, xmin string
					require.NoError(t, db.QueryRowContext(t.Context(), `SELECT payload, to_jsonb(e) - 'payload', xmin::text FROM notification_events e WHERE id=$1`, eventID).Scan(&stored, &attributesAfter, &xmin))

					require.JSONEq(t, string(expected.Data), stored)
					require.JSONEq(t, attributesBefore, attributesAfter)

					after, err := adapter.GetEvent(t.Context(), notification.GetEventInput{Namespace: namespace, ID: eventID})
					require.NoError(t, err)

					afterV1, err := httpdriver.FromEvent(*after)
					require.NoError(t, err)

					// Then V1 responses (and their webhook payloads) retain every historical value.
					beforeJSON, err := json.Marshal(beforeV1)
					require.NoError(t, err)

					afterJSON, err := json.Marshal(afterV1)
					require.NoError(t, err)

					require.JSONEq(t, string(beforeJSON), string(afterJSON))

					_, err = db.ExecContext(t.Context(), string(up))
					require.NoError(t, err)

					var afterXmin string
					require.NoError(t, db.QueryRowContext(t.Context(), `SELECT xmin::text FROM notification_events WHERE id=$1`, eventID).Scan(&afterXmin))

					require.Equal(t, xmin, afterXmin, "reruns must not rewrite canonical snapshots")
				})
			}
		}
	}

	t.Run("preserves opaque data and skips unknown Event types", func(t *testing.T) {
		// Given exact JSON numbers and user keys, conversion must not interpret their values.
		namespace, eventID, unknownID := ulid.Make().String(), ulid.Make().String(), ulid.Make().String()
		envelope := `{"type":"invoice.created","version":1,"invoice":{"id":"i","annotations":{"User_Key":{"largeNumber":9007199254740993,"preciseNumber":0.1234567890123456789}},"metadata":{"User_Key":"v"},"lines":[],"customField":null}}`
		_, err := db.ExecContext(t.Context(), `INSERT INTO notification_events (id, namespace, type, payload, created_at) VALUES
   ($1, $2, 'v1.billing.invoice.created', $3, NOW()), ($4, $2, 'v1.future.event', $3, NOW())`, eventID, namespace, envelope, unknownID)
		require.NoError(t, err)

		// When known snapshots are converted, unrelated contracts are left alone.
		_, err = db.ExecContext(t.Context(), string(up))
		require.NoError(t, err)

		var large, precise, unknown string
		require.NoError(t, db.QueryRowContext(t.Context(), `SELECT payload #>> '{annotations,User_Key,largeNumber}', payload #>> '{annotations,User_Key,preciseNumber}' FROM notification_events WHERE id=$1`, eventID).Scan(&large, &precise))

		require.Equal(t, "9007199254740993", large)
		require.Equal(t, "0.1234567890123456789", precise)

		require.NoError(t, db.QueryRowContext(t.Context(), `SELECT payload FROM notification_events WHERE id=$1`, unknownID).Scan(&unknown))

		require.JSONEq(t, envelope, unknown)
	})

	t.Run("preflight detects invalid envelopes and the migration rolls back", func(t *testing.T) {
		tests := []struct{ name, payload, problem string }{
			{name: "unsupported version", payload: `{"type":"invoice.created","version":99,"invoice":{"id":"i"}}`, problem: "unsupported invoice envelope version"},
			{name: "wrong type", payload: `{"type":"invoice.updated","version":1,"invoice":{"id":"i"}}`, problem: "envelope type differs"},
			{name: "unknown envelope field", payload: `{"type":"invoice.created","version":1,"invoice":{"id":"i"},"extension":true}`, problem: "unexpected envelope fields"},
			{name: "invalid snapshot", payload: `{"type":"invoice.created","version":1,"invoice":[]}`, problem: "invalid envelope snapshot"},
			{name: "missing bare fields", payload: `{}`, problem: "missing required fields"},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				// Given valid and invalid rows, no valid row may be partially migrated.
				namespace, validID, invalidID := ulid.Make().String(), ulid.Make().String(), ulid.Make().String()
				valid := `{"type":"invoice.created","version":1,"invoice":{"id":"i","metadata":{"User_Key":"value"}}}`
				for id, payload := range map[string]string{validID: valid, invalidID: tt.payload} {
					_, err := db.ExecContext(t.Context(), `INSERT INTO notification_events (id, namespace, type, payload, created_at) VALUES ($1, $2, 'v1.billing.invoice.created', $3, NOW())`, id, namespace, payload)
					require.NoError(t, err)
				}

				rows, err := db.QueryContext(t.Context(), string(preflight))
				require.NoError(t, err)

				found := false
				for rows.Next() {
					var ns, id, eventType, status string
					var problem sql.NullString
					require.NoError(t, rows.Scan(&ns, &id, &eventType, &status, &problem))

					if id == invalidID {
						found = true
						require.Equal(t, "invalid", status)
						require.Contains(t, problem.String, tt.problem)
					}
				}

				require.NoError(t, rows.Err())
				require.NoError(t, rows.Close())
				require.True(t, found)

				// When invalid data is encountered, the whole backfill fails without data loss.
				_, err = db.ExecContext(t.Context(), string(up))
				require.ErrorContains(t, err, tt.problem)

				var stored string
				require.NoError(t, db.QueryRowContext(t.Context(), `SELECT payload FROM notification_events WHERE id=$1`, validID).Scan(&stored))

				require.JSONEq(t, valid, stored)

				_, err = db.ExecContext(t.Context(), `DELETE FROM notification_events WHERE namespace=$1`, namespace)
				require.NoError(t, err)
			})
		}
	})
}
