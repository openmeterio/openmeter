package adapter_test

import (
	"database/sql"
	"encoding/json"
	"io/fs"
	"math"
	"path"
	"strings"
	"testing"
	"time"

	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	statusdb "github.com/openmeterio/openmeter/openmeter/ent/db/notificationeventdeliverystatus"
	"github.com/openmeterio/openmeter/openmeter/notification"
	notificationadapter "github.com/openmeterio/openmeter/openmeter/notification/adapter"
	"github.com/openmeterio/openmeter/openmeter/testutils"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/pagination"
	"github.com/openmeterio/openmeter/tools/migrate"
)

func TestDeliveryStatusStateStorageCompatibility(t *testing.T) {
	env := newEventTestEnv(t)
	tests := []struct {
		name  string
		state notification.EventDeliveryStatusState
	}{
		{name: "PENDING", state: notification.EventDeliveryStatusStatePending},
		{name: "SENDING", state: notification.EventDeliveryStatusStateSending},
		{name: "RESENDING", state: notification.EventDeliveryStatusStateResending},
		{name: "SUCCESS", state: notification.EventDeliveryStatusStateSuccess},
		{name: "FAILED", state: notification.EventDeliveryStatusStateFailed},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// given: a delivery and another delivery that must not match its state filter
			namespace := ulid.Make().String()
			channel := seedChannel(t, env, namespace)
			rule := seedRule(t, env, namespace, notification.EventTypeBalanceThreshold, channel.ID)
			event := seedEvent(t, env, namespace, rule, clock.Now(), nil)
			other := seedEvent(t, env, namespace, rule, clock.Now(), nil)
			require.Len(t, event.DeliveryStatus, 1)
			require.Len(t, other.DeliveryStatus, 1)

			if tt.state == notification.EventDeliveryStatusStatePending {
				_, err := env.adapter.UpdateEventDeliveryStatus(t.Context(), notification.UpdateEventDeliveryStatusInput{
					NamespacedID: other.DeliveryStatus[0].NamespacedID,
					State:        notification.EventDeliveryStatusStateFailed,
				})
				require.NoError(t, err)
			}

			at := time.Date(2026, time.November, 4, 10, 0, 0, 0, time.UTC)
			nextAttempt := at.Add(time.Minute)
			attempts := []notification.EventDeliveryAttempt{
				{
					State:     tt.state,
					Timestamp: at,
					Response: notification.EventDeliveryAttemptResponse{
						Body:     "provider response",
						Duration: 250 * time.Millisecond,
					},
				},
			}

			// when: the existing adapter updates state and its accompanying delivery data
			updated, err := env.adapter.UpdateEventDeliveryStatus(t.Context(), notification.UpdateEventDeliveryStatusInput{
				NamespacedID: event.DeliveryStatus[0].NamespacedID,
				State:        tt.state,
				Reason:       "observed delivery",
				NextAttempt:  &nextAttempt,
				Attempts:     attempts,
			})
			require.NoError(t, err)

			var storedState, storedAttempts string
			err = env.sqlDB.QueryRowContext(t.Context(),
				"SELECT state, attempts FROM notification_event_delivery_status WHERE id = $1", updated.ID,
			).Scan(&storedState, &storedAttempts)
			require.NoError(t, err)

			expectedAttempts, err := json.Marshal(attempts)
			require.NoError(t, err)

			assert.Equal(t, tt.state, updated.State)
			assert.Equal(t, tt.name, storedState)
			assert.JSONEq(t, string(expectedAttempts), storedAttempts)

			for _, stored := range []string{tt.name, strings.ToLower(tt.name)} {
				t.Run(stored, func(t *testing.T) {
					// given: the same report in either supported storage format
					_, err := env.sqlDB.ExecContext(t.Context(),
						"UPDATE notification_event_delivery_status SET state = $1 WHERE id = $2", stored, updated.ID,
					)
					require.NoError(t, err)

					fetched, err := env.adapter.GetEventDeliveryStatus(t.Context(), notification.GetEventDeliveryStatusInput{
						NamespacedID: updated.NamespacedID,
					})
					require.NoError(t, err)

					listed, err := env.adapter.ListEventsDeliveryStatus(t.Context(), notification.ListEventsDeliveryStatusInput{
						Namespaces: []string{namespace},
						States:     []notification.EventDeliveryStatusState{tt.state},
						Page:       pagination.NewPage(1, 20),
					})
					require.NoError(t, err)
					require.Len(t, listed.Items, 1)

					// then: the adapter returns the legacy domain state for either stored spelling
					assert.Equal(t, tt.state, fetched.State)
					assert.Equal(t, tt.state, listed.Items[0].State)
					assert.Equal(t, updated.ID, listed.Items[0].ID)
					assert.Equal(t, event.ID, fetched.EventID)
					assert.Equal(t, channel.ID, fetched.ChannelID)
					assert.Equal(t, "observed delivery", fetched.Reason)
					assert.Equal(t, &nextAttempt, fetched.NextAttempt)
					assert.Equal(t, attempts, fetched.Attempts)

					var persisted string
					err = env.sqlDB.QueryRowContext(t.Context(),
						"SELECT state FROM notification_event_delivery_status WHERE id = $1", updated.ID,
					).Scan(&persisted)
					require.NoError(t, err)

					assert.Equal(t, stored, persisted)
				})
			}
		})
	}
}

