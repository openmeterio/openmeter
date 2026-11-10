package service_test

import (
	"context"
	"testing"
	"time"

	"github.com/oklog/ulid/v2"
	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	eventdb "github.com/openmeterio/openmeter/openmeter/ent/db/notificationevent"
	statusdb "github.com/openmeterio/openmeter/openmeter/ent/db/notificationeventdeliverystatus"
	"github.com/openmeterio/openmeter/openmeter/notification"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/filter"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/pagination"
)

// seedBalanceThresholdRule creates a rule targeting the given channels through the
// adapter, for the same reason seedChannel bypasses the service: rule creation itself is
// service-safe, but keeping both on the adapter keeps the seeding path uniform.
func seedBalanceThresholdRule(t *testing.T, env *serviceTestEnv, ns string, channelIDs ...string) notification.Rule {
	t.Helper()

	rule, err := env.adapter.CreateRule(t.Context(), notification.CreateRuleInput{
		NamespacedModel: models.NamespacedModel{Namespace: ns},
		Type:            notification.EventTypeBalanceThreshold,
		Name:            "rule-" + ulid.Make().String(),
		Channels:        channelIDs,
		Config: notification.RuleConfig{
			RuleConfigMeta: notification.RuleConfigMeta{Type: notification.EventTypeBalanceThreshold},
			BalanceThreshold: &notification.BalanceThresholdRuleConfig{
				Thresholds: []notification.BalanceThreshold{
					{Type: notification.BalanceThresholdTypeUsagePercentage, Value: 50},
				},
			},
		},
	})
	require.NoError(t, err, "seeding rule must not fail")
	require.NotNil(t, rule)

	return *rule
}

