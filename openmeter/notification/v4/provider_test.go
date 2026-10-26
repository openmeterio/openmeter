package v4

import (
	"encoding/json"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/pkg/models"
)

func TestGetNamespaceInputValidate(t *testing.T) {
	tests := []struct {
		name       string
		input      GetNamespaceInput
		wantErrors []string
	}{
		{
			name:  "valid ID",
			input: GetNamespaceInput{ID: "ns-test"},
		},
		{
			name:       "missing ID",
			wantErrors: []string{"namespace ID is required"},
		},
		{
			name:       "whitespace ID",
			input:      GetNamespaceInput{ID: " \t\n"},
			wantErrors: []string{"namespace ID is required"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assertProviderInputValidation(t, tt.input, tt.wantErrors)
		})
	}
}

func TestCreateNamespaceInputValidate(t *testing.T) {
	tests := []struct {
		name       string
		modify     func(*CreateNamespaceInput)
		wantErrors []string
	}{
		{
			name: "valid without idempotency key",
		},
		{
			name: "valid with idempotency key",
			modify: func(input *CreateNamespaceInput) {
				input.IdempotencyKey = new("create-namespace")
			},
		},
		{
			name: "missing namespace ID",
			modify: func(input *CreateNamespaceInput) {
				input.ID = ""
			},
			wantErrors: []string{"namespace ID is required"},
		},
		{
			name: "whitespace namespace name",
			modify: func(input *CreateNamespaceInput) {
				input.Name = " \t\n"
			},
			wantErrors: []string{"namespace name is required"},
		},
		{
			name: "empty idempotency key",
			modify: func(input *CreateNamespaceInput) {
				input.IdempotencyKey = new("")
			},
			wantErrors: []string{"idempotency key must not be blank when provided"},
		},
		{
			name: "whitespace idempotency key",
			modify: func(input *CreateNamespaceInput) {
				input.IdempotencyKey = new(" \t\n")
			},
			wantErrors: []string{"idempotency key must not be blank when provided"},
		},
		{
			name: "multiple invalid fields",
			modify: func(input *CreateNamespaceInput) {
				*input = CreateNamespaceInput{IdempotencyKey: new("")}
			},
			wantErrors: []string{
				"namespace ID is required",
				"namespace name is required",
				"idempotency key must not be blank when provided",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := CreateNamespaceInput{
				ProviderNamespace: ProviderNamespace{
					ID:   "ns-test",
					Name: "Namespace",
				},
			}

			if tt.modify != nil {
				tt.modify(&input)
			}

			assertProviderInputValidation(t, input, tt.wantErrors)
		})
	}
}

func TestUpdateNamespaceInputValidate(t *testing.T) {
	tests := []struct {
		name       string
		input      UpdateNamespaceInput
		wantErrors []string
	}{
		{
			name:  "no changes",
			input: UpdateNamespaceInput{ID: "namespace"},
		},
		{
			name:  "name only",
			input: UpdateNamespaceInput{ID: "namespace", Name: new("Updated")},
		},
		{
			name:  "metadata only",
			input: UpdateNamespaceInput{ID: "namespace", Metadata: new(models.Metadata{"key": "value"})},
		},
		{
			name:  "clear metadata with nil map",
			input: UpdateNamespaceInput{ID: "namespace", Metadata: new(models.Metadata)},
		},
		{
			name:  "clear metadata with empty map",
			input: UpdateNamespaceInput{ID: "namespace", Metadata: new(models.Metadata{})},
		},
		{
			name:       "missing identity",
			wantErrors: []string{"namespace ID is required"},
		},
		{
			name:       "empty name",
			input:      UpdateNamespaceInput{ID: "namespace", Name: new("")},
			wantErrors: []string{"invalid namespace name"},
		},
		{
			name:       "multiple invalid fields",
			input:      UpdateNamespaceInput{ID: " \t", Name: new(" \t")},
			wantErrors: []string{"namespace ID is required", "invalid namespace name"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assertProviderInputValidation(t, tt.input, tt.wantErrors)
		})
	}
}

