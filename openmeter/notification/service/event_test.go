package service_test

import (
	"testing"
	"time"

	"github.com/oklog/ulid/v2"
	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

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

// TestListEvents_ServiceValidation covers the filter validation ListEventsInput gained
// alongside the v3 list endpoint. The service calls params.Validate() before reaching
// the adapter, so a malformed filter must fail here rather than producing a query.
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
