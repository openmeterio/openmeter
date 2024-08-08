package e2e

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	notificatione2e "github.com/openmeterio/openmeter/e2e/notification"
)

const (
	PostgresURL    = "postgres://postgres:postgres@127.0.0.1:5432/postgres?sslmode=disable"
	ClickhouseAddr = "127.0.0.1:9000"

	SvixServerURL        = "http://127.0.0.1:8071"
	SvixJWTSigningSecret = "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJpYXQiOjE3MjI5NzYyNzMsImV4cCI6MjAzODMzNjI3MywibmJmIjoxNzIyOTc2MjczLCJpc3MiOiJzdml4LXNlcnZlciIsInN1YiI6Im9yZ18yM3JiOFlkR3FNVDBxSXpwZ0d3ZFhmSGlyTXUifQ.PomP6JWRI62W5N4GtNdJm2h635Q5F54eij0J3BU-_Ds"
)

func TestNotification(t *testing.T) {
	env, err := notificatione2e.NewNotificationTestEnv(PostgresURL, ClickhouseAddr, SvixServerURL, SvixJWTSigningSecret)
	require.NoError(t, err, "NotificationTestEnv() failed")
	require.NotNil(t, env.Notification())
	require.NotNil(t, env.NotificationRepo())
	require.NotNil(t, env.FeatureConn())

	defer func() {
		if err := env.Close(); err != nil {
			t.Errorf("failed to close environment: %v", err)
		}
	}()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Test suite for testing integration with webhook provider (Svix)
	t.Run("Webhook", func(t *testing.T) {
		testSuite := notificatione2e.WebhookTestSuite{
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
		testSuite := notificatione2e.ChannelTestSuite{
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
		testSuite := notificatione2e.RuleTestSuite{
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
		testSuite := notificatione2e.EventTestSuite{
			Env: env,
		}

		testSuite.Setup(ctx, t)

		t.Run("CreateEvent", func(t *testing.T) {
			testSuite.TestCreateEvent(ctx, t)
		})

		t.Run("ListEvents", func(t *testing.T) {
			testSuite.TestListEvents(ctx, t)
		})

		t.Run("CreateDeliveryStatus", func(t *testing.T) {
			testSuite.TestCreateDeliveryStatus(ctx, t)
		})

		t.Run("ListCreateDeliveryStatus", func(t *testing.T) {
			testSuite.TestListCreateDeliveryStatus(ctx, t)
		})
	})
}