func TestUpdateChannelInputValidate(t *testing.T) {
	tests := []struct {
		name       string
		modify     func(*UpdateChannelInput[WebhookConfigUpdate])
		wantErrors []string
	}{
		{
			name: "no changes",
		},
		{
			name: "name only",
			modify: func(input *UpdateChannelInput[WebhookConfigUpdate]) {
				input.Name = new("Updated")
			},
		},
		{
			name: "explicit false",
			modify: func(input *UpdateChannelInput[WebhookConfigUpdate]) {
				input.Disabled = new(false)
			},
		},
		{
			name: "clear event types with nil slice",
			modify: func(input *UpdateChannelInput[WebhookConfigUpdate]) {
				input.EventTypes = new([]EventType)
			},
		},
		{
			name: "clear event types with empty slice",
			modify: func(input *UpdateChannelInput[WebhookConfigUpdate]) {
				input.EventTypes = new([]EventType{})
			},
		},
		{
			name: "headers only",
			modify: func(input *UpdateChannelInput[WebhookConfigUpdate]) {
				input.Config.CustomHeaders = new(map[string]string{"x-test": "updated"})
			},
		},
		{
			name: "missing namespace",
			modify: func(input *UpdateChannelInput[WebhookConfigUpdate]) {
				input.Namespace = ""
			},
			wantErrors: []string{"namespace is required"},
		},
		{
			name: "missing channel ID",
			modify: func(input *UpdateChannelInput[WebhookConfigUpdate]) {
				input.ChannelID = ""
			},
			wantErrors: []string{"channel ID is required"},
		},
		{
			name: "empty name",
			modify: func(input *UpdateChannelInput[WebhookConfigUpdate]) {
				input.Name = new("")
			},
			wantErrors: []string{"invalid channel name"},
		},
		{
			name: "invalid configuration",
			modify: func(input *UpdateChannelInput[WebhookConfigUpdate]) {
				input.Config.URL = new("")
				input.Config.SigningSecret = new("")
			},
			wantErrors: []string{"invalid channel configuration", "invalid endpoint URL", "invalid signing secret"},
		},
		{
			name: "multiple invalid fields",
			modify: func(input *UpdateChannelInput[WebhookConfigUpdate]) {
				input.Namespace = " \t"
				input.ChannelID = " \t"
				input.Name = new(" \t")
				input.Config.URL = new("")
			},
			wantErrors: []string{"namespace is required", "channel ID is required", "invalid channel name", "invalid channel configuration"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := UpdateChannelInput[WebhookConfigUpdate]{
				Namespace: "namespace",
				ChannelID: "channel",
			}

			if tt.modify != nil {
				tt.modify(&input)
			}

			assertProviderInputValidation(t, input, tt.wantErrors)
		})
	}
}

func TestGetChannelInputValidate(t *testing.T) {
	tests := []struct {
		name       string
		input      GetChannelInput
		wantErrors []string
	}{
		{
			name: "valid identity without provider data",
			input: GetChannelInput{
				Namespace: "ns-test",
				ChannelID: "channel-test",
			},
		},
		{
			name: "missing namespace",
			input: GetChannelInput{
				ChannelID: "channel-test",
			},
			wantErrors: []string{"namespace is required"},
		},
		{
			name: "missing channel ID",
			input: GetChannelInput{
				Namespace: "ns-test",
			},
			wantErrors: []string{"channel ID is required"},
		},
		{
			name: "whitespace identity",
			input: GetChannelInput{
				Namespace: " \t\n",
				ChannelID: " \t\n",
			},
			wantErrors: []string{"namespace is required", "channel ID is required"},
		},
		{
			name:       "missing identity",
			wantErrors: []string{"namespace is required", "channel ID is required"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assertProviderInputValidation(t, tt.input, tt.wantErrors)
		})
	}
}

