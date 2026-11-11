package adapter

import (
	"database/sql"
	"encoding/json"
	"io/fs"
	"math"
	"path"
	"testing"
	"time"

	"github.com/oklog/ulid/v2"
	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	entdb "github.com/openmeterio/openmeter/openmeter/ent/db"
	channeldb "github.com/openmeterio/openmeter/openmeter/ent/db/notificationchannel"
	"github.com/openmeterio/openmeter/openmeter/notification"
	"github.com/openmeterio/openmeter/openmeter/testutils"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/filter"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/pagination"
	"github.com/openmeterio/openmeter/pkg/sortx"
	"github.com/openmeterio/openmeter/tools/migrate"
)

// These tests exercise the v3 list-filter wiring added to ListChannels
// (openmeter/notification/adapter/channel.go) directly against the adapter, not
// through notification/service.Service.CreateChannel. Service-level CreateChannel
// also creates a webhook via a webhook.Handler (openmeter/notification/service/channel.go),
// and the only Handler available to tests outside svix is the noop implementation,
// which returns webhook.ErrNotImplemented for CreateWebhook — so a full TestEnv built
// on the real Service cannot create channels at all. adapter.CreateChannel itself does
// not touch the webhook handler, so calling it directly is the practical way to seed
// channels for these list-filter tests.
type channelTestEnv struct {
	db      *entdb.Client
	sqlDB   *sql.DB
	adapter notification.Repository
}

func newChannelTestEnv(t *testing.T) *channelTestEnv {
	t.Helper()

	testdb := testutils.InitPostgresDB(t, testutils.PostgresDBStateEntMigrated)
	dbClient := testdb.EntDriver.Client()

	t.Cleanup(func() {
		_ = dbClient.Close()
		testdb.Close(t)
	})

	adapter, err := New(Config{
		Client: dbClient,
		Logger: testutils.NewDiscardLogger(t),
	})
	require.NoError(t, err, "constructing notification adapter must not fail")

	return &channelTestEnv{db: dbClient, sqlDB: testdb.PGDriver.DB(), adapter: adapter}
}

func TestChannelTypeStorageCompatibility(t *testing.T) {
	env := newChannelTestEnv(t)
	tests := []struct {
		name       string
		columnType string
		configType string
	}{
		{
			name:       "legacy row",
			columnType: "WEBHOOK",
			configType: "WEBHOOK",
		},
		{
			name:       "canonical row",
			columnType: "webhook",
			configType: "webhook",
		},
		{
			name:       "legacy column",
			columnType: "WEBHOOK",
			configType: "webhook",
		},
		{
			name:       "legacy config",
			columnType: "webhook",
			configType: "WEBHOOK",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// given: a canonical service input written for legacy readers
			channel := createChannel(t, env, ulid.Make().String(), clock.Now(), func(input *notification.CreateChannelInput) {
				input.Config.WebHook.CustomHeaders = map[string]string{"X-Test": "preserve"}
				input.Config.WebHook.SigningSecret = "whsec_MfKQ9r8GKYqrTwjUPD8ILPZIo2LaLaSw"
			})

			assert.Equal(t, notification.ChannelTypeWebhook, channel.Type)
			assert.Equal(t, notification.ChannelTypeWebhook, channel.Config.Type)

			var columnType, rawConfig string
			err := env.sqlDB.QueryRowContext(t.Context(),
				"SELECT type, config FROM notification_channels WHERE id = $1", channel.ID,
			).Scan(&columnType, &rawConfig)
			require.NoError(t, err)

			var legacyConfig struct {
				Type string                            `json:"type"`
				Data notification.WebHookChannelConfig `json:"data"`
			}

			require.NoError(t, json.Unmarshal([]byte(rawConfig), &legacyConfig))

			assert.Equal(t, "WEBHOOK", columnType)
			assert.Equal(t, "WEBHOOK", legacyConfig.Type)
			assert.Equal(t, channel.Config.WebHook, legacyConfig.Data)

			// Generated predicates encode canonical operands using the scanner.
			entity, err := env.db.NotificationChannel.Query().Where(
				channeldb.ID(channel.ID),
				channeldb.TypeEQ(notification.ChannelTypeWebhook),
			).Only(t.Context())
			require.NoError(t, err)

			assert.Equal(t, notification.ChannelTypeWebhook, entity.Type)

			// when: either stored discriminator uses its legacy or canonical spelling
			_, err = env.sqlDB.ExecContext(t.Context(), `
				UPDATE notification_channels
				SET type = $1, config = jsonb_set(config, '{type}', to_jsonb($2::text))
				WHERE id = $3`, tt.columnType, tt.configType, channel.ID)
			require.NoError(t, err)

			entity, err = env.db.NotificationChannel.Get(t.Context(), channel.ID)
			require.NoError(t, err)

			assert.Equal(t, notification.ChannelTypeWebhook, entity.Type)

			stored, err := env.adapter.GetChannel(t.Context(), notification.GetChannelInput{
				Namespace: channel.Namespace,
				ID:        channel.ID,
			})
			require.NoError(t, err)

			// then: reads normalize the type without changing webhook settings
			assert.Equal(t, notification.ChannelTypeWebhook, stored.Type)
			assert.Equal(t, channel.Config, stored.Config)

			updated, err := env.adapter.UpdateChannel(t.Context(), notification.UpdateChannelInput{
				NamespacedID: channel.NamespacedID,
				Type:         stored.Type,
				Name:         "updated",
				Config:       stored.Config,
			})
			require.NoError(t, err)

			assert.Equal(t, notification.ChannelTypeWebhook, updated.Type)
			assert.Equal(t, channel.Config, updated.Config)

			err = env.sqlDB.QueryRowContext(t.Context(),
				"SELECT type, config FROM notification_channels WHERE id = $1", channel.ID,
			).Scan(&columnType, &rawConfig)
			require.NoError(t, err)
			require.NoError(t, json.Unmarshal([]byte(rawConfig), &legacyConfig))

			// Updates preserve the immutable column and write a legacy-readable config.
			assert.Equal(t, tt.columnType, columnType)
			assert.Equal(t, "WEBHOOK", legacyConfig.Type)
			assert.Equal(t, channel.Config.WebHook, legacyConfig.Data)
		})
	}
}

