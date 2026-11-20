package adapter

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"path"
	"slices"
	"testing"
	"time"

	"github.com/oklog/ulid/v2"
	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	entdb "github.com/openmeterio/openmeter/openmeter/ent/db"
	"github.com/openmeterio/openmeter/openmeter/notification"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/filter"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/pagination"
	"github.com/openmeterio/openmeter/pkg/sortx"
	"github.com/openmeterio/openmeter/tools/migrate"
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
		versioned string
		eventType notification.EventType
		config    notification.RuleConfig
	}{
		{
			name:      "entitlements.balance.threshold",
			versioned: "v1.entitlement.balance_threshold.triggered",
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
			versioned: "v1.entitlement.usage_period.reset",
			eventType: notification.EventTypeEntitlementReset,
			config: notification.RuleConfig{
				EntitlementReset: &notification.EntitlementResetRuleConfig{Features: []string{"feature-a"}},
			},
		},
		{
			name:      "invoice.created",
			versioned: "v1.billing.invoice.created",
			eventType: notification.EventTypeInvoiceCreated,
			config:    notification.RuleConfig{Invoice: &notification.InvoiceRuleConfig{}},
		},
		{
			name:      "invoice.updated",
			versioned: "v1.billing.invoice.updated",
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

			assert.Equal(t, tt.versioned, storedType)

			for _, storedName := range []string{tt.name, tt.versioned} {
				t.Run(storedName, func(t *testing.T) {
					// when: a supported stored name is paired with the original config
					_, err = env.sqlDB.ExecContext(t.Context(), "UPDATE notification_rules SET type = $1 WHERE id = $2", storedName, created.ID)
					require.NoError(t, err)

					var storedFields map[string]json.RawMessage
					require.NoError(t, json.Unmarshal([]byte(storedConfig), &storedFields))

					assert.NotContains(t, storedFields, "type")
					assert.NotContains(t, storedFields, "data")

					fetched, err := env.adapter.GetRule(t.Context(), notification.GetRuleInput(created.NamespacedID))
					require.NoError(t, err)

					listed, err := env.adapter.ListRules(t.Context(), notification.ListRulesInput{
						Namespace:       created.Namespace,
						IncludeDisabled: true,
						Page:            pagination.NewPage(1, 20),
						Type:            &filter.FilterString{Eq: &tt.name},
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

					// then: the update preserves the stored spelling and the flat config
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

					assert.Equal(t, storedName, updatedType)
					assert.JSONEq(t, storedConfig, updatedConfig)
				})
			}
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

					assert.JSONEq(t, tt.payload, string(entity.Config))

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

					// then: updating either representation writes flat config
					assert.Equal(t, created.Config, updated.Config)

					entity, err = env.db.NotificationRule.Get(t.Context(), created.ID)
					require.NoError(t, err)

					assert.JSONEq(t, tt.payload, string(entity.Config))
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
						Type:      string(eventType),
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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config, err := fromDBRuleConfig(notification.EventTypeEntitlementReset, json.RawMessage(tt.data))
			require.NoError(t, err)

			assert.Equal(t, []string{"legacy"}, config.EntitlementReset.Features)
		})
	}
}

