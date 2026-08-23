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
	"github.com/openmeterio/openmeter/pkg/filter"
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

// TestEventTypeCasing pins the wire/domain split: the v3 enum uses snake_case while the
// column keeps the dotted value written by v1.
func TestEventTypeCasing(t *testing.T) {
	testCases := []struct {
		wire   api.NotificationEventType
		domain notification.EventType
	}{
		{api.NotificationEventTypeEntitlementsBalanceThreshold, notification.EventTypeBalanceThreshold},
		{api.NotificationEventTypeEntitlementsReset, notification.EventTypeEntitlementReset},
		{api.NotificationEventTypeInvoiceCreated, notification.EventTypeInvoiceCreated},
		{api.NotificationEventTypeInvoiceUpdated, notification.EventTypeInvoiceUpdated},
	}

	for _, tc := range testCases {
		t.Run(string(tc.wire), func(t *testing.T) {
			assert.NotEqual(t, string(tc.wire), string(tc.domain), "wire and domain spellings must differ, otherwise the mapping is dead code")

			domain, err := ToDomainEventType(tc.wire)
			require.NoError(t, err)
			assert.Equal(t, tc.domain, domain)

			wire, err := ToAPIEventType(tc.domain)
			require.NoError(t, err)
			assert.Equal(t, tc.wire, wire)
		})
	}

	t.Run("unknown wire value is rejected", func(t *testing.T) {
		_, err := ToDomainEventType(api.NotificationEventType("invoice.created"))
		require.Error(t, err)
	})

	t.Run("unknown domain value is rejected", func(t *testing.T) {
		_, err := ToAPIEventType(notification.EventType("invoice_created"))
		require.Error(t, err)
	})
}

func TestDeliveryStateCasing(t *testing.T) {
	testCases := []struct {
		wire   api.NotificationEventDeliveryState
		domain notification.EventDeliveryStatusState
	}{
		{api.NotificationEventDeliveryStateSuccess, notification.EventDeliveryStatusStateSuccess},
		{api.NotificationEventDeliveryStateFailed, notification.EventDeliveryStatusStateFailed},
		{api.NotificationEventDeliveryStateSending, notification.EventDeliveryStatusStateSending},
		{api.NotificationEventDeliveryStatePending, notification.EventDeliveryStatusStatePending},
		{api.NotificationEventDeliveryStateResending, notification.EventDeliveryStatusStateResending},
	}

	for _, tc := range testCases {
		t.Run(string(tc.wire), func(t *testing.T) {
			domain, err := ToDomainDeliveryState(tc.wire)
			require.NoError(t, err)
			assert.Equal(t, tc.domain, domain)

			wire, err := ToAPIDeliveryState(tc.domain)
			require.NoError(t, err)
			assert.Equal(t, tc.wire, wire)
		})
	}

	t.Run("uppercase wire value is rejected", func(t *testing.T) {
		_, err := ToDomainDeliveryState(api.NotificationEventDeliveryState("FAILED"))
		require.Error(t, err)
	})
}

// TestToAPIBalanceThresholdType covers the deprecated v1 aliases, which are the only
// reason this mapping is not a straight cast.
func TestToAPIBalanceThresholdType(t *testing.T) {
	testCases := []struct {
		in   v1api.NotificationRuleBalanceThresholdValueType
		want api.NotificationEventBalanceThresholdType
	}{
		{v1api.NotificationRuleBalanceThresholdValueTypeBalanceValue, api.NotificationEventBalanceThresholdTypeBalanceValue},
		{v1api.NotificationRuleBalanceThresholdValueTypeUsagePercentage, api.NotificationEventBalanceThresholdTypeUsagePercentage},
		{v1api.NotificationRuleBalanceThresholdValueTypeUsageValue, api.NotificationEventBalanceThresholdTypeUsageValue},
		{v1api.NotificationRuleBalanceThresholdValueTypePercent, api.NotificationEventBalanceThresholdTypeUsagePercentage},
		{v1api.NotificationRuleBalanceThresholdValueTypeNumber, api.NotificationEventBalanceThresholdTypeUsageValue},
	}

	for _, tc := range testCases {
		t.Run(string(tc.in), func(t *testing.T) {
			got, err := ToAPIBalanceThresholdType(tc.in)
			require.NoError(t, err)
			assert.Equal(t, tc.want, got)
		})
	}

	t.Run("unknown value is rejected", func(t *testing.T) {
		_, err := ToAPIBalanceThresholdType(v1api.NotificationRuleBalanceThresholdValueType("nonsense"))
		require.Error(t, err)
	})
}