func TestDeliveryStatusStateDefault(t *testing.T) {
	env := newEventTestEnv(t)
	for _, writer := range []string{"Ent", "SQL"} {
		t.Run(writer, func(t *testing.T) {
			// given: normal Event creation still creates an uppercase PENDING delivery
			namespace := ulid.Make().String()
			channel := seedChannel(t, env, namespace)
			rule := seedRule(t, env, namespace, notification.EventTypeBalanceThreshold, channel.ID)
			event := seedEvent(t, env, namespace, rule, clock.Now(), nil)
			require.Len(t, event.DeliveryStatus, 1)

			assert.Equal(t, notification.EventDeliveryStatusStatePending, event.DeliveryStatus[0].State)

			otherChannel := seedChannel(t, env, namespace)
			id := ulid.Make().String()

			// when: a writer omits state entirely, exercising both Ent and database defaults
			switch writer {
			case "Ent":
				_, err := env.db.NotificationEventDeliveryStatus.Create().
					SetID(id).
					SetNamespace(namespace).
					SetEventID(event.ID).
					SetChannelID(otherChannel.ID).
					AddEventIDs(event.ID).
					Save(t.Context())
				require.NoError(t, err)
			case "SQL":
				_, err := env.sqlDB.ExecContext(t.Context(),
					`INSERT INTO notification_event_delivery_status (id, namespace, event_id, channel_id, created_at, updated_at)
					VALUES ($1, $2, $3, $4, NOW(), NOW())`, id, namespace, event.ID, otherChannel.ID,
				)
				require.NoError(t, err)
			}

			// then: both paths retain the same uppercase default
			var storedState string
			err := env.sqlDB.QueryRowContext(t.Context(),
				"SELECT state FROM notification_event_delivery_status WHERE id = $1", id,
			).Scan(&storedState)
			require.NoError(t, err)

			fetched, err := env.adapter.GetEventDeliveryStatus(t.Context(), notification.GetEventDeliveryStatusInput{
				NamespacedID: models.NamespacedID{Namespace: namespace, ID: id},
			})
			require.NoError(t, err)

			assert.Equal(t, "PENDING", storedState)
			assert.Equal(t, notification.EventDeliveryStatusStatePending, fetched.State)
		})
	}
}

func TestDeliveryStatusStateValidation(t *testing.T) {
	env := newEventTestEnv(t)
	tests := []struct {
		name  string
		state notification.EventDeliveryStatusState
	}{
		{name: "empty"},
		{name: "unknown", state: "UNKNOWN"},
		{name: "lowercase pending", state: "pending"},
		{name: "lowercase sending", state: "sending"},
		{name: "lowercase resending", state: "resending"},
		{name: "lowercase success", state: "success"},
		{name: "target succeeded", state: "succeeded"},
		{name: "lowercase failed", state: "failed"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// given: an existing valid delivery with an unsupported replacement state
			namespace := ulid.Make().String()
			channel := seedChannel(t, env, namespace)
			rule := seedRule(t, env, namespace, notification.EventTypeBalanceThreshold, channel.ID)
			event := seedEvent(t, env, namespace, rule, clock.Now(), nil)
			require.Len(t, event.DeliveryStatus, 1)

			before, err := env.adapter.GetEventDeliveryStatus(t.Context(), notification.GetEventDeliveryStatusInput{
				NamespacedID: event.DeliveryStatus[0].NamespacedID,
			})
			require.NoError(t, err)

			// when: writing through Ent or the adapter without service validation
			_, err = env.db.NotificationEventDeliveryStatus.Create().
				SetNamespace(namespace).
				SetEventID(event.ID).
				SetChannelID(channel.ID).
				SetState(tt.state).
				Save(t.Context())
			require.ErrorContains(t, err, "invalid event delivery status state")

			_, err = env.adapter.UpdateEventDeliveryStatus(t.Context(), notification.UpdateEventDeliveryStatusInput{
				NamespacedID: before.NamespacedID,
				State:        tt.state,
				Reason:       "must not be persisted",
			})
			require.ErrorContains(t, err, "invalid event delivery status state")

			_, err = env.db.NotificationEventDeliveryStatus.Update().
				Where(statusdb.ID(before.ID)).
				SetState(tt.state).
				Save(t.Context())
			require.ErrorContains(t, err, "invalid event delivery status state")

			// then: validation prevents both an extra row and mutation of the existing one
			count, err := env.db.NotificationEventDeliveryStatus.Query().Where(statusdb.Namespace(namespace)).Count(t.Context())
			require.NoError(t, err)

			after, err := env.adapter.GetEventDeliveryStatus(t.Context(), notification.GetEventDeliveryStatusInput{
				NamespacedID: before.NamespacedID,
			})
			require.NoError(t, err)

			assert.Equal(t, 1, count)
			assert.Equal(t, before, after)
		})
	}
}

