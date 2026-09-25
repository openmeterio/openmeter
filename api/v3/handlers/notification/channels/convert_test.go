package channels

import (
	"testing"

	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	api "github.com/openmeterio/openmeter/api/v3"
	"github.com/openmeterio/openmeter/openmeter/notification"
	"github.com/openmeterio/openmeter/pkg/filter"
	"github.com/openmeterio/openmeter/pkg/models"
)

// TestFromAPICreateChannelRequest_TypeCasing covers the type casing mapping: the v3 wire enum value
// ("webhook") must translate to the domain/DB value ("WEBHOOK") on create, and an
// unrecognized wire value must be rejected rather than silently stored — a bad
// ChannelType would otherwise reach ChannelConfig.Validate()'s type switch and fail
// deep inside domain validation with a confusing error instead of a clear one here.
func TestFromAPICreateChannelRequest_TypeCasing(t *testing.T) {
	baseBody := func(channelType api.NotificationChannelType) api.CreateNotificationChannelRequest {
		return api.CreateNotificationChannelRequest{
			Name: "test channel",
			Type: channelType,
			Url:  "https://example.com/hook",
		}
	}

	t.Run("lowercase wire value maps to uppercase domain value", func(t *testing.T) {
		result, err := FromAPICreateChannelRequest("ns1", baseBody(api.NotificationChannelTypeWebhook))
		require.NoError(t, err)
		assert.Equal(t, notification.ChannelTypeWebhook, result.Type)
		assert.Equal(t, notification.ChannelType("WEBHOOK"), result.Type)
		// The nested config's type meta must carry the same mapped value, since
		// ChannelConfig.Validate() switches on it independently of Type above.
		assert.Equal(t, notification.ChannelTypeWebhook, result.Config.ChannelConfigMeta.Type)
	})

	t.Run("unknown type is rejected, not passed through", func(t *testing.T) {
		_, err := FromAPICreateChannelRequest("ns1", baseBody("bogus"))
		require.Error(t, err)
	})

	t.Run("empty type is rejected, not passed through", func(t *testing.T) {
		_, err := FromAPICreateChannelRequest("ns1", baseBody(""))
		require.Error(t, err)
	})

	t.Run("uppercase V1-style value is rejected under the v3 lowercase casing convention", func(t *testing.T) {
		// v3 deliberately spells the wire value "webhook" (lowercase); the domain's
		// own uppercase "WEBHOOK" spelling must not be silently accepted as-is on the
		// v3 wire, or a client copying the V1 casing would get an inconsistent channel.
		_, err := FromAPICreateChannelRequest("ns1", baseBody("WEBHOOK"))
		require.Error(t, err)
	})
}

// TestFromAPIUpdateChannelRequest pins the update request's replace semantics as
// documented on the UpdateNotificationChannelRequest spec model: type/name/url are
// required on the wire, an omitted disabled resets to false, and an omitted
// signing_secret maps to the empty string — which the service layer treats as "keep
// the current secret", never as clearing the credential.
func TestFromAPIUpdateChannelRequest(t *testing.T) {
	baseBody := func() api.UpdateNotificationChannelRequest {
		return api.UpdateNotificationChannelRequest{
			Name: "updated channel",
			Type: api.NotificationChannelTypeWebhook,
			Url:  "https://example.com/hook",
		}
	}

	t.Run("full body maps to the domain input", func(t *testing.T) {
		body := baseBody()
		body.Disabled = lo.ToPtr(true)
		body.SigningSecret = lo.ToPtr("whsec_" + "MfKQ9r8GKYqrTwjUPD8ILPZIo2LaLaSw")

		got, err := FromAPIUpdateChannelRequest("ns1", "chan1", body)
		require.NoError(t, err)
		assert.Equal(t, notification.ChannelTypeWebhook, got.Type)
		assert.Equal(t, notification.ChannelTypeWebhook, got.Config.ChannelConfigMeta.Type)
		assert.Equal(t, "updated channel", got.Name)
		assert.True(t, got.Disabled)
		assert.Equal(t, "https://example.com/hook", got.Config.WebHook.URL)
		assert.Equal(t, *body.SigningSecret, got.Config.WebHook.SigningSecret)
	})

	t.Run("omitted disabled resets to false per replace semantics", func(t *testing.T) {
		got, err := FromAPIUpdateChannelRequest("ns1", "chan1", baseBody())
		require.NoError(t, err)
		assert.False(t, got.Disabled)
	})

	t.Run("omitted signing_secret maps to empty string for the service-side keep-current handling", func(t *testing.T) {
		got, err := FromAPIUpdateChannelRequest("ns1", "chan1", baseBody())
		require.NoError(t, err)
		assert.Empty(t, got.Config.WebHook.SigningSecret)
	})

	t.Run("unknown type is rejected", func(t *testing.T) {
		body := baseBody()
		body.Type = "bogus"
		_, err := FromAPIUpdateChannelRequest("ns1", "chan1", body)
		require.Error(t, err)
	})
}

