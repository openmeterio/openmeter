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
		name     string
		ruleType notification.EventType
		config   notification.RuleConfig
	}{
		{
			name:     "missing invoice config",
			ruleType: notification.EventTypeInvoiceCreated,
			config: notification.RuleConfig{
				RuleConfigMeta: notification.RuleConfigMeta{Type: notification.EventTypeInvoiceCreated},
			},
		},
		{
			name:     "invalid threshold",
			ruleType: notification.EventTypeBalanceThreshold,
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
			name:     "duplicate features",
			ruleType: notification.EventTypeEntitlementReset,
			config: notification.RuleConfig{
				RuleConfigMeta:   notification.RuleConfigMeta{Type: notification.EventTypeEntitlementReset},
				EntitlementReset: &notification.EntitlementResetRuleConfig{Features: []string{"feature-a", "feature-a"}},
			},
		},
		{
			name:     "reset config for balance threshold rule",
			ruleType: notification.EventTypeBalanceThreshold,
			config: notification.RuleConfig{
				RuleConfigMeta:   notification.RuleConfigMeta{Type: notification.EventTypeEntitlementReset},
				EntitlementReset: &notification.EntitlementResetRuleConfig{},
			},
		},
		{
			name:     "invoice config for reset rule",
			ruleType: notification.EventTypeEntitlementReset,
			config: notification.RuleConfig{
				RuleConfigMeta: notification.RuleConfigMeta{Type: notification.EventTypeInvoiceCreated},
				Invoice:        &notification.InvoiceRuleConfig{},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// given: a valid persisted rule and an invalid configuration for its type
			namespace := ulid.Make().String()
			channel := seedChannel(t, env, namespace, clock.Now(), "validation")
			validConfig := notification.RuleConfig{RuleConfigMeta: notification.RuleConfigMeta{Type: tt.ruleType}}
			switch tt.ruleType {
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
				Type:            tt.ruleType,
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
							Type:            tt.ruleType,
							Config:          tt.config,
							Channels:        []string{channel.ID},
						})
					case "update":
						result, err = env.service.UpdateRule(t.Context(), notification.UpdateRuleInput{
							NamespacedID: created.NamespacedID,
							Name:         "invalid",
							Type:         tt.ruleType,
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

func TestRuleInputConfigTypeValidation(t *testing.T) {
	configs := []notification.RuleConfig{
		{
			RuleConfigMeta: notification.RuleConfigMeta{Type: notification.EventTypeBalanceThreshold},
			BalanceThreshold: &notification.BalanceThresholdRuleConfig{
				Thresholds: []notification.BalanceThreshold{
					{Type: notification.BalanceThresholdTypeUsagePercentage, Value: 50},
				},
			},
		},
		{
			RuleConfigMeta:   notification.RuleConfigMeta{Type: notification.EventTypeEntitlementReset},
			EntitlementReset: &notification.EntitlementResetRuleConfig{},
		},
		{
			RuleConfigMeta: notification.RuleConfigMeta{Type: notification.EventTypeInvoiceCreated},
			Invoice:        &notification.InvoiceRuleConfig{},
		},
		{
			RuleConfigMeta: notification.RuleConfigMeta{Type: notification.EventTypeInvoiceUpdated},
			Invoice:        &notification.InvoiceRuleConfig{},
		},
	}

	for _, ruleConfig := range configs {
		for _, config := range configs {
			t.Run(string(ruleConfig.Type)+"/"+string(config.Type), func(t *testing.T) {
				create := notification.CreateRuleInput{
					NamespacedModel: models.NamespacedModel{Namespace: ulid.Make().String()},
					Name:            "type compatibility",
					Type:            ruleConfig.Type,
					Config:          config,
					Channels:        []string{ulid.Make().String()},
				}
				update := notification.UpdateRuleInput{
					NamespacedID: models.NamespacedID{Namespace: create.Namespace, ID: ulid.Make().String()},
					Name:         create.Name,
					Type:         create.Type,
					Config:       config,
					Channels:     create.Channels,
				}

				for _, operation := range []struct {
					name     string
					validate func() error
				}{
					{name: "create", validate: create.Validate},
					{name: "update", validate: update.Validate},
					{
						name: "custom validator",
						validate: func() error {
							return config.ValidateWith(notification.ValidateRuleConfigWithEventType(ruleConfig.Type))
						},
					},
				} {
					t.Run(operation.name, func(t *testing.T) {
						err := operation.validate()
						if ruleConfig.Type == config.Type || (ruleConfig.Invoice != nil && config.Invoice != nil) {
							require.NoError(t, err)

							return
						}

						require.ErrorContains(t, err, "does not match rule type")

						assert.True(t, models.IsGenericValidationError(err))
					})
				}
			})
		}
	}
}

func TestRuleSharedInvoiceConfig(t *testing.T) {
	env := newServiceTestEnv(t)
	tests := []struct {
		name       string
		ruleType   notification.EventType
		configType notification.EventType
	}{
		{
			name:       "created rule with updated config",
			ruleType:   notification.EventTypeInvoiceCreated,
			configType: notification.EventTypeInvoiceUpdated,
		},
		{
			name:       "updated rule with created config",
			ruleType:   notification.EventTypeInvoiceUpdated,
			configType: notification.EventTypeInvoiceCreated,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// given: the two invoice types share one configuration contract
			namespace := ulid.Make().String()
			channel := seedChannel(t, env, namespace, clock.Now(), "invoice")
			config := notification.RuleConfig{
				RuleConfigMeta: notification.RuleConfigMeta{Type: tt.configType},
				Invoice:        &notification.InvoiceRuleConfig{},
			}

			created, err := env.adapter.CreateRule(t.Context(), notification.CreateRuleInput{
				NamespacedModel: models.NamespacedModel{Namespace: namespace},
				Name:            "created",
				Type:            tt.ruleType,
				Config:          config,
				Channels:        []string{channel.ID},
			})
			require.NoError(t, err)

			// when: updating with the other invoice type's config and unchanged channels
			updated, err := env.service.UpdateRule(t.Context(), notification.UpdateRuleInput{
				NamespacedID: created.NamespacedID,
				Name:         "updated",
				Type:         tt.ruleType,
				Config:       config,
				Channels:     []string{channel.ID},
			})
			require.NoError(t, err)

			// then: the stored Rule keeps its type and can be read normally
			stored, err := env.service.GetRule(t.Context(), notification.GetRuleInput(updated.NamespacedID))
			require.NoError(t, err)

			assert.Equal(t, "updated", stored.Name)
			assert.Equal(t, tt.ruleType, stored.Type)
			assert.Equal(t, tt.ruleType, stored.Config.Type)
			assert.Equal(t, &notification.InvoiceRuleConfig{}, stored.Config.Invoice)
		})
	}
}