func TestProviderChannelInputValidation(t *testing.T) {
	tests := []struct {
		name       string
		modify     func(*CreateChannelInput[WebhookConfig])
		wantErrors []string
	}{
		{
			name: "valid identity and name",
		},
		{
			name: "missing namespace",
			modify: func(input *CreateChannelInput[WebhookConfig]) {
				input.Namespace = ""
			},
			wantErrors: []string{"namespace is required"},
		},
		{
			name: "missing channel ID",
			modify: func(input *CreateChannelInput[WebhookConfig]) {
				input.ChannelID = ""
			},
			wantErrors: []string{"channel ID is required"},
		},
		{
			name: "missing name",
			modify: func(input *CreateChannelInput[WebhookConfig]) {
				input.Name = ""
			},
			wantErrors: []string{"channel name is required"},
		},
		{
			name: "whitespace identity and name",
			modify: func(input *CreateChannelInput[WebhookConfig]) {
				input.Namespace = " \t\n"
				input.ChannelID = " \t\n"
				input.Name = " \t\n"
			},
			wantErrors: []string{
				"namespace is required",
				"channel ID is required",
				"channel name is required",
			},
		},
		{
			name: "multiple invalid fields",
			modify: func(input *CreateChannelInput[WebhookConfig]) {
				*input = CreateChannelInput[WebhookConfig]{}
			},
			wantErrors: []string{
				"namespace is required",
				"channel ID is required",
				"channel name is required",
				"invalid channel configuration",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := CreateChannelInput[WebhookConfig]{
				Namespace: "ns-test",
				ChannelID: "channel-test",
				Name:      "Webhook",
				Config: WebhookConfig{
					URL: "https://example.com/webhooks",
				},
			}

			if tt.modify != nil {
				tt.modify(&input)
			}

			assertProviderInputValidation(t, input, tt.wantErrors)
		})
	}
}

func TestCreateChannelInputIdempotencyKeyValidation(t *testing.T) {
	tests := []struct {
		name           string
		idempotencyKey *string
		wantErrors     []string
	}{
		{
			name: "omitted key",
		},
		{
			name:           "valid key",
			idempotencyKey: new("create-channel"),
		},
		{
			name:           "empty key",
			idempotencyKey: new(""),
			wantErrors:     []string{"idempotency key must not be blank when provided"},
		},
		{
			name:           "whitespace key",
			idempotencyKey: new(" \t\n"),
			wantErrors:     []string{"idempotency key must not be blank when provided"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := CreateChannelInput[WebhookConfig]{
				Namespace: "ns-test",
				ChannelID: "channel-test",
				Name:      "Webhook",
				Config: WebhookConfig{
					URL: "https://example.com/webhooks",
				},
				IdempotencyKey: tt.idempotencyKey,
			}

			assertProviderInputValidation(t, input, tt.wantErrors)
		})
	}
}

func TestSendEventInputValidate(t *testing.T) {
	tests := []struct {
		name       string
		modify     func(*SendEventInput)
		wantErrors []string
	}{
		{
			name: "valid without idempotency key",
		},
		{
			name: "valid with idempotency key",
			modify: func(input *SendEventInput) {
				input.IdempotencyKey = new("send-event")
			},
		},
		{
			name: "multiple distinct destinations",
			modify: func(input *SendEventInput) {
				input.Destinations = append(input.Destinations, EventDestination{ChannelID: "channel-other"})
			},
		},
		{
			name: "missing event namespace",
			modify: func(input *SendEventInput) {
				input.Event.Namespace = ""
			},
			wantErrors: []string{"event namespace is required"},
		},
		{
			name: "missing event ID",
			modify: func(input *SendEventInput) {
				input.Event.ID = ""
			},
			wantErrors: []string{"event ID is required"},
		},
		{
			name: "missing destinations",
			modify: func(input *SendEventInput) {
				input.Destinations = nil
			},
			wantErrors: []string{"at least one destination is required"},
		},
		{
			name: "blank destination channel ID",
			modify: func(input *SendEventInput) {
				input.Destinations[0].ChannelID = " \t\n"
			},
			wantErrors: []string{"destinations[0]: channel ID is required"},
		},
		{
			name: "duplicate destinations",
			modify: func(input *SendEventInput) {
				input.Destinations = append(input.Destinations, input.Destinations[0])
			},
			wantErrors: []string{"destinations[1]: duplicate channel ID"},
		},
		{
			name: "missing event type name",
			modify: func(input *SendEventInput) {
				input.Event.EventType.Name = ""
			},
			wantErrors: []string{"event type must have a name and a positive version"},
		},
		{
			name: "whitespace event type name",
			modify: func(input *SendEventInput) {
				input.Event.EventType.Name = " \t\n"
			},
			wantErrors: []string{"event type must have a name and a positive version"},
		},
		{
			name: "zero event type version",
			modify: func(input *SendEventInput) {
				input.Event.EventType.Version = 0
			},
			wantErrors: []string{"event type must have a name and a positive version"},
		},
		{
			name: "negative event type version",
			modify: func(input *SendEventInput) {
				input.Event.EventType.Version = -1
			},
			wantErrors: []string{"event type must have a name and a positive version"},
		},
		{
			name: "missing creation time",
			modify: func(input *SendEventInput) {
				input.Event.CreatedAt = time.Time{}
			},
			wantErrors: []string{"event creation time is required"},
		},
		{
			name: "missing payload",
			modify: func(input *SendEventInput) {
				input.Event.Payload = nil
			},
			wantErrors: []string{"event payload must be valid JSON"},
		},
		{
			name: "malformed payload",
			modify: func(input *SendEventInput) {
				input.Event.Payload = json.RawMessage(`{"invoice_id":`)
			},
			wantErrors: []string{"event payload must be valid JSON"},
		},
		{
			name: "empty idempotency key",
			modify: func(input *SendEventInput) {
				input.IdempotencyKey = new("")
			},
			wantErrors: []string{"idempotency key must not be blank when provided"},
		},
		{
			name: "whitespace idempotency key",
			modify: func(input *SendEventInput) {
				input.IdempotencyKey = new(" \t\n")
			},
			wantErrors: []string{"idempotency key must not be blank when provided"},
		},
		{
			name: "multiple invalid fields",
			modify: func(input *SendEventInput) {
				*input = SendEventInput{IdempotencyKey: new("")}
			},
			wantErrors: []string{
				"event namespace is required",
				"event ID is required",
				"at least one destination is required",
				"event type must have a name and a positive version",
				"event creation time is required",
				"event payload must be valid JSON",
				"idempotency key must not be blank when provided",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := SendEventInput{
				Event: Event{
					NamespacedID: models.NamespacedID{
						Namespace: "ns-test",
						ID:        "event-test",
					},
					EventType: EventType{
						Name:    "invoice.created",
						Version: 1,
					},
					CreatedAt: time.Date(2026, time.October, 20, 12, 0, 0, 0, time.UTC),
					Payload:   json.RawMessage(`{"invoice_id":"invoice-test"}`),
				},
				Destinations: []EventDestination{
					{ChannelID: "channel-test"},
				},
			}

			if tt.modify != nil {
				tt.modify(&input)
			}

			assertProviderInputValidation(t, input, tt.wantErrors)
		})
	}
}