// TestToAPIChannel_TypeCasing covers the reverse type casing direction: a domain channel with
// the uppercase DB value must map back to the lowercase wire value, and an
// unrecognized domain value must error rather than leak the raw string onto the wire.
func TestToAPIChannel_TypeCasing(t *testing.T) {
	t.Run("uppercase domain value maps to lowercase wire value", func(t *testing.T) {
		channel := notification.Channel{
			NamespacedID: models.NamespacedID{Namespace: "ns1", ID: "chan1"},
			Type:         notification.ChannelTypeWebhook,
			Name:         "test",
			Config: notification.ChannelConfig{
				WebHook: notification.WebHookChannelConfig{URL: "https://example.com/hook"},
			},
		}

		result, err := ToAPIChannel(channel)
		require.NoError(t, err)
		assert.Equal(t, api.NotificationChannelTypeWebhook, result.Type)
		assert.Equal(t, api.NotificationChannelType("webhook"), result.Type)
	})

	t.Run("unknown domain type errors instead of leaking the raw value", func(t *testing.T) {
		channel := notification.Channel{
			NamespacedID: models.NamespacedID{Namespace: "ns1", ID: "chan1"},
			Type:         "BOGUS",
			Name:         "test",
		}

		_, err := ToAPIChannel(channel)
		require.Error(t, err)
	})
}

// TestMapAPIChannelTypeFilter covers the filter direction of the type casing: filter[type][...] wire
// values must be translated to the domain/DB value before the predicate reaches the
// adapter, or filter[type][eq]=webhook would silently match zero rows against a
// column that stores "WEBHOOK".
func TestMapAPIChannelTypeFilter(t *testing.T) {
	testCases := []struct {
		name    string
		input   *filter.FilterString
		want    *filter.FilterString
		wantErr bool
	}{
		{
			name:  "nil filter is a no-op",
			input: nil,
			want:  nil,
		},
		{
			name:  "eq translates the wire value to the domain value",
			input: &filter.FilterString{Eq: lo.ToPtr("webhook")},
			want:  &filter.FilterString{Eq: lo.ToPtr("WEBHOOK")},
		},
		{
			name:  "ne translates the wire value to the domain value",
			input: &filter.FilterString{Ne: lo.ToPtr("webhook")},
			want:  &filter.FilterString{Ne: lo.ToPtr("WEBHOOK")},
		},
		{
			name:  "in translates every wire value to its domain value",
			input: &filter.FilterString{In: &[]string{"webhook"}},
			want:  &filter.FilterString{In: &[]string{"WEBHOOK"}},
		},
		{
			name:    "unknown eq value is rejected rather than silently matching zero rows",
			input:   &filter.FilterString{Eq: lo.ToPtr("bogus")},
			wantErr: true,
		},
		{
			name:    "unknown ne value is rejected",
			input:   &filter.FilterString{Ne: lo.ToPtr("bogus")},
			wantErr: true,
		},
		{
			name:    "an unknown value inside in is rejected",
			input:   &filter.FilterString{In: &[]string{"webhook", "bogus"}},
			wantErr: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := mapAPIChannelTypeFilter(tc.input)
			if tc.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tc.want, got)
		})
	}
}