func TestDeliveryStatusAttemptsStorageCompatibility(t *testing.T) {
	env := newEventTestEnv(t)
	at := time.Date(2026, time.November, 4, 10, 0, 0, 0, time.UTC)
	statusCode := 503
	url := "https://example.com/hook"
	tests := []struct {
		name     string
		stored   any
		attempts []notification.EventDeliveryAttempt
		updated  string
	}{
		{
			name:    "SQL NULL",
			updated: "null",
		},
		{
			name:    "JSON null",
			stored:  "null",
			updated: "null",
		},
		{
			name:     "empty array",
			stored:   "[]",
			attempts: []notification.EventDeliveryAttempt{},
			updated:  "[]",
		},
		{
			name: "legacy response fields and duration units",
			stored: `[
				{"state":"FAILED","timestamp":"2026-11-04T10:00:00Z","response":{"status_code":503,"body":"retry later","duration":250000001,"url":"https://example.com/hook"}},
				{"state":"SUCCESS","timestamp":"2026-11-04T10:01:00Z","response":{}}
			]`,
			attempts: []notification.EventDeliveryAttempt{
				{
					State:     notification.EventDeliveryStatusStateFailed,
					Timestamp: at,
					Response: notification.EventDeliveryAttemptResponse{
						StatusCode: &statusCode,
						Body:       "retry later",
						Duration:   250*time.Millisecond + time.Nanosecond,
						URL:        &url,
					},
				},
				{
					State:     notification.EventDeliveryStatusStateSuccess,
					Timestamp: at.Add(time.Minute),
				},
			},
			updated: `[
				{"state":"FAILED","timestamp":"2026-11-04T10:00:00Z","response":{"status_code":503,"body":"retry later","duration":250000001,"url":"https://example.com/hook"}},
				{"state":"SUCCESS","timestamp":"2026-11-04T10:01:00Z","response":{}}
			]`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// given: a delivery containing the JSON representation written by legacy backends
			namespace := ulid.Make().String()
			channel := seedChannel(t, env, namespace)
			rule := seedRule(t, env, namespace, notification.EventTypeBalanceThreshold, channel.ID)
			event := seedEvent(t, env, namespace, rule, at, nil)
			require.Len(t, event.DeliveryStatus, 1)

			id := event.DeliveryStatus[0].NamespacedID
			_, err := env.sqlDB.ExecContext(t.Context(),
				"UPDATE notification_event_delivery_status SET attempts = $1 WHERE id = $2", tt.stored, id.ID,
			)
			require.NoError(t, err)

			// when: both direct delivery reads and Event reads load the attempt history
			fetched, err := env.adapter.GetEventDeliveryStatus(t.Context(), notification.GetEventDeliveryStatusInput{NamespacedID: id})
			require.NoError(t, err)

			listed, err := env.adapter.ListEventsDeliveryStatus(t.Context(), notification.ListEventsDeliveryStatusInput{
				Namespaces: []string{namespace},
				Page:       pagination.NewPage(1, 20),
			})
			require.NoError(t, err)
			require.Len(t, listed.Items, 1)

			fetchedEvent, err := env.adapter.GetEvent(t.Context(), notification.GetEventInput(event.NamespacedID))
			require.NoError(t, err)
			require.Len(t, fetchedEvent.DeliveryStatus, 1)

			listedEvents, err := env.adapter.ListEvents(t.Context(), notification.ListEventsInput{
				Namespaces: []string{namespace},
				Page:       pagination.NewPage(1, 20),
			})
			require.NoError(t, err)
			require.Len(t, listedEvents.Items, 1)
			require.Len(t, listedEvents.Items[0].DeliveryStatus, 1)

			// then: reads preserve order, response fields, nil/empty slices and stored JSON
			assert.Equal(t, tt.attempts, fetched.Attempts)
			assert.Equal(t, tt.attempts, listed.Items[0].Attempts)
			assert.Equal(t, tt.attempts, fetchedEvent.DeliveryStatus[0].Attempts)
			assert.Equal(t, tt.attempts, listedEvents.Items[0].DeliveryStatus[0].Attempts)

			var stored *string
			err = env.sqlDB.QueryRowContext(t.Context(),
				"SELECT attempts FROM notification_event_delivery_status WHERE id = $1", id.ID,
			).Scan(&stored)
			require.NoError(t, err)

			if tt.stored == nil {
				assert.Nil(t, stored)
			} else {
				require.NotNil(t, stored)

				assert.JSONEq(t, tt.stored.(string), *stored)
			}

			updated, err := env.adapter.UpdateEventDeliveryStatus(t.Context(), notification.UpdateEventDeliveryStatusInput{
				NamespacedID: id,
				State:        notification.EventDeliveryStatusStateSuccess,
				Attempts:     tt.attempts,
			})
			require.NoError(t, err)

			assert.Equal(t, tt.attempts, updated.Attempts)

			err = env.sqlDB.QueryRowContext(t.Context(),
				"SELECT attempts FROM notification_event_delivery_status WHERE id = $1", id.ID,
			).Scan(&stored)
			require.NoError(t, err)
			require.NotNil(t, stored)

			assert.JSONEq(t, tt.updated, *stored)
		})
	}
}

