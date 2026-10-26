package noop_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	v4 "github.com/openmeterio/openmeter/openmeter/notification/v4"
	"github.com/openmeterio/openmeter/openmeter/notification/v4/providers/noop"
	"github.com/openmeterio/openmeter/pkg/models"
)

func TestNewWebhookProvider(t *testing.T) {
	tests := []struct {
		name      string
		logger    *slog.Logger
		wantError bool
	}{
		{
			name:   "configured logger",
			logger: slog.New(slog.DiscardHandler),
		},
		{
			name:      "missing logger",
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider, err := noop.NewWebhookProvider(tt.logger)

			switch {
			case tt.wantError:
				require.Nil(t, provider)
				require.ErrorContains(t, err, "invalid logger")
				require.True(t, models.IsGenericValidationError(err))

			default:
				require.NoError(t, err)
				require.NotNil(t, provider)
				require.Equal(t, v4.ProviderRoutingModeExplicit, provider.RoutingMode())

				registry, err := v4.NewProviderRegistry(v4.ProviderRegistryConfig{
					Logger:  tt.logger,
					Webhook: provider,
				})
				require.NoError(t, err)
				require.Same(t, provider, registry.Webhook())
			}
		})
	}
}

func TestWebhookProviderOperations(t *testing.T) {
	provider, err := noop.NewWebhookProvider(slog.New(slog.DiscardHandler))
	require.NoError(t, err)

	tests := []struct {
		name         string
		invalid      bool
		contextError error
	}{
		{
			name: "disabled provider",
		},
		{
			name:    "invalid input",
			invalid: true,
		},
		{
			name:         "canceled context",
			contextError: context.Canceled,
		},
		{
			name:         "expired deadline",
			contextError: context.DeadlineExceeded,
		},
		{
			name:         "cancellation takes precedence over validation",
			invalid:      true,
			contextError: context.Canceled,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Given a shared stateless provider and valid or deliberately empty inputs.
			var namespace v4.ProviderNamespace
			var channel v4.CreateChannelInput[v4.WebhookConfig]
			var event v4.Event
			var destinations []v4.EventDestination
			var observations []v4.GetEventDestination

			if !tt.invalid {
				namespace = v4.ProviderNamespace{ID: "namespace", Name: "Test"}
				channel = v4.CreateChannelInput[v4.WebhookConfig]{
					Namespace: namespace.ID,
					ChannelID: "channel",
					Name:      "Webhook",
					Config:    v4.WebhookConfig{URL: "https://example.com/webhook"},
				}
				event = v4.Event{
					NamespacedID: models.NamespacedID{Namespace: namespace.ID, ID: "event"},
					EventType:    v4.EventType{Name: "invoice.created", Version: 1},
					CreatedAt:    time.Now().UTC(),
					Payload:      json.RawMessage(`{"invoice_id":"invoice"}`),
				}
				destinations = []v4.EventDestination{{ChannelID: channel.ChannelID}}
				observations = []v4.GetEventDestination{{ChannelID: channel.ChannelID, DispatchGeneration: 2}}
			}

			operations := []struct {
				name string
				run  func(context.Context) (any, error)
			}{
				{
					name: "get namespace",
					run: func(ctx context.Context) (any, error) {
						return provider.GetNamespace(ctx, v4.GetNamespaceInput{ID: namespace.ID})
					},
				},
				{
					name: "create namespace",
					run: func(ctx context.Context) (any, error) {
						return provider.CreateNamespace(ctx, v4.CreateNamespaceInput{ProviderNamespace: namespace})
					},
				},
				{
					name: "update namespace",
					run: func(ctx context.Context) (any, error) {
						return provider.UpdateNamespace(ctx, v4.UpdateNamespaceInput{ID: namespace.ID})
					},
				},
				{
					name: "delete namespace",
					run: func(ctx context.Context) (any, error) {
						return nil, provider.DeleteNamespace(ctx, v4.DeleteNamespaceInput{ID: namespace.ID})
					},
				},
				{
					name: "get channel",
					run: func(ctx context.Context) (any, error) {
						return provider.GetChannel(ctx, v4.GetChannelInput{Namespace: channel.Namespace, ChannelID: channel.ChannelID})
					},
				},
				{
					name: "create channel",
					run: func(ctx context.Context) (any, error) {
						return provider.CreateChannel(ctx, channel)
					},
				},
				{
					name: "update channel",
					run: func(ctx context.Context) (any, error) {
						return provider.UpdateChannel(ctx, v4.UpdateChannelInput[v4.WebhookConfigUpdate]{
							Namespace: channel.Namespace,
							ChannelID: channel.ChannelID,
						})
					},
				},
				{
					name: "delete channel",
					run: func(ctx context.Context) (any, error) {
						return nil, provider.DeleteChannel(ctx, v4.DeleteChannelInput{Namespace: channel.Namespace, ChannelID: channel.ChannelID})
					},
				},
				{
					name: "send event",
					run: func(ctx context.Context) (any, error) {
						return provider.SendEvent(ctx, v4.SendEventInput{Event: event, Destinations: destinations})
					},
				},
				{
					name: "get event",
					run: func(ctx context.Context) (any, error) {
						return provider.GetEvent(ctx, v4.GetEventInput{Event: event, Destinations: observations})
					},
				},
				{
					name: "resend event",
					run: func(ctx context.Context) (any, error) {
						return provider.ResendEvent(ctx, v4.ResendEventInput{Event: event, Destinations: observations})
					},
				},
			}

			for _, operation := range operations {
				t.Run(operation.name, func(t *testing.T) {
					ctx := t.Context()

					switch tt.contextError {
					case context.Canceled:
						var cancel context.CancelFunc

						ctx, cancel = context.WithCancel(ctx)
						cancel()

					case context.DeadlineExceeded:
						var cancel context.CancelFunc

						ctx, cancel = context.WithDeadline(ctx, time.Now().Add(-time.Second))
						defer cancel()
					}

					// When any provider operation is invoked.
					result, err := operation.run(ctx)

					// Then no result can be mistaken for acceptance or convergence.
					require.Nil(t, result)
					require.Error(t, err)
					require.ErrorContains(t, err, operation.name)

					var providerErr *v4.ProviderError

					switch {
					case tt.contextError != nil:
						require.ErrorIs(t, err, tt.contextError)
						require.False(t, errors.As(err, &providerErr))

					case tt.invalid:
						require.True(t, models.IsGenericValidationError(err))
						require.False(t, errors.As(err, &providerErr))

					default:
						require.ErrorAs(t, err, &providerErr)
						require.Equal(t, v4.ProviderErrorKindConfiguration, providerErr.Kind)
						require.Equal(t, v4.ProviderErrorCodeConfigurationInvalid, providerErr.Code)
						require.Nil(t, providerErr.RetryAfter)
						require.True(t, models.IsGenericNotImplementedError(err))
					}
				})
			}
		})
	}
}