// TestFromAPIChannelSortField verifies the sort allow-list must accept exactly the
// four documented fields, map each to the matching domain OrderBy constant, and
// reject anything else instead of silently falling back to an unintended order.
func TestFromAPIChannelSortField(t *testing.T) {
	testCases := []struct {
		field   string
		want    notification.OrderBy
		wantErr bool
	}{
		{field: "id", want: notification.OrderByID},
		{field: "type", want: notification.OrderByType},
		{field: "created_at", want: notification.OrderByCreatedAt},
		{field: "updated_at", want: notification.OrderByUpdatedAt},
		{field: "bogus", wantErr: true},
		// FromAPIChannelSortField is only invoked when a sort field was explicitly
		// supplied (see list.go); an empty string reaching this function is therefore
		// itself an unsupported value, not a request for the default.
		{field: "", wantErr: true},
	}

	for _, tc := range testCases {
		name := tc.field
		if name == "" {
			name = "empty"
		}
		t.Run(name, func(t *testing.T) {
			got, err := FromAPIChannelSortField(t.Context(), tc.field)
			if tc.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tc.want, got)
		})
	}
}

// TestToAPIChannel_LabelsRoundTrip covers the labels mapping: domain Metadata and Annotations must
// fold into (and back out of) the single wire `labels` field, with Annotations
// namespaced under the "openmeter_" prefix, per labels.FromMetadataAnnotations.
func TestToAPIChannel_LabelsRoundTrip(t *testing.T) {
	channel := notification.Channel{
		NamespacedID: models.NamespacedID{Namespace: "ns1", ID: "chan1"},
		Type:         notification.ChannelTypeWebhook,
		Name:         "test",
		Config: notification.ChannelConfig{
			WebHook: notification.WebHookChannelConfig{URL: "https://example.com/hook"},
		},
		Metadata:    models.Metadata{"env": "prod"},
		Annotations: models.Annotations{"owner": "sre"},
	}

	apiChannel, err := ToAPIChannel(channel)
	require.NoError(t, err)
	require.NotNil(t, apiChannel.Labels)
	assert.Equal(t, "prod", (*apiChannel.Labels)["env"])
	assert.Equal(t, "sre", (*apiChannel.Labels)["openmeter_owner"])

	// Metadata-derived labels round-trip cleanly: resubmitting just the metadata
	// portion of the API labels reconstructs the same domain Metadata.
	metadataOnly := api.Labels{"env": (*apiChannel.Labels)["env"]}
	created, err := FromAPICreateChannelRequest("ns1", api.CreateNotificationChannelRequest{
		Name:   "test",
		Type:   api.NotificationChannelTypeWebhook,
		Url:    "https://example.com/hook",
		Labels: &metadataOnly,
	})
	require.NoError(t, err)
	assert.Equal(t, channel.Metadata, created.Metadata)

	// Annotation-derived labels do NOT round-trip: resubmitting the full label set
	// exactly as returned by ToAPIChannel — including the server-added "openmeter_"
	// prefix — is rejected by the shared labels.ToMetadataAnnotations reserved-prefix
	// check (api/v3/labels/validate.go), which forbids any incoming key starting with
	// "openmeter" regardless of who produced it. This means a client that reads a
	// channel back via GET and PATCHes the same labels object unmodified will get a
	// validation error, not a no-op update. This is pre-existing shared-package
	// behavior (see api/v3/labels/convert_test.go), not specific to this handler, so
	// it is documented here rather than treated as a defect of FromAPIUpdateChannelRequest.
	_, err = FromAPICreateChannelRequest("ns1", api.CreateNotificationChannelRequest{
		Name:   "test",
		Type:   api.NotificationChannelTypeWebhook,
		Url:    "https://example.com/hook",
		Labels: apiChannel.Labels,
	})
	require.Error(t, err)
}