func TestListChannels_MixedTypeFormats(t *testing.T) {
	env := newChannelTestEnv(t)
	namespace := ulid.Make().String()

	// given: legacy and canonical rows representing the same domain type
	legacy := createChannel(t, env, namespace, clock.Now(), nil)
	canonical := createChannel(t, env, namespace, clock.Now(), nil)
	_, err := env.sqlDB.ExecContext(t.Context(),
		"UPDATE notification_channels SET type = 'webhook' WHERE id = $1", canonical.ID)
	require.NoError(t, err)

	tests := []struct {
		name    string
		filter  *filter.FilterString
		match   bool
		wantErr bool
	}{
		{
			name:  "no filter",
			match: true,
		},
		{
			name:   "empty filter",
			filter: &filter.FilterString{},
			match:  true,
		},
		{
			name:   "equal",
			filter: &filter.FilterString{Eq: new("webhook")},
			match:  true,
		},
		{
			name:   "not equal",
			filter: &filter.FilterString{Ne: new("webhook")},
		},
		{
			name:   "in",
			filter: &filter.FilterString{In: &[]string{"unknown", "webhook"}},
			match:  true,
		},
		{
			name:   "not in",
			filter: &filter.FilterString{Nin: &[]string{"unknown", "webhook"}},
		},
		{
			name:   "unknown",
			filter: &filter.FilterString{Eq: new("unknown")},
		},
		{
			name:    "uppercase is not canonical",
			filter:  &filter.FilterString{Eq: new("WEBHOOK")},
			wantErr: true,
		},
		{
			name:    "like",
			filter:  &filter.FilterString{Like: new("web%")},
			wantErr: true,
		},
		{
			name:    "not like",
			filter:  &filter.FilterString{Nlike: new("web%")},
			wantErr: true,
		},
		{
			name:    "case sensitive like",
			filter:  &filter.FilterString{Like: new("WEB%")},
			wantErr: true,
		},
		{
			name:    "case insensitive like",
			filter:  &filter.FilterString{Ilike: new("WEB%")},
			wantErr: true,
		},
		{
			name:    "contains",
			filter:  &filter.FilterString{Contains: new("HOOK")},
			wantErr: true,
		},
		{
			name:    "not contains",
			filter:  &filter.FilterString{Ncontains: new("HOOK")},
			wantErr: true,
		},
		{
			name:    "range",
			filter:  &filter.FilterString{Gt: new("aaa")},
			wantErr: true,
		},
		{
			name:   "exists",
			filter: &filter.FilterString{Exists: new(true)},
			match:  true,
		},
		{
			name:   "does not exist",
			filter: &filter.FilterString{Exists: new(false)},
		},
		{
			name: "nested expressions",
			filter: &filter.FilterString{And: &[]filter.FilterString{
				{Ne: new("unknown")},
				{Or: &[]filter.FilterString{{Eq: new("unknown")}, {Eq: new("webhook")}}},
			}},
			match: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			before, err := json.Marshal(tt.filter)
			require.NoError(t, err)

			// when: filtering by the canonical domain type
			result, err := env.adapter.ListChannels(t.Context(), notification.ListChannelsInput{
				Namespaces: []string{namespace},
				Type:       tt.filter,
				Page:       pagination.NewPage(1, 20),
			})
			after, marshalErr := json.Marshal(tt.filter)
			require.NoError(t, marshalErr)

			assert.Equal(t, string(before), string(after))

			if tt.wantErr {
				require.ErrorAs(t, err, new(*models.GenericValidationError))

				return
			}

			require.NoError(t, err)

			// then: both representations have the same matching behavior
			var ids []string
			for _, channel := range result.Items {
				ids = append(ids, channel.ID)

				assert.Equal(t, notification.ChannelTypeWebhook, channel.Type)
				assert.Equal(t, notification.ChannelTypeWebhook, channel.Config.Type)
			}

			if tt.match {
				assert.ElementsMatch(t, []string{legacy.ID, canonical.ID}, ids)
				assert.Equal(t, 2, result.TotalCount)
			} else {
				assert.Empty(t, ids)
				assert.Zero(t, result.TotalCount)
			}
		})
	}
}

// createChannel seeds a webhook channel via the adapter directly (see channelTestEnv
// doc comment for why), created at the given frozen time so tests can assert on
// created_at/updated_at deterministically. mutate lets each test case override
// name/disabled/etc. on top of an otherwise-valid channel.
func createChannel(t *testing.T, env *channelTestEnv, ns string, at time.Time, mutate func(*notification.CreateChannelInput)) notification.Channel {
	t.Helper()

	clock.FreezeTime(at)
	defer clock.UnFreeze()

	in := notification.CreateChannelInput{
		NamespacedModel: models.NamespacedModel{Namespace: ns},
		Type:            notification.ChannelTypeWebhook,
		Name:            "channel-" + ulid.Make().String(),
		Config: notification.ChannelConfig{
			ChannelConfigMeta: notification.ChannelConfigMeta{Type: notification.ChannelTypeWebhook},
			WebHook:           notification.WebHookChannelConfig{URL: "https://example.com/hook"},
		},
	}
	if mutate != nil {
		mutate(&in)
	}

	channel, err := env.adapter.CreateChannel(t.Context(), in)
	require.NoError(t, err, "seeding channel must not fail")
	require.NotNil(t, channel)

	return *channel
}