func TestDeliveryStatusInvalidAttempts(t *testing.T) {
	env := newEventTestEnv(t)
	tests := []struct {
		name string
		json string
	}{
		{name: "object instead of array", json: `{}`},
		{name: "double encoded array", json: `"[]"`},
		{name: "invalid response", json: `[{"response":"invalid"}]`},
		{name: "invalid duration", json: `[{"response":{"duration":"250ms"}}]`},
		{name: "invalid timestamp", json: `[{"timestamp":"invalid"}]`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// given: valid JSONB that cannot be decoded as legacy attempts
			namespace := ulid.Make().String()
			channel := seedChannel(t, env, namespace)
			rule := seedRule(t, env, namespace, notification.EventTypeBalanceThreshold, channel.ID)
			event := seedEvent(t, env, namespace, rule, clock.Now(), nil)
			require.Len(t, event.DeliveryStatus, 1)

			id := event.DeliveryStatus[0].NamespacedID
			_, err := env.sqlDB.ExecContext(t.Context(),
				"UPDATE notification_event_delivery_status SET attempts = $1 WHERE id = $2", tt.json, id.ID,
			)
			require.NoError(t, err)

			// when/then: every read path reports the invalid delivery rather than hiding its history
			_, err = env.adapter.GetEventDeliveryStatus(t.Context(), notification.GetEventDeliveryStatusInput{NamespacedID: id})
			require.ErrorContains(t, err, "invalid notification event delivery attempts")
			assert.ErrorContains(t, err, id.ID)

			_, err = env.adapter.ListEventsDeliveryStatus(t.Context(), notification.ListEventsDeliveryStatusInput{
				Namespaces: []string{namespace},
				Page:       pagination.NewPage(1, 20),
			})
			require.ErrorContains(t, err, "invalid notification event delivery attempts")

			_, err = env.adapter.GetEvent(t.Context(), notification.GetEventInput(event.NamespacedID))
			require.ErrorContains(t, err, "invalid notification event delivery attempts")

			_, err = env.adapter.ListEvents(t.Context(), notification.ListEventsInput{
				Namespaces: []string{namespace},
				Page:       pagination.NewPage(1, 20),
			})
			require.ErrorContains(t, err, "invalid notification event delivery attempts")

			var stored string
			err = env.sqlDB.QueryRowContext(t.Context(),
				"SELECT attempts FROM notification_event_delivery_status WHERE id = $1", id.ID,
			).Scan(&stored)
			require.NoError(t, err)

			assert.JSONEq(t, tt.json, stored)
		})
	}
}

func TestDeliveryStatusAttemptSerializationFailure(t *testing.T) {
	env := newEventTestEnv(t)
	tests := []struct {
		name      string
		timestamp time.Time
	}{
		{name: "year above JSON range", timestamp: time.Date(10000, time.January, 1, 0, 0, 0, 0, time.UTC)},
		{name: "negative year", timestamp: time.Date(-1, time.January, 1, 0, 0, 0, 0, time.UTC)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// given: an existing delivery whose state must survive serialization failure
			namespace := ulid.Make().String()
			channel := seedChannel(t, env, namespace)
			rule := seedRule(t, env, namespace, notification.EventTypeBalanceThreshold, channel.ID)
			event := seedEvent(t, env, namespace, rule, clock.Now(), nil)
			require.Len(t, event.DeliveryStatus, 1)

			before := event.DeliveryStatus[0]

			// when: the update includes an attempt that JSON cannot represent
			_, err := env.adapter.UpdateEventDeliveryStatus(t.Context(), notification.UpdateEventDeliveryStatusInput{
				NamespacedID: before.NamespacedID,
				State:        notification.EventDeliveryStatusStateFailed,
				Reason:       "must not persist",
				Attempts: []notification.EventDeliveryAttempt{
					{State: notification.EventDeliveryStatusStateFailed, Timestamp: tt.timestamp},
				},
			})
			require.ErrorContains(t, err, "failed to serialize notification event delivery attempts")

			// then: no part of the update reaches storage
			after, err := env.adapter.GetEventDeliveryStatus(t.Context(), notification.GetEventDeliveryStatusInput{NamespacedID: before.NamespacedID})
			require.NoError(t, err)

			assert.Equal(t, before, *after)
		})
	}
}

