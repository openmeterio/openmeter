package adapter_test

import (
	"database/sql"
	"encoding/json"
	"io/fs"
	"path"
	"testing"
	"time"

	"github.com/oklog/ulid/v2"
	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	entdb "github.com/openmeterio/openmeter/openmeter/ent/db"
	eventdb "github.com/openmeterio/openmeter/openmeter/ent/db/notificationevent"
	"github.com/openmeterio/openmeter/openmeter/notification"
	notificationadapter "github.com/openmeterio/openmeter/openmeter/notification/adapter"
	"github.com/openmeterio/openmeter/openmeter/testutils"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/filter"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/pagination"
	"github.com/openmeterio/openmeter/pkg/sortx"
	"github.com/openmeterio/openmeter/tools/migrate"
)

// These tests exercise the v3 list-filter wiring added to ListEvents
// (openmeter/notification/adapter/event.go) directly against the adapter, for the same
// reason channel_test.go does: notification/service.Service cannot create channels
// outside svix, and events need a rule with channels to exist at all.
type eventTestEnv struct {
	sqlDB   *sql.DB
	db      *entdb.Client
	adapter notification.Repository
}

func newEventTestEnv(t *testing.T) *eventTestEnv {
	t.Helper()

	testdb := testutils.InitPostgresDB(t, testutils.PostgresDBStateEntMigrated)
	dbClient := testdb.EntDriver.Client()

	t.Cleanup(func() {
		_ = dbClient.Close()
		testdb.Close(t)
	})

	adapter, err := notificationadapter.New(notificationadapter.Config{
		Client: dbClient,
		Logger: testutils.NewDiscardLogger(t),
	})
	require.NoError(t, err, "constructing notification adapter must not fail")

	return &eventTestEnv{db: dbClient, sqlDB: testdb.PGDriver.DB(), adapter: adapter}
}

func seedChannel(t *testing.T, env *eventTestEnv, ns string) notification.Channel {
	t.Helper()

	channel, err := env.adapter.CreateChannel(t.Context(), notification.CreateChannelInput{
		NamespacedModel: models.NamespacedModel{Namespace: ns},
		Type:            notification.ChannelTypeWebhook,
		Name:            "channel-" + ulid.Make().String(),
		Config: notification.ChannelConfig{
			ChannelConfigMeta: notification.ChannelConfigMeta{Type: notification.ChannelTypeWebhook},
			WebHook:           notification.WebHookChannelConfig{URL: "https://example.com/hook"},
		},
	})
	require.NoError(t, err, "seeding channel must not fail")
	require.NotNil(t, channel)

	return *channel
}

func seedRule(t *testing.T, env *eventTestEnv, ns string, eventType notification.EventType, channelIDs ...string) notification.Rule {
	t.Helper()

	config := notification.RuleConfig{
		RuleConfigMeta: notification.RuleConfigMeta{Type: eventType},
	}

	switch eventType {
	case notification.EventTypeBalanceThreshold:
		config.BalanceThreshold = &notification.BalanceThresholdRuleConfig{
			Thresholds: []notification.BalanceThreshold{
				{Type: notification.BalanceThresholdTypeUsagePercentage, Value: 50},
			},
		}
	case notification.EventTypeEntitlementReset:
		config.EntitlementReset = &notification.EntitlementResetRuleConfig{}
	case notification.EventTypeInvoiceCreated, notification.EventTypeInvoiceUpdated:
		config.Invoice = &notification.InvoiceRuleConfig{}
	default:
		t.Fatalf("seedRule does not support event type %s", eventType)
	}

	rule, err := env.adapter.CreateRule(t.Context(), notification.CreateRuleInput{
		NamespacedModel: models.NamespacedModel{Namespace: ns},
		Type:            eventType,
		Name:            "rule-" + ulid.Make().String(),
		Channels:        channelIDs,
		Config:          config,
	})
	require.NoError(t, err, "seeding rule must not fail")
	require.NotNil(t, rule)

	return *rule
}

// seedEvent creates an event at the given frozen time so tests can assert on created_at
// deterministically. annotations carry the subject/feature values that the JSONB filters
// match against.
func seedEvent(t *testing.T, env *eventTestEnv, ns string, rule notification.Rule, at time.Time, annotations models.Annotations) notification.Event {
	t.Helper()

	clock.FreezeTime(at)
	defer clock.UnFreeze()

	event, err := env.adapter.CreateEvent(t.Context(), notification.CreateEventInput{
		NamespacedModel: models.NamespacedModel{Namespace: ns},
		Type:            rule.Type,
		RuleID:          rule.ID,
		Annotations:     annotations,
		Payload: notification.EventPayload{
			EventPayloadMeta: notification.EventPayloadMeta{
				Type:    rule.Type,
				Version: notification.EventPayloadVersionCurrent,
			},
			BalanceThreshold: &notification.BalanceThresholdPayload{},
		},
	})
	require.NoError(t, err, "seeding event must not fail")
	require.NotNil(t, event)

	return *event
}