func TestGetEventInputValidate(t *testing.T) {
	tests := []struct {
		name       string
		modify     func(*GetEventInput)
		wantErrors []string
	}{
		{
			name: "valid identity without payload or event type",
		},
		{
			name: "multiple distinct destinations and generations",
			modify: func(input *GetEventInput) {
				input.Destinations = append(input.Destinations, GetEventDestination{
					ChannelID:          "channel-other",
					DispatchGeneration: 2,
				})
			},
		},
		{
			name: "missing event namespace",
			modify: func(input *GetEventInput) {
				input.Event.Namespace = ""
			},
			wantErrors: []string{"event namespace is required"},
		},
		{
			name: "missing event ID",
			modify: func(input *GetEventInput) {
				input.Event.ID = ""
			},
			wantErrors: []string{"event ID is required"},
		},
		{
			name: "whitespace event identity",
			modify: func(input *GetEventInput) {
				input.Event.Namespace = " \t\n"
				input.Event.ID = " \t\n"
			},
			wantErrors: []string{"event namespace is required", "event ID is required"},
		},
		{
			name: "nil destinations",
			modify: func(input *GetEventInput) {
				input.Destinations = nil
			},
			wantErrors: []string{"at least one destination is required"},
		},
		{
			name: "empty destinations",
			modify: func(input *GetEventInput) {
				input.Destinations = []GetEventDestination{}
			},
			wantErrors: []string{"at least one destination is required"},
		},
		{
			name: "missing channel ID",
			modify: func(input *GetEventInput) {
				input.Destinations[0].ChannelID = ""
			},
			wantErrors: []string{"destinations[0]: channel ID is required"},
		},
		{
			name: "whitespace channel ID",
			modify: func(input *GetEventInput) {
				input.Destinations[0].ChannelID = " \t\n"
			},
			wantErrors: []string{"destinations[0]: channel ID is required"},
		},
		{
			name: "duplicate channel ID across generations",
			modify: func(input *GetEventInput) {
				input.Destinations = append(input.Destinations, GetEventDestination{
					ChannelID:          input.Destinations[0].ChannelID,
					DispatchGeneration: 2,
				})
			},
			wantErrors: []string{"destinations[1]: duplicate channel ID"},
		},
		{
			name: "zero dispatch generation",
			modify: func(input *GetEventInput) {
				input.Destinations[0].DispatchGeneration = 0
			},
			wantErrors: []string{"destinations[0]: dispatch generation must be positive"},
		},
		{
			name: "invalid second destination",
			modify: func(input *GetEventInput) {
				input.Destinations = append(input.Destinations, GetEventDestination{})
			},
			wantErrors: []string{
				"destinations[1]: channel ID is required",
				"destinations[1]: dispatch generation must be positive",
			},
		},
		{
			name: "multiple invalid fields",
			modify: func(input *GetEventInput) {
				*input = GetEventInput{}
			},
			wantErrors: []string{
				"event namespace is required",
				"event ID is required",
				"at least one destination is required",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := GetEventInput{
				Event: Event{
					NamespacedID: models.NamespacedID{
						Namespace: "ns-test",
						ID:        "event-test",
					},
				},
				Destinations: []GetEventDestination{
					{
						ChannelID:          "channel-test",
						DispatchGeneration: 1,
					},
				},
			}

			if tt.modify != nil {
				tt.modify(&input)
			}

			assertProviderInputValidation(t, input, tt.wantErrors)
		})
	}
}

