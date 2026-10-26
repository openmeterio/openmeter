package v4_test

import (
	"encoding/base64"
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	v4 "github.com/openmeterio/openmeter/openmeter/notification/v4"
	"github.com/openmeterio/openmeter/pkg/models"
)

func TestWebhookConfigValidate(t *testing.T) {
	secret := base64.StdEncoding.EncodeToString([]byte(strings.Repeat("x", 32)))

	tests := []struct {
		name       string
		config     v4.WebhookConfig
		wantErrors []string
	}{
		{
			name: "HTTPS without explicit secret",
			config: v4.WebhookConfig{
				URL: "https://example.com/webhooks",
			},
		},
		{
			name: "HTTP with custom headers",
			config: v4.WebhookConfig{
				URL:           "http://localhost:8080/webhooks",
				CustomHeaders: map[string]string{"X-Source": "openmeter"},
			},
		},
		{
			name: "prefixed secret",
			config: v4.WebhookConfig{
				URL:           "https://example.com/webhooks",
				SigningSecret: "whsec_" + secret,
			},
		},
		{
			name: "unprefixed secret",
			config: v4.WebhookConfig{
				URL:           "https://example.com/webhooks",
				SigningSecret: secret,
			},
		},
		{
			name:       "missing URL",
			wantErrors: []string{"invalid endpoint URL"},
		},
		{
			name: "relative URL",
			config: v4.WebhookConfig{
				URL: "/webhooks",
			},
			wantErrors: []string{"invalid endpoint URL"},
		},
		{
			name: "unsupported URL scheme",
			config: v4.WebhookConfig{
				URL: "ftp://example.com/webhooks",
			},
			wantErrors: []string{"invalid endpoint URL"},
		},
		{
			name: "missing URL host",
			config: v4.WebhookConfig{
				URL: "https:///webhooks",
			},
			wantErrors: []string{"invalid endpoint URL"},
		},
		{
			name: "malformed URL",
			config: v4.WebhookConfig{
				URL: "https://example.com/%zz",
			},
			wantErrors: []string{"invalid endpoint URL"},
		},
		{
			name: "invalid secret",
			config: v4.WebhookConfig{
				URL:           "https://example.com/webhooks",
				SigningSecret: "invalid",
			},
			wantErrors: []string{"invalid signing secret"},
		},
		{
			name: "multiple invalid fields",
			config: v4.WebhookConfig{
				SigningSecret: "invalid",
			},
			wantErrors: []string{"invalid endpoint URL", "invalid signing secret"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()

			switch {
			case len(tt.wantErrors) > 0:
				require.True(t, models.IsGenericValidationError(err))

				for _, message := range tt.wantErrors {
					require.ErrorContains(t, err, message)
				}

			default:
				require.NoError(t, err)
			}
		})
	}
}