// TestListChannels_Filters table-tests each of the filter fields wired up in
// ListChannels (openmeter/notification/adapter/channel.go).
func TestListChannels_Filters(t *testing.T) {
	env := newChannelTestEnv(t)
	ns := ulid.Make().String()

	frozen := clock.Now()
	t0 := frozen.Add(-2 * time.Hour)
	t1 := frozen.Add(-1 * time.Hour)

	first := createChannel(t, env, ns, t0, func(in *notification.CreateChannelInput) {
		in.Name = "first-channel"
	})

	second := createChannel(t, env, ns, t1, func(in *notification.CreateChannelInput) {
		in.Name = "second-channel"
	})

	disabled := createChannel(t, env, ns, frozen, func(in *notification.CreateChannelInput) {
		in.Name = "disabled-channel"
		in.Disabled = true
	})

	testCases := []struct {
		name    string
		input   notification.ListChannelsInput
		wantIDs []string
	}{
		{
			name: "ID filter matches exactly one channel",
			input: notification.ListChannelsInput{
				ID: &filter.FilterULID{FilterString: filter.FilterString{Eq: lo.ToPtr(first.ID)}},
			},
			wantIDs: []string{first.ID},
		},
		{
			name: "ID in filter matches the selected channels",
			input: notification.ListChannelsInput{
				ID: &filter.FilterULID{FilterString: filter.FilterString{In: lo.ToPtr([]string{first.ID, second.ID})}},
			},
			wantIDs: []string{first.ID, second.ID},
		},
		{
			name: "Name eq matches exactly one channel",
			input: notification.ListChannelsInput{
				Name: &filter.FilterString{Eq: lo.ToPtr("second-channel")},
			},
			wantIDs: []string{second.ID},
		},
		{
			name: "Name contains matches by substring",
			input: notification.ListChannelsInput{
				Name: &filter.FilterString{Contains: lo.ToPtr("first")},
			},
			wantIDs: []string{first.ID},
		},
		{
			name: "Type filter matches every webhook channel",
			input: notification.ListChannelsInput{
				Type: &filter.FilterString{Eq: lo.ToPtr(string(notification.ChannelTypeWebhook))},
			},
			wantIDs: []string{first.ID, second.ID, disabled.ID},
		},
		{
			name: "Disabled eq true matches only the disabled channel",
			input: notification.ListChannelsInput{
				Disabled: &filter.FilterBoolean{Eq: lo.ToPtr(true)},
			},
			wantIDs: []string{disabled.ID},
		},
		{
			name: "Disabled eq false matches only the enabled channels",
			input: notification.ListChannelsInput{
				Disabled: &filter.FilterBoolean{Eq: lo.ToPtr(false)},
			},
			wantIDs: []string{first.ID, second.ID},
		},
		{
			name: "CreatedAt range matches only the channel created inside the window",
			input: notification.ListChannelsInput{
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
			input.Namespaces = []string{ns}
			input.Page = pagination.NewPage(1, 20)

			result, err := env.adapter.ListChannels(t.Context(), input)
			require.NoError(t, err)

			gotIDs := make([]string, 0, len(result.Items))
			for _, item := range result.Items {
				gotIDs = append(gotIDs, item.ID)
			}

			assert.ElementsMatch(t, tc.wantIDs, gotIDs)
		})
	}
}

// TestListChannels_Sort covers the adapter's OrderBy switch for each documented
// sort field.
func TestListChannels_Sort(t *testing.T) {
	env := newChannelTestEnv(t)
	ns := ulid.Make().String()

	frozen := clock.Now()

	older := createChannel(t, env, ns, frozen.Add(-2*time.Hour), func(in *notification.CreateChannelInput) { in.Name = "a-channel" })
	newer := createChannel(t, env, ns, frozen.Add(-1*time.Hour), func(in *notification.CreateChannelInput) { in.Name = "b-channel" })

	t.Run("created_at asc returns creation order", func(t *testing.T) {
		result, err := env.adapter.ListChannels(t.Context(), notification.ListChannelsInput{
			Namespaces: []string{ns},
			OrderBy:    notification.OrderByCreatedAt,
			Order:      sortx.OrderAsc,
			Page:       pagination.NewPage(1, 20),
		})
		require.NoError(t, err)
		require.Len(t, result.Items, 2)
		assert.Equal(t, older.ID, result.Items[0].ID)
		assert.Equal(t, newer.ID, result.Items[1].ID)
	})

	t.Run("created_at desc reverses creation order", func(t *testing.T) {
		result, err := env.adapter.ListChannels(t.Context(), notification.ListChannelsInput{
			Namespaces: []string{ns},
			OrderBy:    notification.OrderByCreatedAt,
			Order:      sortx.OrderDesc,
			Page:       pagination.NewPage(1, 20),
		})
		require.NoError(t, err)
		require.Len(t, result.Items, 2)
		assert.Equal(t, newer.ID, result.Items[0].ID)
		assert.Equal(t, older.ID, result.Items[1].ID)
	})

	t.Run("id asc (default) returns ULID creation order", func(t *testing.T) {
		result, err := env.adapter.ListChannels(t.Context(), notification.ListChannelsInput{
			Namespaces: []string{ns},
			Page:       pagination.NewPage(1, 20),
		})
		require.NoError(t, err)
		require.Len(t, result.Items, 2)
		assert.Equal(t, older.ID, result.Items[0].ID)
		assert.Equal(t, newer.ID, result.Items[1].ID)
	})

	t.Run("type asc returns all rows without erroring", func(t *testing.T) {
		// Only one channel type exists in the domain today, so this cannot assert a
		// meaningful order between rows — it only proves OrderByType is wired to a
		// valid Ent ordering clause instead of erroring or silently dropping rows.
		result, err := env.adapter.ListChannels(t.Context(), notification.ListChannelsInput{
			Namespaces: []string{ns},
			OrderBy:    notification.OrderByType,
			Order:      sortx.OrderAsc,
			Page:       pagination.NewPage(1, 20),
		})
		require.NoError(t, err)
		assert.Len(t, result.Items, 2)
	})

	t.Run("updated_at desc reflects the most recently modified channel first", func(t *testing.T) {
		// Update the older channel after both were created, so its updated_at is now
		// the most recent of the two — this differentiates OrderByUpdatedAt from
		// OrderByCreatedAt, which would still put the older channel first.
		clock.FreezeTime(frozen.Add(time.Hour))
		defer clock.UnFreeze()

		_, err := env.adapter.UpdateChannel(t.Context(), notification.UpdateChannelInput{
			NamespacedID: models.NamespacedID{Namespace: ns, ID: older.ID},
			Type:         older.Type,
			Name:         older.Name,
			Disabled:     older.Disabled,
			Config:       older.Config,
		})
		require.NoError(t, err)

		result, err := env.adapter.ListChannels(t.Context(), notification.ListChannelsInput{
			Namespaces: []string{ns},
			OrderBy:    notification.OrderByUpdatedAt,
			Order:      sortx.OrderDesc,
			Page:       pagination.NewPage(1, 20),
		})
		require.NoError(t, err)
		require.Len(t, result.Items, 2)
		assert.Equal(t, older.ID, result.Items[0].ID, "older channel was updated last, so it must sort first in updated_at desc")
		assert.Equal(t, newer.ID, result.Items[1].ID)
	})
}

