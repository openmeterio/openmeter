package adapter_test

import (
	"testing"
	"time"

	"github.com/oklog/ulid/v2"
	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	entdb "github.com/openmeterio/openmeter/openmeter/ent/db"
	"github.com/openmeterio/openmeter/openmeter/notification"
	notificationadapter "github.com/openmeterio/openmeter/openmeter/notification/adapter"
	"github.com/openmeterio/openmeter/openmeter/testutils"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/filter"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/pagination"
	"github.com/openmeterio/openmeter/pkg/sortx"
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

	adapter, err := notificationadapter.New(notificationadapter.Config{
		Client: dbClient,
		Logger: testutils.NewDiscardLogger(t),
	})
	require.NoError(t, err, "constructing notification adapter must not fail")

	return &channelTestEnv{db: dbClient, adapter: adapter}
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
