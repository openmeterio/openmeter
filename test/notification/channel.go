package notification

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/notification"
	webhooksecret "github.com/openmeterio/openmeter/openmeter/notification/webhook/secret"
	"github.com/openmeterio/openmeter/pkg/models"
)

func NewCreateChannelInput(namespace, name string) notification.CreateChannelInput {
	return notification.CreateChannelInput{
		NamespacedModel: models.NamespacedModel{
			Namespace: namespace,
		},
		Type:     notification.ChannelTypeWebhook,
		Name:     name,
		Disabled: false,
		Config: notification.ChannelConfig{
			ChannelConfigMeta: notification.ChannelConfigMeta{
				Type: notification.ChannelTypeWebhook,
			},
			WebHook: notification.WebHookChannelConfig{
				CustomHeaders: map[string]string{
					"X-TEST-HEADER": "NotificationChannelTest",
				},
				URL:           TestWebhookURL,
				SigningSecret: TestSigningSecret,
			},
		},
		Metadata: models.Metadata{
			"namespace": namespace,
			"name":      name,
		},
		Annotations: models.Annotations{
			"namespace": namespace,
			"name":      name,
		},
	}
}

type ChannelTestSuite struct {
	Env TestEnv
}

func (s *ChannelTestSuite) TestCreate(ctx context.Context, t *testing.T) {
	service := s.Env.Notification()

	createIn := NewCreateChannelInput(s.Env.Namespace(), "NotificationCreateChannel")

	channel, err := service.CreateChannel(ctx, createIn)
	require.NoError(t, err, "Creating channel must not return error")
	require.NotNil(t, channel, "Channel must not be nil")
	assert.NotEmpty(t, channel.ID, "Channel ID must not be empty")
	assert.Equal(t, createIn.Disabled, channel.Disabled, "Channel must not be disabled")
	assert.Equal(t, createIn.Type, channel.Type, "Channel type must be the same")
	assert.EqualValues(t, createIn.Config, channel.Config, "Channel config must be the same")
	assert.Equalf(t, createIn.Annotations, channel.Annotations, "Annotations must be the same")
	assert.Equalf(t, createIn.Metadata, channel.Metadata, "Metadata must be the same")
}

func (s *ChannelTestSuite) TestList(ctx context.Context, t *testing.T) {
	service := s.Env.Notification()

	createIn1 := NewCreateChannelInput(s.Env.Namespace(), "NotificationListChannel1")
	channel1, err := service.CreateChannel(ctx, createIn1)
	require.NoError(t, err, "Creating channel must not return error")
	require.NotNil(t, channel1, "Channel must not be nil")

	createIn2 := NewCreateChannelInput(s.Env.Namespace(), "NotificationListChannel2")
	channel2, err := service.CreateChannel(ctx, createIn2)
	require.NoError(t, err, "Creating channel must not return error")
	require.NotNil(t, channel2, "Channel must not be nil")

	list, err := service.ListChannels(ctx, notification.ListChannelsInput{
		Namespaces: []string{
			createIn1.Namespace,
			createIn2.Namespace,
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
	service := s.Env.Notification()

	createIn := NewCreateChannelInput(s.Env.Namespace(), "NotificationUpdateChannel1")

	channel, err := service.CreateChannel(ctx, createIn)
	require.NoError(t, err, "Creating channel must not return error")
	require.NotNil(t, channel, "Channel must not be nil")

	secret, err := webhooksecret.NewSigningSecretWithDefaultSize()
	require.NoError(t, err, "Generating new signing secret must not return an error")

	updateIn := notification.UpdateChannelInput{
		NamespacedID: models.NamespacedID{
			Namespace: channel.Namespace,
			ID:        channel.ID,
		},
		Type:     channel.Type,
		Name:     "NotificationUpdateChannel2",
		Disabled: true,
		Config: notification.ChannelConfig{
			ChannelConfigMeta: channel.Config.ChannelConfigMeta,
			WebHook: notification.WebHookChannelConfig{
				CustomHeaders: map[string]string{
					"X-TEST-HEADER": "NotificationUpdateChannel2",
				},
				URL:           "http://example.com/update",
				SigningSecret: secret,
			},
		},
		Metadata: models.Metadata{
			"namespace": channel.Namespace,
			"name":      "NotificationUpdateChannel2",
		},
		Annotations: models.Annotations{
			"namespace": channel.Namespace,
			"name":      "NotificationUpdateChannel2",
		},
	}

	channel2, err := service.UpdateChannel(ctx, updateIn)
	require.NoError(t, err, "Creating channel must not return error")
	require.NotNil(t, channel2, "Channel must not be nil")

	assert.Equal(t, updateIn.Disabled, channel2.Disabled, "Channel must not be disabled")
	assert.Equal(t, updateIn.Type, channel2.Type, "Channel type must be the same")
	assert.EqualValues(t, updateIn.Config, channel2.Config, "Channel config must be the same")
	assert.Equalf(t, updateIn.Annotations, channel2.Annotations, "Annotations must be the same")
	assert.Equalf(t, updateIn.Metadata, channel2.Metadata, "Metadata must be the same")
}

func (s *ChannelTestSuite) TestDelete(ctx context.Context, t *testing.T) {
	service := s.Env.Notification()

	createIn := NewCreateChannelInput(s.Env.Namespace(), "NotificationDeleteChannel1")

	channel, err := service.CreateChannel(ctx, createIn)
	require.NoError(t, err, "Creating channel must not return error")
	require.NotNil(t, channel, "Channel must not be nil")

	err = service.DeleteChannel(ctx, notification.DeleteChannelInput{
		Namespace: channel.Namespace,
		ID:        channel.ID,
	})
	require.NoError(t, err, "Deleting channel must not return error")
}

func (s *ChannelTestSuite) TestGet(ctx context.Context, t *testing.T) {
	service := s.Env.Notification()

	createIn := NewCreateChannelInput(s.Env.Namespace(), "NotificationGetChannel1")

	channel, err := service.CreateChannel(ctx, createIn)
	require.NoError(t, err, "Creating channel must not return error")
	require.NotNil(t, channel, "Channel must not be nil")

	channel2, err := service.GetChannel(ctx, notification.GetChannelInput{
		Namespace: channel.Namespace,
		ID:        channel.ID,
	})
	require.NoError(t, err, "Deleting channel must not return error")
	require.NotNil(t, channel2, "Channel must not be nil")
	assert.NotEmpty(t, channel2.ID, "Channel ID must not be empty")
	assert.Equal(t, channel.Namespace, channel2.Namespace, "Channel namespace must be equal")
	assert.Equal(t, channel.ID, channel2.ID, "Channel ID must be equal")
	assert.Equal(t, channel.Disabled, channel2.Disabled, "Channel disabled must be equal")
	assert.Equal(t, channel.Type, channel2.Type, "Channel type must be the same")
	assert.EqualValues(t, channel.Config, channel2.Config, "Channel config must be the same")
	assert.Equalf(t, channel.Annotations, channel2.Annotations, "Annotations must be the same")
	assert.Equalf(t, channel.Metadata, channel2.Metadata, "Metadata must be the same")
}