func TestToDBChannelTypeFilter(t *testing.T) {
	tests := []struct {
		name  string
		input *filter.FilterString
		want  *filter.FilterString
	}{
		{
			name: "nil",
		},
		{
			name:  "empty",
			input: &filter.FilterString{},
			want:  &filter.FilterString{},
		},
		{
			name:  "equal",
			input: &filter.FilterString{Eq: new("webhook")},
			want:  &filter.FilterString{In: &[]string{"webhook", "WEBHOOK"}},
		},
		{
			name:  "not equal",
			input: &filter.FilterString{Ne: new("webhook")},
			want:  &filter.FilterString{Nin: &[]string{"webhook", "WEBHOOK"}},
		},
		{
			name:  "membership preserves unmapped values",
			input: &filter.FilterString{In: &[]string{"webhook", "other"}},
			want:  &filter.FilterString{In: &[]string{"webhook", "WEBHOOK", "other"}},
		},
		{
			name:  "exclusion preserves unmapped values",
			input: &filter.FilterString{Nin: &[]string{"webhook", "other"}},
			want:  &filter.FilterString{Nin: &[]string{"webhook", "WEBHOOK", "other"}},
		},
		{
			name:  "empty membership",
			input: &filter.FilterString{In: &[]string{}},
			want:  &filter.FilterString{In: &[]string{}},
		},
		{
			name:  "empty exclusion",
			input: &filter.FilterString{Nin: &[]string{}},
			want:  &filter.FilterString{Nin: &[]string{}},
		},
		{
			name:  "exists",
			input: &filter.FilterString{Exists: new(true)},
			want:  &filter.FilterString{Exists: new(true)},
		},
		{
			name: "nested expressions",
			input: &filter.FilterString{And: &[]filter.FilterString{
				{Ne: new("other")},
				{Or: &[]filter.FilterString{
					{Eq: new("webhook")},
					{Exists: new(false)},
				}},
			}},
			want: &filter.FilterString{And: &[]filter.FilterString{
				{Nin: &[]string{"other"}},
				{Or: &[]filter.FilterString{
					{In: &[]string{"webhook", "WEBHOOK"}},
					{Exists: new(false)},
				}},
			}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			before, err := json.Marshal(tt.input)
			require.NoError(t, err)

			mapped, err := toDBChannelTypeFilter(tt.input)
			require.NoError(t, err)

			assert.Equal(t, tt.want, mapped)

			// Mutating the returned filter must not change the caller's filter.
			if mapped != nil {
				mapped.Eq = new("changed")

				if mapped.In != nil && len(*mapped.In) > 0 {
					(*mapped.In)[0] = "changed"
				}

				if mapped.Nin != nil && len(*mapped.Nin) > 0 {
					(*mapped.Nin)[0] = "changed"
				}

				if mapped.Exists != nil {
					*mapped.Exists = !*mapped.Exists
				}

				if mapped.And != nil {
					(*mapped.And)[0].Ne = new("changed")
				}
			}

			after, err := json.Marshal(tt.input)
			require.NoError(t, err)

			assert.Equal(t, string(before), string(after))
		})
	}
}

func TestToDBChannelTypeFilter_Invalid(t *testing.T) {
	tests := []struct {
		name  string
		input filter.FilterString
	}{
		{
			name:  "like",
			input: filter.FilterString{Like: new("web%")},
		},
		{
			name:  "not like",
			input: filter.FilterString{Nlike: new("web%")},
		},
		{
			name:  "ilike",
			input: filter.FilterString{Ilike: new("WEB%")},
		},
		{
			name:  "not ilike",
			input: filter.FilterString{Nilike: new("WEB%")},
		},
		{
			name:  "contains",
			input: filter.FilterString{Contains: new("web")},
		},
		{
			name:  "not contains",
			input: filter.FilterString{Ncontains: new("web")},
		},
		{
			name:  "gt",
			input: filter.FilterString{Gt: new("webhook")},
		},
		{
			name:  "gte",
			input: filter.FilterString{Gte: new("webhook")},
		},
		{
			name:  "lt",
			input: filter.FilterString{Lt: new("webhook")},
		},
		{
			name:  "lte",
			input: filter.FilterString{Lte: new("webhook")},
		},
		{
			name:  "legacy operand",
			input: filter.FilterString{Eq: new("WEBHOOK")},
		},
		{
			name:  "empty operand",
			input: filter.FilterString{Eq: new("")},
		},
		{
			name:  "multiple operators",
			input: filter.FilterString{Eq: new("webhook"), Ne: new("other")},
		},
		{
			name: "nested unsupported operator",
			input: filter.FilterString{Or: &[]filter.FilterString{
				{Eq: new("webhook")},
				{Like: new("web%")},
			}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mapped, err := toDBChannelTypeFilter(&tt.input)
			require.ErrorAs(t, err, new(*models.GenericValidationError))

			assert.Nil(t, mapped)
		})
	}
}

