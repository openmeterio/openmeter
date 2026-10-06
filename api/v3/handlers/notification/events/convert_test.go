package events

import (
	"testing"
	"time"

	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	v1api "github.com/openmeterio/openmeter/api"
	api "github.com/openmeterio/openmeter/api/v3"
	"github.com/openmeterio/openmeter/openmeter/notification"
	"github.com/openmeterio/openmeter/pkg/models"
)

func TestFromAPIEventSortField(t *testing.T) {
	testCases := []struct {
		field   string
		want    notification.OrderBy
		wantErr bool
	}{
		{field: "id", want: notification.OrderByID},
		{field: "type", want: notification.OrderByType},
		{field: "created_at", want: notification.OrderByCreatedAt},
		// updated_at is a valid sort on channels but events have no updated_at column,
		// so it must be rejected rather than silently ordered by id.
		{field: "updated_at", wantErr: true},
		{field: "createdAt", wantErr: true},
		{field: "", wantErr: true},
	}

	for _, tc := range testCases {
		t.Run(tc.field, func(t *testing.T) {
			got, err := FromAPIEventSortField(t.Context(), tc.field)
			if tc.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tc.want, got)
		})
	}
}

// TestEventTypeWireEqualsDomain pins that the v3 enum reuses the domain values, which is
// what lets ToAPIBillingNotificationEvent cast between the two types instead of mapping them.
func TestEventTypeWireEqualsDomain(t *testing.T) {
	testCases := []struct {
		wire   api.BillingNotificationEventType
		domain notification.EventType
	}{
		{api.BillingNotificationEventTypeEntitlementsBalanceThreshold, notification.EventTypeBalanceThreshold},
		{api.BillingNotificationEventTypeEntitlementsReset, notification.EventTypeEntitlementReset},
		{api.BillingNotificationEventTypeInvoiceCreated, notification.EventTypeInvoiceCreated},
		{api.BillingNotificationEventTypeInvoiceUpdated, notification.EventTypeInvoiceUpdated},
	}

	for _, tc := range testCases {
		t.Run(string(tc.wire), func(t *testing.T) {
			assert.Equal(t, string(tc.domain), string(tc.wire))
		})
	}
}

func TestDeliveryStateCasing(t *testing.T) {
	testCases := []struct {
		wire   api.BillingNotificationEventDeliveryState
		domain notification.EventDeliveryStatusState
	}{
		{api.BillingNotificationEventDeliveryStateSuccess, notification.EventDeliveryStatusStateSuccess},
		{api.BillingNotificationEventDeliveryStateFailed, notification.EventDeliveryStatusStateFailed},
		{api.BillingNotificationEventDeliveryStateSending, notification.EventDeliveryStatusStateSending},
		{api.BillingNotificationEventDeliveryStatePending, notification.EventDeliveryStatusStatePending},
		{api.BillingNotificationEventDeliveryStateResending, notification.EventDeliveryStatusStateResending},
	}

	for _, tc := range testCases {
		t.Run(string(tc.wire), func(t *testing.T) {
			domain, err := FromAPIBillingNotificationEventDeliveryState(tc.wire)
			require.NoError(t, err)
			assert.Equal(t, tc.domain, domain)

			wire, err := ToAPIBillingNotificationEventDeliveryState(tc.domain)
			require.NoError(t, err)
			assert.Equal(t, tc.wire, wire)
		})
	}

	t.Run("uppercase wire value is rejected", func(t *testing.T) {
		_, err := FromAPIBillingNotificationEventDeliveryState(api.BillingNotificationEventDeliveryState("FAILED"))
		require.Error(t, err)
	})
}

