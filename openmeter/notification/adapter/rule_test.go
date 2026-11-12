package adapter

import (
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/oklog/ulid/v2"
	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	entdb "github.com/openmeterio/openmeter/openmeter/ent/db"
	ruledb "github.com/openmeterio/openmeter/openmeter/ent/db/notificationrule"
	"github.com/openmeterio/openmeter/openmeter/notification"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/filter"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/pagination"
)

// createRule seeds an invoice.created rule via the adapter directly, mirroring
// createChannel: service-level CreateRule also registers the rule on the channels'
// webhooks, which the noop handler cannot do.
func createRule(t *testing.T, env *channelTestEnv, ns string, at time.Time, mutate func(*notification.CreateRuleInput)) notification.Rule {
	t.Helper()

	clock.FreezeTime(at)
	defer clock.UnFreeze()

	in := notification.CreateRuleInput{
		NamespacedModel: models.NamespacedModel{Namespace: ns},
		Type:            notification.EventTypeInvoiceCreated,
		Name:            "rule-" + ulid.Make().String(),
		Config: notification.RuleConfig{
			RuleConfigMeta: notification.RuleConfigMeta{Type: notification.EventTypeInvoiceCreated},
			Invoice:        &notification.InvoiceRuleConfig{},
		},
	}
	if mutate != nil {
		mutate(&in)
	}

	rule, err := env.adapter.CreateRule(t.Context(), in)
	require.NoError(t, err, "seeding rule must not fail")
	require.NotNil(t, rule)

	return *rule
}

func TestListRules_Filters(t *testing.T) {
	env := newChannelTestEnv(t)
	ns := ulid.Make().String()

	frozen := clock.Now()
	t0 := frozen.Add(-2 * time.Hour)
	t1 := frozen.Add(-1 * time.Hour)

	channelA := createChannel(t, env, ns, t0, nil)
	channelB := createChannel(t, env, ns, t0, nil)
	disabledChannel := createChannel(t, env, ns, t0, func(in *notification.CreateChannelInput) {
		in.Disabled = true
	})

	first := createRule(t, env, ns, t0, func(in *notification.CreateRuleInput) {
		in.Name = "first-rule"
		in.Channels = []string{channelA.ID}
	})

	second := createRule(t, env, ns, t1, func(in *notification.CreateRuleInput) {
		in.Name = "second-rule"
		in.Type = notification.EventTypeInvoiceUpdated
		in.Config.Type = notification.EventTypeInvoiceUpdated
		in.Channels = []string{channelB.ID, disabledChannel.ID}
	})

	disabled := createRule(t, env, ns, frozen, func(in *notification.CreateRuleInput) {
		in.Name = "disabled-rule"
		in.Disabled = true
		in.Channels = []string{channelA.ID}
	})

	testCases := []struct {
		name    string
		input   notification.ListRulesInput
		wantIDs []string
	}{
		{
			name:    "no filter returns every rule, disabled included",
			wantIDs: []string{first.ID, second.ID, disabled.ID},
		},
		{
			name: "ID eq matches exactly one rule",
			input: notification.ListRulesInput{
				ID: &filter.FilterULID{FilterString: filter.FilterString{Eq: lo.ToPtr(first.ID)}},
			},
			wantIDs: []string{first.ID},
		},
		{
			name: "ID in matches the selected rules",
			input: notification.ListRulesInput{
				ID: &filter.FilterULID{FilterString: filter.FilterString{In: lo.ToPtr([]string{first.ID, second.ID})}},
			},
			wantIDs: []string{first.ID, second.ID},
		},
		{
			name: "Name eq matches exactly one rule",
			input: notification.ListRulesInput{
				Name: &filter.FilterString{Eq: lo.ToPtr("second-rule")},
			},
			wantIDs: []string{second.ID},
		},
		{
			name: "Name contains matches by substring",
			input: notification.ListRulesInput{
				Name: &filter.FilterString{Contains: lo.ToPtr("first")},
			},
			wantIDs: []string{first.ID},
		},
		{
			name: "Type eq matches rules of that type only",
			input: notification.ListRulesInput{
				Type: &filter.FilterString{Eq: lo.ToPtr(string(notification.EventTypeInvoiceCreated))},
			},
			wantIDs: []string{first.ID, disabled.ID},
		},
		{
			name: "Disabled eq false excludes the disabled rule",
			input: notification.ListRulesInput{
				Disabled: &filter.FilterBoolean{Eq: lo.ToPtr(false)},
			},
			wantIDs: []string{first.ID, second.ID},
		},
		{
			name: "Disabled eq true matches only the disabled rule",
			input: notification.ListRulesInput{
				Disabled: &filter.FilterBoolean{Eq: lo.ToPtr(true)},
			},
			wantIDs: []string{disabled.ID},
		},
		{
			name: "ChannelID eq matches every rule targeting the channel",
			input: notification.ListRulesInput{
				ChannelID: &filter.FilterString{Eq: lo.ToPtr(channelA.ID)},
			},
			wantIDs: []string{first.ID, disabled.ID},
		},
		{
			name: "ChannelID in matches rules targeting any of the channels",
			input: notification.ListRulesInput{
				ChannelID: &filter.FilterString{In: lo.ToPtr([]string{channelA.ID, channelB.ID})},
			},
			wantIDs: []string{first.ID, second.ID, disabled.ID},
		},
		{
			// DeleteChannel refuses to delete a channel still assigned to a rule, so the
			// assignment must stay visible even once the channel is disabled.
			name: "ChannelID matches a disabled channel assignment",
			input: notification.ListRulesInput{
				ChannelID: &filter.FilterString{Eq: lo.ToPtr(disabledChannel.ID)},
			},
			wantIDs: []string{second.ID},
		},
		{
			name: "CreatedAt range matches only the rule created inside the window",
			input: notification.ListRulesInput{
				CreatedAt: &filter.FilterTime{
					And: &[]filter.FilterTime{
						{Gte: lo.ToPtr(t0.Add(-time.Minute))},
						{Lte: lo.ToPtr(t0.Add(time.Minute))},
					},
				},
			},
			wantIDs: []string{first.ID},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			input := tc.input
			input.Namespace = ns
			input.IncludeDisabled = true
			input.Page = pagination.NewPage(1, 20)

			result, err := env.adapter.ListRules(t.Context(), input)
			require.NoError(t, err)

			gotIDs := make([]string, 0, len(result.Items))
			for _, item := range result.Items {
				gotIDs = append(gotIDs, item.ID)
			}

			assert.ElementsMatch(t, tc.wantIDs, gotIDs)
		})
	}
}