// TestListEvents_Filters table-tests each of the filter fields wired up in ListEvents
// (openmeter/notification/adapter/event.go).
func TestListEvents_Filters(t *testing.T) {
	env := newEventTestEnv(t)
	ns := ulid.Make().String()

	frozen := clock.Now()
	t0 := frozen.Add(-2 * time.Hour)
	t1 := frozen.Add(-1 * time.Hour)

	channelA := seedChannel(t, env, ns)
	channelB := seedChannel(t, env, ns)

	ruleA := seedRule(t, env, ns, notification.EventTypeBalanceThreshold, channelA.ID)
	ruleB := seedRule(t, env, ns, notification.EventTypeEntitlementReset, channelB.ID)

	subjectID := ulid.Make().String()
	featureID := ulid.Make().String()

	first := seedEvent(t, env, ns, ruleA, t0, models.Annotations{
		notification.AnnotationEventSubjectKey: "subject-1",
		notification.AnnotationEventFeatureKey: "feature-1",
	})
	second := seedEvent(t, env, ns, ruleA, t1, models.Annotations{
		notification.AnnotationEventSubjectID: subjectID,
		notification.AnnotationEventFeatureID: featureID,
	})
	third := seedEvent(t, env, ns, ruleB, frozen, nil)

	// Mix stored spellings in one namespace. The third Event keeps its legacy PENDING.
	_, err := env.sqlDB.ExecContext(t.Context(), `UPDATE notification_event_delivery_status
		SET state = CASE WHEN event_id = $1 THEN 'failed' ELSE 'pending' END
		WHERE event_id IN ($1, $2)`, first.ID, second.ID)
	require.NoError(t, err)

	testCases := []struct {
		name    string
		input   notification.ListEventsInput
		wantIDs []string
	}{
		{
			name: "ID eq matches exactly one event",
			input: notification.ListEventsInput{
				ID: &filter.FilterULID{FilterString: filter.FilterString{Eq: lo.ToPtr(first.ID)}},
			},
			wantIDs: []string{first.ID},
		},
		{
			name: "ID in matches the selected events",
			input: notification.ListEventsInput{
				ID: &filter.FilterULID{FilterString: filter.FilterString{In: lo.ToPtr([]string{first.ID, third.ID})}},
			},
			wantIDs: []string{first.ID, third.ID},
		},
		{
			name: "Type eq matches every event of that type",
			input: notification.ListEventsInput{
				Type: &filter.FilterString{Eq: lo.ToPtr(string(notification.EventTypeBalanceThreshold))},
			},
			wantIDs: []string{first.ID, second.ID},
		},
		{
			name: "CreatedAt range matches only the event created inside the window",
			input: notification.ListEventsInput{
				CreatedAt: filter.NewFilterTime(
					lo.ToPtr(t0.Add(-time.Minute)),
					lo.ToPtr(t0.Add(time.Minute)),
				),
			},
			wantIDs: []string{first.ID},
		},
		{
			name: "RuleID eq matches every event of that rule",
			input: notification.ListEventsInput{
				RuleID: &filter.FilterULID{FilterString: filter.FilterString{Eq: lo.ToPtr(ruleB.ID)}},
			},
			wantIDs: []string{third.ID},
		},
		{
			name: "ChannelID eq matches events whose rule targets the channel",
			input: notification.ListEventsInput{
				ChannelID: &filter.FilterString{Eq: lo.ToPtr(channelA.ID)},
			},
			wantIDs: []string{first.ID, second.ID},
		},
		{
			name: "DeliveryStatus eq matches events with a status in that state",
			input: notification.ListEventsInput{
				DeliveryStatus: &filter.FilterString{Eq: lo.ToPtr(string(notification.EventDeliveryStatusStateFailed))},
			},
			wantIDs: []string{first.ID},
		},
		{
			name: "DeliveryStatus eq matches both stored spellings",
			input: notification.ListEventsInput{
				DeliveryStatus: &filter.FilterString{Eq: new("PENDING")},
			},
			wantIDs: []string{second.ID, third.ID},
		},
		{
			name: "DeliveryStatus in matches both stored spellings",
			input: notification.ListEventsInput{
				DeliveryStatus: &filter.FilterString{In: new([]string{"SENDING", "PENDING"})},
			},
			wantIDs: []string{second.ID, third.ID},
		},
		{
			name: "V1 delivery state list matches both stored spellings",
			input: notification.ListEventsInput{
				DeliveryStatusStates: []notification.EventDeliveryStatusState{notification.EventDeliveryStatusStatePending},
			},
			wantIDs: []string{second.ID, third.ID},
		},
		{
			name: "SubjectKey eq matches the subject key annotation",
			input: notification.ListEventsInput{
				SubjectKey: &filter.FilterString{Eq: lo.ToPtr("subject-1")},
			},
			wantIDs: []string{first.ID},
		},
		{
			name: "SubjectKey eq does not match the subject id annotation",
			input: notification.ListEventsInput{
				SubjectKey: &filter.FilterString{Eq: lo.ToPtr(subjectID)},
			},
			wantIDs: []string{},
		},
		{
			name: "SubjectKey neq does not match events without the annotation",
			input: notification.ListEventsInput{
				SubjectKey: &filter.FilterString{Ne: lo.ToPtr("other-subject")},
			},
			wantIDs: []string{first.ID},
		},
		{
			name: "SubjectID eq matches the subject id annotation",
			input: notification.ListEventsInput{
				SubjectID: &filter.FilterULID{FilterString: filter.FilterString{Eq: lo.ToPtr(subjectID)}},
			},
			wantIDs: []string{second.ID},
		},
		{
			name: "FeatureKey in matches the listed feature keys",
			input: notification.ListEventsInput{
				FeatureKey: &filter.FilterString{In: lo.ToPtr([]string{"feature-1", "feature-other"})},
			},
			wantIDs: []string{first.ID},
		},
		{
			name: "FeatureID eq matches the feature id annotation",
			input: notification.ListEventsInput{
				FeatureID: &filter.FilterULID{FilterString: filter.FilterString{Eq: lo.ToPtr(featureID)}},
			},
			wantIDs: []string{second.ID},
		},
		{
			name: "deprecated Subjects matches keys and ids from one list",
			input: notification.ListEventsInput{
				Subjects: []string{"subject-1", subjectID},
			},
			wantIDs: []string{first.ID, second.ID},
		},
		{
			name: "deprecated Features matches keys and ids from one list",
			input: notification.ListEventsInput{
				Features: []string{"feature-1", featureID},
			},
			wantIDs: []string{first.ID, second.ID},
		},
		{
			name: "deprecated Events, Rules, Channels, DeliveryStatusStates and period combine",
			input: notification.ListEventsInput{
				Events:               []string{first.ID, second.ID, third.ID},
				Rules:                []string{ruleA.ID},
				Channels:             []string{channelA.ID},
				DeliveryStatusStates: []notification.EventDeliveryStatusState{notification.EventDeliveryStatusStateFailed},
				From:                 t0.Add(-time.Minute),
				To:                   t1,
			},
			wantIDs: []string{first.ID},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			input := tc.input
			input.Namespaces = []string{ns}
			input.Page = pagination.NewPage(1, 20)

			result, err := env.adapter.ListEvents(t.Context(), input)
			require.NoError(t, err)

			gotIDs := make([]string, 0, len(result.Items))
			for _, item := range result.Items {
				gotIDs = append(gotIDs, item.ID)

				for _, status := range item.DeliveryStatus {
					assert.NoError(t, status.State.Validate(), "nested delivery states must use the domain spelling")
				}
			}

			assert.ElementsMatch(t, tc.wantIDs, gotIDs)
		})
	}
}