func TestDeliveryStatusProviderDataPreservation(t *testing.T) {
	schemas := []struct {
		name  string
		state testutils.PostgresDBState
	}{
		{name: "Ent schema", state: testutils.PostgresDBStateEntMigrated},
		{name: "SQL migrations", state: testutils.PostgresDBStateAtlasMigrated},
	}
	tests := []struct {
		name         string
		providerData map[string]string
	}{
		{name: "unset provider data"},
		{name: "empty provider data", providerData: map[string]string{}},
		{
			name: "populated provider data",
			providerData: map[string]string{
				"svix.message_id": "msg_existing",
				"other.entity_id": "other_existing",
			},
		},
	}

	for _, schema := range schemas {
		t.Run(schema.name, func(t *testing.T) {
			testdb := testutils.InitPostgresDB(t, schema.state)
			t.Cleanup(func() { testdb.Close(t) })

			dbClient := testdb.EntDriver.Client()
			adapter, err := notificationadapter.New(notificationadapter.Config{
				Client: dbClient,
				Logger: testutils.NewDiscardLogger(t),
			})
			require.NoError(t, err)

			env := &eventTestEnv{db: dbClient, sqlDB: testdb.PGDriver.DB(), adapter: adapter}

			var nullable, dataType string
			var defaultValue sql.NullString
			err = env.sqlDB.QueryRowContext(t.Context(),
				`SELECT is_nullable, data_type, column_default FROM information_schema.columns
				WHERE table_schema = current_schema() AND table_name = 'notification_event_delivery_status' AND column_name = 'provider_data'`,
			).Scan(&nullable, &dataType, &defaultValue)
			require.NoError(t, err)

			assert.Equal(t, "YES", nullable)
			assert.Equal(t, "jsonb", dataType)
			assert.False(t, defaultValue.Valid)

			for _, tt := range tests {
				t.Run(tt.name, func(t *testing.T) {
					// given: current creation leaves provider data unset
					namespace := ulid.Make().String()
					channel := seedChannel(t, env, namespace)
					rule := seedRule(t, env, namespace, notification.EventTypeBalanceThreshold, channel.ID)
					event := seedEvent(t, env, namespace, rule, clock.Now(), nil)
					require.Len(t, event.DeliveryStatus, 1)

					id := event.DeliveryStatus[0].NamespacedID
					var raw sql.NullString
					err := env.sqlDB.QueryRowContext(t.Context(),
						"SELECT provider_data FROM notification_event_delivery_status WHERE id = $1", id.ID,
					).Scan(&raw)
					require.NoError(t, err)

					assert.False(t, raw.Valid)

					// Seed future data directly; current service inputs do not own this field.
					if tt.providerData != nil {
						_, err := dbClient.NotificationEventDeliveryStatus.UpdateOneID(id.ID).
							SetProviderData(tt.providerData).
							Save(t.Context())
						require.NoError(t, err)
					}

					at := time.Date(2026, time.November, 4, 10, 0, 0, 0, time.UTC)
					nextAttempt := at.Add(time.Minute)
					attempts := []notification.EventDeliveryAttempt{
						{
							State:     notification.EventDeliveryStatusStateFailed,
							Timestamp: at,
							Response:  notification.EventDeliveryAttemptResponse{Body: "retry later"},
						},
					}

					// when: legacy operations update state, attempts and annotations independently
					for _, annotations := range []models.Annotations{
						{"svix.message_id": "annotation-owned"},
						nil,
					} {
						updated, err := adapter.UpdateEventDeliveryStatus(t.Context(), notification.UpdateEventDeliveryStatusInput{
							NamespacedID: id,
							State:        notification.EventDeliveryStatusStateFailed,
							Reason:       "delivery failed",
							NextAttempt:  &nextAttempt,
							Attempts:     attempts,
							Annotations:  annotations,
						})
						require.NoError(t, err)

						fetched, err := adapter.GetEventDeliveryStatus(t.Context(), notification.GetEventDeliveryStatusInput{NamespacedID: id})
						require.NoError(t, err)

						listed, err := adapter.ListEventsDeliveryStatus(t.Context(), notification.ListEventsDeliveryStatusInput{
							Namespaces: []string{namespace},
							Page:       pagination.NewPage(1, 20),
						})
						require.NoError(t, err)
						require.Len(t, listed.Items, 1)

						fetchedEvent, err := adapter.GetEvent(t.Context(), notification.GetEventInput(event.NamespacedID))
						require.NoError(t, err)
						require.Len(t, fetchedEvent.DeliveryStatus, 1)

						assert.Equal(t, *updated, *fetched)
						assert.Equal(t, *updated, listed.Items[0])
						assert.Equal(t, *updated, fetchedEvent.DeliveryStatus[0])
						assert.Equal(t, annotations, fetched.Annotations)
						assert.Equal(t, attempts, fetched.Attempts)
						assert.Equal(t, notification.EventDeliveryStatusStateFailed, fetched.State)
						assert.Equal(t, "delivery failed", fetched.Reason)
						assert.Equal(t, &nextAttempt, fetched.NextAttempt)

						// then: neither writing nor clearing annotations changes provider data
						stored, err := dbClient.NotificationEventDeliveryStatus.Get(t.Context(), id.ID)
						require.NoError(t, err)

						assert.Equal(t, tt.providerData, stored.ProviderData)

						err = env.sqlDB.QueryRowContext(t.Context(),
							"SELECT provider_data FROM notification_event_delivery_status WHERE id = $1", id.ID,
						).Scan(&raw)
						require.NoError(t, err)

						if tt.providerData == nil {
							assert.False(t, raw.Valid)

							continue
						}

						require.True(t, raw.Valid)

						expected, err := json.Marshal(tt.providerData)
						require.NoError(t, err)

						assert.JSONEq(t, string(expected), raw.String)
					}
				})
			}
		})
	}
}

