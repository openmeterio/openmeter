package rules

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

func TestFromAPIRuleSortField(t *testing.T) {
	testCases := []struct {
		field   string
		want    notification.OrderBy
		wantErr bool
	}{
		{field: "id", want: notification.OrderByID},
		{field: "type", want: notification.OrderByType},
		{field: "created_at", want: notification.OrderByCreatedAt},
		{field: "updated_at", want: notification.OrderByUpdatedAt},
		{field: "createdAt", wantErr: true},
		{field: "name", wantErr: true},
		{field: "", wantErr: true},
	}

	for _, tc := range testCases {
		t.Run(tc.field, func(t *testing.T) {
			got, err := FromAPIRuleSortField(t.Context(), tc.field)
			if tc.wantErr {
				require.Error(t, err)

				return
			}

			require.NoError(t, err)
			assert.Equal(t, tc.want, got)
		})
	}
}

// TestToDomainBalanceThresholdType pins that the v3 values map onto the stored v1
// constants and that the deprecated v1 aliases are not accepted on the v3 write path.
func TestToDomainBalanceThresholdType(t *testing.T) {
	testCases := []struct {
		in   api.NotificationBalanceThresholdType
		want v1api.NotificationRuleBalanceThresholdValueType
	}{
		{api.NotificationBalanceThresholdTypeBalanceValue, v1api.NotificationRuleBalanceThresholdValueTypeBalanceValue},
		{api.NotificationBalanceThresholdTypeUsagePercentage, v1api.NotificationRuleBalanceThresholdValueTypeUsagePercentage},
		{api.NotificationBalanceThresholdTypeUsageValue, v1api.NotificationRuleBalanceThresholdValueTypeUsageValue},
	}

	for _, tc := range testCases {
		t.Run(string(tc.in), func(t *testing.T) {
			got, err := ToDomainBalanceThresholdType(tc.in)
			require.NoError(t, err)
			assert.Equal(t, tc.want, got)
		})
	}

	for _, legacy := range []string{"NUMBER", "PERCENT", "nonsense"} {
		t.Run(legacy+" is rejected", func(t *testing.T) {
			_, err := ToDomainBalanceThresholdType(api.NotificationBalanceThresholdType(legacy))
			require.Error(t, err)
			assert.True(t, models.IsGenericValidationError(err))
		})
	}
}

func TestFromAPICreateRuleRequest_BalanceThreshold(t *testing.T) {
	var body api.NotificationRuleRequest
	require.NoError(t, body.FromNotificationRuleBalanceThresholdRequest(api.NotificationRuleBalanceThresholdRequest{
		Type:       api.NotificationRuleBalanceThresholdRequestTypeEntitlementsBalanceThreshold,
		Name:       "threshold rule",
		Disabled:   lo.ToPtr(true),
		ChannelIds: []api.ULID{"01ARZ3NDEKTSV4RRFFQ69G5FAV"},
		Thresholds: []api.NotificationBalanceThreshold{
			{Type: api.NotificationBalanceThresholdTypeUsagePercentage, Value: 90},
			{Type: api.NotificationBalanceThresholdTypeBalanceValue, Value: 10},
		},
		Features: lo.ToPtr([]string{"gpt4_tokens"}),
		Labels:   &api.Labels{"team": "billing"},
	}))

	got, err := FromAPICreateRuleRequest("ns", body)
	require.NoError(t, err)

	assert.Equal(t, "ns", got.Namespace)
	assert.Equal(t, notification.EventTypeBalanceThreshold, got.Type)
	assert.Equal(t, notification.EventTypeBalanceThreshold, got.Config.Type)
	assert.Equal(t, "threshold rule", got.Name)
	assert.True(t, got.Disabled)
	assert.Equal(t, []string{"01ARZ3NDEKTSV4RRFFQ69G5FAV"}, got.Channels)
	assert.Equal(t, models.Metadata{"team": "billing"}, got.Metadata)

	require.NotNil(t, got.Config.BalanceThreshold)
	assert.Equal(t, []string{"gpt4_tokens"}, got.Config.BalanceThreshold.Features)
	assert.Equal(t, []notification.BalanceThreshold{
		{Type: v1api.NotificationRuleBalanceThresholdValueTypeUsagePercentage, Value: 90},
		{Type: v1api.NotificationRuleBalanceThresholdValueTypeBalanceValue, Value: 10},
	}, got.Config.BalanceThreshold.Thresholds)
	assert.Nil(t, got.Config.EntitlementReset)
	assert.Nil(t, got.Config.Invoice)
}

