package notification

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNotification(t *testing.T) {
	env, err := NewTestEnv()
	require.NoError(t, err, "NotificationTestEnv() failed")
	require.NotNil(t, env.Notification())
	require.NotNil(t, env.NotificationRepo())
	require.NotNil(t, env.Feature())

	defer func() {
		if err := env.Close(); err != nil {
			t.Errorf("failed to close environment: %v", err)
		}
	}()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Test suite for testing integration with webhook provider (Svix)
	t.Run("Webhook", func(t *testing.T) {
		testSuite := WebhookTestSuite{
			Env: env,
		}

		testSuite.Setup(ctx, t)

		t.Run("CreateWebhook", func(t *testing.T) {
			testSuite.TestCreateWebhook(ctx, t)
		})

		t.Run("UpdateWebhook", func(t *testing.T) {
			testSuite.TestUpdateWebhook(ctx, t)
		})

		t.Run("DeleteWebhook", func(t *testing.T) {
			testSuite.TestDeleteWebhook(ctx, t)
		})

		t.Run("GetWebhook", func(t *testing.T) {
			testSuite.TestGetWebhook(ctx, t)
		})

		t.Run("ListWebhook", func(t *testing.T) {
			testSuite.TestListWebhook(ctx, t)
		})
	})

	// Test suite covering notification channels
	t.Run("Channel", func(t *testing.T) {
		testSuite := ChannelTestSuite{
			Env: env,
		}

		t.Run("Create", func(t *testing.T) {
			testSuite.TestCreate(ctx, t)
		})

		t.Run("List", func(t *testing.T) {
			testSuite.TestList(ctx, t)
		})

		t.Run("Update", func(t *testing.T) {
			testSuite.TestUpdate(ctx, t)
		})

		t.Run("Delete", func(t *testing.T) {
			testSuite.TestDelete(ctx, t)
		})

		t.Run("Get", func(t *testing.T) {
			testSuite.TestGet(ctx, t)
		})
	})

	// Test suite covering notification rules
	t.Run("Rule", func(t *testing.T) {
		testSuite := RuleTestSuite{
			Env: env,
		}

		testSuite.Setup(ctx, t)

		t.Run("Create", func(t *testing.T) {
			testSuite.TestCreate(ctx, t)
		})

		t.Run("List", func(t *testing.T) {
			testSuite.TestList(ctx, t)
		})

		t.Run("Update", func(t *testing.T) {
			testSuite.TestUpdate(ctx, t)
		})

		t.Run("Delete", func(t *testing.T) {
			testSuite.TestDelete(ctx, t)
		})

		t.Run("Get", func(t *testing.T) {
			testSuite.TestGet(ctx, t)
		})
	})

	// Test suite covering notification events
	t.Run("Event", func(t *testing.T) {
		testSuite := EventTestSuite{
			Env: env,
		}

		testSuite.Setup(ctx, t)

		t.Run("CreateEvent", func(t *testing.T) {
			testSuite.TestCreateEvent(ctx, t)
		})

		t.Run("GetEvent", func(t *testing.T) {
			testSuite.TestGetEvent(ctx, t)
		})

		t.Run("ListEvents", func(t *testing.T) {
			testSuite.TestListEvents(ctx, t)
		})

		t.Run("TestListDeliveryStatus", func(t *testing.T) {
			testSuite.TestListDeliveryStatus(ctx, t)
		})

		t.Run("TestUpdateDeliveryStatus", func(t *testing.T) {
			testSuite.TestUpdateDeliveryStatus(ctx, t)
		})
	})

	// Test suite for repo methods
	t.Run("Repository", func(t *testing.T) {
		testSuite := RepositoryTestSuite{
			Env: env,
		}

		testSuite.Setup(ctx, t)

		t.Run("TestFilterEventByFeature", func(t *testing.T) {
			testSuite.TestFilterEventByFeature(t)
		})

		t.Run("TestFilterEventBySubject", func(t *testing.T) {
			testSuite.TestFilterEventBySubject(t)
		})
	})
}
