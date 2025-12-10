package subscriptionaddon_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/addon"
	"github.com/openmeterio/openmeter/openmeter/session"
	subscriptionaddon "github.com/openmeterio/openmeter/openmeter/subscription/addon"
	"github.com/openmeterio/openmeter/pkg/models"
)

func TestEvents(t *testing.T) {
	// Create test data
	now := time.Now().UTC()
	customer := customer.Customer{
		ManagedResource: models.ManagedResource{
			NamespacedModel: models.NamespacedModel{
				Namespace: "test-namespace",
			},
			ManagedModel: models.ManagedModel{
				CreatedAt: now,
				UpdatedAt: now,
			},
			ID:   "test-customer-id",
			Name: "Test Customer",
		},
		UsageAttribution: &customer.CustomerUsageAttribution{
			SubjectKeys: []string{"test-subject-key"},
		},
		PrimaryEmail: lo.ToPtr("test@example.com"),
	}

	addonMeta := productcatalog.AddonMeta{
		Key:          "test-key",
		Version:      1,
		Name:         "Test Addon",
		Description:  lo.ToPtr("Test Description"),
		InstanceType: productcatalog.AddonInstanceTypeSingle,
	}

	addon := addon.Addon{
		NamespacedID: models.NamespacedID{
			Namespace: "test-namespace",
			ID:        "test-addon-id",
		},
		ManagedModel: models.ManagedModel{
			CreatedAt: now,
			UpdatedAt: now,
		},
		AddonMeta: addonMeta,
	}

	subscriptionAddon := subscriptionaddon.SubscriptionAddon{
		NamespacedID: models.NamespacedID{
			Namespace: "test-namespace",
			ID:        "test-subscription-addon-id",
		},
		ManagedModel: models.ManagedModel{
			CreatedAt: now,
			UpdatedAt: now,
		},
		Name:           "Test Subscription Addon",
		Description:    lo.ToPtr("Test Description"),
		Addon:          addon,
		SubscriptionID: "test-subscription-id",
	}

	userID := "test-user-id"

	t.Run("CreatedEvent", func(t *testing.T) {
		// Create context with authentication session
		ctx := context.WithValue(
			context.Background(),
			session.AuthenticationSessionKey, &session.AuthenticationSession{
				UserID: userID,
			},
		)

		// Create event
		event := subscriptionaddon.NewCreatedEvent(ctx, customer, subscriptionAddon)
		event.UserID = &userID

		// Test event name
		assert.Equal(t, "io.openmeter.subscriptionaddon.v1.subscriptionaddon.created", event.EventName())

		// Test event metadata
		metadata := event.EventMetadata()
		assert.Equal(t, "//openmeter.io/namespace/test-namespace/subscriptionAddon/test-subscription-addon-id", metadata.Source)
		assert.Equal(t, "//openmeter.io/namespace/test-namespace/customer/test-customer-id", metadata.Subject)

		// Test serialization
		data, err := json.Marshal(event)
		require.NoError(t, err)

		// Test deserialization
		var decodedEvent subscriptionaddon.CreatedEvent
		err = json.Unmarshal(data, &decodedEvent)
		require.NoError(t, err)

		// Compare fields
		assert.Equal(t, event.Customer, decodedEvent.Customer)
		assert.Equal(t, event.SubscriptionAddon, decodedEvent.SubscriptionAddon)
		assert.Equal(t, event.UserID, decodedEvent.UserID)
	})

	t.Run("ChangeQuantityEvent", func(t *testing.T) {
		// Create context with authentication session
		ctx := context.WithValue(
			context.Background(),
			session.AuthenticationSessionKey, &session.AuthenticationSession{
				UserID: userID,
			},
		)

		// Create event
		event := subscriptionaddon.NewChangeQuantityEvent(ctx, customer, subscriptionAddon)
		event.UserID = &userID

		// Test event name
		assert.Equal(t, "io.openmeter.subscriptionaddon.v1.subscriptionaddon.changequantity", event.EventName())

		// Test event metadata
		metadata := event.EventMetadata()
		assert.Equal(t, "//openmeter.io/namespace/test-namespace/subscriptionAddon/test-subscription-addon-id", metadata.Source)
		assert.Equal(t, "//openmeter.io/namespace/test-namespace/customer/test-customer-id", metadata.Subject)

		// Test serialization
		data, err := json.Marshal(event)
		require.NoError(t, err)

		// Test deserialization
		var decodedEvent subscriptionaddon.ChangeQuantityEvent
		err = json.Unmarshal(data, &decodedEvent)
		require.NoError(t, err)

		// Compare fields
		assert.Equal(t, event.Customer, decodedEvent.Customer)
		assert.Equal(t, event.SubscriptionAddon, decodedEvent.SubscriptionAddon)
		assert.Equal(t, event.UserID, decodedEvent.UserID)
	})
}