func TestResendEventInputValidate(t *testing.T) {
	tests := []struct {
		name       string
		modify     func(*ResendEventInput)
		wantErrors []string
	}{
		{
			name: "first resend without payload or event type",
		},
		{
			name: "valid with idempotency key",
			modify: func(input *ResendEventInput) {
				input.IdempotencyKey = new("resend-event")
			},
		},
		{
			name: "multiple distinct destinations and generations",
			modify: func(input *ResendEventInput) {
				input.Destinations = append(input.Destinations, GetEventDestination{
					ChannelID:          "channel-other",
					DispatchGeneration: 3,
				})
			},
		},
		{
			name: "initial dispatch generation",
			modify: func(input *ResendEventInput) {
				input.Destinations[0].DispatchGeneration = 1
			},
			wantErrors: []string{"destinations[0]: resend dispatch generation must be at least 2"},
		},
		{
			name: "zero dispatch generation",
			modify: func(input *ResendEventInput) {
				input.Destinations[0].DispatchGeneration = 0
			},
			wantErrors: []string{
				"destinations[0]: dispatch generation must be positive",
				"destinations[0]: resend dispatch generation must be at least 2",
			},
		},
		{
			name: "initial generation on second destination",
			modify: func(input *ResendEventInput) {
				input.Destinations = append(input.Destinations, GetEventDestination{
					ChannelID:          "channel-other",
					DispatchGeneration: 1,
				})
			},
			wantErrors: []string{"destinations[1]: resend dispatch generation must be at least 2"},
		},
		{
			name: "missing event namespace",
			modify: func(input *ResendEventInput) {
				input.Event.Namespace = ""
			},
			wantErrors: []string{"event namespace is required"},
		},
		{
			name: "missing event ID",
			modify: func(input *ResendEventInput) {
				input.Event.ID = ""
			},
			wantErrors: []string{"event ID is required"},
		},
		{
			name: "missing destinations",
			modify: func(input *ResendEventInput) {
				input.Destinations = nil
			},
			wantErrors: []string{"at least one destination is required"},
		},
		{
			name: "missing channel ID",
			modify: func(input *ResendEventInput) {
				input.Destinations[0].ChannelID = ""
			},
			wantErrors: []string{"destinations[0]: channel ID is required"},
		},
		{
			name: "duplicate destinations",
			modify: func(input *ResendEventInput) {
				input.Destinations = append(input.Destinations, input.Destinations[0])
			},
			wantErrors: []string{"destinations[1]: duplicate channel ID"},
		},
		{
			name: "empty idempotency key",
			modify: func(input *ResendEventInput) {
				input.IdempotencyKey = new("")
			},
			wantErrors: []string{"idempotency key must not be blank when provided"},
		},
		{
			name: "whitespace idempotency key",
			modify: func(input *ResendEventInput) {
				input.IdempotencyKey = new(" \t\n")
			},
			wantErrors: []string{"idempotency key must not be blank when provided"},
		},
		{
			name: "multiple invalid fields",
			modify: func(input *ResendEventInput) {
				*input = ResendEventInput{
					Destinations:   []GetEventDestination{{}},
					IdempotencyKey: new(""),
				}
			},
			wantErrors: []string{
				"idempotency key must not be blank when provided",
				"event namespace is required",
				"event ID is required",
				"destinations[0]: channel ID is required",
				"destinations[0]: dispatch generation must be positive",
				"destinations[0]: resend dispatch generation must be at least 2",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := ResendEventInput{
				Event: Event{
					NamespacedID: models.NamespacedID{
						Namespace: "ns-test",
						ID:        "event-test",
					},
				},
				Destinations: []GetEventDestination{
					{
						ChannelID:          "channel-test",
						DispatchGeneration: 2,
					},
				},
			}

			if tt.modify != nil {
				tt.modify(&input)
			}

			assertProviderInputValidation(t, input, tt.wantErrors)
		})
	}
}