func TestChannelConfigMapping(t *testing.T) {
	tests := []struct {
		name string
		data string
	}{
		{
			name: "legacy uppercase",
			data: `{"type":"WEBHOOK","data":{"url":"https://example.com/hook","customHeaders":{"X-Team":"billing"},"signingSecret":"original-secret"}}`,
		},
		{
			name: "legacy lowercase",
			data: `{"type":"webhook","data":{"url":"https://example.com/hook","customHeaders":{"X-Team":"billing"},"signingSecret":"original-secret"}}`,
		},
		{
			name: "flat",
			data: `{"url":"https://example.com/hook","custom_headers":{"X-Team":"billing"},"signing_secret":"original-secret"}`,
		},
		{
			name: "invalid envelope falls back to flat config",
			data: `{"type":"WEBHOOK","data":[],"url":"https://example.com/hook","custom_headers":{"X-Team":"billing"},"signing_secret":"original-secret"}`,
		},
		{
			name: "invalid discriminator falls back to flat config",
			data: `{"type":"email","data":{},"url":"https://example.com/hook","custom_headers":{"X-Team":"billing"},"signing_secret":"original-secret"}`,
		},
		{
			name: "valid envelope takes precedence",
			data: `{"type":"WEBHOOK","data":{"url":"https://example.com/hook","customHeaders":{"X-Team":"billing"},"signingSecret":"original-secret"},"url":"https://ignored.example.com","signing_secret":123}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// given: either supported storage format for a webhook channel
			row := entdb.NotificationChannel{
				ID:        ulid.Make().String(),
				Namespace: ulid.Make().String(),
				Type:      notification.ChannelTypeWebhook,
				Config:    json.RawMessage(tt.data),
			}

			// when: the adapter maps the stored configuration
			channel, err := ChannelFromDBEntity(row)
			require.NoError(t, err)

			// then: domain values are preserved and writes remain legacy-readable
			assert.Equal(t, notification.ChannelTypeWebhook, channel.Config.Type)
			assert.Equal(t, "https://example.com/hook", channel.Config.WebHook.URL)
			assert.Equal(t, map[string]string{"X-Team": "billing"}, channel.Config.WebHook.CustomHeaders)
			assert.Equal(t, "original-secret", channel.Config.WebHook.SigningSecret)

			encoded, err := toDBChannelConfig(channel.Config)
			require.NoError(t, err)

			assert.JSONEq(t, `{"type":"WEBHOOK","data":{"url":"https://example.com/hook","customHeaders":{"X-Team":"billing"},"signingSecret":"original-secret"}}`, string(encoded))
			assert.JSONEq(t, tt.data, string(row.Config))
		})
	}
}

func TestChannelConfigMapping_Invalid(t *testing.T) {
	tests := []struct {
		name        string
		channelType notification.ChannelType
		data        string
	}{
		{
			name:        "empty flat config",
			channelType: notification.ChannelTypeWebhook,
			data:        `{}`,
		},
		{
			name:        "invalid JSON",
			channelType: notification.ChannelTypeWebhook,
			data:        `{`,
		},
		{
			name:        "null",
			channelType: notification.ChannelTypeWebhook,
			data:        `null`,
		},
		{
			name:        "array",
			channelType: notification.ChannelTypeWebhook,
			data:        `[]`,
		},
		{
			name:        "missing discriminator",
			channelType: notification.ChannelTypeWebhook,
			data:        `{"data":{}}`,
		},
		{
			name:        "null discriminator",
			channelType: notification.ChannelTypeWebhook,
			data:        `{"type":null,"data":{}}`,
		},
		{
			name:        "missing data",
			channelType: notification.ChannelTypeWebhook,
			data:        `{"type":"WEBHOOK"}`,
		},
		{
			name:        "null data",
			channelType: notification.ChannelTypeWebhook,
			data:        `{"type":"WEBHOOK","data":null}`,
		},
		{
			name:        "array data",
			channelType: notification.ChannelTypeWebhook,
			data:        `{"type":"WEBHOOK","data":[]}`,
		},
		{
			name:        "mismatched discriminator",
			channelType: notification.ChannelTypeWebhook,
			data:        `{"type":"email","data":{}}`,
		},
		{
			name:        "unknown channel type",
			channelType: "email",
			data:        `{"url":"https://example.com/hook"}`,
		},
		{
			name:        "flat headers have wrong type",
			channelType: notification.ChannelTypeWebhook,
			data:        `{"url":"https://example.com/hook","custom_headers":[]}`,
		},
		{
			name:        "flat secret has wrong type",
			channelType: notification.ChannelTypeWebhook,
			data:        `{"url":"https://example.com/hook","signing_secret":123}`,
		},
		{
			name:        "legacy headers have wrong type",
			channelType: notification.ChannelTypeWebhook,
			data:        `{"type":"WEBHOOK","data":{"customHeaders":[]}}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			channel, err := ChannelFromDBEntity(entdb.NotificationChannel{
				ID: "test-channel", Namespace: "test-namespace",
				Type: tt.channelType, Config: json.RawMessage(tt.data),
			})
			require.ErrorContains(t, err, "invalid")
			require.ErrorContains(t, err, "channel.id=test-channel")

			assert.Nil(t, channel)
		})
	}
}