func TestToAPIBillingNotificationBalanceThresholdType(t *testing.T) {
	testCases := []struct {
		in   v1api.NotificationRuleBalanceThresholdValueType
		want api.BillingNotificationBalanceThresholdType
	}{
		{v1api.NotificationRuleBalanceThresholdValueTypeBalanceValue, api.BillingNotificationBalanceThresholdTypeBalanceValue},
		{v1api.NotificationRuleBalanceThresholdValueTypeUsagePercentage, api.BillingNotificationBalanceThresholdTypeUsagePercentage},
		{v1api.NotificationRuleBalanceThresholdValueTypeUsageValue, api.BillingNotificationBalanceThresholdTypeUsageValue},
		{v1api.NotificationRuleBalanceThresholdValueTypePercent, api.BillingNotificationBalanceThresholdTypeUsagePercentage},
		{v1api.NotificationRuleBalanceThresholdValueTypeNumber, api.BillingNotificationBalanceThresholdTypeUsageValue},
	}

	for _, tc := range testCases {
		t.Run(string(tc.in), func(t *testing.T) {
			got, err := ToAPIBillingNotificationBalanceThresholdType(tc.in)
			require.NoError(t, err)
			assert.Equal(t, tc.want, got)
		})
	}

	t.Run("unknown value is rejected", func(t *testing.T) {
		_, err := ToAPIBillingNotificationBalanceThresholdType(v1api.NotificationRuleBalanceThresholdValueType("nonsense"))
		require.Error(t, err)
	})
}

func TestToAPIBillingNotificationEvent_BalanceThreshold(t *testing.T) {
	createdAt := time.Date(2024, 5, 1, 12, 0, 0, 0, time.UTC)

	event := notification.Event{
		NamespacedID: models.NamespacedID{Namespace: "ns", ID: "01ARZ3NDEKTSV4RRFFQ69G5FAV"},
		Type:         notification.EventTypeBalanceThreshold,
		CreatedAt:    createdAt,
		Rule: notification.Rule{
			NamespacedID: models.NamespacedID{Namespace: "ns", ID: "01ARZ3NDEKTSV4RRFFQ69G5FAW"},
			Type:         notification.EventTypeBalanceThreshold,
			Name:         "threshold rule",
		},
		Payload: notification.EventPayload{
			EventPayloadMeta: notification.EventPayloadMeta{Type: notification.EventTypeBalanceThreshold},
			BalanceThreshold: &notification.BalanceThresholdPayload{
				EntitlementValuePayloadBase: notification.EntitlementValuePayloadBase{
					Entitlement: v1api.EntitlementMetered{Id: "01ARZ3NDEKTSV4RRFFQ69G5FAX"},
					Feature:     v1api.Feature{Id: "01ARZ3NDEKTSV4RRFFQ69G5FAY", Key: "gpt4_tokens"},
					Subject:     v1api.Subject{Key: "customer-1"},
					Customer:    v1api.Customer{Id: "01ARZ3NDEKTSV4RRFFQ69G5FAZ"},
					Value: v1api.EntitlementValue{
						HasAccess: true,
						Balance:   lo.ToPtr(100.0),
						Usage:     lo.ToPtr(900.0),
						Overage:   lo.ToPtr(0.0),
					},
				},
				// The legacy PERCENT spelling must normalize to usage_percentage.
				Threshold: v1api.NotificationRuleBalanceThresholdValue{
					Type:  v1api.NotificationRuleBalanceThresholdValueTypePercent,
					Value: 90,
				},
			},
		},
	}

	got, err := ToAPIBillingNotificationEvent(event)
	require.NoError(t, err)

	assert.Equal(t, event.ID, got.Id)
	assert.Equal(t, api.BillingNotificationEventTypeEntitlementsBalanceThreshold, got.Type)
	assert.Equal(t, createdAt, got.CreatedAt)
	assert.Equal(t, event.Rule.ID, got.Rule.Id)
	assert.Equal(t, "threshold rule", got.Rule.Name)
	assert.Equal(t, api.BillingNotificationEventTypeEntitlementsBalanceThreshold, got.Rule.Type)

	payload, err := got.Payload.AsBillingNotificationEventBalanceThresholdPayload()
	require.NoError(t, err)

	assert.Equal(t, event.ID, payload.Id)
	assert.Equal(t, createdAt, payload.Timestamp)
	assert.Equal(t, "01ARZ3NDEKTSV4RRFFQ69G5FAX", payload.Data.EntitlementId)
	assert.Equal(t, "01ARZ3NDEKTSV4RRFFQ69G5FAY", payload.Data.Feature.Id)
	assert.Equal(t, "gpt4_tokens", payload.Data.Feature.Key)
	assert.Equal(t, "customer-1", payload.Data.SubjectKey)
	assert.Equal(t, "01ARZ3NDEKTSV4RRFFQ69G5FAZ", lo.FromPtr(payload.Data.CustomerId))
	assert.Equal(t, api.BillingEntitlementValueResult{
		Type:       api.BillingEntitlementTypeMetered,
		FeatureKey: "gpt4_tokens",
		HasAccess:  true,
		Value: &api.BillingEntitlementAccessValue{
			Balance:                   "100",
			Usage:                     "900",
			Overage:                   "0",
			TotalAvailableGrantAmount: "0",
			GrantBalances:             map[string]api.Numeric{},
		},
	}, payload.Data.Value)
	assert.Equal(t, api.BillingNotificationBalanceThresholdTypeUsagePercentage, payload.Data.Threshold.Type)
	assert.Equal(t, 90.0, payload.Data.Threshold.Value)
}