func TestDeliveryStatusDispatchGenerationPreservation(t *testing.T) {
	schemas := []struct {
		name  string
		state testutils.PostgresDBState
	}{
		{name: "Ent schema", state: testutils.PostgresDBStateEntMigrated},
		{name: "SQL migrations", state: testutils.PostgresDBStateAtlasMigrated},
	}
	tests := []struct {
		name       string
		generation int
	}{
		{name: "initial dispatch", generation: 1},
		{name: "later dispatch", generation: 2},
		{name: "maximum generation", generation: math.MaxInt32},
	}

	for _, schema := range schemas {
		t.Run(schema.name, func(t *testing.T) {
			testdb := testutils.InitPostgresDB(t, schema.state)
			t.Cleanup(func() { testdb.Close(t) })

			dbClient := testdb.EntDriver.Client()
			adapter, err := notificationadapter.New(notificationadapter.Config{
				Client: dbClient,
				Logger: testutils.NewDiscardLogger(t),
			})
			require.NoError(t, err)

			env := &eventTestEnv{db: dbClient, sqlDB: testdb.PGDriver.DB(), adapter: adapter}
			var dataType, nullable, defaultValue string
			err = env.sqlDB.QueryRowContext(t.Context(),
				`SELECT data_type, is_nullable, column_default FROM information_schema.columns
				WHERE table_schema = current_schema() AND table_name = 'notification_event_delivery_status' AND column_name = 'dispatch_generation'`,
			).Scan(&dataType, &nullable, &defaultValue)
			require.NoError(t, err)

			assert.Equal(t, "integer", dataType)
			assert.Equal(t, "NO", nullable)
			assert.Equal(t, "1", defaultValue)

			for _, tt := range tests {
				t.Run(tt.name, func(t *testing.T) {
					// given: legacy creation relies on the initial generation default
					namespace := ulid.Make().String()
					channel := seedChannel(t, env, namespace)
					rule := seedRule(t, env, namespace, notification.EventTypeBalanceThreshold, channel.ID)
					event := seedEvent(t, env, namespace, rule, clock.Now(), nil)
					require.Len(t, event.DeliveryStatus, 1)

					id := event.DeliveryStatus[0].NamespacedID
					stored, err := dbClient.NotificationEventDeliveryStatus.Get(t.Context(), id.ID)
					require.NoError(t, err)

					assert.Equal(t, 1, stored.DispatchGeneration)

					// Simulate an old binary that omits the new column from its INSERT.
					var sqlDefault int
					err = env.sqlDB.QueryRowContext(t.Context(),
						`INSERT INTO notification_event_delivery_status (id, namespace, event_id, channel_id, created_at, updated_at)
						VALUES ($1, $2, $3, $4, NOW(), NOW()) RETURNING dispatch_generation`,
						ulid.Make().String(), namespace, event.ID, channel.ID,
					).Scan(&sqlDefault)
					require.NoError(t, err)

					assert.Equal(t, 1, sqlDefault)

					_, err = dbClient.NotificationEventDeliveryStatus.UpdateOneID(id.ID).
						SetDispatchGeneration(tt.generation).
						Save(t.Context())
					require.NoError(t, err)

					// when: existing writes change delivery state, including the current resend transition
					for _, state := range []notification.EventDeliveryStatusState{
						notification.EventDeliveryStatusStateSending,
						notification.EventDeliveryStatusStateSuccess,
						notification.EventDeliveryStatusStateResending,
						notification.EventDeliveryStatusStateFailed,
					} {
						attempts := []notification.EventDeliveryAttempt{
							{State: state, Timestamp: time.Date(2026, time.November, 4, 10, 0, 0, 0, time.UTC)},
						}
						updated, err := adapter.UpdateEventDeliveryStatus(t.Context(), notification.UpdateEventDeliveryStatusInput{
							NamespacedID: id,
							State:        state,
							Reason:       "legacy update",
							Attempts:     attempts,
							Annotations:  models.Annotations{"legacy": "unchanged behavior"},
						})
						require.NoError(t, err)

						fetched, err := adapter.GetEventDeliveryStatus(t.Context(), notification.GetEventDeliveryStatusInput{NamespacedID: id})
						require.NoError(t, err)

						assert.Equal(t, *updated, *fetched)
						assert.Equal(t, state, fetched.State)
						assert.Equal(t, attempts, fetched.Attempts)

						// then: the dormant generation is neither reset nor incremented
						stored, err = dbClient.NotificationEventDeliveryStatus.Get(t.Context(), id.ID)
						require.NoError(t, err)

						assert.Equal(t, tt.generation, stored.DispatchGeneration)
					}
				})
			}
		})
	}
}

