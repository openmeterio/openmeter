package service_test

import (
	"errors"
	"testing"
	"time"

	"github.com/oklog/ulid/v2"
	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	entdb "github.com/openmeterio/openmeter/openmeter/ent/db"
	meteradapter "github.com/openmeterio/openmeter/openmeter/meter/mockadapter"
	"github.com/openmeterio/openmeter/openmeter/notification"
	notificationadapter "github.com/openmeterio/openmeter/openmeter/notification/adapter"
	notificationservice "github.com/openmeterio/openmeter/openmeter/notification/service"
	webhooknoop "github.com/openmeterio/openmeter/openmeter/notification/webhook/noop"
	productcatalogadapter "github.com/openmeterio/openmeter/openmeter/productcatalog/adapter"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/feature"
	"github.com/openmeterio/openmeter/openmeter/testutils"
	"github.com/openmeterio/openmeter/openmeter/watermill/eventbus"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/filter"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/pagination"
)

// These tests exercise notification/service.Service.ListChannels (which calls
// params.Validate() before delegating to the adapter, see channel.go) rather than
// the adapter directly. Service-level CreateChannel also creates a webhook via a
// webhook.Handler, and the only Handler available here is the noop implementation,
// which returns webhook.ErrNotImplemented for CreateWebhook — so channels are seeded
// through the adapter directly (adapter.CreateChannel does not touch the webhook
// handler) and then listed through the real Service, matching the pattern documented
// on channelTestEnv in openmeter/notification/adapter/channel_test.go.
type serviceTestEnv struct {
	db      *entdb.Client
	adapter notification.Repository
	service notification.Service
}

func newServiceTestEnv(t *testing.T) *serviceTestEnv {
	t.Helper()

	testdb := testutils.InitPostgresDB(t, testutils.PostgresDBStateEntMigrated)
	dbClient := testdb.EntDriver.Client()

	t.Cleanup(func() {
		_ = dbClient.Close()
		testdb.Close(t)
	})

	logger := testutils.NewDiscardLogger(t)

	adapter, err := notificationadapter.New(notificationadapter.Config{
		Client: dbClient,
		Logger: logger,
	})
	require.NoError(t, err, "constructing notification adapter must not fail")

	// service.Config requires a non-nil FeatureConnector even though ListChannels
	// never calls it; wire a real one backed by the same test DB (mirrors
	// test/notification/testenv.go) instead of leaving it nil, since notification
	// service.New rejects a nil FeatureConnector outright.
	meterService, err := meteradapter.New(nil)
	require.NoError(t, err, "constructing mock meter adapter must not fail")
	require.NoError(t, meterService.SetDBClient(dbClient), "setting meter adapter DB client must not fail")

	featureRepo := productcatalogadapter.NewPostgresFeatureRepo(dbClient, logger)
	featureConnector := feature.NewFeatureConnector(featureRepo, meterService, eventbus.NewMock(t))

	svc, err := notificationservice.New(notificationservice.Config{
		Adapter:          adapter,
		FeatureConnector: featureConnector,
		Webhook:          webhooknoop.New(logger),
		Logger:           logger,
	})
	require.NoError(t, err, "constructing notification service must not fail")

	return &serviceTestEnv{db: dbClient, adapter: adapter, service: svc}
}

// seedChannel creates a webhook channel via the adapter directly (see serviceTestEnv
// doc comment for why), freezing the clock at `at` for the duration of creation so
// CreatedAt/UpdatedAt land on a caller-chosen instant. The freeze/unfreeze pair is
// scoped to this call (defer runs even if require.NoError below triggers
// t.Goexit()), so a failed seed can never leak frozen time into later tests in the
// package.
func seedChannel(t *testing.T, env *serviceTestEnv, ns string, at time.Time, name string) notification.Channel {
	t.Helper()

	clock.FreezeTime(at)
	defer clock.UnFreeze()

	in := notification.CreateChannelInput{
		NamespacedModel: models.NamespacedModel{Namespace: ns},
		Type:            notification.ChannelTypeWebhook,
		Name:            name,
		Config: notification.ChannelConfig{
			ChannelConfigMeta: notification.ChannelConfigMeta{Type: notification.ChannelTypeWebhook},
			WebHook:           notification.WebHookChannelConfig{URL: "https://example.com/hook"},
		},
	}

	channel, err := env.adapter.CreateChannel(t.Context(), in)
	require.NoError(t, err, "seeding channel must not fail")
	require.NotNil(t, channel)

	return *channel
}