func TestRuleChannelConfigCompatibility(t *testing.T) {
	tests := []struct {
		name    string
		data    string
		wantErr bool
	}{
		{
			name: "flat config",
			data: `{"url":"https://example.com/hook","custom_headers":{"X-Team":"billing"},"signing_secret":"original-secret"}`,
		},
		{
			name:    "malformed config",
			data:    `{"type":"WEBHOOK","data":[]}`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// given: a rule referencing a channel whose config uses a new or malformed shape
			env := newChannelTestEnv(t)
			channel := createChannel(t, env, ulid.Make().String(), clock.Now(), nil)
			created := createRule(t, env, channel.Namespace, clock.Now(), func(input *notification.CreateRuleInput) {
				input.Channels = []string{channel.ID}
			})

			_, err := env.db.NotificationChannel.UpdateOneID(channel.ID).
				SetConfig(json.RawMessage(tt.data)).Save(t.Context())
			require.NoError(t, err)

			// when: reading the rule with its nested channels
			item, getErr := env.adapter.GetRule(t.Context(), notification.GetRuleInput(created.NamespacedID))
			listed, listErr := env.adapter.ListRules(t.Context(), notification.ListRulesInput{
				Namespace: created.Namespace, Page: pagination.NewPage(1, 20),
			})

			// then: config values or decoding errors propagate through both read paths
			if tt.wantErr {
				require.ErrorContains(t, getErr, "invalid notification channel config")
				require.ErrorContains(t, listErr, "invalid notification channel config")

				assert.Nil(t, item)

				return
			}

			require.NoError(t, getErr)
			require.NoError(t, listErr)
			require.Len(t, listed.Items, 1)

			require.Len(t, item.Channels, 1)
			require.Len(t, listed.Items[0].Channels, 1)

			assert.Equal(t, "original-secret", item.Channels[0].Config.WebHook.SigningSecret)
			assert.Equal(t, map[string]string{"X-Team": "billing"}, item.Channels[0].Config.WebHook.CustomHeaders)
			assert.Equal(t, item.Channels[0].Config, listed.Items[0].Channels[0].Config)
		})
	}
}

