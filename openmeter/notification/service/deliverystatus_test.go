package service_test

import (
	"testing"

	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	statusdb "github.com/openmeterio/openmeter/openmeter/ent/db/notificationeventdeliverystatus"
	"github.com/openmeterio/openmeter/openmeter/notification"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/models"
)

func TestDeliveryStatusStateValidation(t *testing.T) {
	env := newServiceTestEnv(t)
	tests := []struct {
		name  string
		state notification.EventDeliveryStatusState
	}{
		{name: "empty"},
		{name: "unknown", state: "UNKNOWN"},
		{name: "lowercase pending", state: "pending"},
		{name: "lowercase sending", state: "sending"},
		{name: "lowercase resending", state: "resending"},
		{name: "lowercase success", state: "success"},
		{name: "target succeeded", state: "succeeded"},
		{name: "lowercase failed", state: "failed"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// given: an existing valid delivery with an unsupported replacement state
			namespace := ulid.Make().String()
			channel := seedChannel(t, env, namespace, clock.Now(), "validation")
			rule := seedBalanceThresholdRule(t, env, namespace, channel.ID)
			event := seedEvent(t, env, namespace, rule)
			require.Len(t, event.DeliveryStatus, 1)

			before, err := env.service.GetEventDeliveryStatus(t.Context(), notification.GetEventDeliveryStatusInput{
				NamespacedID: event.DeliveryStatus[0].NamespacedID,
			})
			require.NoError(t, err)

			// when: the service receives an unsupported replacement state
			_, err = env.service.UpdateEventDeliveryStatus(t.Context(), notification.UpdateEventDeliveryStatusInput{
				NamespacedID: before.NamespacedID,
				State:        tt.state,
				Reason:       "must not be persisted",
			})
			require.ErrorContains(t, err, "invalid event delivery status state")

			assert.True(t, models.IsGenericValidationError(err))

			// then: validation prevents both an extra row and mutation of the existing one
			count, err := env.db.NotificationEventDeliveryStatus.Query().Where(statusdb.Namespace(namespace)).Count(t.Context())
			require.NoError(t, err)

			after, err := env.service.GetEventDeliveryStatus(t.Context(), notification.GetEventDeliveryStatusInput{
				NamespacedID: before.NamespacedID,
			})
			require.NoError(t, err)

			assert.Equal(t, 1, count)
			assert.Equal(t, before, after)
		})
	}
}