// TestListChannels_ServiceFilters covers the two filter fields with no test coverage
// at any layer before this file: oeq (one-of-equal, modeled as filter.FilterString.In)
// on Type and Name, and the UpdatedAt filter. All three are exercised through
// notification/service.Service.ListChannels rather than the adapter directly.
func TestListChannels_ServiceFilters(t *testing.T) {
	env := newServiceTestEnv(t)
	ns := ulid.Make().String()

	frozen := clock.Now()
	tAlpha := frozen.Add(-3 * time.Hour)
	tBeta := frozen.Add(-2 * time.Hour)
	tGamma := frozen.Add(-90 * time.Minute)
	tAlphaUpdated := frozen.Add(-1 * time.Hour)

	alpha := seedChannel(t, env, ns, tAlpha, "channel-alpha")
	beta := seedChannel(t, env, ns, tBeta, "channel-beta")
	gamma := seedChannel(t, env, ns, tGamma, "channel-gamma")

	// Touch alpha's updated_at independently of its creation time so an UpdatedAt
	// filter window can distinguish it from beta and gamma, which keep their
	// creation-time updated_at (Ent's UpdateDefault mirrors CreatedAt at insert).
	clock.FreezeTime(tAlphaUpdated)
	_, err := env.adapter.UpdateChannel(t.Context(), notification.UpdateChannelInput{
		NamespacedID: models.NamespacedID{Namespace: ns, ID: alpha.ID},
		Type:         alpha.Type,
		Name:         alpha.Name,
		Disabled:     alpha.Disabled,
		Config:       alpha.Config,
	})
	clock.UnFreeze()
	require.NoError(t, err, "updating alpha to advance its updated_at must not fail")

	testCases := []struct {
		name    string
		input   notification.ListChannelsInput
		wantIDs []string
	}{
		{
			// given: three webhook channels, all of type WEBHOOK
			// when: filtering with Type oeq against ["WEBHOOK", "UNKNOWN_TYPE"]
			// then: every channel matching the real value comes back; the bogus
			// value in the list contributes no matches but must not error or
			// narrow the result, proving oeq performs an OR/IN match rather than
			// requiring every listed value to match.
			//
			// This row alone would pass even if the Type predicate were dropped
			// entirely (WEBHOOK is the only channel type in the domain, so "match
			// everything" is indistinguishable from "no filter applied"). The
			// negative row directly below is what actually proves the predicate
			// is wired in: a list containing only the bogus value must exclude
			// every real channel.
			name: "Type oeq matches every channel via the real value in the list",
			input: notification.ListChannelsInput{
				Type: &filter.FilterString{In: &[]string{string(notification.ChannelTypeWebhook), "UNKNOWN_TYPE"}},
			},
			wantIDs: []string{alpha.ID, beta.ID, gamma.ID},
		},
		{
			// given: three webhook channels, all of type WEBHOOK
			// when: filtering with Type oeq against a list that excludes the real value
			// then: no channels match, proving the In predicate is actually applied
			// rather than being a no-op that happens to return everything
			name: "Type oeq excluding the real value matches nothing",
			input: notification.ListChannelsInput{
				Type: &filter.FilterString{In: &[]string{"UNKNOWN_TYPE"}},
			},
			wantIDs: nil,
		},
		{
			// given: three distinctly named channels
			// when: filtering with Name oeq against two of the three names
			// then: exactly those two channels come back, not the third
			name: "Name oeq matches exactly the two listed names",
			input: notification.ListChannelsInput{
				Name: &filter.FilterString{In: &[]string{"channel-alpha", "channel-beta"}},
			},
			wantIDs: []string{alpha.ID, beta.ID},
		},
		{
			// given: alpha was updated at tAlphaUpdated; beta and gamma still carry
			// their creation-time updated_at, both outside the window below
			// when: filtering UpdatedAt to a window tight around tAlphaUpdated
			// then: only alpha comes back
			name: "UpdatedAt range matches only the channel updated inside the window",
			input: notification.ListChannelsInput{
				UpdatedAt: &filter.FilterTime{
					And: &[]filter.FilterTime{
						{Gte: lo.ToPtr(tAlphaUpdated.Add(-time.Minute))},
						{Lte: lo.ToPtr(tAlphaUpdated.Add(time.Minute))},
					},
				},
			},
			wantIDs: []string{alpha.ID},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			input := tc.input
			input.Namespaces = []string{ns}
			input.Page = pagination.NewPage(1, 20)

			result, err := env.service.ListChannels(t.Context(), input)
			require.NoError(t, err)

			gotIDs := make([]string, 0, len(result.Items))
			for _, item := range result.Items {
				gotIDs = append(gotIDs, item.ID)
			}
			assert.ElementsMatch(t, tc.wantIDs, gotIDs)
		})
	}
}