func assertProviderInputValidation(t *testing.T, input models.Validator, wantErrors []string) {
	t.Helper()

	err := input.Validate()

	switch {
	case len(wantErrors) > 0:
		require.True(t, models.IsGenericValidationError(err))

		for _, message := range wantErrors {
			require.ErrorContains(t, err, message)
		}

	default:
		require.NoError(t, err)
	}
}

func TestProviderChannelConfigValidation(t *testing.T) {
	tests := []struct {
		name      string
		config    WebhookConfig
		wantError string
	}{
		{
			name: "configuration value",
			config: WebhookConfig{
				URL: "https://example.com/webhooks",
			},
		},
		{
			name:      "zero configuration",
			wantError: "invalid endpoint URL",
		},
		{
			name: "invalid URL",
			config: WebhookConfig{
				URL: "/webhooks",
			},
			wantError: "invalid endpoint URL",
		},
		{
			name: "invalid signing secret",
			config: WebhookConfig{
				URL:           "https://example.com/webhooks",
				SigningSecret: "invalid",
			},
			wantError: "invalid signing secret",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := CreateChannelInput[WebhookConfig]{
				Namespace: "namespace",
				ChannelID: "channel",
				Name:      "Webhook",
				Config:    tt.config,
			}

			var wantErrors []string
			if tt.wantError != "" {
				wantErrors = []string{"invalid channel configuration", tt.wantError}
			}

			assertProviderInputValidation(t, input, wantErrors)
		})
	}
}

func TestProviderNamespaceValidation(t *testing.T) {
	tests := []struct {
		name       string
		namespace  ProviderNamespace
		wantErrors []string
	}{
		{
			name: "valid namespace",
			namespace: ProviderNamespace{
				ID:   "ns-test",
				Name: "Namespace",
			},
		},
		{
			name: "missing ID and name",
			wantErrors: []string{
				"namespace ID is required",
				"namespace name is required",
			},
		},
		{
			name: "missing ID",
			namespace: ProviderNamespace{
				Name: "Namespace",
			},
			wantErrors: []string{"namespace ID is required"},
		},
		{
			name: "missing name",
			namespace: ProviderNamespace{
				ID: "ns-test",
			},
			wantErrors: []string{"namespace name is required"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.namespace.Validate()

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

func TestProviderError(t *testing.T) {
	tests := []struct {
		name        string
		providerErr *ProviderError
		wantError   string
	}{
		{
			name: "with cause",
			providerErr: &ProviderError{
				Kind: ProviderErrorKindRetryable,
				Code: ProviderErrorCodeUnavailable,
				Err:  errors.New("provider request failed"),
			},
			wantError: "provider_unavailable: provider request failed",
		},
		{
			name: "without cause",
			providerErr: &ProviderError{
				Code: ProviderErrorCodeUnavailable,
			},
			wantError: "provider_unavailable",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wrapped := fmt.Errorf("namespace operation: %w", tt.providerErr)

			var got *ProviderError

			require.ErrorAs(t, wrapped, &got)
			require.Same(t, tt.providerErr, got)
			require.EqualError(t, tt.providerErr, tt.wantError)
			require.Equal(t, tt.providerErr.Err, errors.Unwrap(tt.providerErr))

			if tt.providerErr.Err != nil {
				require.ErrorIs(t, wrapped, tt.providerErr.Err)
			}
		})
	}
}