func TestRuleTypeStorageCompatibility(t *testing.T) {
	env := newChannelTestEnv(t)
	tests := []struct {
		name      string
		eventType notification.EventType
		config    notification.RuleConfig
	}{
		{
			name:      "entitlements.balance.threshold",
			eventType: notification.EventTypeBalanceThreshold,
			config: notification.RuleConfig{
				BalanceThreshold: &notification.BalanceThresholdRuleConfig{
					Features: []string{"feature-a"},
					Thresholds: []notification.BalanceThreshold{
						{Type: notification.BalanceThresholdTypeUsagePercentage, Value: 50},
					},
				},
			},
		},
		{
			name:      "entitlements.reset",
			eventType: notification.EventTypeEntitlementReset,
			config: notification.RuleConfig{
				EntitlementReset: &notification.EntitlementResetRuleConfig{Features: []string{"feature-a"}},
			},
		},
		{
			name:      "invoice.created",
			eventType: notification.EventTypeInvoiceCreated,
			config:    notification.RuleConfig{Invoice: &notification.InvoiceRuleConfig{}},
		},
		{
			name:      "invoice.updated",
			eventType: notification.EventTypeInvoiceUpdated,
			config:    notification.RuleConfig{Invoice: &notification.InvoiceRuleConfig{}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// given: a rule using an existing unversioned type and a nonmatching rule
			channel := createChannel(t, env, ulid.Make().String(), clock.Now(), nil)
			createRule(t, env, channel.Namespace, clock.Now(), func(input *notification.CreateRuleInput) {
				if tt.eventType == notification.EventTypeInvoiceCreated {
					input.Type = notification.EventTypeInvoiceUpdated
					input.Config.Type = notification.EventTypeInvoiceUpdated
				}
			})

			created := createRule(t, env, channel.Namespace, clock.Now(), func(input *notification.CreateRuleInput) {
				input.Type = tt.eventType
				input.Config = tt.config
				input.Config.Type = tt.eventType
				input.Channels = []string{channel.ID}
			})

			var storedType, storedConfig string
			err := env.sqlDB.QueryRowContext(t.Context(),
				"SELECT type, config FROM notification_rules WHERE id = $1", created.ID,
			).Scan(&storedType, &storedConfig)
			require.NoError(t, err)

			assert.Equal(t, tt.name, storedType)

			// when: getting and filtering rules through the existing adapter
			fetched, err := env.adapter.GetRule(t.Context(), notification.GetRuleInput(created.NamespacedID))
			require.NoError(t, err)

			listed, err := env.adapter.ListRules(t.Context(), notification.ListRulesInput{
				Namespace: created.Namespace,
				Page:      pagination.NewPage(1, 20),
				Type:      &filter.FilterString{Eq: &tt.name},
			})
			require.NoError(t, err)
			require.Len(t, listed.Items, 1)

			// then: the type, configuration, and channel assignment retain their meaning
			assert.Equal(t, created.ID, listed.Items[0].ID)
			assert.Equal(t, tt.eventType, fetched.Type)
			assert.Equal(t, created.Config, fetched.Config)
			assert.Equal(t, created.Config, listed.Items[0].Config)

			require.Len(t, fetched.Channels, 1)

			assert.Equal(t, channel.ID, fetched.Channels[0].ID)

			// when: updating mutable fields through the legacy path
			updated, err := env.adapter.UpdateRule(t.Context(), notification.UpdateRuleInput{
				NamespacedID: created.NamespacedID,
				Type:         created.Type,
				Name:         "updated",
				Disabled:     true,
				Config:       created.Config,
				Channels:     []string{channel.ID},
			})
			require.NoError(t, err)

			// then: storage remains readable by old backends without type/config conversion
			assert.Equal(t, created.ID, updated.ID)
			assert.Equal(t, created.CreatedAt, updated.CreatedAt)
			assert.Equal(t, tt.eventType, updated.Type)
			assert.Equal(t, "updated", updated.Name)
			assert.True(t, updated.Disabled)

			require.Len(t, updated.Channels, 1)

			assert.Equal(t, channel.ID, updated.Channels[0].ID)

			var updatedType, updatedConfig string
			err = env.sqlDB.QueryRowContext(t.Context(),
				"SELECT type, config FROM notification_rules WHERE id = $1", updated.ID,
			).Scan(&updatedType, &updatedConfig)
			require.NoError(t, err)

			assert.Equal(t, storedType, updatedType)
			assert.JSONEq(t, storedConfig, updatedConfig)
		})
	}
}