func TestListRules_MixedTypeSpellingsAcrossPages(t *testing.T) {
	env := newChannelTestEnv(t)
	namespace := ulid.Make().String()

	// given: all four public types with both legacy and versioned rows
	idsByType := make(map[string][]string)
	var allIDs []string
	for _, eventType := range []notification.EventType{notification.EventTypeBalanceThreshold, notification.EventTypeEntitlementReset, notification.EventTypeInvoiceCreated, notification.EventTypeInvoiceUpdated} {
		var ids []string
		for _, legacy := range []bool{true, false} {
			created := createRule(t, env, namespace, clock.Now(), func(input *notification.CreateRuleInput) {
				input.Type = eventType
				input.Config = notification.RuleConfig{RuleConfigMeta: notification.RuleConfigMeta{Type: eventType}}
				switch eventType {
				case notification.EventTypeBalanceThreshold:
					input.Config.BalanceThreshold = &notification.BalanceThresholdRuleConfig{Thresholds: []notification.BalanceThreshold{{Type: notification.BalanceThresholdTypeUsagePercentage, Value: 50}}}
				case notification.EventTypeEntitlementReset:
					input.Config.EntitlementReset = &notification.EntitlementResetRuleConfig{}
				case notification.EventTypeInvoiceCreated, notification.EventTypeInvoiceUpdated:
					input.Config.Invoice = &notification.InvoiceRuleConfig{}
				}
			})

			if legacy {
				_, err := env.sqlDB.ExecContext(t.Context(),
					"UPDATE notification_rules SET type = $1 WHERE id = $2", string(eventType), created.ID)
				require.NoError(t, err)
			}

			ids = append(ids, created.ID)
		}

		slices.Sort(ids)
		idsByType[string(eventType)] = ids
		allIDs = append(allIDs, ids...)
	}

	excludingInvoiceCreated := lo.Filter(allIDs, func(id string, _ int) bool {
		return !slices.Contains(idsByType["invoice.created"], id)
	})

	tests := []struct {
		name   string
		filter *filter.FilterString
		want   []string
	}{
		{name: "all", want: allIDs},
		{name: "equal", filter: &filter.FilterString{Eq: new("invoice.created")}, want: idsByType["invoice.created"]},
		{name: "not equal", filter: &filter.FilterString{Ne: new("invoice.created")}, want: excludingInvoiceCreated},
		{
			name:   "in",
			filter: &filter.FilterString{In: &[]string{"entitlements.reset", "invoice.created"}},
			want:   slices.Concat(idsByType["entitlements.reset"], idsByType["invoice.created"]),
		},
		{name: "not in", filter: &filter.FilterString{Nin: &[]string{"invoice.created"}}, want: excludingInvoiceCreated},
	}
	for _, stage := range []struct {
		name, direction string
	}{
		{name: "mixed spellings"},
		{name: "after upgrade", direction: "up"},
		{name: "after rollback", direction: "down"},
		{name: "after re-upgrade", direction: "up"},
	} {
		t.Run(stage.name, func(t *testing.T) {
			if stage.direction != "" {
				migration, err := fs.ReadFile(migrate.OMMigrationsConfig.FS, path.Join(
					migrate.OMMigrationsConfig.FSPath,
					"20261119151900_notification_event_type_normalization."+stage.direction+".sql",
				))
				require.NoError(t, err)

				_, err = env.sqlDB.ExecContext(t.Context(), string(migration))
				require.NoError(t, err)
			}

			for _, tt := range tests {
				t.Run(tt.name, func(t *testing.T) {
					for _, order := range []sortx.Order{sortx.OrderAsc, sortx.OrderDesc} {
						t.Run(string(order), func(t *testing.T) {
							want := slices.Clone(tt.want)
							if order == sortx.OrderDesc {
								slices.Reverse(want)
							}

							// when: each page contains one row ordered by public type and ID
							var got []string
							for page := 1; page <= len(want)+1; page++ {
								listed, err := env.adapter.ListRules(t.Context(), notification.ListRulesInput{
									Namespace: namespace,
									Type:      tt.filter, OrderBy: notification.OrderByType, Order: order,
									Page: pagination.NewPage(page, 1),
								})
								require.NoError(t, err)

								for _, item := range listed.Items {
									got = append(got, item.ID)
								}
							}

							// then: pagination includes each matching row once, regardless of storage spelling
							assert.Equal(t, want, got)
						})
					}
				})
			}
		})
	}
}

