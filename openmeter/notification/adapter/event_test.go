package adapter_test

import (
	"database/sql"
	"encoding/json"
	"io/fs"
	"os"
	"path"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/oklog/ulid/v2"
	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apievents "github.com/openmeterio/openmeter/api/v3/handlers/notification/events"
	entdb "github.com/openmeterio/openmeter/openmeter/ent/db"
	"github.com/openmeterio/openmeter/openmeter/notification"
	notificationadapter "github.com/openmeterio/openmeter/openmeter/notification/adapter"
	"github.com/openmeterio/openmeter/openmeter/notification/httpdriver"
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
			input.Expand = &notification.EventExpand{DeliveryStatus: true}
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

func TestListEvents_BalanceEventDeduplicationHashes(t *testing.T) {
	env := newEventTestEnv(t)
	ns := ulid.Make().String()
	rule := seedRule(t, env, ns, notification.EventTypeBalanceThreshold, seedChannel(t, env, ns).ID)

	const (
		v1 = "bsnap_v1_legacy"
		v2 = "bsnap_v2_current"
	)

	// Given a usage period with both hash versions and newer Events that must be excluded.
	start := clock.Now().UTC().Truncate(time.Second)
	end := start.Add(time.Hour)
	legacy := seedEvent(t, env, ns, rule, start, models.Annotations{notification.AnnotationBalanceEventDedupeHash: v1})
	current := seedEvent(t, env, ns, rule, start.Add(time.Minute), models.Annotations{notification.AnnotationBalanceEventDedupeHash: v2})
	seedEvent(t, env, ns, rule, end, models.Annotations{"other.key": v2})
	seedEvent(t, env, ns, rule, end.Add(time.Second), models.Annotations{notification.AnnotationBalanceEventDedupeHash: v2})

	otherNS := ulid.Make().String()
	otherRule := seedRule(t, env, otherNS, notification.EventTypeBalanceThreshold, seedChannel(t, env, otherNS).ID)
	seedEvent(t, env, otherNS, otherRule, end, models.Annotations{notification.AnnotationBalanceEventDedupeHash: v1})

	tests := []struct {
		name   string
		hashes []string
		count  int
		latest string
	}{
		{name: "V1", hashes: []string{v1}, count: 1, latest: legacy.ID},
		{name: "V2", hashes: []string{v2}, count: 1, latest: current.ID},
		{name: "both versions", hashes: []string{v1, v2}, count: 2, latest: current.ID},
		{name: "unknown hash", hashes: []string{"bsnap_v2_unknown"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// When the worker requests the latest matching Event in this usage period.
			result, err := env.adapter.ListEvents(t.Context(), notification.ListEventsInput{
				Page:                pagination.NewPage(1, 1),
				Namespaces:          []string{ns},
				CreatedAt:           filter.NewFilterTime(&start, &end),
				DeduplicationHashes: tt.hashes,
				OrderBy:             notification.OrderByCreatedAt,
				Order:               sortx.OrderDesc,
			})
			require.NoError(t, err)

			// Then pagination counts all matches, even though only one Event is requested.
			assert.EqualValues(t, tt.count, result.TotalCount)

			if tt.latest == "" {
				require.Empty(t, result.Items)
				return
			}

			require.Len(t, result.Items, 1)
			assert.Equal(t, tt.latest, result.Items[0].ID)
			assert.Empty(t, result.Items[0].DeliveryStatus)
		})
	}
}

func TestEventReads_ExpandDeliveryStatus(t *testing.T) {
	env := newEventTestEnv(t)
	ns := ulid.Make().String()
	rule := seedRule(t, env, ns, notification.EventTypeBalanceThreshold, seedChannel(t, env, ns).ID)
	at := clock.Now().UTC().Truncate(time.Second)
	event := seedEvent(t, env, ns, rule, at, nil)

	for _, tt := range []struct {
		name   string
		expand *notification.EventExpand
		count  int
	}{
		{name: "omitted"},
		{name: "empty", expand: &notification.EventExpand{}},
		{name: "requested", expand: &notification.EventExpand{DeliveryStatus: true}, count: 1},
	} {
		t.Run(tt.name, func(t *testing.T) {
			got, err := env.adapter.GetEvent(t.Context(), notification.GetEventInput{
				Namespace: ns,
				ID:        event.ID,
				Expand:    tt.expand,
			})
			require.NoError(t, err)
			require.NotNil(t, got)
			assert.Len(t, got.DeliveryStatus, tt.count)

			// The filter must still match pending deliveries even when the caller omits the edge.
			page, err := env.adapter.ListEvents(t.Context(), notification.ListEventsInput{
				Namespaces:     []string{ns},
				Expand:         tt.expand,
				DeliveryStatus: &filter.FilterString{Eq: new(string(notification.EventDeliveryStatusStatePending))},
			})
			require.NoError(t, err)
			require.Len(t, page.Items, 1)
			assert.Equal(t, 1, page.TotalCount)
			assert.Equal(t, event.ID, page.Items[0].ID)
			assert.Len(t, page.Items[0].DeliveryStatus, tt.count)
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

			for _, stage := range []struct {
				name, direction string
			}{
				{name: "mixed states"},
				{name: "after upgrade", direction: "up"},
				{name: "after rollback", direction: "down"},
				{name: "after re-upgrade", direction: "up"},
			} {
				t.Run(stage.name, func(t *testing.T) {
					if stage.direction != "" {
						migration, err := fs.ReadFile(migrate.OMMigrationsConfig.FS, path.Join(
							migrate.OMMigrationsConfig.FSPath,
							"20261124090300_notification_delivery_state_normalization."+stage.direction+".sql",
						))
						require.NoError(t, err)

						_, err = env.sqlDB.ExecContext(t.Context(), string(migration))
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

	t.Run("created_at desc pages events with the same timestamp by id", func(t *testing.T) {
		// given events created at the same instant
		tiedNS := ulid.Make().String()
		tiedChannel := seedChannel(t, env, tiedNS)
		tiedRule := seedRule(t, env, tiedNS, notification.EventTypeBalanceThreshold, tiedChannel.ID)

		var want []string
		for range 3 {
			want = append(want, seedEvent(t, env, tiedNS, tiedRule, frozen, nil).ID)
		}

		slices.Sort(want)
		slices.Reverse(want)

		// when they are listed one per page
		var got []string
		for page := 1; page <= len(want); page++ {
			result, err := env.adapter.ListEvents(t.Context(), notification.ListEventsInput{
				Namespaces: []string{tiedNS},
				OrderBy:    notification.OrderByCreatedAt,
				Order:      sortx.OrderDesc,
				Page:       pagination.NewPage(page, 1),
			})
			require.NoError(t, err)
			require.Len(t, result.Items, 1)
			got = append(got, result.Items[0].ID)
		}

		// then id breaks the tie, so no event is skipped or repeated
		assert.Equal(t, want, got)
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
			item, getErr := env.adapter.GetEvent(t.Context(), notification.GetEventInput{Namespace: created.Namespace, ID: created.ID})
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
			item, getErr := env.adapter.GetEvent(t.Context(), notification.GetEventInput{Namespace: created.Namespace, ID: created.ID})
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
		versioned string
		eventType notification.EventType
		payload   notification.EventPayload
	}{
		{
			name:      "entitlements.balance.threshold",
			versioned: "v1.entitlement.balance_threshold.triggered",
			eventType: notification.EventTypeBalanceThreshold,
			payload:   notification.EventPayload{BalanceThreshold: &notification.BalanceThresholdPayload{}},
		},
		{
			name:      "entitlements.reset",
			versioned: "v1.entitlement.usage_period.reset",
			eventType: notification.EventTypeEntitlementReset,
			payload:   notification.EventPayload{EntitlementReset: &notification.EntitlementResetPayload{}},
		},
		{
			name:      "invoice.created",
			versioned: "v1.billing.invoice.created",
			eventType: notification.EventTypeInvoiceCreated,
			payload:   notification.EventPayload{Invoice: &notification.InvoicePayload{}},
		},
		{
			name:      "invoice.updated",
			versioned: "v1.billing.invoice.updated",
			eventType: notification.EventTypeInvoiceUpdated,
			payload:   notification.EventPayload{Invoice: &notification.InvoicePayload{}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// given: a rule and a different event type in the same namespace
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

			// then: storage uses the versioned name and the original payload representation
			assert.Equal(t, tt.versioned, storedType)
			assert.JSONEq(t, string(expectedPayload), storedPayload)

			for _, eventType := range []string{tt.name, tt.versioned} {
				for _, ruleType := range []string{tt.name, tt.versioned} {
					t.Run(eventType+"/"+ruleType, func(t *testing.T) {
						// when: the Event and Rule have independently selected storage spellings
						_, err = env.sqlDB.ExecContext(t.Context(), "UPDATE notification_events SET type = $1 WHERE id = $2", eventType, created.ID)
						require.NoError(t, err)

						_, err = env.sqlDB.ExecContext(t.Context(), "UPDATE notification_rules SET type = $1 WHERE id = $2", ruleType, rule.ID)
						require.NoError(t, err)

						fetched, err := env.adapter.GetEvent(t.Context(), notification.GetEventInput{Namespace: created.Namespace, ID: created.ID})
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
						assert.Equal(t, tt.eventType, fetched.Rule.Type)
						assert.Equal(t, tt.eventType, fetched.Rule.Config.Type)
						assert.Equal(t, created.CreatedAt.Truncate(time.Microsecond), fetched.CreatedAt)
						assert.Equal(t, created.Payload, fetched.Payload)
						assert.Equal(t, created.Payload, listed.Items[0].Payload)

						// Internal filters can combine exact and pattern comparisons on the public type.
						mixed, err := env.adapter.ListEvents(t.Context(), notification.ListEventsInput{
							Namespaces: []string{namespace},
							Type: &filter.FilterString{And: &[]filter.FilterString{
								{Like: &tt.name},
								{Ne: new(string(otherType))},
							}},
						})
						require.NoError(t, err)
						require.Len(t, mixed.Items, 1)
						assert.Equal(t, created.ID, mixed.Items[0].ID)

						var readType, readPayload string
						err = env.sqlDB.QueryRowContext(t.Context(),
							"SELECT type, payload FROM notification_events WHERE id = $1", created.ID,
						).Scan(&readType, &readPayload)
						require.NoError(t, err)

						assert.Equal(t, eventType, readType)
						assert.JSONEq(t, storedPayload, readPayload)
					})
				}
			}
		})
	}
}

func TestRuleEventTypeNormalizationMigration(t *testing.T) {
	env := newEventTestEnv(t)
	migrations := make(map[string]string)
	for _, direction := range []string{"up", "down"} {
		migration, err := fs.ReadFile(migrate.OMMigrationsConfig.FS, path.Join(
			migrate.OMMigrationsConfig.FSPath,
			"20261124090200_notification_event_type_normalization."+direction+".sql",
		))
		require.NoError(t, err)

		migrations[direction] = string(migration)
	}

	// Compare raw PostgreSQL JSON text so annotations and snapshots retain large integers exactly.
	const snapshotSQL = `SELECT e.type, r.type, e.xmin::text, r.xmin::text, jsonb_build_object(
		'event', to_jsonb(e) - 'type', 'rule', to_jsonb(r) - 'type',
		'channels', (SELECT jsonb_agg(to_jsonb(c) ORDER BY c.id) FROM notification_channels c WHERE c.namespace = e.namespace),
		'rule_channels', (SELECT jsonb_agg(to_jsonb(rc) ORDER BY rc.notification_channel_id) FROM notification_channel_rules rc WHERE rc.notification_rule_id = r.id),
		'deliveries', (SELECT jsonb_agg(to_jsonb(d) ORDER BY d.id) FROM notification_event_delivery_status d WHERE d.event_id = e.id),
		'event_deliveries', (SELECT jsonb_agg(to_jsonb(ed) ORDER BY ed.notification_event_delivery_status_id) FROM notification_event_delivery_status_events ed WHERE ed.notification_event_id = e.id)
	)::text FROM notification_events e JOIN notification_rules r ON r.id = e.rule_id WHERE e.id = $1`

	for _, tc := range []struct {
		legacy    string
		versioned string
	}{
		{legacy: "entitlements.balance.threshold", versioned: "v1.entitlement.balance_threshold.triggered"},
		{legacy: "entitlements.reset", versioned: "v1.entitlement.usage_period.reset"},
		{legacy: "invoice.created", versioned: "v1.billing.invoice.created"},
		{legacy: "invoice.updated", versioned: "v1.billing.invoice.updated"},
		{legacy: "future.event", versioned: "future.event"},
	} {
		t.Run(tc.legacy, func(t *testing.T) {
			type snapshot struct {
				id, namespace, before string
				eventType, ruleType   string
				eventXmin, ruleXmin   string
			}

			// Seed through the adapter, then reproduce historical spellings and inactive Rules.
			seedPair := func(t *testing.T, storedRuleType, storedEventType string, disabled, deleted bool) snapshot {
				t.Helper()

				eventType := notification.EventType(tc.legacy)
				if tc.legacy == "future.event" {
					eventType = notification.EventTypeInvoiceCreated
				}

				namespace := ulid.Make().String()
				channel := seedChannel(t, env, namespace)
				rule := seedRule(t, env, namespace, eventType, channel.ID)
				payload := notification.EventPayload{
					EventPayloadMeta: notification.EventPayloadMeta{Type: eventType, Version: notification.EventPayloadVersionCurrent},
				}
				switch eventType {
				case notification.EventTypeBalanceThreshold:
					payload.BalanceThreshold = &notification.BalanceThresholdPayload{}
				case notification.EventTypeEntitlementReset:
					payload.EntitlementReset = &notification.EntitlementResetPayload{}
				case notification.EventTypeInvoiceCreated, notification.EventTypeInvoiceUpdated:
					payload.Invoice = &notification.InvoicePayload{}
				}

				event, err := env.adapter.CreateEvent(t.Context(), notification.CreateEventInput{
					NamespacedModel: models.NamespacedModel{Namespace: namespace},
					Type:            eventType, RuleID: rule.ID, Payload: payload,
				})
				require.NoError(t, err)

				// Historical Rules use the envelope; current writers use flat configs.
				_, err = env.sqlDB.ExecContext(t.Context(), `
					UPDATE notification_rules SET type = $1, disabled = $2,
						deleted_at = CASE WHEN $3 THEN '2026-01-02T00:00:00Z'::timestamptz ELSE NULL END,
						config = CASE WHEN $5 THEN jsonb_build_object('type', $6::text, 'data', config) ELSE config END,
						metadata = '{"team":"billing"}', annotations = '{"preserve":9007199254740993}'
					WHERE id = $4`, storedRuleType, disabled, deleted, rule.ID, storedRuleType == tc.legacy, string(eventType))
				require.NoError(t, err)

				_, err = env.sqlDB.ExecContext(t.Context(), `
					UPDATE notification_events SET type = $1, annotations = '{"preserve":9007199254740993}',
						rule_snapshot = '{"preserve":9007199254740993}', provider_data = '{"svix.message_id":"msg_preserve"}'
					WHERE id = $2`, storedEventType, event.ID)
				require.NoError(t, err)

				_, err = env.sqlDB.ExecContext(t.Context(), `
					UPDATE notification_event_delivery_status SET state = 'sending',
						next_attempt_at = '2026-01-03T00:00:00Z', dispatch_generation = 3,
						dispatch_queued_at = '2026-01-01T00:00:00Z', attempts = '[]',
						provider_data = '{"svix.message_id":"msg_preserve"}', annotations = '{"preserve":9007199254740993}'
					WHERE event_id = $1`, event.ID)
				require.NoError(t, err)

				result := snapshot{id: event.ID, namespace: namespace}
				err = env.sqlDB.QueryRowContext(t.Context(), snapshotSQL, event.ID).Scan(
					&result.eventType, &result.ruleType, &result.eventXmin, &result.ruleXmin, &result.before)
				require.NoError(t, err)

				return result
			}

			// given: mixed spellings, multiple namespaces, and disabled/deleted Rules
			snapshots := []snapshot{
				seedPair(t, tc.legacy, tc.legacy, false, false),
				seedPair(t, tc.versioned, tc.versioned, false, false),
				seedPair(t, tc.legacy, tc.legacy, true, false),
				seedPair(t, tc.legacy, tc.legacy, true, true),
			}

			if tc.legacy != tc.versioned {
				snapshots = append(snapshots,
					seedPair(t, tc.legacy, tc.versioned, false, false),
					seedPair(t, tc.versioned, tc.legacy, false, false),
				)
			}

			for _, step := range []struct {
				name, direction string
				createAfterUp   bool
			}{
				{name: "upgrade", direction: "up"},
				{name: "repeat upgrade with new records", direction: "up", createAfterUp: true},
				{name: "rollback", direction: "down"},
				{name: "repeat rollback", direction: "down"},
				{name: "upgrade after rollback", direction: "up"},
			} {
				t.Run(step.name, func(t *testing.T) {
					if step.createAfterUp {
						snapshots = append(snapshots, seedPair(t, tc.versioned, tc.versioned, false, false))
					}

					// when: applying both table updates through the embedded migration
					_, err := env.sqlDB.ExecContext(t.Context(), migrations[step.direction])
					require.NoError(t, err)

					wantType := tc.versioned
					if step.direction == "down" {
						wantType = tc.legacy
					}

					// then: only the two type columns change, including for newly created records
					for i := range snapshots {
						before := &snapshots[i]
						var eventType, ruleType, eventXmin, ruleXmin, after string
						err := env.sqlDB.QueryRowContext(t.Context(), snapshotSQL, before.id).Scan(
							&eventType, &ruleType, &eventXmin, &ruleXmin, &after)
						require.NoError(t, err)

						assert.Equal(t, wantType, eventType)
						assert.Equal(t, wantType, ruleType)
						assert.Equal(t, before.before, after)

						if before.eventType == wantType {
							assert.Equal(t, before.eventXmin, eventXmin)
						} else {
							assert.NotEqual(t, before.eventXmin, eventXmin)
						}

						if before.ruleType == wantType {
							assert.Equal(t, before.ruleXmin, ruleXmin)
						} else {
							assert.NotEqual(t, before.ruleXmin, ruleXmin)
						}

						before.eventType, before.ruleType = eventType, ruleType
						before.eventXmin, before.ruleXmin = eventXmin, ruleXmin
						if tc.legacy != "future.event" {
							fetched, err := env.adapter.GetEvent(t.Context(), notification.GetEventInput{
								Namespace: before.namespace, ID: before.id, Expand: &notification.EventExpand{DeliveryStatus: true},
							})
							require.NoError(t, err)

							assert.Equal(t, notification.EventType(tc.legacy), fetched.Type)
							assert.Equal(t, notification.EventType(tc.legacy), fetched.Rule.Type)
							assert.Equal(t, notification.EventType(tc.legacy), fetched.Payload.Type)
							assert.Len(t, fetched.Rule.Channels, 1)
							assert.Len(t, fetched.DeliveryStatus, 1)
						}
					}
				})
			}
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

					before, err := env.adapter.GetEvent(t.Context(), notification.GetEventInput{
						Namespace: created.Namespace,
						ID:        created.ID,
						Expand:    &notification.EventExpand{DeliveryStatus: true},
					})
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

					fetched, err := env.adapter.GetEvent(t.Context(), notification.GetEventInput{
						Namespace: created.Namespace,
						ID:        created.ID,
						Expand:    &notification.EventExpand{DeliveryStatus: true},
					})
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
			_, err = env.adapter.GetEvent(t.Context(), notification.GetEventInput{Namespace: created.Namespace, ID: created.ID})
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
				SetType(string(rule.Type)).
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
				SetType(string(rule.Type)).
				SetPayload(payload).
				SetRuleID(ulid.Make().String()).
				Save(t.Context())
			require.Error(t, err)

			assert.True(t, entdb.IsConstraintError(err))

			err = dbClient.NotificationRule.DeleteOneID(rule.ID).Exec(t.Context())
			require.ErrorContains(t, err, "notification_events_notification_rules_events")

			fetched, err := adapter.GetEvent(t.Context(), notification.GetEventInput{Namespace: legacy.Namespace, ID: legacy.ID})
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
						SetType(string(rule.Type)).
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
						SetState(string(notification.EventDeliveryStatusStatePending)).
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

					fetched, err := adapter.GetEvent(t.Context(), notification.GetEventInput{
						Namespace: namespace,
						ID:        entity.ID,
						Expand:    &notification.EventExpand{DeliveryStatus: true},
					})
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

					fetched, err := adapter.GetEvent(t.Context(), notification.GetEventInput{
						Namespace: event.Namespace,
						ID:        event.ID,
						Expand:    &notification.EventExpand{DeliveryStatus: true},
					})
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

func TestListEvents_MixedTypeSpellingsAcrossPages(t *testing.T) {
	env := newEventTestEnv(t)
	namespace := ulid.Make().String()
	channel := seedChannel(t, env, namespace)

	// given: all four public types with both legacy and versioned rows
	idsByType := make(map[string][]string)
	var allIDs []string
	for _, eventType := range []notification.EventType{notification.EventTypeBalanceThreshold, notification.EventTypeEntitlementReset, notification.EventTypeInvoiceCreated, notification.EventTypeInvoiceUpdated} {
		var ids []string
		for _, legacy := range []bool{true, false} {
			rule := seedRule(t, env, namespace, eventType, channel.ID)
			payload := notification.EventPayload{
				EventPayloadMeta: notification.EventPayloadMeta{Type: eventType, Version: notification.EventPayloadVersionCurrent},
			}
			switch eventType {
			case notification.EventTypeBalanceThreshold:
				payload.BalanceThreshold = &notification.BalanceThresholdPayload{}
			case notification.EventTypeEntitlementReset:
				payload.EntitlementReset = &notification.EntitlementResetPayload{}
			case notification.EventTypeInvoiceCreated, notification.EventTypeInvoiceUpdated:
				payload.Invoice = &notification.InvoicePayload{}
			}

			created, err := env.adapter.CreateEvent(t.Context(), notification.CreateEventInput{
				NamespacedModel: models.NamespacedModel{Namespace: namespace},
				Type:            eventType, RuleID: rule.ID, Payload: payload,
			})
			require.NoError(t, err)

			if legacy {
				_, err := env.sqlDB.ExecContext(t.Context(),
					"UPDATE notification_events SET type = $1 WHERE id = $2", string(eventType), created.ID)
				require.NoError(t, err)
			}

			ids = append(ids, created.ID)
		}

		slices.Sort(ids)
		idsByType[string(eventType)] = ids
		allIDs = append(allIDs, ids...)
	}

	excludingInvoiceCreated := lo.Filter(allIDs, func(id string, _ int) bool {
		return !slices.Contains(idsByType["invoice.created"], id)
	})

	tests := []struct {
		name   string
		filter *filter.FilterString
		want   []string
	}{
		{name: "all", want: allIDs},
		{name: "equal", filter: &filter.FilterString{Eq: new("invoice.created")}, want: idsByType["invoice.created"]},
		{name: "not equal", filter: &filter.FilterString{Ne: new("invoice.created")}, want: excludingInvoiceCreated},
		{
			name:   "in",
			filter: &filter.FilterString{In: &[]string{"entitlements.reset", "invoice.created"}},
			want:   slices.Concat(idsByType["entitlements.reset"], idsByType["invoice.created"]),
		},
		{name: "not in", filter: &filter.FilterString{Nin: &[]string{"invoice.created"}}, want: excludingInvoiceCreated},
	}
	for _, stage := range []struct {
		name, direction string
	}{
		{name: "mixed spellings"},
		{name: "after upgrade", direction: "up"},
		{name: "after rollback", direction: "down"},
		{name: "after re-upgrade", direction: "up"},
	} {
		t.Run(stage.name, func(t *testing.T) {
			if stage.direction != "" {
				migration, err := fs.ReadFile(migrate.OMMigrationsConfig.FS, path.Join(
					migrate.OMMigrationsConfig.FSPath,
					"20261124090200_notification_event_type_normalization."+stage.direction+".sql",
				))
				require.NoError(t, err)

				_, err = env.sqlDB.ExecContext(t.Context(), string(migration))
				require.NoError(t, err)
			}

			for _, tt := range tests {
				t.Run(tt.name, func(t *testing.T) {
					for _, order := range []sortx.Order{sortx.OrderAsc, sortx.OrderDesc} {
						t.Run(string(order), func(t *testing.T) {
							want := slices.Clone(tt.want)
							if order == sortx.OrderDesc {
								slices.Reverse(want)
							}

							// when: each page contains one row ordered by public type and ID
							var got []string
							for page := 1; page <= len(want)+1; page++ {
								listed, err := env.adapter.ListEvents(t.Context(), notification.ListEventsInput{
									Namespaces: []string{namespace},
									Type:       tt.filter, OrderBy: notification.OrderByType, Order: order,
									Page: pagination.NewPage(page, 1),
								})
								require.NoError(t, err)

								for _, item := range listed.Items {
									got = append(got, item.ID)
								}
							}

							// then: pagination includes each matching row once, regardless of storage spelling
							assert.Equal(t, want, got)
						})
					}
				})
			}
		})
	}
}

func TestEventPayloadDualReader(t *testing.T) {
	env := newEventTestEnv(t)
	fixturesJSON, err := os.ReadFile("../../../api/v3/handlers/notification/events/testdata/legacy_payloads.json")
	require.NoError(t, err)

	var fixtures map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(fixturesJSON, &fixtures))

	cases := []struct {
		eventType notification.EventType
		prefix    string
		field     string
	}{
		{
			eventType: notification.EventTypeInvoiceCreated,
			prefix:    "invoice_",
			field:     "invoice",
		},
		{
			eventType: notification.EventTypeInvoiceUpdated,
			prefix:    "invoice_",
			field:     "invoice",
		},
		{
			eventType: notification.EventTypeBalanceThreshold,
			prefix:    "balance_threshold_",
			field:     "balanceThreshold",
		},
		{
			eventType: notification.EventTypeEntitlementReset,
			prefix:    "entitlement_reset_",
			field:     "entitlementReset",
		},
	}

	for _, tc := range cases {
		t.Run(string(tc.eventType), func(t *testing.T) {
			for name, raw := range fixtures {
				if !strings.HasPrefix(name, tc.prefix) {
					continue
				}

				t.Run(name, func(t *testing.T) {
					// Given a historical snapshot with every supported property and union variant.
					ns := ulid.Make().String()
					channel := seedChannel(t, env, ns)
					rule := seedRule(t, env, ns, tc.eventType, channel.ID)
					envelope, err := json.Marshal(map[string]any{
						"type": tc.eventType, "version": notification.EventPayloadVersionCurrent, tc.field: raw,
					})
					require.NoError(t, err)

					var payload notification.EventPayload
					require.NoError(t, json.Unmarshal(envelope, &payload))

					created, err := env.adapter.CreateEvent(t.Context(), notification.CreateEventInput{
						NamespacedModel: models.NamespacedModel{Namespace: ns},
						Type:            tc.eventType, RuleID: rule.ID, Payload: payload,
					})
					require.NoError(t, err)

					// PostgreSQL stores timestamps at microsecond precision. Keep the original
					// payload as the expectation, but use persisted Event and delivery metadata.
					expected, err := env.adapter.GetEvent(t.Context(), notification.GetEventInput{
						Namespace: ns,
						ID:        created.ID,
						Expand:    &notification.EventExpand{DeliveryStatus: true},
					})
					require.NoError(t, err)

					expected.Payload = payload

					wantV1, err := httpdriver.FromEvent(*expected)
					require.NoError(t, err)

					wantV3, err := apievents.ToAPIBillingNotificationEvent(*expected)
					require.NoError(t, err)

					wireJSON, err := json.Marshal(wantV3.Payload)
					require.NoError(t, err)

					var wire struct {
						Data json.RawMessage `json:"data"`
					}
					require.NoError(t, json.Unmarshal(wireJSON, &wire))

					for _, storage := range []struct {
						name    string
						payload json.RawMessage
					}{
						{
							name:    "legacy envelope",
							payload: envelope,
						},
						{
							name:    "canonical bare payload",
							payload: wire.Data,
						},
					} {
						t.Run(storage.name, func(t *testing.T) {
							// When either representation is stored, reads must not change the snapshot.
							_, err := env.db.NotificationEvent.UpdateOneID(created.ID).SetPayload(storage.payload).Save(t.Context())
							require.NoError(t, err)

							got, err := env.adapter.GetEvent(t.Context(), notification.GetEventInput{Namespace: ns, ID: created.ID, Expand: &notification.EventExpand{DeliveryStatus: true}})
							require.NoError(t, err)

							gotV1, err := httpdriver.FromEvent(*got)
							require.NoError(t, err)

							gotV3, err := apievents.ToAPIBillingNotificationEvent(*got)
							require.NoError(t, err)

							// Then both APIs and the webhook mapper retain the same content and Event metadata.
							wantV1JSON, err := json.Marshal(wantV1)
							require.NoError(t, err)

							gotV1JSON, err := json.Marshal(gotV1)
							require.NoError(t, err)

							wantV3JSON, err := json.Marshal(wantV3)
							require.NoError(t, err)

							gotV3JSON, err := json.Marshal(gotV3)
							require.NoError(t, err)

							assert.JSONEq(t, string(wantV1JSON), string(gotV1JSON))
							assert.JSONEq(t, string(wantV3JSON), string(gotV3JSON))

							row, err := env.db.NotificationEvent.Get(t.Context(), created.ID)
							require.NoError(t, err)

							assert.JSONEq(t, string(storage.payload), string(row.Payload))
						})
					}
				})
			}
		})
	}
}