func TestListEvents_MixedDeliveryStatesDueForReconciliation(t *testing.T) {
	env := newEventTestEnv(t)
	ns := ulid.Make().String()
	channelA := seedChannel(t, env, ns)
	channelB := seedChannel(t, env, ns)
	rule := seedRule(t, env, ns, notification.EventTypeBalanceThreshold, channelA.ID, channelB.ID)

	cutoff := time.Date(2026, time.November, 9, 10, 0, 0, 0, time.UTC)
	later := cutoff.Add(time.Minute)
	tests := []struct {
		name       string
		state      string
		next       *time.Time
		otherState string
		due        bool
	}{
		{name: "pending without a schedule", state: "pending", otherState: "SUCCESS", due: true},
		{name: "sending at the cutoff", state: "sending", next: &cutoff, otherState: "FAILED", due: true},
		{name: "resending at the cutoff", state: "resending", next: &cutoff, otherState: "success", due: true},
		{name: "legacy pending at the cutoff", state: "PENDING", next: &cutoff, otherState: "failed", due: true},
		{name: "completed destination cannot make a future send due", state: "SENDING", next: &later, otherState: "success"},
		{name: "failed destination cannot make a future resend due", state: "RESENDING", next: &later, otherState: "failed"},
		{name: "all destinations terminal", state: "FAILED", otherState: "success"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// given: two destinations for the same Event, with an unscheduled terminal sibling
			event := seedEvent(t, env, ns, rule, cutoff, nil)
			require.Len(t, event.DeliveryStatus, 2)

			for _, dest := range []struct {
				id    string
				state string
				next  *time.Time
			}{
				{id: event.DeliveryStatus[0].ID, state: tt.state, next: tt.next},
				{id: event.DeliveryStatus[1].ID, state: tt.otherState},
			} {
				_, err := env.sqlDB.ExecContext(t.Context(), `UPDATE notification_event_delivery_status
					SET state = $2, next_attempt_at = $3 WHERE id = $1`, dest.id, dest.state, dest.next)
				require.NoError(t, err)
			}

			// when: selecting this Event with the poller's active-state filter and cutoff
			result, err := env.adapter.ListEvents(t.Context(), notification.ListEventsInput{
				ID:                &filter.FilterULID{FilterString: filter.FilterString{Eq: &event.ID}},
				DeliveryStatus:    &filter.FilterString{In: new([]string{"PENDING", "SENDING", "RESENDING"})},
				NextAttemptBefore: cutoff,
				Page:              pagination.NewPage(1, 20),
			})
			require.NoError(t, err)

			// then: the same destination must be active and due
			if tt.due {
				require.Len(t, result.Items, 1)

				assert.Equal(t, event.ID, result.Items[0].ID)
			} else {
				assert.Empty(t, result.Items)
			}
		})
	}
}

// TestListEvents_AnnotationFilterOperators pins the JSONB predicate semantics of the
// annotation-backed filters: negated operators treat a missing annotation like a NULL
// column, so events without the annotation never match, and operators the API cannot
// produce are rejected instead of being silently ignored.
func TestListEvents_AnnotationFilterOperators(t *testing.T) {
	env := newEventTestEnv(t)
	ns := ulid.Make().String()

	channel := seedChannel(t, env, ns)
	rule := seedRule(t, env, ns, notification.EventTypeBalanceThreshold, channel.ID)

	frozen := clock.Now()

	annotated := seedEvent(t, env, ns, rule, frozen.Add(-2*time.Hour), models.Annotations{
		notification.AnnotationEventSubjectKey: "customer-42",
	})
	other := seedEvent(t, env, ns, rule, frozen.Add(-time.Hour), models.Annotations{
		notification.AnnotationEventSubjectKey: "internal-7",
	})
	seedEvent(t, env, ns, rule, frozen, nil)

	testCases := []struct {
		name    string
		input   notification.ListEventsInput
		wantIDs []string
		wantErr bool
	}{
		{
			name: "eq matches the annotation value",
			input: notification.ListEventsInput{
				SubjectKey: &filter.FilterString{Eq: lo.ToPtr("customer-42")},
			},
			wantIDs: []string{annotated.ID},
		},
		{
			name: "neq excludes the matching event and events without the annotation",
			input: notification.ListEventsInput{
				SubjectKey: &filter.FilterString{Ne: lo.ToPtr("customer-42")},
			},
			wantIDs: []string{other.ID},
		},
		{
			name: "unsupported operator is rejected",
			input: notification.ListEventsInput{
				SubjectKey: &filter.FilterString{Contains: lo.ToPtr("CUSTOMER")},
			},
			wantErr: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			input := tc.input
			input.Namespaces = []string{ns}
			input.Page = pagination.NewPage(1, 20)

			result, err := env.adapter.ListEvents(t.Context(), input)
			if tc.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)

			gotIDs := make([]string, 0, len(result.Items))
			for _, item := range result.Items {
				gotIDs = append(gotIDs, item.ID)
			}

			assert.ElementsMatch(t, tc.wantIDs, gotIDs)
		})
	}
}

// TestListEvents_Sort covers the adapter's OrderBy switch for each sort field the v3 API
// exposes.
func TestListEvents_Sort(t *testing.T) {
	env := newEventTestEnv(t)
	ns := ulid.Make().String()

	frozen := clock.Now()

	channel := seedChannel(t, env, ns)
	rule := seedRule(t, env, ns, notification.EventTypeBalanceThreshold, channel.ID)

	older := seedEvent(t, env, ns, rule, frozen.Add(-2*time.Hour), nil)
	newer := seedEvent(t, env, ns, rule, frozen.Add(-1*time.Hour), nil)

	t.Run("created_at asc returns creation order", func(t *testing.T) {
		result, err := env.adapter.ListEvents(t.Context(), notification.ListEventsInput{
			Namespaces: []string{ns},
			OrderBy:    notification.OrderByCreatedAt,
			Order:      sortx.OrderAsc,
			Page:       pagination.NewPage(1, 20),
		})
		require.NoError(t, err)
		require.Len(t, result.Items, 2)
		assert.Equal(t, older.ID, result.Items[0].ID)
		assert.Equal(t, newer.ID, result.Items[1].ID)
	})

	t.Run("created_at desc reverses creation order", func(t *testing.T) {
		result, err := env.adapter.ListEvents(t.Context(), notification.ListEventsInput{
			Namespaces: []string{ns},
			OrderBy:    notification.OrderByCreatedAt,
			Order:      sortx.OrderDesc,
			Page:       pagination.NewPage(1, 20),
		})
		require.NoError(t, err)
		require.Len(t, result.Items, 2)
		assert.Equal(t, newer.ID, result.Items[0].ID)
		assert.Equal(t, older.ID, result.Items[1].ID)
	})

	t.Run("id asc returns ULID creation order", func(t *testing.T) {
		result, err := env.adapter.ListEvents(t.Context(), notification.ListEventsInput{
			Namespaces: []string{ns},
			OrderBy:    notification.OrderByID,
			Order:      sortx.OrderAsc,
			Page:       pagination.NewPage(1, 20),
		})
		require.NoError(t, err)
		require.Len(t, result.Items, 2)
		assert.Equal(t, older.ID, result.Items[0].ID)
		assert.Equal(t, newer.ID, result.Items[1].ID)
	})

	t.Run("type asc returns all rows without erroring", func(t *testing.T) {
		// Both events share a type, so this cannot assert a meaningful order between
		// rows — it only proves OrderByType reaches a valid Ent ordering clause instead
		// of falling through to the order-by-id default.
		result, err := env.adapter.ListEvents(t.Context(), notification.ListEventsInput{
			Namespaces: []string{ns},
			OrderBy:    notification.OrderByType,
			Order:      sortx.OrderAsc,
			Page:       pagination.NewPage(1, 20),
		})
		require.NoError(t, err)
		assert.Len(t, result.Items, 2)
	})
}