func TestMapAPIEnumFilter(t *testing.T) {
	toDomainState := func(v string) (string, error) {
		domain, err := ToDomainDeliveryState(api.NotificationEventDeliveryState(v))
		return string(domain), err
	}

	t.Run("nil filter maps to nil", func(t *testing.T) {
		got, err := mapAPIEnumFilter(nil, toDomainState)
		require.NoError(t, err)
		assert.Nil(t, got)
	})

	t.Run("eq is translated to the domain value", func(t *testing.T) {
		got, err := mapAPIEnumFilter(&filter.FilterString{Eq: lo.ToPtr("failed")}, toDomainState)
		require.NoError(t, err)
		require.NotNil(t, got)
		assert.Equal(t, "FAILED", lo.FromPtr(got.Eq))
	})

	t.Run("in translates every element", func(t *testing.T) {
		got, err := mapAPIEnumFilter(&filter.FilterString{In: lo.ToPtr([]string{"failed", "pending"})}, toDomainState)
		require.NoError(t, err)
		require.NotNil(t, got)
		assert.Equal(t, []string{"FAILED", "PENDING"}, lo.FromPtr(got.In))
	})

	t.Run("an unknown value fails the whole filter", func(t *testing.T) {
		_, err := mapAPIEnumFilter(&filter.FilterString{In: lo.ToPtr([]string{"failed", "nonsense"})}, toDomainState)
		require.Error(t, err)
	})

	t.Run("the input filter is not mutated", func(t *testing.T) {
		in := &filter.FilterString{Eq: lo.ToPtr("failed")}
		_, err := mapAPIEnumFilter(in, toDomainState)
		require.NoError(t, err)
		assert.Equal(t, "failed", lo.FromPtr(in.Eq))
	})
}

func TestRequireExactFilter(t *testing.T) {
	testCases := []struct {
		name    string
		in      *filter.FilterString
		wantErr bool
	}{
		{name: "nil is allowed", in: nil},
		{name: "empty is allowed", in: &filter.FilterString{}},
		{name: "eq is allowed", in: &filter.FilterString{Eq: lo.ToPtr("a")}},
		{name: "in is allowed", in: &filter.FilterString{In: lo.ToPtr([]string{"a", "b"})}},
		{name: "neq is rejected", in: &filter.FilterString{Ne: lo.ToPtr("a")}, wantErr: true},
		{name: "contains is rejected", in: &filter.FilterString{Contains: lo.ToPtr("a")}, wantErr: true},
		{name: "exists is rejected", in: &filter.FilterString{Exists: lo.ToPtr(true)}, wantErr: true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := requireExactFilter("subject", tc.in)
			if tc.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "filter[subject]")
				return
			}

			require.NoError(t, err)
		})
	}
}