func TestChannelConfigStorageCompatibility(t *testing.T) {
	env := newChannelTestEnv(t)
	tests := []struct {
		name    string
		data    string
		wantErr bool
	}{
		{
			name: "legacy",
			data: `{"type":"WEBHOOK","data":{"url":"https://example.com/hook","customHeaders":{"X-Team":"billing"},"signingSecret":"original-secret"}}`,
		},
		{
			name: "flat",
			data: `{"url":"https://example.com/hook","custom_headers":{"X-Team":"billing"},"signing_secret":"original-secret"}`,
		},
		{
			name:    "malformed",
			data:    `{"type":"WEBHOOK","data":[]}`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// given: an existing channel with a stored configuration
			created := createChannel(t, env, ulid.Make().String(), clock.Now(), nil)
			_, err := env.sqlDB.ExecContext(t.Context(),
				"UPDATE notification_channels SET config = $1 WHERE id = $2", tt.data, created.ID)
			require.NoError(t, err)

			// when: fetching and listing the channel through the adapter
			channel, getErr := env.adapter.GetChannel(t.Context(), notification.GetChannelInput{
				Namespace: created.Namespace, ID: created.ID,
			})
			listed, listErr := env.adapter.ListChannels(t.Context(), notification.ListChannelsInput{
				Namespaces: []string{created.Namespace}, Page: pagination.NewPage(1, 20),
			})

			// then: malformed configurations propagate errors through both paths
			if tt.wantErr {
				require.Error(t, getErr)
				require.Error(t, listErr)

				assert.Nil(t, channel)

				return
			}

			require.NoError(t, getErr)
			require.NoError(t, listErr)
			require.Len(t, listed.Items, 1)

			assert.Equal(t, channel.Config, listed.Items[0].Config)
			assert.Equal(t, created.ID, channel.ID)
			assert.Equal(t, created.CreatedAt, channel.CreatedAt)
			assert.Equal(t, created.UpdatedAt, channel.UpdatedAt)
			assert.Equal(t, "original-secret", channel.Config.WebHook.SigningSecret)
			assert.Equal(t, map[string]string{"X-Team": "billing"}, channel.Config.WebHook.CustomHeaders)

			entity, err := env.db.NotificationChannel.Get(t.Context(), created.ID)
			require.NoError(t, err)

			assert.JSONEq(t, tt.data, string(entity.Config))

			t.Run("update preserves settings and writes a legacy envelope", func(t *testing.T) {
				// when: updating a channel read from either format
				updated, err := env.adapter.UpdateChannel(t.Context(), notification.UpdateChannelInput{
					NamespacedID: channel.NamespacedID, Type: channel.Type,
					Name: "updated", Config: channel.Config,
				})
				require.NoError(t, err)

				// then: settings survive without rotating the secret
				assert.Equal(t, channel.Config, updated.Config)

				entity, err := env.db.NotificationChannel.Get(t.Context(), created.ID)
				require.NoError(t, err)

				assert.JSONEq(t, `{"type":"WEBHOOK","data":{"url":"https://example.com/hook","customHeaders":{"X-Team":"billing"},"signingSecret":"original-secret"}}`, string(entity.Config))
			})
		})
	}
}

func TestChannelEventTypeFilterPreservation(t *testing.T) {
	env := newChannelTestEnv(t)
	tests := []struct {
		name   string
		filter []string
	}{
		{
			name: "unset filter",
		},
		{
			name:   "empty filter",
			filter: []string{},
		},
		{
			name:   "populated filter",
			filter: []string{"v1.billing.invoice.created", "v1.billing.invoice.updated"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// given: legacy creation leaves the future routing filter unset
			channel := createChannel(t, env, ulid.Make().String(), clock.Now(), nil)

			var stored sql.NullString
			err := env.sqlDB.QueryRowContext(t.Context(),
				"SELECT event_type_filter FROM notification_channels WHERE id = $1", channel.ID,
			).Scan(&stored)
			require.NoError(t, err)

			assert.False(t, stored.Valid)

			// Seed future data directly; current service inputs do not own this field.
			if tt.filter != nil {
				_, err := env.db.NotificationChannel.UpdateOneID(channel.ID).
					SetEventTypeFilter(tt.filter).
					Save(t.Context())
				require.NoError(t, err)
			}

			// when: existing adapter code updates the channel's desired state
			config := channel.Config
			config.WebHook.URL = "https://example.com/updated"

			updated, err := env.adapter.UpdateChannel(t.Context(), notification.UpdateChannelInput{
				NamespacedID: channel.NamespacedID,
				Type:         channel.Type,
				Name:         "updated",
				Disabled:     true,
				Config:       config,
				Annotations:  channel.Annotations,
				Metadata:     channel.Metadata,
			})
			require.NoError(t, err)

			// then: owned fields change while the stored filter is preserved
			assert.Equal(t, "updated", updated.Name)
			assert.True(t, updated.Disabled)
			assert.Equal(t, config, updated.Config)

			entity, err := env.db.NotificationChannel.Get(t.Context(), channel.ID)
			require.NoError(t, err)

			assert.Equal(t, tt.filter, entity.EventTypeFilter)

			err = env.sqlDB.QueryRowContext(t.Context(),
				"SELECT event_type_filter FROM notification_channels WHERE id = $1", channel.ID,
			).Scan(&stored)
			require.NoError(t, err)

			if tt.filter == nil {
				assert.False(t, stored.Valid)

				return
			}

			require.True(t, stored.Valid)

			expected, err := json.Marshal(tt.filter)
			require.NoError(t, err)

			assert.JSONEq(t, string(expected), stored.String)
		})
	}
}