func TestEventChannelConfigCompatibility(t *testing.T) {
	tests := []struct {
		name    string
		data    string
		wantErr bool
	}{
		{
			name: "flat config",
			data: `{"url":"https://example.com/hook","custom_headers":{"X-Team":"billing"},"signing_secret":"original-secret"}`,
		},
		{
			name:    "malformed config",
			data:    `{"type":"WEBHOOK","data":[]}`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// given: an event referencing a channel whose config uses a new or malformed shape
			env := newEventTestEnv(t)
			channel := seedChannel(t, env, ulid.Make().String())
			rule := seedRule(t, env, channel.Namespace, notification.EventTypeBalanceThreshold, channel.ID)
			created := seedEvent(t, env, channel.Namespace, rule, clock.Now(), nil)

			_, err := env.db.NotificationChannel.UpdateOneID(channel.ID).
				SetConfig(json.RawMessage(tt.data)).Save(t.Context())
			require.NoError(t, err)

			// when: reading the event with its nested channels
			item, getErr := env.adapter.GetEvent(t.Context(), notification.GetEventInput(created.NamespacedID))
			listed, listErr := env.adapter.ListEvents(t.Context(), notification.ListEventsInput{
				Namespaces: []string{created.Namespace}, Page: pagination.NewPage(1, 20),
			})

			// then: config values or decoding errors propagate through both read paths
			if tt.wantErr {
				require.ErrorContains(t, getErr, "invalid notification channel config")
				require.ErrorContains(t, listErr, "invalid notification channel config")

				assert.Nil(t, item)

				return
			}

			require.NoError(t, getErr)
			require.NoError(t, listErr)
			require.Len(t, listed.Items, 1)

			require.Len(t, item.Rule.Channels, 1)
			require.Len(t, listed.Items[0].Rule.Channels, 1)

			assert.Equal(t, "original-secret", item.Rule.Channels[0].Config.WebHook.SigningSecret)
			assert.Equal(t, map[string]string{"X-Team": "billing"}, item.Rule.Channels[0].Config.WebHook.CustomHeaders)
			assert.Equal(t, item.Rule.Channels[0].Config, listed.Items[0].Rule.Channels[0].Config)
		})
	}
}

func TestEventRuleConfigCompatibility(t *testing.T) {
	tests := []struct {
		name    string
		data    string
		wantErr bool
	}{
		{
			name: "legacy envelope",
			data: `{"type":"entitlements.balance.threshold","data":{"features":["feature-a"],"thresholds":[{"type":"usage_percentage","value":50}]}}`,
		},
		{
			name: "flat config",
			data: `{"features":["feature-a"],"thresholds":[{"type":"usage_percentage","value":50}]}`,
		},
		{
			name:    "malformed envelope",
			data:    `{"type":"entitlements.balance.threshold","data":[]}`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// given: an existing event whose rule has a stored config in either format
			env := newEventTestEnv(t)
			channel := seedChannel(t, env, ulid.Make().String())
			rule := seedRule(t, env, channel.Namespace, notification.EventTypeBalanceThreshold, channel.ID)
			created := seedEvent(t, env, channel.Namespace, rule, clock.Now(), nil)

			_, err := env.db.NotificationRule.UpdateOneID(rule.ID).
				SetConfig(json.RawMessage(tt.data)).Save(t.Context())
			require.NoError(t, err)

			// when: reading the rule directly and through its event
			_, ruleGetErr := env.adapter.GetRule(t.Context(), notification.GetRuleInput(rule.NamespacedID))
			_, ruleListErr := env.adapter.ListRules(t.Context(), notification.ListRulesInput{
				Namespace: rule.Namespace,
				Page:      pagination.NewPage(1, 20),
			})
			item, getErr := env.adapter.GetEvent(t.Context(), notification.GetEventInput(created.NamespacedID))
			listed, listErr := env.adapter.ListEvents(t.Context(), notification.ListEventsInput{
				Namespaces: []string{created.Namespace},
				Page:       pagination.NewPage(1, 20),
			})

			// then: values and decoding errors propagate through every read path
			if tt.wantErr {
				require.ErrorContains(t, ruleGetErr, "invalid notification rule config")
				require.ErrorContains(t, ruleListErr, "invalid notification rule config")
				require.ErrorContains(t, getErr, "invalid notification rule config")
				require.ErrorContains(t, listErr, "invalid notification rule config")

				assert.Nil(t, item)

				return
			}

			require.NoError(t, ruleGetErr)
			require.NoError(t, ruleListErr)
			require.NoError(t, getErr)
			require.NoError(t, listErr)
			require.Len(t, listed.Items, 1)

			assert.Equal(t, []string{"feature-a"}, item.Rule.Config.BalanceThreshold.Features)
			assert.Equal(t, []notification.BalanceThreshold{
				{Type: notification.BalanceThresholdTypeUsagePercentage, Value: 50},
			}, item.Rule.Config.BalanceThreshold.Thresholds)
			assert.Equal(t, item.Rule.Config, listed.Items[0].Rule.Config)

			entity, err := env.db.NotificationRule.Get(t.Context(), rule.ID)
			require.NoError(t, err)

			assert.JSONEq(t, tt.data, string(entity.Config))
		})
	}
}

