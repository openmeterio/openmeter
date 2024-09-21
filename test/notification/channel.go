// Copyright © 2024 Tailfin Cloud Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package notification

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/notification"
	notificationwebhook "github.com/openmeterio/openmeter/openmeter/notification/webhook"
	"github.com/openmeterio/openmeter/pkg/models"
)

func NewCreateChannelInput(name string) notification.CreateChannelInput {
	return notification.CreateChannelInput{
		NamespacedModel: models.NamespacedModel{
			Namespace: TestNamespace,
		},
		Type:     notification.ChannelTypeWebhook,
		Name:     name,
		Disabled: false,
		Config: notification.ChannelConfig{
			ChannelConfigMeta: notification.ChannelConfigMeta{
				Type: notification.ChannelTypeWebhook,
			},
			WebHook: notification.WebHookChannelConfig{
				CustomHeaders: map[string]interface{}{
					"X-TEST-HEADER": "NotificationChannelTest",
				},
				URL:           TestWebhookURL,
				SigningSecret: TestSigningSecret,
			},
		},
	}
}

type ChannelTestSuite struct {
	Env TestEnv
}

func (s *ChannelTestSuite) TestCreate(ctx context.Context, t *testing.T) {
	service := s.Env.Notification()

	createIn := NewCreateChannelInput("NotificationCreateChannel")

	channel, err := service.CreateChannel(ctx, createIn)
	require.NoError(t, err, "Creating channel must not return error")
	require.NotNil(t, channel, "Channel must not be nil")
	assert.NotEmpty(t, channel.ID, "Channel ID must not be empty")
	assert.Equal(t, createIn.Disabled, channel.Disabled, "Channel must not be disabled")
	assert.Equal(t, createIn.Type, channel.Type, "Channel type must be the same")
	assert.EqualValues(t, createIn.Config, channel.Config, "Channel config must be the same")
}

func (s *ChannelTestSuite) TestList(ctx context.Context, t *testing.T) {
	service := s.Env.Notification()

	createIn1 := NewCreateChannelInput("NotificationListChannel1")
	channel1, err := service.CreateChannel(ctx, createIn1)
	require.NoError(t, err, "Creating channel must not return error")
	require.NotNil(t, channel1, "Channel must not be nil")

	createIn2 := NewCreateChannelInput("NotificationListChannel2")
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

	createIn := NewCreateChannelInput("NotificationUpdateChannel1")

	channel, err := service.CreateChannel(ctx, createIn)
	require.NoError(t, err, "Creating channel must not return error")
	require.NotNil(t, channel, "Channel must not be nil")

	secret, err := notificationwebhook.NewSigningSecretWithDefaultSize()
	require.NoError(t, err, "Generating new signing secret must not return an error")

	updateIn := notification.UpdateChannelInput{
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

	channel2, err := service.UpdateChannel(ctx, updateIn)
	require.NoError(t, err, "Creating channel must not return error")
	require.NotNil(t, channel2, "Channel must not be nil")

	assert.Equal(t, updateIn.Disabled, channel2.Disabled, "Channel must not be disabled")
	assert.Equal(t, updateIn.Type, channel2.Type, "Channel type must be the same")
	assert.EqualValues(t, updateIn.Config, channel2.Config, "Channel config must be the same")
}

func (s *ChannelTestSuite) TestDelete(ctx context.Context, t *testing.T) {
	service := s.Env.Notification()

	createIn := NewCreateChannelInput("NotificationDeleteChannel1")

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

	createIn := NewCreateChannelInput("NotificationGetChannel1")

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
	assert.Equal(t, channel.Disabled, channel2.Disabled, "Channel disabled must not be equal")
	assert.Equal(t, channel.Type, channel2.Type, "Channel type must be the same")
	assert.EqualValues(t, channel.Config, channel2.Config, "Channel config must be the same")
}
