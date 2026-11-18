package service_test

import (
	"testing"

	"github.com/oklog/ulid/v2"
	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	ruledb "github.com/openmeterio/openmeter/openmeter/ent/db/notificationrule"
	"github.com/openmeterio/openmeter/openmeter/notification"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/filter"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/pagination"
)

func TestListRules_RequiresNamespace(t *testing.T) {
	env := newServiceTestEnv(t)

	_, err := env.service.ListRules(t.Context(), notification.ListRulesInput{
		Page: pagination.NewPage(1, 20),
	})
	require.Error(t, err)
	assert.True(t, models.IsGenericValidationError(err), "expected a models.GenericValidationError, got: %v", err)
}

func TestListRules_InvalidFilterCombinationRejected(t *testing.T) {
	env := newServiceTestEnv(t)

	_, err := env.service.ListRules(t.Context(), notification.ListRulesInput{
		Namespace: ulid.Make().String(),
		Page:      pagination.NewPage(1, 20),
		Name: &filter.FilterString{
			Eq:       lo.ToPtr("a"),
			Contains: lo.ToPtr("b"),
		},
	})
	require.Error(t, err)
	assert.True(t, models.IsGenericValidationError(err), "expected a models.GenericValidationError, got: %v", err)
	assert.ErrorIs(t, err, filter.ErrFilterMultipleOperators)
}

func TestCreateRule_InvalidType(t *testing.T) {
	env := newServiceTestEnv(t)
	tests := []struct {
		name      string
		eventType notification.EventType
	}{
		{name: "empty"},
		{name: "unknown", eventType: "unknown.event"},
		{name: "unknown versioned name", eventType: "v1.invoice.created"},
		{name: "balance storage alias", eventType: "v1.entitlement.balance_threshold.triggered"},
		{name: "reset storage alias", eventType: "v1.entitlement.usage_period.reset"},
		{name: "invoice created storage alias", eventType: "v1.billing.invoice.created"},
		{name: "invoice updated storage alias", eventType: "v1.billing.invoice.updated"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// given: valid dependencies and config with an unsupported public type
			namespace := ulid.Make().String()
			channel := seedChannel(t, env, namespace, clock.Now(), "validation")

			// when: creating through the service boundary
			created, err := env.service.CreateRule(t.Context(), notification.CreateRuleInput{
				NamespacedModel: models.NamespacedModel{Namespace: namespace},
				Type:            tt.eventType,
				Name:            "invalid type",
				Channels:        []string{channel.ID},
				Config: notification.RuleConfig{
					RuleConfigMeta: notification.RuleConfigMeta{Type: notification.EventTypeInvoiceCreated},
					Invoice:        &notification.InvoiceRuleConfig{},
				},
			})
			require.ErrorContains(t, err, "invalid notification event type")

			assert.True(t, models.IsGenericValidationError(err))
			assert.Nil(t, created)

			// then: validation prevents persistence
			count, err := env.db.NotificationRule.Query().Where(ruledb.Namespace(namespace)).Count(t.Context())
			require.NoError(t, err)

			assert.Zero(t, count)
		})
	}
}

func TestRuleConfigWriteValidation(t *testing.T) {
	env := newServiceTestEnv(t)
	tests := []struct {
		name   string
		config notification.RuleConfig
	}{
		{
			name: "missing invoice config",
			config: notification.RuleConfig{
				RuleConfigMeta: notification.RuleConfigMeta{Type: notification.EventTypeInvoiceCreated},
			},
		},
		{
			name: "invalid threshold",
			config: notification.RuleConfig{
				RuleConfigMeta: notification.RuleConfigMeta{Type: notification.EventTypeBalanceThreshold},
				BalanceThreshold: &notification.BalanceThresholdRuleConfig{
					Thresholds: []notification.BalanceThreshold{
						{Type: notification.BalanceThresholdTypeUsagePercentage, Value: -1},
					},
				},
			},
		},
		{
			name: "duplicate features",
			config: notification.RuleConfig{
				RuleConfigMeta:   notification.RuleConfigMeta{Type: notification.EventTypeEntitlementReset},
				EntitlementReset: &notification.EntitlementResetRuleConfig{Features: []string{"feature-a", "feature-a"}},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// given: a valid persisted rule and an invalid configuration for its type
			namespace := ulid.Make().String()
			channel := seedChannel(t, env, namespace, clock.Now(), "validation")
			validConfig := notification.RuleConfig{RuleConfigMeta: notification.RuleConfigMeta{Type: tt.config.Type}}
			switch tt.config.Type {
			case notification.EventTypeInvoiceCreated:
				validConfig.Invoice = &notification.InvoiceRuleConfig{}
			case notification.EventTypeBalanceThreshold:
				validConfig.BalanceThreshold = &notification.BalanceThresholdRuleConfig{
					Thresholds: []notification.BalanceThreshold{
						{Type: notification.BalanceThresholdTypeUsagePercentage, Value: 50},
					},
				}
			case notification.EventTypeEntitlementReset:
				validConfig.EntitlementReset = &notification.EntitlementResetRuleConfig{}
			}

			created, err := env.adapter.CreateRule(t.Context(), notification.CreateRuleInput{
				NamespacedModel: models.NamespacedModel{Namespace: namespace},
				Name:            "original",
				Type:            tt.config.Type,
				Config:          validConfig,
				Channels:        []string{channel.ID},
			})
			require.NoError(t, err)

			before, err := env.service.GetRule(t.Context(), notification.GetRuleInput(created.NamespacedID))
			require.NoError(t, err)

			for _, operation := range []string{"create", "update"} {
				t.Run(operation, func(t *testing.T) {
					// when: the service receives the invalid config
					var result *notification.Rule
					var err error

					switch operation {
					case "create":
						result, err = env.service.CreateRule(t.Context(), notification.CreateRuleInput{
							NamespacedModel: models.NamespacedModel{Namespace: namespace},
							Name:            "invalid",
							Type:            tt.config.Type,
							Config:          tt.config,
							Channels:        []string{channel.ID},
						})
					case "update":
						result, err = env.service.UpdateRule(t.Context(), notification.UpdateRuleInput{
							NamespacedID: created.NamespacedID,
							Name:         "invalid",
							Type:         tt.config.Type,
							Config:       tt.config,
							Channels:     []string{channel.ID},
						})
					}

					require.Error(t, err)

					assert.True(t, models.IsGenericValidationError(err))
					assert.Nil(t, result)

					// then: the existing rule survives and no additional rule is persisted
					after, err := env.service.GetRule(t.Context(), notification.GetRuleInput(created.NamespacedID))
					require.NoError(t, err)

					count, err := env.db.NotificationRule.Query().Where(ruledb.Namespace(namespace)).Count(t.Context())
					require.NoError(t, err)

					assert.Equal(t, before, after)
					assert.Equal(t, 1, count)
				})
			}
		})
	}
}
