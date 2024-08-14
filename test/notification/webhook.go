package notification

import (
	"context"
	"crypto/rand"
	"testing"
	"time"

	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	notificationwebhook "github.com/openmeterio/openmeter/internal/notification/webhook"
	"github.com/openmeterio/openmeter/pkg/convert"
	"github.com/openmeterio/openmeter/pkg/defaultx"
)

func NewCreateWebhookInput(id *string, desc string) notificationwebhook.CreateWebhookInput {
	if id == nil || *id == "" {
		uid := ulid.MustNew(ulid.Timestamp(time.Now()), rand.Reader)
		id = convert.ToPointer(uid.String())
	}

	return notificationwebhook.CreateWebhookInput{
		Namespace:     TestNamespace,
		ID:            id,
		URL:           TestWebhookURL,
		CustomHeaders: nil,
		Disabled:      false,
		Secret:        convert.ToPointer(TestSigningSecret),
		RateLimit:     nil,
		Description:   convert.ToPointer(desc),
		EventTypes:    nil,
		Channels:      nil,
	}
}

type WebhookTestSuite struct {
	Env TestEnv
}

func (s *WebhookTestSuite) Setup(ctx context.Context, t *testing.T) {
	err := s.Env.NotificationWebhook().RegisterEventTypes(ctx, notificationwebhook.RegisterEventTypesInputs{
		EvenTypes: notificationwebhook.NotificationEventTypes,
	})
	assert.NoError(t, err, "Registering event types must not fail")
}

func (s *WebhookTestSuite) TestCreateWebhook(ctx context.Context, t *testing.T) {
	wb := s.Env.NotificationWebhook()

	input := NewCreateWebhookInput(nil, "TestCreateWebhook")

	webhook, err := wb.CreateWebhook(ctx, input)
	require.NoError(t, err, "Creating webhook must not return error")
	require.NotNil(t, wb, "Webhook must not be nil")

	assert.Equal(t, input.Namespace, webhook.Namespace, "Webhook namespace must match")
	assert.Equal(t, input.URL, webhook.URL, "Webhook url must match")
	assert.Equal(t, input.Disabled, webhook.Disabled, "Webhook disabled must match")
	if input.Secret != nil {
		assert.Equal(t, defaultx.WithDefault(input.Secret, ""), webhook.Secret, "Webhook secret must match")
	}
	assert.Equal(t, defaultx.WithDefault(input.Description, ""), webhook.Description, "Webhook description must match")
	assert.Equal(t, input.EventTypes, webhook.EventTypes, "Webhook event types must match")
	assert.Equal(t, input.Channels, webhook.Channels, "Webhook channels must match")
	assert.NotZero(t, webhook.CreatedAt, "Webhook created at timestamp must not be empty")
	assert.NotZero(t, webhook.UpdatedAt, "Webhook updated at timestamp must not be empty")
}

func (s *WebhookTestSuite) TestUpdateWebhook(ctx context.Context, t *testing.T) {
	wb := s.Env.NotificationWebhook()

	createIn := NewCreateWebhookInput(nil, "TestUpdateWebhook")

	webhook, err := wb.CreateWebhook(ctx, createIn)
	require.NoError(t, err, "Creating webhook must not return error", "createIn", createIn)
	require.NotNil(t, webhook, "Webhook must not be nil")

	updateIn := notificationwebhook.UpdateWebhookInput{
		Namespace: TestNamespace,
		ID:        webhook.ID,
		URL:       "http://example2.com/",
		CustomHeaders: map[string]string{
			"X-Test-Header": "test-value",
		},
		Disabled:    true,
		Secret:      convert.ToPointer("whsec_mCP4QSwe52D0IEU/UXLSD6Fif1RykRRMFHL0KJnGeQg="),
		RateLimit:   convert.ToPointer[int32](50),
		Description: convert.ToPointer(webhook.Description),
		EventTypes:  nil,
		Channels:    []string{"test-channel"},
	}

	updatedWebhook, err := wb.UpdateWebhook(ctx, updateIn)
	require.NoError(t, err, "Updating webhook must not return error", "updateIn", updateIn)
	require.NotNil(t, updatedWebhook, "Webhook must not be nil")

	assert.Equal(t, updateIn.Namespace, updatedWebhook.Namespace, "Webhook namespace must match")
	assert.Equal(t, updateIn.URL, updatedWebhook.URL, "Webhook url must match")
	assert.Equal(t, updateIn.Disabled, updatedWebhook.Disabled, "Webhook disabled must match")
	assert.Equal(t, defaultx.WithDefault(updateIn.Secret, ""), updatedWebhook.Secret, "Webhook secret must match")
	assert.Equal(t, defaultx.WithDefault(updateIn.Description, ""), updatedWebhook.Description, "Webhook description must match")
	assert.Equal(t, updateIn.EventTypes, updatedWebhook.EventTypes, "Webhook event types must match")
	assert.Equal(t, updateIn.Channels, updatedWebhook.Channels, "Webhook channels must match")
	assert.NotZero(t, updatedWebhook.CreatedAt, "Webhook channels must match")
	assert.NotZero(t, updatedWebhook.UpdatedAt, "Webhook channels must match")
}