func TestWebhookProviderDoesNotLogInputs(t *testing.T) {
	tests := []struct {
		name string
		run  func(context.Context, *noop.WebhookProvider) error
	}{
		{
			name: "channel configuration",
			run: func(ctx context.Context, provider *noop.WebhookProvider) error {
				_, err := provider.CreateChannel(ctx, v4.CreateChannelInput[v4.WebhookConfig]{
					Namespace: "namespace",
					ChannelID: "channel",
					Name:      "sensitive-name",
					Config: v4.WebhookConfig{
						URL:           "https://example.com/sensitive-url",
						CustomHeaders: map[string]string{"Authorization": "sensitive-token"},
					},
				})

				return err
			},
		},
		{
			name: "event payload",
			run: func(ctx context.Context, provider *noop.WebhookProvider) error {
				_, err := provider.SendEvent(ctx, v4.SendEventInput{
					Event: v4.Event{
						NamespacedID: models.NamespacedID{Namespace: "namespace", ID: "event"},
						EventType:    v4.EventType{Name: "invoice.created", Version: 1},
						CreatedAt:    time.Now().UTC(),
						Payload:      json.RawMessage(`{"value":"sensitive-payload"}`),
					},
					Destinations: []v4.EventDestination{{ChannelID: "channel"}},
				})

				return err
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var logs bytes.Buffer

			provider, err := noop.NewWebhookProvider(slog.New(slog.NewJSONHandler(&logs, &slog.HandlerOptions{Level: slog.LevelDebug})))
			require.NoError(t, err)

			err = tt.run(t.Context(), provider)
			require.True(t, models.IsGenericNotImplementedError(err))
			require.Contains(t, logs.String(), "webhook provider is disabled")
			require.NotContains(t, logs.String(), "sensitive-")
			require.NotContains(t, err.Error(), "sensitive-")
		})
	}
}