func TestToAPIBillingNotificationEvent_EntitlementReset(t *testing.T) {
	event := notification.Event{
		NamespacedID: models.NamespacedID{Namespace: "ns", ID: "01ARZ3NDEKTSV4RRFFQ69G5FAV"},
		Type:         notification.EventTypeEntitlementReset,
		Rule: notification.Rule{
			NamespacedID: models.NamespacedID{Namespace: "ns", ID: "01ARZ3NDEKTSV4RRFFQ69G5FAW"},
			Type:         notification.EventTypeEntitlementReset,
			Name:         "reset rule",
		},
		Payload: notification.EventPayload{
			EventPayloadMeta: notification.EventPayloadMeta{Type: notification.EventTypeEntitlementReset},
			EntitlementReset: &notification.EntitlementResetPayload{
				Entitlement: v1api.EntitlementMetered{Id: "01ARZ3NDEKTSV4RRFFQ69G5FAX"},
				Feature:     v1api.Feature{Id: "01ARZ3NDEKTSV4RRFFQ69G5FAY", Key: "gpt4_tokens"},
				Subject:     v1api.Subject{Key: "customer-1"},
				Value:       v1api.EntitlementValue{HasAccess: true},
			},
		},
	}

	got, err := ToAPIBillingNotificationEvent(event)
	require.NoError(t, err)

	payload, err := got.Payload.AsBillingNotificationEventResetPayload()
	require.NoError(t, err)

	assert.Equal(t, api.BillingNotificationEventResetPayloadTypeEntitlementsReset, payload.Type)
	assert.Equal(t, "01ARZ3NDEKTSV4RRFFQ69G5FAX", payload.Data.EntitlementId)
	assert.Equal(t, "gpt4_tokens", payload.Data.Feature.Key)
	// The customer is absent on this payload, so the optional field must stay unset
	// rather than reporting an empty id.
	assert.Nil(t, payload.Data.CustomerId)
}

func TestToAPIBillingNotificationEvent_InvoiceCreated(t *testing.T) {
	event := notification.Event{
		NamespacedID: models.NamespacedID{Namespace: "ns", ID: "01ARZ3NDEKTSV4RRFFQ69G5FAV"},
		Type:         notification.EventTypeInvoiceCreated,
		Rule: notification.Rule{
			NamespacedID: models.NamespacedID{Namespace: "ns", ID: "01ARZ3NDEKTSV4RRFFQ69G5FAW"},
			Type:         notification.EventTypeInvoiceCreated,
			Name:         "invoice rule",
		},
		Payload: notification.EventPayload{
			EventPayloadMeta: notification.EventPayloadMeta{Type: notification.EventTypeInvoiceCreated},
			Invoice: &notification.InvoicePayload{
				Invoice: v1api.Invoice{
					Id:       "01ARZ3NDEKTSV4RRFFQ69G5FB0",
					Number:   "INV-2024-0001",
					Currency: "USD",
					Status:   v1api.InvoiceStatusDraft,
					Customer: v1api.BillingInvoiceCustomerExtendedDetails{
						Id: lo.ToPtr("01ARZ3NDEKTSV4RRFFQ69G5FB1"),
					},
					Totals: v1api.InvoiceTotals{Total: "123.45"},
				},
			},
		},
	}

	got, err := ToAPIBillingNotificationEvent(event)
	require.NoError(t, err)

	payload, err := got.Payload.AsBillingNotificationEventInvoiceCreatedPayload()
	require.NoError(t, err)

	assert.Equal(t, api.BillingNotificationEventInvoiceCreatedPayloadTypeInvoiceCreated, payload.Type)
	assert.Equal(t, "01ARZ3NDEKTSV4RRFFQ69G5FB0", payload.Data.Invoice.Id)
	assert.Equal(t, "INV-2024-0001", payload.Data.Invoice.Number)
	assert.Equal(t, "01ARZ3NDEKTSV4RRFFQ69G5FB1", lo.FromPtr(payload.Data.CustomerId))
	assert.Equal(t, "USD", payload.Data.Currency)
	assert.Equal(t, string(v1api.InvoiceStatusDraft), payload.Data.Status)
	assert.Equal(t, "123.45", payload.Data.Total)
}

