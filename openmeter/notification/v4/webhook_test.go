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

func TestWebhookConfigUpdateJSON(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		want      v4.WebhookConfigUpdate
		wantError string
	}{
		{
			name:  "omitted headers preserve existing headers",
			input: `{}`,
		},
		{
			name:  "null headers remove all headers",
			input: `{"custom_headers":null}`,
			want: v4.WebhookConfigUpdate{
				CustomHeaders: new(map[string]string),
			},
		},
		{
			name:  "empty headers remove all headers",
			input: `{"custom_headers":{}}`,
			want: v4.WebhookConfigUpdate{
				CustomHeaders: new(map[string]string{}),
			},
		},
		{
			name:  "populated headers replace existing headers",
			input: `{"custom_headers":{"X-Source":"test"}}`,
			want: v4.WebhookConfigUpdate{
				CustomHeaders: new(map[string]string{"X-Source": "test"}),
			},
		},
		{
			name:  "other fields remain available",
			input: `{"url":"https://example.com/webhooks","signing_secret":"test-secret","custom_headers":null}`,
			want: v4.WebhookConfigUpdate{
				URL:           new("https://example.com/webhooks"),
				SigningSecret: new("test-secret"),
				CustomHeaders: new(map[string]string),
			},
		},
		{
			name:      "non-string header value",
			input:     `{"url":"https://example.com/changed","custom_headers":{"X-Source":42}}`,
			wantError: "invalid custom headers",
		},
		{
			name:      "headers must be an object",
			input:     `{"custom_headers":[]}`,
			wantError: "invalid custom headers",
		},
		{
			name:      "invalid URL field type",
			input:     `{"url":42}`,
			wantError: "invalid webhook configuration update",
		},
		{
			name:      "malformed JSON",
			input:     `{"custom_headers":`,
			wantError: "unexpected end of JSON input",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given a previously decoded update, stale fields must not leak into a new request.
			original := v4.WebhookConfigUpdate{
				URL:           new("https://example.com/original"),
				SigningSecret: new("original-secret"),
				CustomHeaders: new(map[string]string{"X-Original": "value"}),
			}
			decoded := original

			// When decoding the next update.
			err := json.Unmarshal([]byte(tt.input), &decoded)

			// Then failed decoding preserves the receiver and valid updates retain their intent.
			if tt.wantError != "" {
				require.ErrorContains(t, err, tt.wantError)
				require.Equal(t, original, decoded)

				return
			}

			require.NoError(t, err)
			require.Equal(t, tt.want, decoded)

			encoded, err := json.Marshal(decoded)
			require.NoError(t, err)
			require.JSONEq(t, tt.input, string(encoded))

			var roundTripped v4.WebhookConfigUpdate

			require.NoError(t, json.Unmarshal(encoded, &roundTripped))
			require.Equal(t, tt.want, roundTripped)
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