func TestDeliveryStatusDispatchGenerationValidation(t *testing.T) {
	env := newEventTestEnv(t)
	tests := []struct {
		name       string
		generation int
		valid      bool
	}{
		{name: "negative", generation: -1},
		{name: "zero", generation: 0},
		{name: "minimum", generation: 1, valid: true},
		{name: "maximum", generation: math.MaxInt32, valid: true},
		{name: "above PostgreSQL integer range", generation: math.MaxInt32 + 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// given: a valid delivery and a candidate generation
			namespace := ulid.Make().String()
			channel := seedChannel(t, env, namespace)
			rule := seedRule(t, env, namespace, notification.EventTypeBalanceThreshold, channel.ID)
			event := seedEvent(t, env, namespace, rule, clock.Now(), nil)
			require.Len(t, event.DeliveryStatus, 1)

			id := event.DeliveryStatus[0].ID
			createdID := ulid.Make().String()

			// when: Ent create, single update and bulk update receive the candidate value
			_, createErr := env.db.NotificationEventDeliveryStatus.Create().
				SetID(createdID).
				SetNamespace(namespace).
				SetEventID(event.ID).
				SetChannelID(channel.ID).
				SetDispatchGeneration(tt.generation).
				Save(t.Context())

			_, updateErr := env.db.NotificationEventDeliveryStatus.UpdateOneID(id).
				SetDispatchGeneration(tt.generation).
				Save(t.Context())

			_, bulkErr := env.db.NotificationEventDeliveryStatus.Update().
				Where(statusdb.ID(id)).
				SetDispatchGeneration(tt.generation).
				Save(t.Context())

			// then: only values within PostgreSQL's positive integer range are accepted
			if tt.valid {
				require.NoError(t, createErr)
				require.NoError(t, updateErr)
				require.NoError(t, bulkErr)
			} else {
				require.ErrorContains(t, createErr, "validator failed for field")
				require.ErrorContains(t, updateErr, "validator failed for field")
				require.ErrorContains(t, bulkErr, "validator failed for field")
			}

			stored, err := env.db.NotificationEventDeliveryStatus.Get(t.Context(), id)
			require.NoError(t, err)

			exists, err := env.db.NotificationEventDeliveryStatus.Query().Where(statusdb.ID(createdID)).Exist(t.Context())
			require.NoError(t, err)

			assert.Equal(t, tt.valid, exists)

			if tt.valid {
				assert.Equal(t, tt.generation, stored.DispatchGeneration)
			} else {
				assert.Equal(t, 1, stored.DispatchGeneration)
			}
		})
	}
}

