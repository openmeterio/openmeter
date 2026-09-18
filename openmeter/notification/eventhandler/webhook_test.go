package eventhandler

import (
	"context"
	"log/slog"
	"testing"
	"time"

	"cirello.io/pglock"
	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/require"
	oteltracenoop "go.opentelemetry.io/otel/trace/noop"

	"github.com/openmeterio/openmeter/openmeter/notification"
	notificationadapter "github.com/openmeterio/openmeter/openmeter/notification/adapter"
	"github.com/openmeterio/openmeter/openmeter/notification/webhook"
	webhooknoop "github.com/openmeterio/openmeter/openmeter/notification/webhook/noop"
	"github.com/openmeterio/openmeter/openmeter/testutils"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/models"
)

// fakeWebhookHandler overrides ListWebhooks and GetMessage on top of the no-op webhook.Handler
// so the reconciler can be driven with provider responses without a live Svix server.
type fakeWebhookHandler struct {
	*webhooknoop.Handler

	listWebhooksFn func(ctx context.Context, params webhook.ListWebhooksInput) ([]webhook.Webhook, error)
	getMessageFn   func(ctx context.Context, params webhook.GetMessageInput) (*webhook.Message, error)
}

func (h *fakeWebhookHandler) ListWebhooks(ctx context.Context, params webhook.ListWebhooksInput) ([]webhook.Webhook, error) {
	return h.listWebhooksFn(ctx, params)
}

func (h *fakeWebhookHandler) GetMessage(ctx context.Context, params webhook.GetMessageInput) (*webhook.Message, error) {
	return h.getMessageFn(ctx, params)
}

// newTestHandler wires a Handler backed by a real Postgres-backed repository and the given
// fake webhook provider responses. It calls t.Skip via testutils.InitPostgresDB when
// POSTGRES_HOST is unset.
func newTestHandler(t *testing.T, fake *fakeWebhookHandler) (*Handler, notification.Repository) {
	t.Helper()

	testDB := testutils.InitPostgresDB(t, testutils.PostgresDBStateEntMigrated)
	t.Cleanup(func() { testDB.Close(t) })

	logger := slog.New(slog.DiscardHandler)

	repo, err := notificationadapter.New(notificationadapter.Config{
		Client: testDB.EntDriver.Client(),
		Logger: logger,
	})
	require.NoError(t, err)

	// The lock client is never exercised: tests call reconcileWebhookEvent directly,
	// bypassing the leader lock used by Reconcile/Start.
	lockClient, err := pglock.UnsafeNew(testDB.PGDriver.DB())
	require.NoError(t, err)

	fake.Handler = webhooknoop.New(logger)

	h, err := New(Config{
		Repository: repo,
		Webhook:    fake,
		Logger:     logger,
		Tracer:     oteltracenoop.NewTracerProvider().Tracer("test"),
		LockClient: lockClient,
	})
	require.NoError(t, err)

	return h, repo
}

// createWebhookChannel creates a webhook channel in the given disabled state and with the given annotations.
func createWebhookChannel(t *testing.T, ctx context.Context, repo notification.Repository, namespace string, disabled bool, annotations models.Annotations) *notification.Channel {
	t.Helper()

	channel, err := repo.CreateChannel(ctx, notification.CreateChannelInput{
		NamespacedModel: models.NamespacedModel{Namespace: namespace},
		Type:            notification.ChannelTypeWebhook,
		Name:            "test channel",
		Disabled:        disabled,
		Config: notification.ChannelConfig{
			ChannelConfigMeta: notification.ChannelConfigMeta{Type: notification.ChannelTypeWebhook},
			WebHook: notification.WebHookChannelConfig{
				URL: "https://example.com/webhook",
			},
		},
		Annotations: annotations,
	})
	require.NoError(t, err)

	return channel
}

// createWebhookEventSending creates a rule bound to the given channel and a balance threshold
// event on it, then moves the resulting delivery status for the channel into SENDING state.
// It returns the event (re-fetched so Rule.Channels is populated) and the SENDING delivery status.
func createWebhookEventSending(t *testing.T, ctx context.Context, repo notification.Repository, namespace string, channelID string) (*notification.Event, notification.EventDeliveryStatus) {
	t.Helper()

	rule, err := repo.CreateRule(ctx, notification.CreateRuleInput{
		NamespacedModel: models.NamespacedModel{Namespace: namespace},
		Type:            notification.EventTypeBalanceThreshold,
		Name:            "test rule",
		Config: notification.RuleConfig{
			RuleConfigMeta: notification.RuleConfigMeta{Type: notification.EventTypeBalanceThreshold},
			BalanceThreshold: &notification.BalanceThresholdRuleConfig{
				Features: []string{"feature-1"},
				Thresholds: []notification.BalanceThreshold{
					{Type: notification.BalanceThresholdTypeNumber, Value: 100},
				},
			},
		},
		Channels: []string{channelID},
	})
	require.NoError(t, err)

	event, err := repo.CreateEvent(ctx, notification.CreateEventInput{
		NamespacedModel: models.NamespacedModel{Namespace: namespace},
		Type:            notification.EventTypeBalanceThreshold,
		RuleID:          rule.ID,
		Payload: notification.EventPayload{
			EventPayloadMeta: notification.EventPayloadMeta{Type: notification.EventTypeBalanceThreshold},
			BalanceThreshold: &notification.BalanceThresholdPayload{},
		},
	})
	require.NoError(t, err)
	require.Len(t, event.DeliveryStatus, 1)

	status := event.DeliveryStatus[0]

	_, err = repo.UpdateEventDeliveryStatus(ctx, notification.UpdateEventDeliveryStatusInput{
		NamespacedID: status.NamespacedID,
		State:        notification.EventDeliveryStatusStateSending,
		Annotations:  status.Annotations,
		Attempts:     status.Attempts,
	})
	require.NoError(t, err)

	// Re-fetch: CreateEvent's response does not eager load Rule.Channels the same way GetEvent does,
	// and reconcileWebhookEvent relies on event.Rule.Channels to know the channel type.
	refetched, err := repo.GetEvent(ctx, notification.GetEventInput{Namespace: namespace, ID: event.ID})
	require.NoError(t, err)

	refetchedStatus, err := repo.GetEventDeliveryStatus(ctx, notification.GetEventDeliveryStatusInput{
		NamespacedID: models.NamespacedID{Namespace: namespace, ID: status.ID},
	})
	require.NoError(t, err)

	return refetched, *refetchedStatus
}

