package rules

import (
	"testing"
	"time"

	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apiv1models "github.com/openmeterio/openmeter/api/models"
	apiv3models "github.com/openmeterio/openmeter/api/v3/models"
	"github.com/openmeterio/openmeter/openmeter/notification"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/feature"
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

// TestFromAPIBillingNotificationBalanceThresholdType pins that the v3 values map onto the stored v1
// constants and that the deprecated v1 aliases are not accepted on the v3 write path.
func TestFromAPIBillingNotificationBalanceThresholdType(t *testing.T) {
	testCases := []struct {
		in   apiv3models.BillingNotificationBalanceThresholdType
		want apiv1models.NotificationRuleBalanceThresholdValueType
	}{
		{apiv3models.BillingNotificationBalanceThresholdTypeBalanceValue, apiv1models.NotificationRuleBalanceThresholdValueTypeBalanceValue},
		{apiv3models.BillingNotificationBalanceThresholdTypeUsagePercentage, apiv1models.NotificationRuleBalanceThresholdValueTypeUsagePercentage},
		{apiv3models.BillingNotificationBalanceThresholdTypeUsageValue, apiv1models.NotificationRuleBalanceThresholdValueTypeUsageValue},
	}

	for _, tc := range testCases {
		t.Run(string(tc.in), func(t *testing.T) {
			got, err := FromAPIBillingNotificationBalanceThresholdType(tc.in)
			require.NoError(t, err)
			assert.Equal(t, tc.want, got)
		})
	}

	for _, legacy := range []string{"NUMBER", "PERCENT", "nonsense"} {
		t.Run(legacy+" is rejected", func(t *testing.T) {
			_, err := FromAPIBillingNotificationBalanceThresholdType(apiv3models.BillingNotificationBalanceThresholdType(legacy))
			require.Error(t, err)
			assert.True(t, models.IsGenericValidationError(err))
		})
	}
}

func TestFromAPICreateRuleRequest_BalanceThreshold(t *testing.T) {
	var body apiv3models.BillingNotificationRuleRequest
	require.NoError(t, body.FromBillingNotificationRuleBalanceThresholdRequest(apiv3models.BillingNotificationRuleBalanceThresholdRequest{
		Type:     apiv3models.BillingNotificationRuleBalanceThresholdRequestTypeEntitlementsBalanceThreshold,
		Name:     "threshold rule",
		Disabled: lo.ToPtr(true),
		Channels: []apiv3models.NotificationChannelReference{{Id: "01ARZ3NDEKTSV4RRFFQ69G5FAV"}},
		Thresholds: []apiv3models.BillingNotificationBalanceThreshold{
			{Type: apiv3models.BillingNotificationBalanceThresholdTypeUsagePercentage, Value: 90},
			{Type: apiv3models.BillingNotificationBalanceThresholdTypeBalanceValue, Value: 10},
		},
		Features: &[]apiv3models.FeatureReference{{Id: "01ARZ3NDEKTSV4RRFFQ69G5FAX"}},
		Labels:   &apiv3models.Labels{"team": "billing"},
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
	assert.Equal(t, []string{"01ARZ3NDEKTSV4RRFFQ69G5FAX"}, got.Config.BalanceThreshold.Features)
	assert.Equal(t, []notification.BalanceThreshold{
		{Type: apiv1models.NotificationRuleBalanceThresholdValueTypeUsagePercentage, Value: 90},
		{Type: apiv1models.NotificationRuleBalanceThresholdValueTypeBalanceValue, Value: 10},
	}, got.Config.BalanceThreshold.Thresholds)
	assert.Nil(t, got.Config.EntitlementReset)
	assert.Nil(t, got.Config.Invoice)
}

func TestFromAPIUpdateRuleRequest_InvoiceCreated(t *testing.T) {
	var body apiv3models.BillingNotificationRuleRequest
	require.NoError(t, body.FromBillingNotificationRuleInvoiceCreatedRequest(apiv3models.BillingNotificationRuleInvoiceCreatedRequest{
		Type:     apiv3models.BillingNotificationRuleInvoiceCreatedRequestTypeInvoiceCreated,
		Name:     "invoice rule",
		Channels: []apiv3models.NotificationChannelReference{{Id: "01ARZ3NDEKTSV4RRFFQ69G5FAV"}, {Id: "01ARZ3NDEKTSV4RRFFQ69G5FAW"}},
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
	var body apiv3models.BillingNotificationRuleRequest
	require.NoError(t, body.FromBillingNotificationRuleEntitlementResetRequest(apiv3models.BillingNotificationRuleEntitlementResetRequest{
		Type:     apiv3models.BillingNotificationRuleEntitlementResetRequestTypeEntitlementsReset,
		Name:     "reset rule",
		Channels: []apiv3models.NotificationChannelReference{{Id: "01ARZ3NDEKTSV4RRFFQ69G5FAV"}},
	}))

	got, err := FromAPICreateRuleRequest("ns", body)
	require.NoError(t, err)

	require.NotNil(t, got.Config.EntitlementReset)
	assert.Empty(t, got.Config.EntitlementReset.Features)
}

func TestFromAPIRuleRequest_UnknownTypeIsRejected(t *testing.T) {
	var body apiv3models.BillingNotificationRuleRequest
	require.NoError(t, body.UnmarshalJSON([]byte(`{"type":"nonsense","name":"x","channels":[]}`)))

	_, err := FromAPICreateRuleRequest("ns", body)
	require.Error(t, err)
	assert.True(t, models.IsGenericValidationError(err))
}

func TestToAPIBillingNotificationRule_BalanceThreshold(t *testing.T) {
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
					{Type: apiv1models.NotificationRuleBalanceThresholdValueTypePercent, Value: 90},
					{Type: apiv1models.NotificationRuleBalanceThresholdValueTypeBalanceValue, Value: 10},
				},
			},
		},
		Channels: []notification.Channel{
			{NamespacedID: models.NamespacedID{Namespace: "ns", ID: "01ARZ3NDEKTSV4RRFFQ69G5FAW"}},
		},
	}

	got, err := ToAPIBillingNotificationRule(notification.RuleView{Rule: rule, Features: []feature.Feature{{ID: "01ARZ3NDEKTSV4RRFFQ69G5FAX", Key: "gpt4_tokens"}}})
	require.NoError(t, err)

	discriminator, err := got.Discriminator()
	require.NoError(t, err)
	assert.Equal(t, string(notification.EventTypeBalanceThreshold), discriminator)

	v, err := got.AsBillingNotificationRuleBalanceThreshold()
	require.NoError(t, err)
	assert.Equal(t, rule.ID, v.Id)
	assert.Equal(t, "threshold rule", v.Name)
	assert.True(t, lo.FromPtr(v.Disabled))
	assert.Equal(t, []apiv3models.NotificationChannelReference{{Id: "01ARZ3NDEKTSV4RRFFQ69G5FAW"}}, v.Channels)
	assert.Equal(t, createdAt, v.CreatedAt)
	assert.Equal(t, &[]apiv3models.FeatureReference{{Id: "01ARZ3NDEKTSV4RRFFQ69G5FAX"}}, v.Features)
	assert.Equal(t, apiv3models.Labels{"team": "billing"}, lo.FromPtr(v.Labels))
	assert.Equal(t, []apiv3models.BillingNotificationBalanceThreshold{
		{Type: apiv3models.BillingNotificationBalanceThresholdTypeUsagePercentage, Value: 90},
		{Type: apiv3models.BillingNotificationBalanceThresholdTypeBalanceValue, Value: 10},
	}, v.Thresholds)
}

func TestToAPIBillingNotificationRule_EntitlementResetOmitsEmptyFeatures(t *testing.T) {
	rule := notification.Rule{
		NamespacedID: models.NamespacedID{Namespace: "ns", ID: "01ARZ3NDEKTSV4RRFFQ69G5FAV"},
		Type:         notification.EventTypeEntitlementReset,
		Name:         "reset rule",
		Config: notification.RuleConfig{
			RuleConfigMeta:   notification.RuleConfigMeta{Type: notification.EventTypeEntitlementReset},
			EntitlementReset: &notification.EntitlementResetRuleConfig{},
		},
	}

	got, err := ToAPIBillingNotificationRule(notification.RuleView{Rule: rule})
	require.NoError(t, err)

	v, err := got.AsBillingNotificationRuleEntitlementReset()
	require.NoError(t, err)
	assert.Nil(t, v.Features)
	assert.Empty(t, v.Channels)
	assert.False(t, lo.FromPtr(v.Disabled))
}

func TestToAPIBillingNotificationRule_Invoice(t *testing.T) {
	for _, ruleType := range []notification.EventType{notification.EventTypeInvoiceCreated, notification.EventTypeInvoiceUpdated} {
		t.Run(string(ruleType), func(t *testing.T) {
			got, err := ToAPIBillingNotificationRule(notification.RuleView{Rule: notification.Rule{
				NamespacedID: models.NamespacedID{Namespace: "ns", ID: "01ARZ3NDEKTSV4RRFFQ69G5FAV"},
				Type:         ruleType,
				Name:         "invoice rule",
				Config: notification.RuleConfig{
					RuleConfigMeta: notification.RuleConfigMeta{Type: ruleType},
					Invoice:        &notification.InvoiceRuleConfig{},
				},
			}})
			require.NoError(t, err)

			discriminator, err := got.Discriminator()
			require.NoError(t, err)
			assert.Equal(t, string(ruleType), discriminator)
		})
	}
}

func TestToAPIBillingNotificationRule_MissingConfigIsRejected(t *testing.T) {
	_, err := ToAPIBillingNotificationRule(notification.RuleView{Rule: notification.Rule{
		NamespacedID: models.NamespacedID{Namespace: "ns", ID: "01ARZ3NDEKTSV4RRFFQ69G5FAV"},
		Type:         notification.EventTypeBalanceThreshold,
		Config:       notification.RuleConfig{RuleConfigMeta: notification.RuleConfigMeta{Type: notification.EventTypeBalanceThreshold}},
	}})
	require.Error(t, err)
}