func TestDeliveryStatusDispatchGenerationMigration(t *testing.T) {
	tests := []struct {
		name  string
		state notification.EventDeliveryStatusState
	}{
		{name: "completed delivery", state: notification.EventDeliveryStatusStateSuccess},
		{name: "resend in progress", state: notification.EventDeliveryStatusStateResending},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// given: an existing delivery before the generation column is added
			testdb := testutils.InitPostgresDB(t, testutils.PostgresDBStateAtlasMigrated)
			t.Cleanup(func() { testdb.Close(t) })

			dbClient := testdb.EntDriver.Client()
			adapter, err := notificationadapter.New(notificationadapter.Config{
				Client: dbClient,
				Logger: testutils.NewDiscardLogger(t),
			})
			require.NoError(t, err)

			env := &eventTestEnv{db: dbClient, sqlDB: testdb.PGDriver.DB(), adapter: adapter}
			namespace := ulid.Make().String()
			channel := seedChannel(t, env, namespace)
			rule := seedRule(t, env, namespace, notification.EventTypeBalanceThreshold, channel.ID)
			event := seedEvent(t, env, namespace, rule, clock.Now(), nil)
			require.Len(t, event.DeliveryStatus, 1)

			before, err := adapter.UpdateEventDeliveryStatus(t.Context(), notification.UpdateEventDeliveryStatusInput{
				NamespacedID: event.DeliveryStatus[0].NamespacedID,
				State:        tt.state,
				Reason:       "historical delivery",
				Attempts: []notification.EventDeliveryAttempt{
					{State: notification.EventDeliveryStatusStateSuccess, Timestamp: time.Date(2026, time.November, 4, 10, 0, 0, 0, time.UTC)},
				},
			})
			require.NoError(t, err)

			// when: applying the generated down/up migration around the existing row
			for _, direction := range []string{"down", "up"} {
				migration, err := fs.ReadFile(migrate.OMMigrationsConfig.FS, path.Join(
					migrate.OMMigrationsConfig.FSPath,
					"20261110144149_notification_event_compatibility."+direction+".sql",
				))
				require.NoError(t, err)

				_, err = env.sqlDB.ExecContext(t.Context(), string(migration))
				require.NoError(t, err)
			}

			// then: even historical resends receive baseline 1 without rewriting delivery history
			stored, err := dbClient.NotificationEventDeliveryStatus.Get(t.Context(), before.ID)
			require.NoError(t, err)

			after, err := adapter.GetEventDeliveryStatus(t.Context(), notification.GetEventDeliveryStatusInput{NamespacedID: before.NamespacedID})
			require.NoError(t, err)

			assert.Equal(t, 1, stored.DispatchGeneration)
			assert.Equal(t, *before, *after)
		})
	}
}

func TestDeliveryStatusDispatchQueuedAtMigration(t *testing.T) {
	testdb := testutils.InitPostgresDB(t, testutils.PostgresDBStateAtlasMigrated)
	t.Cleanup(func() { testdb.Close(t) })

	db := testdb.PGDriver.DB()
	applyMigration := func(direction string) {
		t.Helper()

		migration, err := fs.ReadFile(migrate.OMMigrationsConfig.FS, path.Join(
			migrate.OMMigrationsConfig.FSPath,
			"20261110144149_notification_event_compatibility."+direction+".sql",
		))
		require.NoError(t, err)

		_, err = db.ExecContext(t.Context(), string(migration))
		require.NoError(t, err)
	}

	// given: deliveries written by a legacy worker before dispatch_queued_at exists
	applyMigration("down")

	next := time.Date(2026, time.November, 1, 13, 0, 0, 0, time.UTC)
	tests := []struct {
		name        string
		id          string
		state       notification.EventDeliveryStatusState
		nextAttempt *time.Time
	}{
		{name: "scheduled resend", id: ulid.Make().String(), state: notification.EventDeliveryStatusStateResending, nextAttempt: &next},
		{name: "completed delivery", id: ulid.Make().String(), state: notification.EventDeliveryStatusStateSuccess},
	}

	for _, tt := range tests {
		_, err := db.ExecContext(t.Context(), `INSERT INTO notification_event_delivery_status
			(id, namespace, event_id, channel_id, state, next_attempt_at, created_at, updated_at)
			VALUES ($1, $2, $3, $4, $5, $6, now(), now())`,
			tt.id, ulid.Make().String(), ulid.Make().String(), ulid.Make().String(), tt.state, tt.nextAttempt)
		require.NoError(t, err)
	}

	// when: upgrading a database with existing deliveries
	applyMigration("up")

	// then: each delivery keeps its schedule and has no inferred queue time
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			row, err := testdb.EntDriver.Client().NotificationEventDeliveryStatus.Get(t.Context(), tt.id)
			require.NoError(t, err)

			assert.Nil(t, row.DispatchQueuedAt)
			assert.Equal(t, tt.state, row.State)

			if tt.nextAttempt == nil {
				require.Nil(t, row.NextAttemptAt)
				return
			}

			require.NotNil(t, row.NextAttemptAt)

			assert.WithinDuration(t, *tt.nextAttempt, *row.NextAttemptAt, 0)
		})
	}
}