func TestToAPIEvent_BalanceThreshold(t *testing.T) {
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

	got, err := ToAPIEvent(event)
	require.NoError(t, err)

	assert.Equal(t, event.ID, got.Id)
	assert.Equal(t, api.NotificationEventTypeEntitlementsBalanceThreshold, got.Type)
	assert.Equal(t, createdAt, got.CreatedAt)
	assert.Equal(t, event.Rule.ID, got.Rule.Id)
	assert.Equal(t, "threshold rule", got.Rule.Name)
	assert.Equal(t, api.NotificationEventTypeEntitlementsBalanceThreshold, got.Rule.Type)

	payload, err := got.Payload.AsNotificationEventBalanceThresholdPayload()
	require.NoError(t, err)

	assert.Equal(t, event.ID, payload.Id)
	assert.Equal(t, createdAt, payload.Timestamp)
	assert.Equal(t, "01ARZ3NDEKTSV4RRFFQ69G5FAX", payload.Data.EntitlementId)
	assert.Equal(t, "01ARZ3NDEKTSV4RRFFQ69G5FAY", payload.Data.Feature.Id)
	assert.Equal(t, "gpt4_tokens", payload.Data.Feature.Key)
	assert.Equal(t, "customer-1", payload.Data.SubjectKey)
	assert.Equal(t, "01ARZ3NDEKTSV4RRFFQ69G5FAZ", lo.FromPtr(payload.Data.CustomerId))
	assert.True(t, payload.Data.Value.HasAccess)
	assert.Equal(t, 100.0, lo.FromPtr(payload.Data.Value.Balance))
	assert.Equal(t, api.NotificationEventBalanceThresholdTypeUsagePercentage, payload.Data.Threshold.Type)
	assert.Equal(t, 90.0, payload.Data.Threshold.Value)
}

func TestToAPIEvent_EntitlementReset(t *testing.T) {
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

	got, err := ToAPIEvent(event)
	require.NoError(t, err)

	payload, err := got.Payload.AsNotificationEventResetPayload()
	require.NoError(t, err)

	assert.Equal(t, api.NotificationEventResetPayloadTypeEntitlementsReset, payload.Type)
	assert.Equal(t, "01ARZ3NDEKTSV4RRFFQ69G5FAX", payload.Data.EntitlementId)
	assert.Equal(t, "gpt4_tokens", payload.Data.Feature.Key)
	// The customer is absent on this payload, so the optional field must stay unset
	// rather than reporting an empty id.
	assert.Nil(t, payload.Data.CustomerId)
}

func TestToAPIEvent_InvoiceCreated(t *testing.T) {
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

	got, err := ToAPIEvent(event)
	require.NoError(t, err)

	payload, err := got.Payload.AsNotificationEventInvoiceCreatedPayload()
	require.NoError(t, err)

	assert.Equal(t, api.NotificationEventInvoiceCreatedPayloadTypeInvoiceCreated, payload.Type)
	assert.Equal(t, "01ARZ3NDEKTSV4RRFFQ69G5FB0", payload.Data.Invoice.Id)
	assert.Equal(t, "INV-2024-0001", payload.Data.Invoice.Number)
	assert.Equal(t, "01ARZ3NDEKTSV4RRFFQ69G5FB1", lo.FromPtr(payload.Data.CustomerId))
	assert.Equal(t, "USD", payload.Data.Currency)
	assert.Equal(t, string(v1api.InvoiceStatusDraft), payload.Data.Status)
	assert.Equal(t, "123.45", payload.Data.Total)
}

func TestToAPIEvent_MissingPayloadIsRejected(t *testing.T) {
	event := notification.Event{
		NamespacedID: models.NamespacedID{Namespace: "ns", ID: "01ARZ3NDEKTSV4RRFFQ69G5FAV"},
		Type:         notification.EventTypeBalanceThreshold,
		Rule: notification.Rule{
			NamespacedID: models.NamespacedID{Namespace: "ns", ID: "01ARZ3NDEKTSV4RRFFQ69G5FAW"},
			Type:         notification.EventTypeBalanceThreshold,
		},
	}

	_, err := ToAPIEvent(event)
	require.Error(t, err)
}

// TestToAPIDeliveryStatuses covers the two behaviors that differ from v1: the channel is
// reported by id only (no lookup into the rule's channel list), and attempts come back
// newest first.
func TestToAPIDeliveryStatuses(t *testing.T) {
	older := time.Date(2024, 5, 1, 10, 0, 0, 0, time.UTC)
	newer := time.Date(2024, 5, 1, 11, 0, 0, 0, time.UTC)
	nextAttempt := time.Date(2024, 5, 1, 12, 0, 0, 0, time.UTC)

	got, err := ToAPIDeliveryStatuses([]notification.EventDeliveryStatus{
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
	assert.Equal(t, api.NotificationEventDeliveryStateFailed, status.State)
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