func TestRuleTypeValidation(t *testing.T) {
	env := newChannelTestEnv(t)
	tests := []struct {
		name      string
		eventType notification.EventType
	}{
		{name: "empty"},
		{name: "unknown", eventType: "unknown.event"},
		{name: "versioned types remain unsupported", eventType: "v1.invoice.created"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// given: an unsupported type with otherwise valid rule configuration
			namespace := ulid.Make().String()

			// when: writing through the adapter, without service validation
			_, err := env.adapter.CreateRule(t.Context(), notification.CreateRuleInput{
				NamespacedModel: models.NamespacedModel{Namespace: namespace},
				Type:            tt.eventType,
				Name:            "invalid type",
				Config: notification.RuleConfig{
					RuleConfigMeta: notification.RuleConfigMeta{Type: notification.EventTypeInvoiceCreated},
					Invoice:        &notification.InvoiceRuleConfig{},
				},
			})
			require.ErrorContains(t, err, "invalid notification event type")

			// then: the existing domain validator still prevents persistence
			count, err := env.db.NotificationRule.Query().Where(ruledb.Namespace(namespace)).Count(t.Context())
			require.NoError(t, err)

			assert.Zero(t, count)
		})
	}
}

func TestRuleConfigStorageCompatibility(t *testing.T) {
	env := newChannelTestEnv(t)
	tests := []struct {
		eventType notification.EventType
		payload   string
		config    notification.RuleConfig
	}{
		{
			eventType: notification.EventTypeBalanceThreshold,
			payload:   `{"features":["feature-a"],"thresholds":[{"type":"usage_percentage","value":50}]}`,
			config: notification.RuleConfig{
				BalanceThreshold: &notification.BalanceThresholdRuleConfig{
					Features: []string{"feature-a"},
					Thresholds: []notification.BalanceThreshold{
						{Type: notification.BalanceThresholdTypeUsagePercentage, Value: 50},
					},
				},
			},
		},
		{
			eventType: notification.EventTypeEntitlementReset,
			payload:   `{"features":["feature-a"]}`,
			config: notification.RuleConfig{
				EntitlementReset: &notification.EntitlementResetRuleConfig{Features: []string{"feature-a"}},
			},
		},
		{
			eventType: notification.EventTypeInvoiceCreated,
			payload:   `{}`,
			config:    notification.RuleConfig{Invoice: &notification.InvoiceRuleConfig{}},
		},
		{
			eventType: notification.EventTypeInvoiceUpdated,
			payload:   `{}`,
			config:    notification.RuleConfig{Invoice: &notification.InvoiceRuleConfig{}},
		},
	}

	for _, tt := range tests {
		t.Run(string(tt.eventType), func(t *testing.T) {
			legacy := fmt.Sprintf(`{"type":%q,"data":%s}`, tt.eventType, tt.payload)
			formats := []struct {
				name string
				data string
			}{
				{name: "legacy envelope", data: legacy},
				{name: "flat config", data: tt.payload},
			}

			for _, format := range formats {
				t.Run(format.name, func(t *testing.T) {
					// given: a rule created with the legacy contract
					channel := createChannel(t, env, ulid.Make().String(), clock.Now(), nil)
					created := createRule(t, env, channel.Namespace, clock.Now(), func(input *notification.CreateRuleInput) {
						input.Type = tt.eventType
						input.Config = tt.config
						input.Config.Type = tt.eventType
						input.Channels = []string{channel.ID}
					})

					entity, err := env.db.NotificationRule.Get(t.Context(), created.ID)
					require.NoError(t, err)

					assert.JSONEq(t, legacy, string(entity.Config))

					_, err = env.sqlDB.ExecContext(t.Context(),
						"UPDATE notification_rules SET config = $1 WHERE id = $2", format.data, created.ID)
					require.NoError(t, err)

					// when: getting and listing a stored config in either format
					fetched, err := env.adapter.GetRule(t.Context(), notification.GetRuleInput(created.NamespacedID))
					require.NoError(t, err)

					listed, err := env.adapter.ListRules(t.Context(), notification.ListRulesInput{
						Namespace: created.Namespace,
						Page:      pagination.NewPage(1, 20),
					})
					require.NoError(t, err)
					require.Len(t, listed.Items, 1)

					// then: decoded values and identity are preserved without rewriting the row
					assert.Equal(t, created.Config, fetched.Config)
					assert.Equal(t, created.Config, listed.Items[0].Config)
					assert.Equal(t, created.ID, fetched.ID)
					assert.Equal(t, created.CreatedAt, fetched.CreatedAt)
					assert.Equal(t, created.UpdatedAt, fetched.UpdatedAt)

					entity, err = env.db.NotificationRule.Get(t.Context(), created.ID)
					require.NoError(t, err)

					assert.JSONEq(t, format.data, string(entity.Config))

					// when: updating a rule read from either format
					updated, err := env.adapter.UpdateRule(t.Context(), notification.UpdateRuleInput{
						NamespacedID: fetched.NamespacedID,
						Type:         fetched.Type,
						Name:         "updated",
						Config:       fetched.Config,
						Channels:     []string{channel.ID},
					})
					require.NoError(t, err)

					// then: writes remain readable by old backends
					assert.Equal(t, created.Config, updated.Config)

					entity, err = env.db.NotificationRule.Get(t.Context(), created.ID)
					require.NoError(t, err)

					assert.JSONEq(t, legacy, string(entity.Config))
				})
			}
		})
	}
}