func TestEventTypeStorageCompatibility(t *testing.T) {
	env := newEventTestEnv(t)
	tests := []struct {
		name      string
		eventType notification.EventType
		payload   notification.EventPayload
	}{
		{
			name:      "entitlements.balance.threshold",
			eventType: notification.EventTypeBalanceThreshold,
			payload:   notification.EventPayload{BalanceThreshold: &notification.BalanceThresholdPayload{}},
		},
		{
			name:      "entitlements.reset",
			eventType: notification.EventTypeEntitlementReset,
			payload:   notification.EventPayload{EntitlementReset: &notification.EntitlementResetPayload{}},
		},
		{
			name:      "invoice.created",
			eventType: notification.EventTypeInvoiceCreated,
			payload:   notification.EventPayload{Invoice: &notification.InvoicePayload{}},
		},
		{
			name:      "invoice.updated",
			eventType: notification.EventTypeInvoiceUpdated,
			payload:   notification.EventPayload{Invoice: &notification.InvoicePayload{}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// given: a legacy rule and a different event type in the same namespace
			namespace := ulid.Make().String()
			channel := seedChannel(t, env, namespace)
			rule := seedRule(t, env, namespace, tt.eventType, channel.ID)

			otherType := notification.EventTypeBalanceThreshold
			if tt.eventType == otherType {
				otherType = notification.EventTypeEntitlementReset
			}

			otherRule := seedRule(t, env, namespace, otherType, channel.ID)
			seedEvent(t, env, namespace, otherRule, clock.Now(), nil)

			payload := tt.payload
			payload.EventPayloadMeta = notification.EventPayloadMeta{
				Type:    tt.eventType,
				Version: notification.EventPayloadVersionCurrent,
			}

			// when: creating through the existing adapter
			// Include sub-microsecond precision even on hosts with a coarser clock.
			clock.FreezeTime(time.Date(2026, time.November, 10, 14, 51, 17, 323346255, time.UTC))
			defer clock.UnFreeze()

			created, err := env.adapter.CreateEvent(t.Context(), notification.CreateEventInput{
				NamespacedModel: models.NamespacedModel{Namespace: namespace},
				Type:            tt.eventType,
				RuleID:          rule.ID,
				Payload:         payload,
			})
			require.NoError(t, err)
			require.NotNil(t, created)

			var storedType, storedPayload string
			err = env.sqlDB.QueryRowContext(t.Context(),
				"SELECT type, payload FROM notification_events WHERE id = $1", created.ID,
			).Scan(&storedType, &storedPayload)
			require.NoError(t, err)

			expectedPayload, err := json.Marshal(payload)
			require.NoError(t, err)

			// then: old readers still receive the same type and payload representation
			assert.Equal(t, tt.name, storedType)
			assert.JSONEq(t, string(expectedPayload), storedPayload)

			fetched, err := env.adapter.GetEvent(t.Context(), notification.GetEventInput(created.NamespacedID))
			require.NoError(t, err)
			require.NotNil(t, fetched)

			listed, err := env.adapter.ListEvents(t.Context(), notification.ListEventsInput{
				Namespaces: []string{namespace},
				Page:       pagination.NewPage(1, 20),
				Type:       &filter.FilterString{Eq: &tt.name},
			})
			require.NoError(t, err)
			require.Len(t, listed.Items, 1)

			assert.Equal(t, created.ID, listed.Items[0].ID)
			assert.Equal(t, tt.eventType, fetched.Type)
			assert.Equal(t, tt.eventType, listed.Items[0].Type)
			assert.Equal(t, rule.ID, fetched.Rule.ID)
			assert.Equal(t, created.CreatedAt.Truncate(time.Microsecond), fetched.CreatedAt)
			assert.Equal(t, created.Payload, fetched.Payload)
			assert.Equal(t, created.Payload, listed.Items[0].Payload)

			var readType, readPayload string
			err = env.sqlDB.QueryRowContext(t.Context(),
				"SELECT type, payload FROM notification_events WHERE id = $1", created.ID,
			).Scan(&readType, &readPayload)
			require.NoError(t, err)

			assert.Equal(t, storedType, readType)
			assert.JSONEq(t, storedPayload, readPayload)
		})
	}
}

func TestEventTypeValidation(t *testing.T) {
	env := newEventTestEnv(t)
	tests := []struct {
		name      string
		eventType notification.EventType
	}{
		{name: "empty"},
		{name: "unknown", eventType: "unknown.event"},
		{name: "versioned types remain unsupported", eventType: "v1.invoice.created"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// given: an existing rule and payload, but an unsupported event type
			namespace := ulid.Make().String()
			channel := seedChannel(t, env, namespace)
			rule := seedRule(t, env, namespace, notification.EventTypeBalanceThreshold, channel.ID)

			// when: writing directly through the adapter without service validation
			_, err := env.adapter.CreateEvent(t.Context(), notification.CreateEventInput{
				NamespacedModel: models.NamespacedModel{Namespace: namespace},
				Type:            tt.eventType,
				RuleID:          rule.ID,
				Payload: notification.EventPayload{
					EventPayloadMeta: notification.EventPayloadMeta{
						Type:    rule.Type,
						Version: notification.EventPayloadVersionCurrent,
					},
					BalanceThreshold: &notification.BalanceThresholdPayload{},
				},
			})
			require.ErrorContains(t, err, "invalid notification event type")

			// then: the domain validator still prevents persistence
			count, err := env.db.NotificationEvent.Query().Where(eventdb.Namespace(namespace)).Count(t.Context())
			require.NoError(t, err)

			assert.Zero(t, count)
		})
	}
}

