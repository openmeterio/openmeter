package adapter

import (
	"log/slog"
	"testing"

	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/notification"
	"github.com/openmeterio/openmeter/openmeter/testutils"
	"github.com/openmeterio/openmeter/pkg/models"
)

func TestDisableChannel(t *testing.T) {
	testDB := testutils.InitPostgresDB(t, testutils.PostgresDBStateEntMigrated)
	t.Cleanup(func() { testDB.Close(t) })

	repo, err := New(Config{
		Client: testDB.EntDriver.Client(),
		Logger: slog.New(slog.DiscardHandler),
	})
	require.NoError(t, err)

	ctx := t.Context()
	namespace := ulid.Make().String()

	createChannel := func(t *testing.T, disabled bool, annotations models.Annotations) *notification.Channel {
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

	t.Run("disables an enabled channel and replaces its annotations", func(t *testing.T) {
		// given: an enabled channel
		channel := createChannel(t, false, nil)

		// when: the channel is disabled
		err := repo.DisableChannel(ctx, notification.DisableChannelInput{
			NamespacedID: channel.NamespacedID,
			Annotations:  models.Annotations{notification.AnnotationChannelProviderDisabledTimestamp: "stamp"},
		})
		require.NoError(t, err)

		// then: the channel is disabled and carries the annotations
		updatedChannel, err := repo.GetChannel(ctx, notification.GetChannelInput{Namespace: namespace, ID: channel.ID})
		require.NoError(t, err)
		require.True(t, updatedChannel.Disabled)
		require.Equal(t, "stamp", updatedChannel.Annotations[notification.AnnotationChannelProviderDisabledTimestamp])
	})

	t.Run("is a no-op on an already disabled channel", func(t *testing.T) {
		// given: a channel that is already disabled
		channel := createChannel(t, true, models.Annotations{"foo": "bar"})

		// when: the disable runs against it
		err := repo.DisableChannel(ctx, notification.DisableChannelInput{
			NamespacedID: channel.NamespacedID,
			Annotations:  models.Annotations{notification.AnnotationChannelProviderDisabledTimestamp: "stamp"},
		})
		require.NoError(t, err)

		// then: the disabled predicate matched no row, so the annotations are unchanged
		updatedChannel, err := repo.GetChannel(ctx, notification.GetChannelInput{Namespace: namespace, ID: channel.ID})
		require.NoError(t, err)
		require.Equal(t, "bar", updatedChannel.Annotations["foo"])
		require.NotContains(t, updatedChannel.Annotations, notification.AnnotationChannelProviderDisabledTimestamp)
	})
}