func TestRuleConfigNormalizationMigration(t *testing.T) {
	env := newChannelTestEnv(t)

	migrations := make(map[string]string)
	for _, direction := range []string{"up", "down"} {
		name := "20261119152230_notification_rule_config_normalization." + direction + ".sql"
		migration, err := fs.ReadFile(migrate.OMMigrationsConfig.FS, path.Join(migrate.OMMigrationsConfig.FSPath, name))
		require.NoError(t, err)

		migrations[direction] = string(migration)
	}

	t.Run("preserves Rule data across upgrade and rollback", func(t *testing.T) {
		type fixture struct {
			storedType  string
			legacyType  string
			data        string
			flat        bool
			annotations string
			disabled    bool
			deleted     bool
		}

		fixtures := []fixture{
			{
				storedType: "v1.entitlement.balance_threshold.triggered", legacyType: "entitlements.balance.threshold",
				data: `{"thresholds":[{"type":"usage_percentage","value":50}],"features":[]}`, disabled: true,
			},
			{
				storedType: "entitlements.reset", legacyType: "entitlements.reset", data: `{"features":null}`, deleted: true,
				annotations: `{"keep":9007199254740993}`,
			},
			{
				storedType: "invoice.created", legacyType: "invoice.created", data: `{}`,
			},
			{storedType: "entitlements.balance.threshold", legacyType: "entitlements.balance.threshold", data: `{"thresholds":[]}`},
			{storedType: "v1.entitlement.usage_period.reset", legacyType: "entitlements.reset", data: `{"features":[]}`},
			{storedType: "invoice.updated", legacyType: "invoice.updated", data: `{}`},
			{storedType: "v1.billing.invoice.updated", legacyType: "invoice.updated", data: `{}`, annotations: `null`},
			{storedType: "v1.billing.invoice.created", legacyType: "invoice.created", data: `{}`, flat: true},
			{storedType: "future.event", data: `{"type":"future.event","data":{"mode":"future"}}`},
		}

		for _, f := range fixtures {
			t.Run(f.storedType, func(t *testing.T) {
				// given: a persisted Rule and the fields that must survive config conversion
				tx, err := env.sqlDB.BeginTx(t.Context(), nil)
				require.NoError(t, err)

				defer func() { require.NoError(t, tx.Rollback()) }()

				id := ulid.Make().String()
				original := f.data
				envelope := fmt.Sprintf(`{"type":%q,"data":%s}`, f.legacyType, f.data)
				if f.legacyType != "" && !f.flat {
					original = envelope
				}

				var otherFields string
				err = tx.QueryRowContext(t.Context(), `
					INSERT INTO notification_rules AS r
						(id, namespace, type, name, config, annotations, disabled, deleted_at, created_at, updated_at)
					VALUES ($1, $2, $3, 'rule', $4, NULLIF($5, '')::jsonb, $6,
						CASE WHEN $7 THEN now() END, now(), now())
					RETURNING (to_jsonb(r) - 'config' - 'annotations')::text`,
					id, ulid.Make().String(), f.storedType, original, f.annotations, f.disabled, f.deleted).Scan(&otherFields)
				require.NoError(t, err)

				for _, direction := range []string{"up", "up", "down", "down", "up"} {
					// when: applying the same migration twice, rolling back, then upgrading again
					_, err := tx.ExecContext(t.Context(), migrations[direction])
					require.NoError(t, err)

					// then: only the config representation changes
					wantConfig := f.data
					if f.legacyType != "" && direction == "down" {
						wantConfig = envelope
					}

					var configMatches, annotationsMatch bool
					var currentFields string
					err = tx.QueryRowContext(t.Context(), `
						SELECT config = $2::jsonb, annotations IS NOT DISTINCT FROM NULLIF($3, '')::jsonb,
							(to_jsonb(r) - 'config' - 'annotations')::text
						FROM notification_rules AS r WHERE id = $1`, id, wantConfig, f.annotations).
						Scan(&configMatches, &annotationsMatch, &currentFields)
					require.NoError(t, err)

					assert.True(t, configMatches, "%s config", direction)
					assert.True(t, annotationsMatch, "%s annotations", direction)
					assert.Equal(t, otherFields, currentFields)
				}
			})
		}
	})

	t.Run("rollback wraps the latest config and newly created Rules", func(t *testing.T) {
		// given: a threshold Rule with a legacy config envelope
		namespace := ulid.Make().String()
		channel := createChannel(t, env, namespace, clock.Now(), nil)
		rule := createRule(t, env, namespace, clock.Now(), func(input *notification.CreateRuleInput) {
			input.Type = notification.EventTypeBalanceThreshold
			input.Config = notification.RuleConfig{
				RuleConfigMeta: notification.RuleConfigMeta{Type: input.Type},
				BalanceThreshold: &notification.BalanceThresholdRuleConfig{
					Thresholds: []notification.BalanceThreshold{{Type: notification.BalanceThresholdTypeUsagePercentage, Value: 50}},
				},
			}
			input.Channels = []string{channel.ID}
		})

		_, err := env.sqlDB.ExecContext(t.Context(), `
			UPDATE notification_rules SET config = jsonb_build_object('type', 'entitlements.balance.threshold', 'data', config)
			WHERE id = $1`, rule.ID)
		require.NoError(t, err)

		_, err = env.sqlDB.ExecContext(t.Context(), migrations["up"])
		require.NoError(t, err)

		// when: changing the threshold after upgrade and creating another Rule
		current, err := env.adapter.GetRule(t.Context(), notification.GetRuleInput(rule.NamespacedID))
		require.NoError(t, err)

		current.Config.BalanceThreshold.Thresholds[0].Value = 80
		_, err = env.adapter.UpdateRule(t.Context(), notification.UpdateRuleInput{
			NamespacedID: current.NamespacedID,
			Type:         current.Type,
			Name:         current.Name,
			Config:       current.Config,
			Channels:     []string{channel.ID},
		})
		require.NoError(t, err)

		created := createRule(t, env, namespace, clock.Now(), nil)

		_, err = env.sqlDB.ExecContext(t.Context(), migrations["down"])
		require.NoError(t, err)

		// then: both Rules have legacy envelopes and the edited threshold is preserved
		var discriminator, threshold string
		err = env.sqlDB.QueryRowContext(t.Context(), `
			SELECT config->>'type', config #>> '{data,thresholds,0,value}'
			FROM notification_rules WHERE id = $1`, rule.ID).Scan(&discriminator, &threshold)
		require.NoError(t, err)

		assert.Equal(t, "entitlements.balance.threshold", discriminator)
		assert.Equal(t, "80", threshold)

		var config string
		err = env.sqlDB.QueryRowContext(t.Context(), "SELECT config::text FROM notification_rules WHERE id = $1", created.ID).Scan(&config)
		require.NoError(t, err)

		assert.JSONEq(t, `{"type":"invoice.created","data":{}}`, config)

		_, err = env.sqlDB.ExecContext(t.Context(), migrations["up"])
		require.NoError(t, err)

		err = env.sqlDB.QueryRowContext(t.Context(), `
			SELECT config #>> '{thresholds,0,value}'
			FROM notification_rules WHERE id = $1`, rule.ID).Scan(&threshold)
		require.NoError(t, err)

		assert.Equal(t, "80", threshold)
	})

	t.Run("invalid envelopes abort all conversions", func(t *testing.T) {
		for _, tc := range []struct{ name, config string }{
			{name: "wrong discriminator", config: `{"type":"invoice.updated","data":{}}`},
			{name: "missing discriminator", config: `{"data":{}}`},
			{name: "missing data", config: `{"type":"invoice.created"}`},
			{name: "array data", config: `{"type":"invoice.created","data":[]}`},
		} {
			t.Run(tc.name, func(t *testing.T) {
				// given: a valid Rule ordered before an invalid Rule
				namespace := ulid.Make().String()
				firstID, invalidID := ulid.Make().String(), ulid.Make().String()
				_, err := env.sqlDB.ExecContext(t.Context(), `
					INSERT INTO notification_rules (id, namespace, name, type, config, created_at, updated_at)
					VALUES ($1, $2, 'valid', 'invoice.created', '{"type":"invoice.created","data":{}}', now(), now()),
						($3, $2, 'invalid', 'v1.billing.invoice.created', $4, now(), now())`, firstID, namespace, invalidID, tc.config)
				require.NoError(t, err)

				defer func() {
					_, err := env.sqlDB.ExecContext(t.Context(), "DELETE FROM notification_rules WHERE namespace = $1", namespace)
					require.NoError(t, err)
				}()

				// when: the migration reaches the invalid Rule
				_, err = env.sqlDB.ExecContext(t.Context(), migrations["up"])
				require.ErrorContains(t, err, "invalid rule config envelope")

				assert.Contains(t, err.Error(), namespace)
				assert.Contains(t, err.Error(), invalidID)
				assert.Contains(t, err.Error(), `expected type "invoice.created"`)
				assert.Contains(t, err.Error(), "If schema_om shows version 20261119152230 as dirty")
				assert.Contains(t, err.Error(), "force 20261119152000 (the previous successful version), then rerun the migration")

				// then: the failed migration preserves both rows
				var preserved int
				err = env.sqlDB.QueryRowContext(t.Context(), `
					SELECT count(*) FROM notification_rules WHERE namespace = $1 AND annotations IS NULL
					AND ((id = $2 AND config = '{"type":"invoice.created","data":{}}') OR (id = $3 AND config = $4::jsonb))`,
					namespace, firstID, invalidID, tc.config).Scan(&preserved)
				require.NoError(t, err)

				assert.Equal(t, 2, preserved)
			})
		}
	})
}