func TestEventPayloadStorageCompatibility(t *testing.T) {
	env := newEventTestEnv(t)
	tests := []struct {
		name      string
		eventType notification.EventType
		payload   string
	}{
		{
			name:      "legacy balance threshold without version",
			eventType: notification.EventTypeBalanceThreshold,
			payload:   `{"type":"entitlements.balance.threshold","balanceThreshold":{},"extension":{"number":9007199254740993,"fraction":0.1234567890123456789,"text":"<tag> & value"}}`,
		},
		{
			name:      "legacy entitlement reset without version",
			eventType: notification.EventTypeEntitlementReset,
			payload:   `{"type":"entitlements.reset","entitlementReset":{},"extension":{"number":9007199254740993,"fraction":0.1234567890123456789,"text":"<tag> & value"}}`,
		},
		{
			name:      "invoice created version one",
			eventType: notification.EventTypeInvoiceCreated,
			payload:   `{"type":"invoice.created","version":1,"invoice":{"id":"invoice-created","currency":"USD"},"extension":{"number":9007199254740993,"fraction":0.1234567890123456789,"text":"<tag> & value"}}`,
		},
		{
			name:      "invoice updated version one",
			eventType: notification.EventTypeInvoiceUpdated,
			payload:   `{"type":"invoice.updated","version":1,"invoice":{"id":"invoice-updated","currency":"EUR"},"extension":{"number":9007199254740993,"fraction":0.1234567890123456789,"text":"<tag> & value"}}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for _, writer := range []string{"legacy SQL", "raw Ent"} {
				t.Run(writer, func(t *testing.T) {
					// given: an event with the existing typed payload and delivery records
					namespace := ulid.Make().String()
					channel := seedChannel(t, env, namespace)
					rule := seedRule(t, env, namespace, tt.eventType, channel.ID)

					var expected notification.EventPayload
					require.NoError(t, json.Unmarshal([]byte(tt.payload), &expected))

					created, err := env.adapter.CreateEvent(t.Context(), notification.CreateEventInput{
						NamespacedModel: models.NamespacedModel{Namespace: namespace},
						Type:            tt.eventType,
						RuleID:          rule.ID,
						Payload:         expected,
					})
					require.NoError(t, err)

					before, err := env.adapter.GetEvent(t.Context(), notification.GetEventInput(created.NamespacedID))
					require.NoError(t, err)

					// when: storing JSON with fields unknown to the current domain model
					switch writer {
					case "legacy SQL":
						_, err = env.sqlDB.ExecContext(t.Context(),
							"UPDATE notification_events SET payload = $1 WHERE id = $2", tt.payload, created.ID,
						)
					case "raw Ent":
						_, err = env.db.NotificationEvent.UpdateOneID(created.ID).
							SetPayload(json.RawMessage(tt.payload)).Save(t.Context())
					}

					require.NoError(t, err)

					var stored, kind, number, fraction, textValue string
					err = env.sqlDB.QueryRowContext(t.Context(),
						`SELECT payload, jsonb_typeof(payload), payload #>> '{extension,number}',
						payload #>> '{extension,fraction}', payload #>> '{extension,text}'
						FROM notification_events WHERE id = $1`, created.ID,
					).Scan(&stored, &kind, &number, &fraction, &textValue)
					require.NoError(t, err)

					// then: storage is an object with exact numbers, not a quoted or base64 string
					assert.Equal(t, "object", kind)
					assert.Equal(t, "9007199254740993", number)
					assert.Equal(t, "0.1234567890123456789", fraction)
					assert.Equal(t, "<tag> & value", textValue)
					assert.JSONEq(t, tt.payload, stored)

					entity, err := env.db.NotificationEvent.Get(t.Context(), created.ID)
					require.NoError(t, err)

					assert.Equal(t, json.RawMessage(stored), entity.Payload)

					fetched, err := env.adapter.GetEvent(t.Context(), notification.GetEventInput(created.NamespacedID))
					require.NoError(t, err)

					listed, err := env.adapter.ListEvents(t.Context(), notification.ListEventsInput{
						Namespaces: []string{namespace},
						Page:       pagination.NewPage(1, 20),
					})
					require.NoError(t, err)
					require.Len(t, listed.Items, 1)

					assert.Equal(t, expected, fetched.Payload)
					assert.Equal(t, expected, listed.Items[0].Payload)
					assert.Equal(t, created.ID, fetched.ID)
					assert.Equal(t, before.CreatedAt, fetched.CreatedAt)
					assert.Equal(t, rule.ID, fetched.Rule.ID)
					assert.Equal(t, before.DeliveryStatus, fetched.DeliveryStatus)

					var afterRead string
					err = env.sqlDB.QueryRowContext(t.Context(),
						"SELECT payload FROM notification_events WHERE id = $1", created.ID,
					).Scan(&afterRead)
					require.NoError(t, err)

					assert.Equal(t, stored, afterRead, "reads must not rewrite unknown fields or historical payloads")
				})
			}
		})
	}
}

func TestEventPayloadReadErrors(t *testing.T) {
	env := newEventTestEnv(t)
	tests := []struct {
		name    string
		payload string
		wantErr string
	}{
		{
			name:    "missing invoice version",
			payload: `{"type":"invoice.created","invoice":{}}`,
			wantErr: "unsupported notification event payload version: 0",
		},
		{
			name:    "unsupported invoice version",
			payload: `{"type":"invoice.created","version":99,"invoice":{}}`,
			wantErr: "unsupported notification event payload version: 99",
		},
		{
			name:    "invalid metadata",
			payload: `{"type":"invoice.created","version":"invalid","invoice":{}}`,
			wantErr: "failed to deserialize notification event payload meta",
		},
		{
			name:    "invalid invoice payload",
			payload: `{"type":"invoice.created","version":1,"invoice":[]}`,
			wantErr: "failed to deserialize notification event payload:",
		},
		{
			name:    "double encoded payload",
			payload: `"{\"type\":\"invoice.created\",\"version\":1,\"invoice\":{}}"`,
			wantErr: "failed to deserialize notification event payload meta",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// given: an existing event containing JSONB the typed mapper cannot decode
			namespace := ulid.Make().String()
			channel := seedChannel(t, env, namespace)
			rule := seedRule(t, env, namespace, notification.EventTypeInvoiceCreated, channel.ID)
			created, err := env.adapter.CreateEvent(t.Context(), notification.CreateEventInput{
				NamespacedModel: models.NamespacedModel{Namespace: namespace},
				Type:            rule.Type,
				RuleID:          rule.ID,
				Payload: notification.EventPayload{
					EventPayloadMeta: notification.EventPayloadMeta{
						Type:    rule.Type,
						Version: notification.EventPayloadVersionCurrent,
					},
					Invoice: &notification.InvoicePayload{},
				},
			})
			require.NoError(t, err)

			_, err = env.sqlDB.ExecContext(t.Context(),
				"UPDATE notification_events SET payload = $1 WHERE id = $2", tt.payload, created.ID,
			)
			require.NoError(t, err)

			// when: reading through either adapter entry point
			_, err = env.adapter.GetEvent(t.Context(), notification.GetEventInput(created.NamespacedID))
			require.ErrorContains(t, err, tt.wantErr)

			_, err = env.adapter.ListEvents(t.Context(), notification.ListEventsInput{
				Namespaces: []string{namespace},
				Page:       pagination.NewPage(1, 20),
			})
			require.ErrorContains(t, err, tt.wantErr)

			// then: failed reads leave the stored payload intact
			entity, err := env.db.NotificationEvent.Get(t.Context(), created.ID)
			require.NoError(t, err)

			assert.JSONEq(t, tt.payload, string(entity.Payload))
		})
	}
}

func TestEventRuleReferenceStorage(t *testing.T) {
	tests := []struct {
		name  string
		state testutils.PostgresDBState
	}{
		{name: "Ent schema", state: testutils.PostgresDBStateEntMigrated},
		{name: "SQL migrations", state: testutils.PostgresDBStateAtlasMigrated},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// given: a legacy Rule-backed Event in either schema installation path
			testdb := testutils.InitPostgresDB(t, tt.state)
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
			legacy := seedEvent(t, env, namespace, rule, clock.Now(), nil)

			// Exercise the generated migration with existing data, starting from
			// the old NOT NULL constraint restored by its down migration.
			for _, direction := range []string{"down", "up"} {
				migration, err := fs.ReadFile(migrate.OMMigrationsConfig.FS, path.Join(
					migrate.OMMigrationsConfig.FSPath,
					"20261110144149_notification_event_compatibility."+direction+".sql",
				))
				require.NoError(t, err)

				_, err = env.sqlDB.ExecContext(t.Context(), string(migration))
				require.NoError(t, err)

				var nullable string
				err = env.sqlDB.QueryRowContext(t.Context(),
					`SELECT is_nullable FROM information_schema.columns
					WHERE table_schema = current_schema() AND table_name = 'notification_events' AND column_name = 'rule_id'`,
				).Scan(&nullable)
				require.NoError(t, err)

				if direction == "down" {
					assert.Equal(t, "NO", nullable)
				} else {
					assert.Equal(t, "YES", nullable)
				}
			}

			payload, err := json.Marshal(legacy.Payload)
			require.NoError(t, err)

			// when: bypassing current creation paths to exercise the nullable column
			withoutRule, err := dbClient.NotificationEvent.Create().
				SetNamespace(namespace).
				SetType(rule.Type).
				SetPayload(payload).
				Save(t.Context())
			require.NoError(t, err)

			var storedRuleID sql.NullString
			err = env.sqlDB.QueryRowContext(t.Context(),
				"SELECT rule_id FROM notification_events WHERE id = $1", withoutRule.ID,
			).Scan(&storedRuleID)
			require.NoError(t, err)

			// then: the schema accepts SQL NULL, while non-null references remain protected
			assert.False(t, storedRuleID.Valid)

			_, err = dbClient.NotificationEvent.Create().
				SetNamespace(namespace).
				SetType(rule.Type).
				SetPayload(payload).
				SetRuleID(ulid.Make().String()).
				Save(t.Context())
			require.Error(t, err)

			assert.True(t, entdb.IsConstraintError(err))

			err = dbClient.NotificationRule.DeleteOneID(rule.ID).Exec(t.Context())
			require.ErrorContains(t, err, "notification_events_notification_rules_events")

			fetched, err := adapter.GetEvent(t.Context(), notification.GetEventInput(legacy.NamespacedID))
			require.NoError(t, err)

			assert.Equal(t, legacy.ID, fetched.ID)
			assert.Equal(t, rule.ID, fetched.Rule.ID)
			assert.Equal(t, legacy.CreatedAt.Truncate(time.Microsecond), fetched.CreatedAt)
			assert.Equal(t, legacy.Payload, fetched.Payload)
		})
	}
}

func TestEventRuleSnapshotStorage(t *testing.T) {
	schemas := []struct {
		name  string
		state testutils.PostgresDBState
	}{
		{name: "Ent schema", state: testutils.PostgresDBStateEntMigrated},
		{name: "SQL migrations", state: testutils.PostgresDBStateAtlasMigrated},
	}
	tests := []struct {
		name     string
		snapshot json.RawMessage
	}{
		{name: "unset"},
		{name: "empty object", snapshot: json.RawMessage(`{}`)},
		{
			name: "evaluated Rule state",
			snapshot: json.RawMessage(`{
				"name":"evaluated rule",
				"event_type":"entitlements.balance.threshold",
				"updated_at":"2026-11-03T10:00:00Z",
				"config":{"thresholds":[{"type":"usage_percentage","value":50}]}
			}`),
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
				WHERE table_schema = current_schema() AND table_name = 'notification_events' AND column_name = 'rule_snapshot'`,
			).Scan(&nullable, &dataType, &defaultValue)
			require.NoError(t, err)

			assert.Equal(t, "YES", nullable)
			assert.Equal(t, "jsonb", dataType)
			assert.False(t, defaultValue.Valid)

			for _, tt := range tests {
				t.Run(tt.name, func(t *testing.T) {
					// given: current creation paths omit snapshots, even for Rule-backed Events
					namespace := ulid.Make().String()
					channel := seedChannel(t, env, namespace)
					rule := seedRule(t, env, namespace, notification.EventTypeBalanceThreshold, channel.ID)
					legacy := seedEvent(t, env, namespace, rule, clock.Now(), nil)

					legacyRow, err := dbClient.NotificationEvent.Get(t.Context(), legacy.ID)
					require.NoError(t, err)

					assert.Nil(t, legacyRow.RuleSnapshot)

					// Seed through Ent because the current service/adapter inputs do not expose snapshots.
					create := dbClient.NotificationEvent.Create().
						SetNamespace(namespace).
						SetType(rule.Type).
						SetRuleID(rule.ID).
						SetPayload(legacyRow.Payload)
					if tt.snapshot != nil {
						create.SetRuleSnapshot(tt.snapshot)
					}

					entity, err := create.Save(t.Context())
					require.NoError(t, err)

					status, err := dbClient.NotificationEventDeliveryStatus.Create().
						SetNamespace(namespace).
						SetEventID(entity.ID).
						SetChannelID(channel.ID).
						SetState(notification.EventDeliveryStatusStatePending).
						AddEvents(entity).
						Save(t.Context())
					require.NoError(t, err)

					// when: the live Rule changes and delivery reconciliation updates its status
					_, err = adapter.UpdateRule(t.Context(), notification.UpdateRuleInput{
						NamespacedID: rule.NamespacedID,
						Type:         rule.Type,
						Name:         "updated live rule",
						Channels:     []string{channel.ID},
						Config: notification.RuleConfig{
							RuleConfigMeta: notification.RuleConfigMeta{Type: rule.Type},
							BalanceThreshold: &notification.BalanceThresholdRuleConfig{
								Thresholds: []notification.BalanceThreshold{
									{Type: notification.BalanceThresholdTypeUsagePercentage, Value: 90},
								},
							},
						},
					})
					require.NoError(t, err)

					_, err = adapter.UpdateEventDeliveryStatus(t.Context(), notification.UpdateEventDeliveryStatusInput{
						NamespacedID: models.NamespacedID{Namespace: namespace, ID: status.ID},
						State:        notification.EventDeliveryStatusStateFailed,
						Reason:       "delivery failed",
					})
					require.NoError(t, err)

					fetched, err := adapter.GetEvent(t.Context(), notification.GetEventInput{Namespace: namespace, ID: entity.ID})
					require.NoError(t, err)

					listed, err := adapter.ListEvents(t.Context(), notification.ListEventsInput{
						Namespaces: []string{namespace},
						Page:       pagination.NewPage(1, 20),
						ID:         &filter.FilterULID{FilterString: filter.FilterString{Eq: &entity.ID}},
					})
					require.NoError(t, err)
					require.Len(t, listed.Items, 1)
					require.Len(t, fetched.DeliveryStatus, 1)

					assert.Equal(t, entity.ID, listed.Items[0].ID)
					assert.Equal(t, "updated live rule", fetched.Rule.Name)
					assert.Equal(t, notification.EventDeliveryStatusStateFailed, fetched.DeliveryStatus[0].State)
					assert.Equal(t, legacy.Payload, fetched.Payload)

					// then: snapshot storage retains the evaluated state, independently of the live Rule
					stored, err := dbClient.NotificationEvent.Get(t.Context(), entity.ID)
					require.NoError(t, err)

					var raw sql.NullString
					err = env.sqlDB.QueryRowContext(t.Context(),
						"SELECT rule_snapshot FROM notification_events WHERE id = $1", entity.ID,
					).Scan(&raw)
					require.NoError(t, err)

					if tt.snapshot == nil {
						assert.Nil(t, stored.RuleSnapshot)
						assert.False(t, raw.Valid)
					} else {
						require.True(t, raw.Valid)

						assert.JSONEq(t, string(tt.snapshot), string(stored.RuleSnapshot))
						assert.JSONEq(t, string(tt.snapshot), raw.String)
					}

					legacyRow, err = dbClient.NotificationEvent.Get(t.Context(), legacy.ID)
					require.NoError(t, err)

					assert.Nil(t, legacyRow.RuleSnapshot, "Rule updates must not populate historical snapshots")
				})
			}
		})
	}
}