func TestFromAPIUpdateRuleRequest_InvoiceCreated(t *testing.T) {
	var body api.NotificationRuleRequest
	require.NoError(t, body.FromNotificationRuleInvoiceCreatedRequest(api.NotificationRuleInvoiceCreatedRequest{
		Type:       api.NotificationRuleInvoiceCreatedRequestTypeInvoiceCreated,
		Name:       "invoice rule",
		ChannelIds: []api.ULID{"01ARZ3NDEKTSV4RRFFQ69G5FAV", "01ARZ3NDEKTSV4RRFFQ69G5FAW"},
	}))

	got, err := FromAPIUpdateRuleRequest("ns", "01ARZ3NDEKTSV4RRFFQ69G5FAX", body)
	require.NoError(t, err)

	assert.Equal(t, models.NamespacedID{Namespace: "ns", ID: "01ARZ3NDEKTSV4RRFFQ69G5FAX"}, got.NamespacedID)
	assert.Equal(t, notification.EventTypeInvoiceCreated, got.Type)
	// An omitted disabled resets the flag: updates are full replacements.
	assert.False(t, got.Disabled)
	assert.Len(t, got.Channels, 2)
	require.NotNil(t, got.Config.Invoice)
	assert.Nil(t, got.Config.BalanceThreshold)
}

func TestFromAPIRuleRequest_EntitlementResetWithoutFeatures(t *testing.T) {
	var body api.NotificationRuleRequest
	require.NoError(t, body.FromNotificationRuleEntitlementResetRequest(api.NotificationRuleEntitlementResetRequest{
		Type:       api.NotificationRuleEntitlementResetRequestTypeEntitlementsReset,
		Name:       "reset rule",
		ChannelIds: []api.ULID{"01ARZ3NDEKTSV4RRFFQ69G5FAV"},
	}))

	got, err := FromAPICreateRuleRequest("ns", body)
	require.NoError(t, err)

	require.NotNil(t, got.Config.EntitlementReset)
	assert.Empty(t, got.Config.EntitlementReset.Features)
}

func TestFromAPIRuleRequest_UnknownTypeIsRejected(t *testing.T) {
	var body api.NotificationRuleRequest
	require.NoError(t, body.UnmarshalJSON([]byte(`{"type":"nonsense","name":"x","channel_ids":[]}`)))

	_, err := FromAPICreateRuleRequest("ns", body)
	require.Error(t, err)
	assert.True(t, models.IsGenericValidationError(err))
}