func TestReconcileWebhookEventDisablesChannel(t *testing.T) {
	namespace := ulid.Make().String()

	now := time.Now().UTC().Truncate(time.Second)
	clock.FreezeTime(now)
	defer clock.UnFreeze()

	fake := &fakeWebhookHandler{}
	h, repo := newTestHandler(t, fake)

	ctx := t.Context()

	t.Run("provider disabled endpoint disables channel and fails delivery status", func(t *testing.T) {
		// given: an enabled webhook channel with an event whose delivery status is SENDING
		channel := createWebhookChannel(t, ctx, repo, namespace, false, nil)
		event, status := createWebhookEventSending(t, ctx, repo, namespace, channel.ID)

		fake.listWebhooksFn = func(ctx context.Context, params webhook.ListWebhooksInput) ([]webhook.Webhook, error) {
			return []webhook.Webhook{
				{
					ID:       channel.ID,
					Disabled: true,
					Channels: []string{event.Rule.ID},
				},
			}, nil
		}
		fake.getMessageFn = func(ctx context.Context, params webhook.GetMessageInput) (*webhook.Message, error) {
			return &webhook.Message{
				// Timestamp must not be after the delivery status' UpdatedAt, otherwise
				// reconciliation is skipped waiting for the provider to populate delivery statuses.
				Timestamp: now.Add(-time.Minute),
			}, nil
		}

		// when: the webhook event is reconciled
		err := h.reconcileWebhookEvent(ctx, event)
		require.NoError(t, err)

		// then: the delivery status is FAILED with the system channel disabled reason
		updatedStatus, err := repo.GetEventDeliveryStatus(ctx, notification.GetEventDeliveryStatusInput{
			NamespacedID: models.NamespacedID{Namespace: namespace, ID: status.ID},
		})
		require.NoError(t, err)
		require.Equal(t, notification.EventDeliveryStatusStateFailed, updatedStatus.State)
		require.Equal(t, ErrSystemChannelDisabled.Error(), updatedStatus.Reason)

		// then: the channel is disabled and stamped with the provider-disabled annotation
		updatedChannel, err := repo.GetChannel(ctx, notification.GetChannelInput{Namespace: namespace, ID: channel.ID})
		require.NoError(t, err)
		require.True(t, updatedChannel.Disabled)
		require.Equal(t, now.Format(time.RFC3339), updatedChannel.Annotations[notification.AnnotationChannelProviderDisabledTimestamp])
	})
}

func TestDisableChannelForProvider(t *testing.T) {
	namespace := ulid.Make().String()

	now := time.Now().UTC().Truncate(time.Second)
	clock.FreezeTime(now)
	defer clock.UnFreeze()

	h, repo := newTestHandler(t, &fakeWebhookHandler{})

	ctx := t.Context()

	t.Run("existing annotations survive the merge", func(t *testing.T) {
		// given: an enabled channel with a pre-existing annotation
		channel := createWebhookChannel(t, ctx, repo, namespace, false, models.Annotations{"foo": "bar"})

		// when: the provider-side disable is mirrored back
		err := h.disableChannelForProvider(ctx, channel.NamespacedID)
		require.NoError(t, err)

		// then: the channel is disabled, the pre-existing annotation is preserved, and the
		// provider-disabled timestamp annotation is added
		updatedChannel, err := repo.GetChannel(ctx, notification.GetChannelInput{Namespace: namespace, ID: channel.ID})
		require.NoError(t, err)
		require.True(t, updatedChannel.Disabled)
		require.Equal(t, "bar", updatedChannel.Annotations["foo"])
		require.Equal(t, now.Format(time.RFC3339), updatedChannel.Annotations[notification.AnnotationChannelProviderDisabledTimestamp])
	})

	t.Run("already disabled channel is left untouched", func(t *testing.T) {
		// given: a channel that is already disabled
		channel := createWebhookChannel(t, ctx, repo, namespace, true, models.Annotations{"foo": "bar"})

		// when: the provider-side disable is mirrored back again
		err := h.disableChannelForProvider(ctx, channel.NamespacedID)
		require.NoError(t, err)

		// then: the write is skipped, so the annotations are unchanged
		updatedChannel, err := repo.GetChannel(ctx, notification.GetChannelInput{Namespace: namespace, ID: channel.ID})
		require.NoError(t, err)
		require.Equal(t, "bar", updatedChannel.Annotations["foo"])
		require.NotContains(t, updatedChannel.Annotations, notification.AnnotationChannelProviderDisabledTimestamp)
	})
}