func TestChannelProviderDataPreservation(t *testing.T) {
	env := newChannelTestEnv(t)
	tests := []struct {
		name         string
		providerData map[string]string
	}{
		{
			name: "unset provider data",
		},
		{
			name:         "empty provider data",
			providerData: map[string]string{},
		},
		{
			name: "populated provider data",
			providerData: map[string]string{
				"svix.endpoint_id": "ep_existing",
				"other.entity_id":  "other_existing",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// given: legacy creation leaves provider data unset
			channel := createChannel(t, env, ulid.Make().String(), clock.Now(), nil)

			var stored sql.NullString
			err := env.sqlDB.QueryRowContext(t.Context(),
				"SELECT provider_data FROM notification_channels WHERE id = $1", channel.ID,
			).Scan(&stored)
			require.NoError(t, err)

			assert.False(t, stored.Valid)

			// Seed future data directly; current service inputs do not own this field.
			if tt.providerData != nil {
				_, err := env.db.NotificationChannel.UpdateOneID(channel.ID).
					SetProviderData(tt.providerData).
					Save(t.Context())
				require.NoError(t, err)
			}

			// when: existing adapter code updates configuration, metadata, and annotations
			config := channel.Config
			config.WebHook.URL = "https://example.com/updated"
			metadata := models.Metadata{"svix.endpoint_id": "user-owned"}
			annotations := models.Annotations{"svix.endpoint_id": "system-owned"}

			updated, err := env.adapter.UpdateChannel(t.Context(), notification.UpdateChannelInput{
				NamespacedID: channel.NamespacedID,
				Type:         channel.Type,
				Name:         "updated",
				Disabled:     true,
				Config:       config,
				Annotations:  annotations,
				Metadata:     metadata,
			})
			require.NoError(t, err)

			// then: owned fields change independently of provider data
			assert.Equal(t, "updated", updated.Name)
			assert.True(t, updated.Disabled)
			assert.Equal(t, config, updated.Config)
			assert.Equal(t, metadata, updated.Metadata)
			assert.Equal(t, annotations, updated.Annotations)

			entity, err := env.db.NotificationChannel.Get(t.Context(), channel.ID)
			require.NoError(t, err)

			assert.Equal(t, tt.providerData, entity.ProviderData)

			err = env.sqlDB.QueryRowContext(t.Context(),
				"SELECT provider_data FROM notification_channels WHERE id = $1", channel.ID,
			).Scan(&stored)
			require.NoError(t, err)

			if tt.providerData == nil {
				assert.False(t, stored.Valid)

				return
			}

			require.True(t, stored.Valid)

			expected, err := json.Marshal(tt.providerData)
			require.NoError(t, err)

			assert.JSONEq(t, string(expected), stored.String)
		})
	}
}

func TestChannelReconciliationPreservation(t *testing.T) {
	schemas := []struct {
		name  string
		state testutils.PostgresDBState
	}{
		{name: "Ent schema", state: testutils.PostgresDBStateEntMigrated},
		{name: "SQL migrations", state: testutils.PostgresDBStateAtlasMigrated},
	}
	at := time.Date(2026, time.November, 4, 10, 0, 0, 0, time.UTC)
	tests := []struct {
		name       string
		generation int
		status     notification.ChannelReconcileStatus
		next       *time.Time
	}{
		{name: "unknown", generation: 1, status: notification.ChannelReconcileStatusUnknown},
		{name: "pending", generation: 2, status: notification.ChannelReconcileStatusPending},
		{name: "in sync", generation: math.MaxInt32, status: notification.ChannelReconcileStatusInSync},
		{name: "scheduled retry", generation: 3, status: notification.ChannelReconcileStatusError, next: &at},
	}

	for _, schema := range schemas {
		t.Run(schema.name, func(t *testing.T) {
			testdb := testutils.InitPostgresDB(t, schema.state)
			t.Cleanup(func() { testdb.Close(t) })

			dbClient := testdb.EntDriver.Client()
			adapter, err := New(Config{Client: dbClient, Logger: testutils.NewDiscardLogger(t)})
			require.NoError(t, err)

			env := &channelTestEnv{db: dbClient, sqlDB: testdb.PGDriver.DB(), adapter: adapter}

			for _, tt := range tests {
				t.Run(tt.name, func(t *testing.T) {
					// given: legacy creation leaves reconciliation inactive
					channel := createChannel(t, env, ulid.Make().String(), at, nil)
					stored, err := dbClient.NotificationChannel.Get(t.Context(), channel.ID)
					require.NoError(t, err)

					assert.Equal(t, 1, stored.Generation)
					assert.Equal(t, notification.ChannelReconcileStatusUnknown, stored.ReconcileStatus)
					assert.Nil(t, stored.ReconcileNextAt)

					_, err = dbClient.NotificationChannel.UpdateOneID(channel.ID).
						SetGeneration(tt.generation).
						SetReconcileStatus(tt.status).
						SetOrClearReconcileNextAt(tt.next).
						Save(t.Context())
					require.NoError(t, err)

					// when: current update and soft-delete operations change desired Channel state
					config := channel.Config
					config.WebHook.URL = "https://example.com/updated"
					updated, err := adapter.UpdateChannel(t.Context(), notification.UpdateChannelInput{
						NamespacedID: channel.NamespacedID,
						Type:         channel.Type,
						Name:         "updated",
						Disabled:     true,
						Config:       config,
						Metadata:     models.Metadata{"key": "user-owned"},
						Annotations:  models.Annotations{"key": "system-owned"},
					})
					require.NoError(t, err)

					fetched, err := adapter.GetChannel(t.Context(), notification.GetChannelInput(channel.NamespacedID))
					require.NoError(t, err)

					assert.Equal(t, *updated, *fetched)
					assert.Equal(t, config, fetched.Config)
					assert.True(t, fetched.Disabled)

					stored, err = dbClient.NotificationChannel.Get(t.Context(), channel.ID)
					require.NoError(t, err)

					assert.Equal(t, tt.generation, stored.Generation)
					assert.Equal(t, tt.status, stored.ReconcileStatus)
					if tt.next == nil {
						assert.Nil(t, stored.ReconcileNextAt)
					} else {
						require.NotNil(t, stored.ReconcileNextAt)

						assert.WithinDuration(t, *tt.next, *stored.ReconcileNextAt, 0)
					}

					err = adapter.DeleteChannel(t.Context(), notification.DeleteChannelInput(channel.NamespacedID))
					require.NoError(t, err)

					// then: reconciliation state is preserved, including on soft deletion
					stored, err = dbClient.NotificationChannel.Get(t.Context(), channel.ID)
					require.NoError(t, err)

					assert.NotNil(t, stored.DeletedAt)
					assert.Equal(t, tt.generation, stored.Generation)
					assert.Equal(t, tt.status, stored.ReconcileStatus)
					if tt.next == nil {
						assert.Nil(t, stored.ReconcileNextAt)
					} else {
						require.NotNil(t, stored.ReconcileNextAt)

						assert.WithinDuration(t, *tt.next, *stored.ReconcileNextAt, 0)
					}
				})
			}
		})
	}
}