func TestWebhookConfigUpdateValidate(t *testing.T) {
	secret := base64.StdEncoding.EncodeToString([]byte(strings.Repeat("x", 32)))

	tests := []struct {
		name       string
		config     v4.WebhookConfigUpdate
		wantErrors []string
	}{
		{
			name: "no changes",
		},
		{
			name:   "HTTPS URL only",
			config: v4.WebhookConfigUpdate{URL: new("https://example.com/webhooks")},
		},
		{
			name:   "HTTP URL only",
			config: v4.WebhookConfigUpdate{URL: new("http://localhost:8080/webhooks")},
		},
		{
			name:   "headers only",
			config: v4.WebhookConfigUpdate{CustomHeaders: new(map[string]string{"x-test": "value"})},
		},
		{
			name:   "clear headers with nil map",
			config: v4.WebhookConfigUpdate{CustomHeaders: new(map[string]string)},
		},
		{
			name:   "clear headers with empty map",
			config: v4.WebhookConfigUpdate{CustomHeaders: new(map[string]string{})},
		},
		{
			name:   "prefixed secret only",
			config: v4.WebhookConfigUpdate{SigningSecret: new("whsec_" + secret)},
		},
		{
			name:   "unprefixed secret only",
			config: v4.WebhookConfigUpdate{SigningSecret: new(secret)},
		},
		{
			name:       "empty URL",
			config:     v4.WebhookConfigUpdate{URL: new("")},
			wantErrors: []string{"invalid endpoint URL"},
		},
		{
			name:       "relative URL",
			config:     v4.WebhookConfigUpdate{URL: new("/webhooks")},
			wantErrors: []string{"invalid endpoint URL"},
		},
		{
			name:       "unsupported URL scheme",
			config:     v4.WebhookConfigUpdate{URL: new("ftp://example.com/webhooks")},
			wantErrors: []string{"invalid endpoint URL"},
		},
		{
			name:       "missing URL host",
			config:     v4.WebhookConfigUpdate{URL: new("https:///webhooks")},
			wantErrors: []string{"invalid endpoint URL"},
		},
		{
			name:       "malformed URL",
			config:     v4.WebhookConfigUpdate{URL: new("https://example.com/%zz")},
			wantErrors: []string{"invalid endpoint URL"},
		},
		{
			name:       "empty secret",
			config:     v4.WebhookConfigUpdate{SigningSecret: new("")},
			wantErrors: []string{"invalid signing secret"},
		},
		{
			name:       "invalid secret",
			config:     v4.WebhookConfigUpdate{SigningSecret: new("invalid")},
			wantErrors: []string{"invalid signing secret"},
		},
		{
			name:       "multiple invalid fields",
			config:     v4.WebhookConfigUpdate{URL: new(""), SigningSecret: new("invalid")},
			wantErrors: []string{"invalid endpoint URL", "invalid signing secret"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()

			switch {
			case len(tt.wantErrors) > 0:
				require.True(t, models.IsGenericValidationError(err))

				for _, message := range tt.wantErrors {
					require.ErrorContains(t, err, message)
				}

			default:
				require.NoError(t, err)
			}
		})
	}
}

func TestWebhookChannel(t *testing.T) {
	tests := []struct {
		name   string
		config v4.WebhookConfig
	}{
		{
			name: "minimal configuration",
			config: v4.WebhookConfig{
				URL: "https://example.com/webhooks",
			},
		},
		{
			name: "complete configuration",
			config: v4.WebhookConfig{
				URL:           "https://example.com/webhooks",
				CustomHeaders: map[string]string{"X-Source": "openmeter"},
				SigningSecret: "whsec_" + base64.StdEncoding.EncodeToString([]byte(strings.Repeat("x", 32))),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given a directly initialized channel.
			channel := &v4.WebhookChannel{
				Name:   tt.name,
				Config: tt.config,
			}

			// When round-tripping its serialized representation.
			encoded, err := json.Marshal(channel)
			require.NoError(t, err)

			var decoded v4.WebhookChannel

			require.NoError(t, json.Unmarshal(encoded, &decoded))

			// Then channel fields and configuration are retained.
			require.Equal(t, *channel, decoded)

			var typedChannel v4.Channel = channel

			require.Equal(t, v4.ChannelTypeWebhook, typedChannel.ChannelType())

			// Configuration changes cannot change the channel's identity.
			channel.Config = v4.WebhookConfig{}
			channel.Disabled = true

			require.Equal(t, v4.ChannelTypeWebhook, typedChannel.ChannelType())
		})
	}
}

func TestWebhookConfigJSON(t *testing.T) {
	tests := []struct {
		name   string
		config v4.WebhookConfig
		want   string
	}{
		{
			name: "optional fields omitted",
			config: v4.WebhookConfig{
				URL: "https://example.com/webhooks",
			},
			want: `{"url":"https://example.com/webhooks"}`,
		},
		{
			name: "existing snake case keys",
			config: v4.WebhookConfig{
				URL:           "https://example.com/webhooks",
				CustomHeaders: map[string]string{"X-Source": "openmeter"},
				SigningSecret: "test-secret",
			},
			want: `{"url":"https://example.com/webhooks","custom_headers":{"X-Source":"openmeter"},"signing_secret":"test-secret"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			encoded, err := json.Marshal(tt.config)
			require.NoError(t, err)
			require.JSONEq(t, tt.want, string(encoded))

			var decoded v4.WebhookConfig

			require.NoError(t, json.Unmarshal(encoded, &decoded))
			require.Equal(t, tt.config, decoded)
		})
	}
}