func TestRuleConfigMappingInvalid(t *testing.T) {
	tests := []struct {
		name string
		data string
	}{
		{name: "invalid JSON", data: `{`},
		{name: "null", data: `null`},
		{name: "array", data: `[]`},
		{name: "scalar", data: `"config"`},
		{name: "trailing JSON", data: `{} {}`},
		{name: "missing discriminator", data: `{"data":{}}`},
		{name: "wrong discriminator", data: `{"type":"unknown.event","data":{}}`},
		{name: "missing data", data: `{"type":"invoice.created"}`},
		{name: "null data", data: `{"type":"invoice.created","data":null}`},
		{name: "array data", data: `{"type":"invoice.created","data":[]}`},
		{name: "invalid features", data: `{"features":"invalid"}`},
		{name: "unknown flat fields", data: `{"unexpected":true}`},
	}

	for _, eventType := range notification.EventTypes() {
		t.Run(string(eventType), func(t *testing.T) {
			for _, tt := range tests {
				t.Run(tt.name, func(t *testing.T) {
					rule, err := RuleFromDBEntity(entdb.NotificationRule{
						ID:        "test-rule",
						Namespace: "test-namespace",
						Type:      eventType,
						Config:    json.RawMessage(tt.data),
					})
					require.ErrorContains(t, err, "invalid notification rule config")
					require.ErrorContains(t, err, "rule.id=test-rule")

					assert.Nil(t, rule)
				})
			}
		})
	}
}

func TestRuleConfigMappingPrecedence(t *testing.T) {
	tests := []struct {
		name string
		data string
	}{
		{
			name: "envelope takes precedence over flat fields",
			data: `{"type":"entitlements.reset","data":{"features":["legacy"]},"features":["flat"]}`,
		},
		{
			name: "legacy unknown fields remain readable",
			data: `{"type":"entitlements.reset","data":{"features":["legacy"],"historical_field":true}}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config, err := fromDBRuleConfig(notification.EventTypeEntitlementReset, json.RawMessage(tt.data))
			require.NoError(t, err)

			assert.Equal(t, []string{"legacy"}, config.EntitlementReset.Features)
		})
	}
}

func TestRuleConfigWriteValidation(t *testing.T) {
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
			encoded, err := toDBRuleConfig(tt.config)
			require.ErrorContains(t, err, "invalid rule config")

			assert.Nil(t, encoded)
		})
	}
}