func TestChannelReconciliationValidation(t *testing.T) {
	env := newChannelTestEnv(t)
	tests := []struct {
		name       string
		generation int
		status     notification.ChannelReconcileStatus
		valid      bool
	}{
		{name: "minimum", generation: 1, status: notification.ChannelReconcileStatusUnknown, valid: true},
		{name: "maximum", generation: math.MaxInt32, status: notification.ChannelReconcileStatusInSync, valid: true},
		{name: "negative generation", generation: -1, status: notification.ChannelReconcileStatusUnknown},
		{name: "zero generation", generation: 0, status: notification.ChannelReconcileStatusUnknown},
		{name: "generation above integer range", generation: math.MaxInt32 + 1, status: notification.ChannelReconcileStatusUnknown},
		{name: "empty status", generation: 1},
		{name: "unsupported status", generation: 1, status: "invalid"},
		{name: "uppercase status", generation: 1, status: "PENDING"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// given: a Channel with valid default reconciliation state
			channel := createChannel(t, env, ulid.Make().String(), clock.Now(), nil)
			before, err := env.db.NotificationChannel.Get(t.Context(), channel.ID)
			require.NoError(t, err)

			createdID := ulid.Make().String()

			// when: create, single update and bulk update receive candidate values
			_, createErr := env.db.NotificationChannel.Create().
				SetID(createdID).
				SetNamespace(channel.Namespace).
				SetType(channel.Type).
				SetName("validation").
				SetConfig(before.Config).
				SetGeneration(tt.generation).
				SetReconcileStatus(tt.status).
				Save(t.Context())

			_, updateErr := env.db.NotificationChannel.UpdateOneID(channel.ID).
				SetGeneration(tt.generation).
				SetReconcileStatus(tt.status).
				Save(t.Context())

			_, bulkErr := env.db.NotificationChannel.Update().
				Where(channeldb.ID(channel.ID)).
				SetGeneration(tt.generation).
				SetReconcileStatus(tt.status).
				Save(t.Context())

			// then: invalid values fail before writing and valid boundary values round-trip
			if tt.valid {
				require.NoError(t, createErr)
				require.NoError(t, updateErr)
				require.NoError(t, bulkErr)
			} else {
				require.ErrorContains(t, createErr, "validator failed for field")
				require.ErrorContains(t, updateErr, "validator failed for field")
				require.ErrorContains(t, bulkErr, "validator failed for field")
			}

			stored, err := env.db.NotificationChannel.Get(t.Context(), channel.ID)
			require.NoError(t, err)

			exists, err := env.db.NotificationChannel.Query().Where(channeldb.ID(createdID)).Exist(t.Context())
			require.NoError(t, err)

			assert.Equal(t, tt.valid, exists)

			if tt.valid {
				assert.Equal(t, tt.generation, stored.Generation)
				assert.Equal(t, tt.status, stored.ReconcileStatus)
			} else {
				assert.Equal(t, before.Generation, stored.Generation)
				assert.Equal(t, before.ReconcileStatus, stored.ReconcileStatus)
				assert.Equal(t, before.UpdatedAt, stored.UpdatedAt)
			}
		})
	}
}

func TestChannelReconciliationMigration(t *testing.T) {
	tests := []struct {
		name    string
		deleted bool
	}{
		{name: "active Channel"},
		{name: "soft-deleted Channel", deleted: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// given: an existing Channel before reconciliation storage is introduced
			testdb := testutils.InitPostgresDB(t, testutils.PostgresDBStateAtlasMigrated)
			t.Cleanup(func() { testdb.Close(t) })

			dbClient := testdb.EntDriver.Client()
			adapter, err := New(Config{Client: dbClient, Logger: testutils.NewDiscardLogger(t)})
			require.NoError(t, err)

			env := &channelTestEnv{db: dbClient, sqlDB: testdb.PGDriver.DB(), adapter: adapter}
			channel := createChannel(t, env, ulid.Make().String(), clock.Now(), nil)
			if tt.deleted {
				err = adapter.DeleteChannel(t.Context(), notification.DeleteChannelInput(channel.NamespacedID))
				require.NoError(t, err)
			}

			before, err := dbClient.NotificationChannel.Get(t.Context(), channel.ID)
			require.NoError(t, err)

			// when: the generated migration is rolled back and applied to the existing row
			for _, direction := range []string{"down", "up"} {
				migration, err := fs.ReadFile(migrate.OMMigrationsConfig.FS, path.Join(
					migrate.OMMigrationsConfig.FSPath,
					"20261110143316_notification_channel_compatibility."+direction+".sql",
				))
				require.NoError(t, err)

				_, err = env.sqlDB.ExecContext(t.Context(), string(migration))
				require.NoError(t, err)
			}

			// then: existing Channels receive inactive defaults without altering desired state
			after, err := dbClient.NotificationChannel.Get(t.Context(), channel.ID)
			require.NoError(t, err)

			assert.Equal(t, 1, after.Generation)
			assert.Equal(t, notification.ChannelReconcileStatusUnknown, after.ReconcileStatus)
			assert.Nil(t, after.ReconcileNextAt)
			assert.Equal(t, before.Config, after.Config)
			assert.Equal(t, before.Type, after.Type)
			assert.Equal(t, before.Disabled, after.Disabled)
			assert.Equal(t, before.DeletedAt, after.DeletedAt)
			assert.Equal(t, before.UpdatedAt, after.UpdatedAt)

			// Old writers omit all three columns and must receive the same defaults.
			var generation int
			var status string
			var next sql.NullTime
			err = env.sqlDB.QueryRowContext(t.Context(),
				`INSERT INTO notification_channels (id, namespace, created_at, updated_at, type, name, config)
				SELECT $1, namespace, NOW(), NOW(), type, name, config FROM notification_channels WHERE id = $2
				RETURNING generation, reconcile_status, reconcile_next_at`, ulid.Make().String(), channel.ID,
			).Scan(&generation, &status, &next)
			require.NoError(t, err)

			assert.Equal(t, 1, generation)
			assert.Equal(t, "unknown", status)
			assert.False(t, next.Valid)
		})
	}
}