func (s *WebhookTestSuite) TestDeleteWebhook(ctx context.Context, t *testing.T) {
	wb := s.Env.NotificationWebhook()

	createIn := NewCreateWebhookInput(nil, "TestDeleteWebhook")

	webhook, err := wb.CreateWebhook(ctx, createIn)
	require.NoError(t, err, "Creating webhook must not return error")
	require.NotNil(t, webhook, "Webhook must not be nil")

	deleteIn := notificationwebhook.DeleteWebhookInput{
		Namespace: webhook.Namespace,
		ID:        webhook.ID,
	}

	err = wb.DeleteWebhook(ctx, deleteIn)
	require.NoError(t, err, "Creating webhook must not return error")
}

func (s *WebhookTestSuite) TestGetWebhook(ctx context.Context, t *testing.T) {
	wb := s.Env.NotificationWebhook()

	createIn := NewCreateWebhookInput(nil, "TestGetWebhook")

	webhook, err := wb.CreateWebhook(ctx, createIn)
	require.NoError(t, err, "Creating webhook must not return error")
	require.NotNil(t, wb, "Webhook must not be nil")

	webhook2, err := wb.GetWebhook(ctx, notificationwebhook.GetWebhookInput{
		Namespace: webhook.Namespace,
		ID:        webhook.ID,
	})
	require.NoError(t, err, "Fetching webhook must not return error")
	require.NotNil(t, wb, "Webhook must not be nil")

	assert.Equal(t, webhook.Namespace, webhook2.Namespace, "Webhook namespace must match")
	assert.Equal(t, webhook.URL, webhook2.URL, "Webhook url must match")
	assert.Equal(t, webhook.Disabled, webhook2.Disabled, "Webhook disabled must match")
	assert.Equal(t, webhook.Secret, webhook2.Secret, "Webhook secret must match")
	assert.Equal(t, webhook.Description, webhook2.Description, "Webhook description must match")
	assert.Equal(t, webhook.EventTypes, webhook2.EventTypes, "Webhook event types must match")
	assert.Equal(t, webhook.Channels, webhook2.Channels, "Webhook channels must match")
	assert.Equal(t, webhook.CreatedAt, webhook2.CreatedAt, "Webhook created at must match")
}

func (s *WebhookTestSuite) TestListWebhook(ctx context.Context, t *testing.T) {
	wb := s.Env.NotificationWebhook()

	createIn1 := NewCreateWebhookInput(nil, "TestListWebhook1")

	webhook1, err := wb.CreateWebhook(ctx, createIn1)
	require.NoError(t, err, "Creating webhook must not return error")
	require.NotNil(t, wb, "Webhook must not be nil")

	createIn2 := NewCreateWebhookInput(nil, "TestListWebhook2")
	createIn2.EventTypes = []string{
		notificationwebhook.EntitlementsBalanceThresholdType,
	}

	webhook2, err := wb.CreateWebhook(ctx, createIn2)
	require.NoError(t, err, "Creating webhook must not return error")
	require.NotNil(t, wb, "Webhook must not be nil")

	createIn3 := NewCreateWebhookInput(nil, "TestListWebhook3")
	createIn3.Channels = []string{
		"test-channel",
	}

	webhook3, err := wb.CreateWebhook(ctx, createIn3)
	require.NoError(t, err, "Creating webhook must not return error")
	require.NotNil(t, wb, "Webhook must not be nil")

	list, err := wb.ListWebhooks(ctx, notificationwebhook.ListWebhooksInput{
		Namespace:  TestNamespace,
		IDs:        []string{webhook1.ID},
		EventTypes: webhook2.EventTypes,
		Channels:   webhook3.Channels,
	})
	require.NoError(t, err, "Creating webhook must not return error")
	require.NotNil(t, list, "Webhook list must not be nil")

	expectedWebhooks := map[string]notificationwebhook.Webhook{
		webhook1.ID: *webhook1,
		webhook2.ID: *webhook2,
		webhook3.ID: *webhook3,
	}

	for _, webhook := range list {
		expectedWebhook, ok := expectedWebhooks[webhook.ID]
		require.True(t, ok, "Expected webhook to exist")

		assert.Equal(t, webhook.Namespace, expectedWebhook.Namespace, "Webhook namespace must match")
		assert.Equal(t, webhook.URL, expectedWebhook.URL, "Webhook url must match")
		assert.Equal(t, webhook.Disabled, expectedWebhook.Disabled, "Webhook disabled must match")
		assert.Equal(t, webhook.Secret, expectedWebhook.Secret, "Webhook secret must match")
		assert.Equal(t, webhook.Description, expectedWebhook.Description, "Webhook description must match")
		assert.Equal(t, webhook.EventTypes, expectedWebhook.EventTypes, "Webhook event types must match")
		assert.Equal(t, webhook.Channels, expectedWebhook.Channels, "Webhook channels must match")
		assert.Equal(t, webhook.CreatedAt, expectedWebhook.CreatedAt, "Webhook created at must match")
	}
}