func TestToAPIRule_BalanceThreshold(t *testing.T) {
	createdAt := time.Date(2024, 5, 1, 12, 0, 0, 0, time.UTC)

	rule := notification.Rule{
		NamespacedID: models.NamespacedID{Namespace: "ns", ID: "01ARZ3NDEKTSV4RRFFQ69G5FAV"},
		ManagedModel: models.ManagedModel{CreatedAt: createdAt, UpdatedAt: createdAt},
		Metadata:     models.Metadata{"team": "billing"},
		Type:         notification.EventTypeBalanceThreshold,
		Name:         "threshold rule",
		Disabled:     true,
		Config: notification.RuleConfig{
			RuleConfigMeta: notification.RuleConfigMeta{Type: notification.EventTypeBalanceThreshold},
			BalanceThreshold: &notification.BalanceThresholdRuleConfig{
				Features: []string{"gpt4_tokens"},
				Thresholds: []notification.BalanceThreshold{
					// The legacy PERCENT spelling must normalize to usage_percentage.
					{Type: v1api.NotificationRuleBalanceThresholdValueTypePercent, Value: 90},
					{Type: v1api.NotificationRuleBalanceThresholdValueTypeBalanceValue, Value: 10},
				},
			},
		},
		Channels: []notification.Channel{
			{NamespacedID: models.NamespacedID{Namespace: "ns", ID: "01ARZ3NDEKTSV4RRFFQ69G5FAW"}},
		},
	}

	got, err := ToAPIRule(rule)
	require.NoError(t, err)

	discriminator, err := got.Discriminator()
	require.NoError(t, err)
	assert.Equal(t, string(notification.EventTypeBalanceThreshold), discriminator)

	v, err := got.AsNotificationRuleBalanceThreshold()
	require.NoError(t, err)
	assert.Equal(t, rule.ID, v.Id)
	assert.Equal(t, "threshold rule", v.Name)
	assert.True(t, lo.FromPtr(v.Disabled))
	assert.Equal(t, []api.ULID{"01ARZ3NDEKTSV4RRFFQ69G5FAW"}, v.ChannelIds)
	assert.Equal(t, createdAt, v.CreatedAt)
	assert.Equal(t, lo.ToPtr([]string{"gpt4_tokens"}), v.Features)
	assert.Equal(t, api.Labels{"team": "billing"}, lo.FromPtr(v.Labels))
	assert.Equal(t, []api.NotificationBalanceThreshold{
		{Type: api.NotificationBalanceThresholdTypeUsagePercentage, Value: 90},
		{Type: api.NotificationBalanceThresholdTypeBalanceValue, Value: 10},
	}, v.Thresholds)
}

func TestToAPIRule_EntitlementResetOmitsEmptyFeatures(t *testing.T) {
	rule := notification.Rule{
		NamespacedID: models.NamespacedID{Namespace: "ns", ID: "01ARZ3NDEKTSV4RRFFQ69G5FAV"},
		Type:         notification.EventTypeEntitlementReset,
		Name:         "reset rule",
		Config: notification.RuleConfig{
			RuleConfigMeta:   notification.RuleConfigMeta{Type: notification.EventTypeEntitlementReset},
			EntitlementReset: &notification.EntitlementResetRuleConfig{},
		},
	}

	got, err := ToAPIRule(rule)
	require.NoError(t, err)

	v, err := got.AsNotificationRuleEntitlementReset()
	require.NoError(t, err)
	assert.Nil(t, v.Features)
	assert.Empty(t, v.ChannelIds)
	assert.False(t, lo.FromPtr(v.Disabled))
}

func TestToAPIRule_Invoice(t *testing.T) {
	for _, ruleType := range []notification.EventType{notification.EventTypeInvoiceCreated, notification.EventTypeInvoiceUpdated} {
		t.Run(string(ruleType), func(t *testing.T) {
			got, err := ToAPIRule(notification.Rule{
				NamespacedID: models.NamespacedID{Namespace: "ns", ID: "01ARZ3NDEKTSV4RRFFQ69G5FAV"},
				Type:         ruleType,
				Name:         "invoice rule",
				Config: notification.RuleConfig{
					RuleConfigMeta: notification.RuleConfigMeta{Type: ruleType},
					Invoice:        &notification.InvoiceRuleConfig{},
				},
			})
			require.NoError(t, err)

			discriminator, err := got.Discriminator()
			require.NoError(t, err)
			assert.Equal(t, string(ruleType), discriminator)
		})
	}
}

func TestToAPIRule_MissingConfigIsRejected(t *testing.T) {
	_, err := ToAPIRule(notification.Rule{
		NamespacedID: models.NamespacedID{Namespace: "ns", ID: "01ARZ3NDEKTSV4RRFFQ69G5FAV"},
		Type:         notification.EventTypeBalanceThreshold,
		Config:       notification.RuleConfig{RuleConfigMeta: notification.RuleConfigMeta{Type: notification.EventTypeBalanceThreshold}},
	})
	require.Error(t, err)
}