func seedEvent(t *testing.T, env *serviceTestEnv, ns string, rule notification.Rule) notification.Event {
	t.Helper()

	event, err := env.service.CreateEvent(t.Context(), notification.CreateEventInput{
		NamespacedModel: models.NamespacedModel{Namespace: ns},
		Type:            rule.Type,
		RuleID:          rule.ID,
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

func TestListEvents_ServiceValidation(t *testing.T) {
	env := newServiceTestEnv(t)
	ns := ulid.Make().String()

	testCases := []struct {
		name  string
		input notification.ListEventsInput
	}{
		{
			name: "two operators on one filter node",
			input: notification.ListEventsInput{
				Type: &filter.FilterString{
					Eq: lo.ToPtr("invoice.created"),
					Ne: lo.ToPtr("invoice.updated"),
				},
			},
		},
		{
			name: "malformed ULID in the id filter",
			input: notification.ListEventsInput{
				ID: &filter.FilterULID{FilterString: filter.FilterString{Eq: lo.ToPtr("not-a-ulid")}},
			},
		},
		{
			name: "unsupported order by",
			input: notification.ListEventsInput{
				OrderBy: notification.OrderByUpdatedAt,
			},
		},
		{
			name: "deprecated period start after its end",
			input: notification.ListEventsInput{
				From: clock.Now(),
				To:   clock.Now().Add(-time.Hour),
			},
		},
		{
			name: "negated channel_id filter",
			input: notification.ListEventsInput{
				ChannelID: &filter.FilterString{Ne: lo.ToPtr(ulid.Make().String())},
			},
		},
		{
			name: "negated delivery_status filter",
			input: notification.ListEventsInput{
				DeliveryStatus: &filter.FilterString{Nin: lo.ToPtr([]string{"FAILED"})},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			input := tc.input
			input.Namespaces = []string{ns}
			input.Page = pagination.NewPage(1, 20)

			_, err := env.service.ListEvents(t.Context(), input)
			require.Error(t, err)
			assert.True(t, models.IsGenericValidationError(err), "expected a validation error, got: %v", err)
		})
	}
}

// TestListEvents_WithoutNamespaceIsAllowed pins the deliberate difference from
// ListChannelsInput: the delivery reconciliation loop lists events across every tenant
// to find deliveries due for retry, so an unscoped list must stay valid.
func TestListEvents_WithoutNamespaceIsAllowed(t *testing.T) {
	env := newServiceTestEnv(t)
	ns := ulid.Make().String()

	channel := seedChannel(t, env, ns, clock.Now(), "channel-"+ulid.Make().String())
	rule := seedBalanceThresholdRule(t, env, ns, channel.ID)
	event := seedEvent(t, env, ns, rule)

	result, err := env.service.ListEvents(t.Context(), notification.ListEventsInput{
		Page: pagination.NewPage(1, 20),
	})
	require.NoError(t, err)

	ids := make([]string, 0, len(result.Items))
	for _, item := range result.Items {
		ids = append(ids, item.ID)
	}

	assert.Contains(t, ids, event.ID)
}

// TestResendEvent covers the action behind POST /notification/events/{id}/resend: an
// eligible delivery status moves to RESENDING and its next attempt is cleared so the
// delivery worker picks it up immediately.
func TestResendEvent(t *testing.T) {
	env := newServiceTestEnv(t)
	ns := ulid.Make().String()

	channel := seedChannel(t, env, ns, clock.Now(), "channel-"+ulid.Make().String())
	rule := seedBalanceThresholdRule(t, env, ns, channel.ID)
	event := seedEvent(t, env, ns, rule)

	statuses, err := env.service.ListEventsDeliveryStatus(t.Context(), notification.ListEventsDeliveryStatusInput{
		Namespaces: []string{ns},
		Events:     []string{event.ID},
		Page:       pagination.NewPage(1, 20),
	})
	require.NoError(t, err)
	require.Len(t, statuses.Items, 1)

	// A freshly created status is PENDING, which resend skips; move it to FAILED first
	// so there is something eligible to re-send.
	_, err = env.service.UpdateEventDeliveryStatus(t.Context(), notification.UpdateEventDeliveryStatusInput{
		NamespacedID: models.NamespacedID{Namespace: ns, ID: statuses.Items[0].ID},
		State:        notification.EventDeliveryStatusStateFailed,
		Reason:       "delivery failed",
		NextAttempt:  lo.ToPtr(clock.Now().Add(time.Hour)),
	})
	require.NoError(t, err)

	t.Run("an unknown channel is rejected", func(t *testing.T) {
		err := env.service.ResendEvent(t.Context(), notification.ResendEventInput{
			NamespacedID: models.NamespacedID{Namespace: ns, ID: event.ID},
			Channels:     []string{ulid.Make().String()},
		})
		require.Error(t, err)
		assert.True(t, models.IsGenericValidationError(err), "expected a validation error, got: %v", err)
	})

	t.Run("an eligible status moves to resending", func(t *testing.T) {
		err := env.service.ResendEvent(t.Context(), notification.ResendEventInput{
			NamespacedID: models.NamespacedID{Namespace: ns, ID: event.ID},
		})
		require.NoError(t, err)

		status, err := env.service.GetEventDeliveryStatus(t.Context(), notification.GetEventDeliveryStatusInput{
			NamespacedID: models.NamespacedID{Namespace: ns, ID: statuses.Items[0].ID},
		})
		require.NoError(t, err)
		require.NotNil(t, status)

		assert.Equal(t, notification.EventDeliveryStatusStateResending, status.State)
		assert.Nil(t, status.NextAttempt, "resend must clear the scheduled next attempt")
	})
}

func TestDispatchQueuedAtPreservation(t *testing.T) {
	env := newServiceTestEnv(t)
	queued := time.Date(2026, time.November, 1, 10, 0, 0, 0, time.UTC)
	tests := []struct {
		name     string
		queuedAt *time.Time
	}{
		{name: "legacy delivery"},
		{name: "delivery with a recorded queue time", queuedAt: &queued},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// given: a delivery created through the current service
			ns := ulid.Make().String()
			channel := seedChannel(t, env, ns, clock.Now(), "channel-"+ulid.Make().String())
			rule := seedBalanceThresholdRule(t, env, ns, channel.ID)
			event := seedEvent(t, env, ns, rule)
			require.Len(t, event.DeliveryStatus, 1)

			id := event.DeliveryStatus[0].NamespacedID
			deliveries := env.db.NotificationEventDeliveryStatus
			row, err := deliveries.Get(t.Context(), id.ID)
			require.NoError(t, err)
			require.Nil(t, row.DispatchQueuedAt)

			if tt.queuedAt != nil {
				err = deliveries.UpdateOneID(id.ID).SetDispatchQueuedAt(*tt.queuedAt).Exec(t.Context())
				require.NoError(t, err)
			}

			assertDispatchQueuedAt := func(got *time.Time) {
				t.Helper()

				if tt.queuedAt == nil {
					require.Nil(t, got)
					return
				}

				require.NotNil(t, got)

				assert.WithinDuration(t, *tt.queuedAt, *got, 0)
			}

			// when: reconciliation writes a report with a next attempt and the caller later requests a resend
			next := queued.Add(3 * time.Hour)
			_, err = env.service.UpdateEventDeliveryStatus(t.Context(), notification.UpdateEventDeliveryStatusInput{
				NamespacedID: id,
				State:        notification.EventDeliveryStatusStateFailed,
				Reason:       "delivery failed",
				Attempts: []notification.EventDeliveryAttempt{
					{State: notification.EventDeliveryStatusStateFailed, Timestamp: queued.Add(time.Hour)},
				},
				NextAttempt: &next,
			})
			require.NoError(t, err)

			row, err = deliveries.Get(t.Context(), id.ID)
			require.NoError(t, err)
			require.NotNil(t, row.NextAttemptAt)

			assert.WithinDuration(t, next, *row.NextAttemptAt, 0)
			assertDispatchQueuedAt(row.DispatchQueuedAt)

			err = env.service.ResendEvent(t.Context(), notification.ResendEventInput{
				NamespacedID: event.NamespacedID,
				Channels:     []string{channel.ID},
			})
			require.NoError(t, err)

			// then: resend clears the retry schedule and preserves the stored queue time
			row, err = deliveries.Get(t.Context(), id.ID)
			require.NoError(t, err)

			assert.Equal(t, notification.EventDeliveryStatusStateResending, row.State)
			assert.Nil(t, row.NextAttemptAt)
			assertDispatchQueuedAt(row.DispatchQueuedAt)
		})
	}
}

func TestResendEvent_SkipsChannelDisabledAfterDelivery(t *testing.T) {
	env := newServiceTestEnv(t)
	ns := ulid.Make().String()

	// given: a rule with two channels, an event delivered (and failed) on both, and one
	// channel disabled afterwards
	active := seedChannel(t, env, ns, clock.Now(), "active-"+ulid.Make().String())
	disabled := seedChannel(t, env, ns, clock.Now(), "disabled-"+ulid.Make().String())
	rule := seedBalanceThresholdRule(t, env, ns, active.ID, disabled.ID)
	event := seedEvent(t, env, ns, rule)

	statuses, err := env.service.ListEventsDeliveryStatus(t.Context(), notification.ListEventsDeliveryStatusInput{
		Namespaces: []string{ns},
		Events:     []string{event.ID},
		Page:       pagination.NewPage(1, 20),
	})
	require.NoError(t, err)
	require.Len(t, statuses.Items, 2)

	for _, status := range statuses.Items {
		_, err = env.service.UpdateEventDeliveryStatus(t.Context(), notification.UpdateEventDeliveryStatusInput{
			NamespacedID: models.NamespacedID{Namespace: ns, ID: status.ID},
			State:        notification.EventDeliveryStatusStateFailed,
			Reason:       "delivery failed",
		})
		require.NoError(t, err)
	}

	_, err = env.adapter.UpdateChannel(t.Context(), notification.UpdateChannelInput{
		NamespacedID: models.NamespacedID{Namespace: ns, ID: disabled.ID},
		Type:         disabled.Type,
		Name:         disabled.Name,
		Disabled:     true,
		Config:       disabled.Config,
	})
	require.NoError(t, err)

	// when: the event is resent without naming channels
	err = env.service.ResendEvent(t.Context(), notification.ResendEventInput{
		NamespacedID: models.NamespacedID{Namespace: ns, ID: event.ID},
	})
	require.NoError(t, err)

	// then: only the active channel's status is marked for redelivery
	statuses, err = env.service.ListEventsDeliveryStatus(t.Context(), notification.ListEventsDeliveryStatusInput{
		Namespaces: []string{ns},
		Events:     []string{event.ID},
		Page:       pagination.NewPage(1, 20),
	})
	require.NoError(t, err)

	stateByChannel := lo.SliceToMap(statuses.Items, func(s notification.EventDeliveryStatus) (string, notification.EventDeliveryStatusState) {
		return s.ChannelID, s.State
	})
	assert.Equal(t, notification.EventDeliveryStatusStateResending, stateByChannel[active.ID])
	assert.Equal(t, notification.EventDeliveryStatusStateFailed, stateByChannel[disabled.ID])
}

func TestCreateEventRequiresRule(t *testing.T) {
	env := newServiceTestEnv(t)
	paths := []struct {
		name   string
		create func(context.Context, notification.CreateEventInput) (*notification.Event, error)
	}{
		{name: "service", create: env.service.CreateEvent},
		{name: "adapter", create: env.adapter.CreateEvent},
	}
	tests := []struct {
		name   string
		ruleID string
		valid  bool
	}{
		{name: "omitted Rule"},
		{name: "unknown Rule", ruleID: ulid.Make().String()},
		{name: "existing Rule", valid: true},
	}

	for _, path := range paths {
		t.Run(path.name, func(t *testing.T) {
			for _, tt := range tests {
				t.Run(tt.name, func(t *testing.T) {
					// given: a nullable Rule column, but unchanged legacy creation requirements
					namespace := ulid.Make().String()
					channel := seedChannel(t, env, namespace, clock.Now(), "test-channel")
					rule := seedBalanceThresholdRule(t, env, namespace, channel.ID)
					ruleID := tt.ruleID
					if tt.valid {
						ruleID = rule.ID
					}

					// when: creating through the service or directly through the adapter
					event, err := path.create(t.Context(), notification.CreateEventInput{
						NamespacedModel: models.NamespacedModel{Namespace: namespace},
						Type:            rule.Type,
						RuleID:          ruleID,
						Payload: notification.EventPayload{
							EventPayloadMeta: notification.EventPayloadMeta{
								Type:    rule.Type,
								Version: notification.EventPayloadVersionCurrent,
							},
							BalanceThreshold: &notification.BalanceThresholdPayload{},
						},
					})

					// then: only an existing Rule permits Event and delivery creation
					wantCount := 0
					if tt.valid {
						require.NoError(t, err)
						require.NotNil(t, event)

						assert.Equal(t, rule.ID, event.Rule.ID)

						require.Len(t, event.DeliveryStatus, 1)

						assert.Equal(t, channel.ID, event.DeliveryStatus[0].ChannelID)

						wantCount = 1
					} else {
						require.Error(t, err)

						assert.Nil(t, event)
					}

					eventCount, err := env.db.NotificationEvent.Query().Where(eventdb.Namespace(namespace)).Count(t.Context())
					require.NoError(t, err)

					statusCount, err := env.db.NotificationEventDeliveryStatus.Query().Where(statusdb.Namespace(namespace)).Count(t.Context())
					require.NoError(t, err)

					assert.Equal(t, wantCount, eventCount)
					assert.Equal(t, wantCount, statusCount)
				})
			}
		})
	}
}
