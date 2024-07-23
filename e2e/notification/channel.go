package notification

import (
	"context"
	"testing"

	"github.com/huandu/go-clone"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/internal/notification"
	notificationwebhook "github.com/openmeterio/openmeter/internal/notification/webhook"
	"github.com/openmeterio/openmeter/pkg/models"
)

var createChannelInput = notification.CreateChannelInput{
	NamespacedModel: models.NamespacedModel{
		Namespace: TestNamespace,
	},
	Type:     notification.ChannelTypeWebhook,
	Name:     "NotificationChannelTest",
	Disabled: false,
	Config: notification.ChannelConfig{
		ChannelConfigMeta: notification.ChannelConfigMeta{
			Type: notification.ChannelTypeWebhook,
		},
		WebHook: notification.WebHookChannelConfig{
			CustomHeaders: map[string]interface{}{
				"X-TEST-HEADER": "NotificationTestCreate1",
			},
			URL:           "http://example.com",
			SigningSecret: "whsec_Fk5kgr5qTdPdQIDniFv+6K0WN2bUpdGjjGtaNeAx8N8=",
		},
	},
}

// TODO: test channels with features

type ChannelTestSuite struct {
	Env TestEnv
}

func (s *ChannelTestSuite) TestCreate(ctx context.Context, t *testing.T) {
	connector := s.Env.NotificationConn()

	input := clone.Clone(createChannelInput).(notification.CreateChannelInput)

	channel, err := connector.CreateChannel(ctx, input)
	require.NoError(t, err, "Creating channel must not return error")
	require.NotNil(t, channel, "Channel must not be nil")
	assert.NotEmpty(t, channel.ID, "Channel ID must not be empty")
	assert.Equal(t, input.Disabled, channel.Disabled, "Channel must not be disabled")
	assert.Equal(t, input.Type, channel.Type, "Channel type must be the same")
	assert.EqualValues(t, input.Config, channel.Config, "Channel config must be the same")
}

func (s *ChannelTestSuite) TestList(ctx context.Context, t *testing.T) {
	connector := s.Env.NotificationConn()

	input1 := clone.Clone(createChannelInput).(notification.CreateChannelInput)
	input1.Name = "NotificationListChannel1"
	channel1, err := connector.CreateChannel(ctx, input1)
	require.NoError(t, err, "Creating channel must not return error")
	require.NotNil(t, channel1, "Channel must not be nil")

	input2 := clone.Clone(createChannelInput).(notification.CreateChannelInput)
	input2.Name = "NotificationListChannel2"
	channel2, err := connector.CreateChannel(ctx, input2)
	require.NoError(t, err, "Creating channel must not return error")
	require.NotNil(t, channel2, "Channel must not be nil")

	list, err := connector.ListChannels(ctx, notification.ListChannelsInput{
		Namespaces: []string{
			input1.Namespace,
			input2.Namespace,
		},
		Channels: []string{
			channel1.ID,
			channel2.ID,
		},
		OrderBy:         "id",
		IncludeDisabled: false,
	})
	require.NoError(t, err, "Listing channels must not return error")
	assert.NotEmpty(t, list.Items, "List of channels must not be empty")

	expectedList := []notification.Channel{
		*channel1,
		*channel2,
	}

	assert.EqualValues(t, expectedList, list.Items, "Unexpected items returned by listing channels")
}

func (s *ChannelTestSuite) TestUpdate(ctx context.Context, t *testing.T) {
	connector := s.Env.NotificationConn()

	input := clone.Clone(createChannelInput).(notification.CreateChannelInput)
	input.Name = "NotificationUpdateChannel1"

	channel, err := connector.CreateChannel(ctx, input)
	require.NoError(t, err, "Creating channel must not return error")
	require.NotNil(t, channel, "Channel must not be nil")

	secret, err := notificationwebhook.NewSigningSecretWithDefaultSize()
	require.NoError(t, err, "Generating new signing secret must not return an error")

	input2 := notification.UpdateChannelInput{
		NamespacedModel: channel.NamespacedModel,
		Type:            channel.Type,
		Name:            "NotificationUpdateChannel2",
		Disabled:        true,
		Config: notification.ChannelConfig{
			ChannelConfigMeta: channel.Config.ChannelConfigMeta,
			WebHook: notification.WebHookChannelConfig{
				CustomHeaders: map[string]interface{}{
					"X-TEST-HEADER": "NotificationUpdateChannel2",
				},
				URL:           "http://example.com/update",
				SigningSecret: secret,
			},
		},
		ID: channel.ID,
	}

	channel2, err := connector.UpdateChannel(ctx, input2)
	require.NoError(t, err, "Creating channel must not return error")
	require.NotNil(t, channel2, "Channel must not be nil")

	assert.Equal(t, input2.Disabled, channel2.Disabled, "Channel must not be disabled")
	assert.Equal(t, input2.Type, channel2.Type, "Channel type must be the same")
	assert.EqualValues(t, input2.Config, channel2.Config, "Channel config must be the same")
}

func (s *ChannelTestSuite) TestDelete(ctx context.Context, t *testing.T) {
	connector := s.Env.NotificationConn()

	input := clone.Clone(createChannelInput).(notification.CreateChannelInput)
	input.Name = "NotificationDeleteChannel1"

	channel, err := connector.CreateChannel(ctx, input)
	require.NoError(t, err, "Creating channel must not return error")
	require.NotNil(t, channel, "Channel must not be nil")

	err = connector.DeleteChannel(ctx, notification.DeleteChannelInput{
		Namespace: channel.Namespace,
		ID:        channel.ID,
	})
	require.NoError(t, err, "Deleting channel must not return error")
}

func (s *ChannelTestSuite) TestGet(ctx context.Context, t *testing.T) {
	connector := s.Env.NotificationConn()

	input := clone.Clone(createChannelInput).(notification.CreateChannelInput)
	input.Name = "NotificationGetChannel1"

	channel, err := connector.CreateChannel(ctx, input)
	require.NoError(t, err, "Creating channel must not return error")
	require.NotNil(t, channel, "Channel must not be nil")

	channel2, err := connector.GetChannel(ctx, notification.GetChannelInput{
		Namespace: channel.Namespace,
		ID:        channel.ID,
	})
	require.NoError(t, err, "Deleting channel must not return error")
	require.NotNil(t, channel2, "Channel must not be nil")
	assert.NotEmpty(t, channel2.ID, "Channel ID must not be empty")
	assert.Equal(t, channel.Namespace, channel2.Namespace, "Channel namespace must be equal")
	assert.Equal(t, channel.ID, channel2.ID, "Channel ID must be equal")
	assert.Equal(t, channel.Disabled, channel2.Disabled, "Channel disabled must not be equal")
	assert.Equal(t, channel.Type, channel2.Type, "Channel type must be the same")
	assert.EqualValues(t, channel.Config, channel2.Config, "Channel config must be the same")
}