func TestToAPIBillingNotificationEvent_MissingPayloadIsRejected(t *testing.T) {
	event := notification.Event{
		NamespacedID: models.NamespacedID{Namespace: "ns", ID: "01ARZ3NDEKTSV4RRFFQ69G5FAV"},
		Type:         notification.EventTypeBalanceThreshold,
		Rule: notification.Rule{
			NamespacedID: models.NamespacedID{Namespace: "ns", ID: "01ARZ3NDEKTSV4RRFFQ69G5FAW"},
			Type:         notification.EventTypeBalanceThreshold,
		},
	}

	_, err := ToAPIBillingNotificationEvent(event)
	require.Error(t, err)
}

// TestToAPIBillingNotificationEventDeliveryStatuses covers the two behaviors that differ from v1: the channel is
// reported by id only (no lookup into the rule's channel list), and attempts come back
// newest first.
func TestToAPIBillingNotificationEventDeliveryStatuses(t *testing.T) {
	older := time.Date(2024, 5, 1, 10, 0, 0, 0, time.UTC)
	newer := time.Date(2024, 5, 1, 11, 0, 0, 0, time.UTC)
	nextAttempt := time.Date(2024, 5, 1, 12, 0, 0, 0, time.UTC)

	got, err := ToAPIBillingNotificationEventDeliveryStatuses([]notification.EventDeliveryStatus{
		{
			ChannelID:   "01ARZ3NDEKTSV4RRFFQ69G5FAV",
			State:       notification.EventDeliveryStatusStateFailed,
			Reason:      "connection refused",
			UpdatedAt:   newer,
			NextAttempt: lo.ToPtr(nextAttempt),
			Attempts: []notification.EventDeliveryAttempt{
				{
					State:     notification.EventDeliveryStatusStateFailed,
					Timestamp: older,
					Response: notification.EventDeliveryAttemptResponse{
						Body:     "first",
						Duration: 1500 * time.Millisecond,
					},
				},
				{
					State:     notification.EventDeliveryStatusStateFailed,
					Timestamp: newer,
					Response: notification.EventDeliveryAttemptResponse{
						StatusCode: lo.ToPtr(500),
						Body:       "second",
						URL:        lo.ToPtr("https://example.com/hook"),
						Duration:   2 * time.Second,
					},
				},
			},
		},
	})
	require.NoError(t, err)
	require.Len(t, got, 1)

	status := got[0]
	assert.Equal(t, "01ARZ3NDEKTSV4RRFFQ69G5FAV", status.ChannelId)
	assert.Equal(t, api.BillingNotificationEventDeliveryStateFailed, status.State)
	assert.Equal(t, "connection refused", status.Reason)
	assert.Equal(t, newer, status.UpdatedAt)
	assert.Equal(t, nextAttempt, lo.FromPtr(status.NextAttempt))

	require.Len(t, status.Attempts, 2)
	assert.Equal(t, newer, status.Attempts[0].Timestamp, "attempts must be newest first")
	assert.Equal(t, int32(500), lo.FromPtr(status.Attempts[0].Response.StatusCode))
	assert.Equal(t, int64(2000), status.Attempts[0].Response.DurationMs)
	assert.Equal(t, "https://example.com/hook", lo.FromPtr(status.Attempts[0].Response.Url))

	assert.Equal(t, older, status.Attempts[1].Timestamp)
	assert.Nil(t, status.Attempts[1].Response.StatusCode, "a missing status code must stay absent, not become 0")
	assert.Equal(t, int64(1500), status.Attempts[1].Response.DurationMs)
}