func TestEventProviderDataPreservation(t *testing.T) {
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
				WHERE table_schema = current_schema() AND table_name = 'notification_events' AND column_name = 'provider_data'`,
			).Scan(&nullable, &dataType, &defaultValue)
			require.NoError(t, err)

			assert.Equal(t, "YES", nullable)
			assert.Equal(t, "jsonb", dataType)
			assert.False(t, defaultValue.Valid)

			for _, tt := range tests {
				t.Run(tt.name, func(t *testing.T) {
					// given: legacy creation leaves provider data unset, independently of annotations
					namespace := ulid.Make().String()
					channel := seedChannel(t, env, namespace)
					rule := seedRule(t, env, namespace, notification.EventTypeBalanceThreshold, channel.ID)
					annotations := models.Annotations{"svix.message.id": "legacy-message"}
					event := seedEvent(t, env, namespace, rule, clock.Now(), annotations)

					var raw sql.NullString
					err := env.sqlDB.QueryRowContext(t.Context(),
						"SELECT provider_data FROM notification_events WHERE id = $1", event.ID,
					).Scan(&raw)
					require.NoError(t, err)

					assert.False(t, raw.Valid)

					// Seed future data directly; current service inputs do not own this field.
					if tt.providerData != nil {
						_, err := dbClient.NotificationEvent.UpdateOneID(event.ID).
							SetProviderData(tt.providerData).
							Save(t.Context())
						require.NoError(t, err)
					}

					// when: annotations and delivery status change, followed by legacy reads
					annotations["svix.message_id"] = "annotation-owned"
					_, err = dbClient.NotificationEvent.UpdateOneID(event.ID).
						SetAnnotations(annotations).
						Save(t.Context())
					require.NoError(t, err)
					require.Len(t, event.DeliveryStatus, 1)

					_, err = adapter.UpdateEventDeliveryStatus(t.Context(), notification.UpdateEventDeliveryStatusInput{
						NamespacedID: event.DeliveryStatus[0].NamespacedID,
						State:        notification.EventDeliveryStatusStateFailed,
						Reason:       "delivery failed",
						Annotations:  models.Annotations{"svix.message.id": "delivery-owned"},
					})
					require.NoError(t, err)

					fetched, err := adapter.GetEvent(t.Context(), notification.GetEventInput(event.NamespacedID))
					require.NoError(t, err)

					listed, err := adapter.ListEvents(t.Context(), notification.ListEventsInput{
						Namespaces: []string{namespace},
						Page:       pagination.NewPage(1, 20),
					})
					require.NoError(t, err)
					require.Len(t, listed.Items, 1)
					require.Len(t, fetched.DeliveryStatus, 1)

					assert.Equal(t, event.ID, listed.Items[0].ID)
					assert.Equal(t, annotations, fetched.Annotations)
					assert.Equal(t, annotations, listed.Items[0].Annotations)
					assert.Equal(t, event.Payload, fetched.Payload)
					assert.Equal(t, rule.ID, fetched.Rule.ID)
					assert.Equal(t, notification.EventDeliveryStatusStateFailed, fetched.DeliveryStatus[0].State)
					assert.Equal(t, "delivery-owned", fetched.DeliveryStatus[0].Annotations["svix.message.id"])

					// then: no operation changes provider data or copies annotations into it
					stored, err := dbClient.NotificationEvent.Get(t.Context(), event.ID)
					require.NoError(t, err)

					assert.Equal(t, tt.providerData, stored.ProviderData)

					err = env.sqlDB.QueryRowContext(t.Context(),
						"SELECT provider_data FROM notification_events WHERE id = $1", event.ID,
					).Scan(&raw)
					require.NoError(t, err)

					if tt.providerData == nil {
						assert.False(t, raw.Valid)

						return
					}

					require.True(t, raw.Valid)

					expected, err := json.Marshal(tt.providerData)
					require.NoError(t, err)

					assert.JSONEq(t, string(expected), raw.String)
				})
			}
		})
	}
}
