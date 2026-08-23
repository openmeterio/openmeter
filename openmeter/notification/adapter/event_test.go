package adapter_test

import (
	"testing"
	"time"

	"github.com/oklog/ulid/v2"
	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	entdb "github.com/openmeterio/openmeter/openmeter/ent/db"
	"github.com/openmeterio/openmeter/openmeter/notification"
	notificationadapter "github.com/openmeterio/openmeter/openmeter/notification/adapter"
	"github.com/openmeterio/openmeter/openmeter/testutils"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/filter"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/pagination"
	"github.com/openmeterio/openmeter/pkg/sortx"
)

// These tests exercise the v3 list-filter wiring added to ListEvents
// (openmeter/notification/adapter/event.go) directly against the adapter, for the same
// reason channel_test.go does: notification/service.Service cannot create channels
// outside svix, and events need a rule with channels to exist at all.
type eventTestEnv struct {
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

	return &eventTestEnv{db: dbClient, adapter: adapter}
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

	first := seedEvent(t, env, ns, ruleA, t0, models.Annotations{
		notification.AnnotationEventSubjectKey: "subject-1",
		notification.AnnotationEventFeatureKey: "feature-1",
	})
	second := seedEvent(t, env, ns, ruleA, t1, models.Annotations{
		notification.AnnotationEventSubjectID: "subject-2-id",
		notification.AnnotationEventFeatureID: "feature-2-id",
	})
	third := seedEvent(t, env, ns, ruleB, frozen, nil)

	// Move one of ruleA's delivery statuses to FAILED so the delivery-state filter has
	// something to distinguish; everything else stays PENDING as created.
	firstStatuses, err := env.adapter.ListEventsDeliveryStatus(t.Context(), notification.ListEventsDeliveryStatusInput{
		Namespaces: []string{ns},
		Events:     []string{first.ID},
		Page:       pagination.NewPage(1, 20),
	})
	require.NoError(t, err)
	require.Len(t, firstStatuses.Items, 1)

	_, err = env.adapter.UpdateEventDeliveryStatus(t.Context(), notification.UpdateEventDeliveryStatusInput{
		NamespacedID: models.NamespacedID{Namespace: ns, ID: firstStatuses.Items[0].ID},
		State:        notification.EventDeliveryStatusStateFailed,
		Reason:       "test",
	})
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
			name: "Subject eq matches the subject key annotation",
			input: notification.ListEventsInput{
				Subject: &filter.FilterString{Eq: lo.ToPtr("subject-1")},
			},
			wantIDs: []string{first.ID},
		},
		{
			name: "Subject eq also matches the subject id annotation",
			input: notification.ListEventsInput{
				Subject: &filter.FilterString{Eq: lo.ToPtr("subject-2-id")},
			},
			wantIDs: []string{second.ID},
		},
		{
			name: "Subject in matches every listed subject",
			input: notification.ListEventsInput{
				Subject: &filter.FilterString{In: lo.ToPtr([]string{"subject-1", "subject-2-id"})},
			},
			wantIDs: []string{first.ID, second.ID},
		},
		{
			name: "Feature eq matches the feature key annotation",
			input: notification.ListEventsInput{
				Feature: &filter.FilterString{Eq: lo.ToPtr("feature-1")},
			},
			wantIDs: []string{first.ID},
		},
		{
			name: "Feature eq also matches the feature id annotation",
			input: notification.ListEventsInput{
				Feature: &filter.FilterString{Eq: lo.ToPtr("feature-2-id")},
			},
			wantIDs: []string{second.ID},
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
			}
			assert.ElementsMatch(t, tc.wantIDs, gotIDs)
		})
	}
}

// TestListEvents_AnnotationFilterRejectsInexactOperators pins the adapter's refusal to
// silently widen a subject/feature filter it cannot express against the JSONB column.
// Returning every row for filter[subject][contains]=... would look like a successful
// query while answering a different question.
func TestListEvents_AnnotationFilterRejectsInexactOperators(t *testing.T) {
	env := newEventTestEnv(t)
	ns := ulid.Make().String()

	testCases := []struct {
		name  string
		input notification.ListEventsInput
	}{
		{
			name: "subject contains",
			input: notification.ListEventsInput{
				Subject: &filter.FilterString{Contains: lo.ToPtr("subject")},
			},
		},
		{
			name: "subject neq",
			input: notification.ListEventsInput{
				Subject: &filter.FilterString{Ne: lo.ToPtr("subject-1")},
			},
		},
		{
			name: "feature neq",
			input: notification.ListEventsInput{
				Feature: &filter.FilterString{Ne: lo.ToPtr("feature-1")},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			input := tc.input
			input.Namespaces = []string{ns}
			input.Page = pagination.NewPage(1, 20)

			_, err := env.adapter.ListEvents(t.Context(), input)
			require.Error(t, err)
			assert.Contains(t, err.Error(), "only supports equality and set membership")
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