// TestListChannels_InvalidFilterCombinationRejected covers the highest-value gap in
// filter coverage: Service.ListChannels must reject an invalid filter predicate via
// params.Validate() before ever reaching the adapter/database (Service.ListChannels
// wraps Validate() failures as "invalid params: %w" without calling the adapter).
//
// The invalid combination is two operators (Eq and Contains) set directly on one
// filter.FilterString node instead of being combined through And/Or.
// pkg/filter's validateSingleOperator counts non-nil operator fields on a single
// node and returns ErrFilterMultipleOperators once more than one is set — this test
// asserts the exact sentinel error surfaces through Service.ListChannels.
func TestListChannels_InvalidFilterCombinationRejected(t *testing.T) {
	env := newServiceTestEnv(t)
	ns := ulid.Make().String()

	// given: a Name filter with both Eq and Contains set on the same node
	// when: Service.ListChannels validates the input
	// then: it returns a models.GenericValidationError wrapping
	// filter.ErrFilterMultipleOperators. Service.ListChannels calls
	// params.Validate() and returns on error before ever calling s.adapter.ListChannels,
	// so this also exercises that the adapter is never reached — not separately
	// asserted here, but guaranteed by that early-return structure.
	_, err := env.service.ListChannels(t.Context(), notification.ListChannelsInput{
		Namespaces: []string{ns},
		Name: &filter.FilterString{
			Eq:       lo.ToPtr("channel-name"),
			Contains: lo.ToPtr("channel"),
		},
		Page: pagination.NewPage(1, 20),
	})

	require.Error(t, err)
	assert.True(t, models.IsGenericValidationError(err), "expected a models.GenericValidationError, got: %v", err)
	assert.ErrorIs(t, err, filter.ErrFilterMultipleOperators)
}

// TestUpdateChannel_AfterDeleteReturnsUpdateAfterDeleteError covers the read-before-write
// path in Service.UpdateChannel: it fetches the channel via
// adapter.GetChannel before validating/persisting the update, and adapter.GetChannel has
// no deleted_at filter (see notification/adapter/channel.go GetChannel, which only filters
// on ID and Namespace), so a soft-deleted channel is still fetched and the DeletedAt check
// fires. The v3 handler's error encoder (api/v3/handlers/notification/channels/error_encoder.go)
// maps this error to HTTP 409, so this test exercises the condition that mapping depends on.
func TestUpdateChannel_AfterDeleteReturnsUpdateAfterDeleteError(t *testing.T) {
	env := newServiceTestEnv(t)
	ns := ulid.Make().String()

	channel := seedChannel(t, env, ns, clock.Now(), "channel-to-delete")

	// Deleted via the adapter directly rather than Service.DeleteChannel: the service
	// method calls s.webhook.DeleteWebhook before adapter.DeleteChannel,
	// and the only webhook.Handler available in this harness is the noop implementation,
	// which unconditionally returns webhook.ErrNotImplemented (see
	// notification/webhook/noop/noop.go DeleteWebhook) — the same constraint documented on
	// serviceTestEnv above for why channels are seeded through the adapter instead of
	// Service.CreateChannel.
	err := env.adapter.DeleteChannel(t.Context(), notification.DeleteChannelInput{
		Namespace: ns,
		ID:        channel.ID,
	})
	require.NoError(t, err, "deleting channel via the adapter must not fail")

	// given: channel is soft-deleted (DeletedAt set, adapter.GetChannel still returns it)
	// when: Service.UpdateChannel is called on the deleted channel with otherwise-valid params
	// then: it returns a notification.UpdateAfterDeleteError, not a NotFoundError, proving the
	// deleted channel is actually fetched and the DeletedAt branch is reached
	_, err = env.service.UpdateChannel(t.Context(), notification.UpdateChannelInput{
		NamespacedID: models.NamespacedID{Namespace: ns, ID: channel.ID},
		Type:         channel.Type,
		Name:         channel.Name,
		Disabled:     channel.Disabled,
		Config:       channel.Config,
	})

	require.Error(t, err)

	var updateAfterDeleteErr notification.UpdateAfterDeleteError
	assert.True(t, errors.As(err, &updateAfterDeleteErr),
		"expected a notification.UpdateAfterDeleteError, got: %v", err)
}
