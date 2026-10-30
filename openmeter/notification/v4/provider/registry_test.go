package provider_test

import (
	"log/slog"
	"testing"

	"github.com/stretchr/testify/require"

	notificationprovider "github.com/openmeterio/openmeter/openmeter/notification/v4/provider"
	"github.com/openmeterio/openmeter/openmeter/notification/v4/provider/noop"
	"github.com/openmeterio/openmeter/pkg/models"
)

func TestNewRegistry(t *testing.T) {
	logger := slog.New(slog.DiscardHandler)

	provider, err := noop.NewWebhookProvider(logger)
	require.NoError(t, err)

	tests := []struct {
		name       string
		config     notificationprovider.RegistryConfig
		wantErrors []string
	}{
		{
			name: "configured webhook provider",
			config: notificationprovider.RegistryConfig{
				Logger:  logger,
				Webhook: provider,
			},
		},
		{
			name: "missing webhook provider",
			config: notificationprovider.RegistryConfig{
				Logger: logger,
			},
			wantErrors: []string{"webhook provider is required"},
		},
		{
			name: "typed nil webhook provider",
			config: notificationprovider.RegistryConfig{
				Logger:  logger,
				Webhook: (*noop.WebhookProvider)(nil),
			},
			wantErrors: []string{"webhook provider is required"},
		},
		{
			name: "typed nil webhook provider and missing logger",
			config: notificationprovider.RegistryConfig{
				Webhook: (*noop.WebhookProvider)(nil),
			},
			wantErrors: []string{
				"logger is required",
				"webhook provider is required",
			},
		},
		{
			name: "missing logger",
			config: notificationprovider.RegistryConfig{
				Webhook: provider,
			},
			wantErrors: []string{"logger is required"},
		},
		{
			name: "all dependencies missing",
			wantErrors: []string{
				"logger is required",
				"webhook provider is required",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given a registry assembled by application composition.
			config := tt.config

			// When constructing the registry.
			registry, err := notificationprovider.NewRegistry(config)

			// Then invalid wiring fails before a registry is returned.
			switch {
			case len(tt.wantErrors) > 0:
				require.ErrorContains(t, err, "invalid provider registry configuration")
				require.True(t, models.IsGenericValidationError(err))
				require.Nil(t, registry)

				for _, message := range tt.wantErrors {
					require.ErrorContains(t, err, message)
				}

			default:
				require.NoError(t, err)
				require.NotNil(t, registry)
				require.Same(t, provider, registry.Webhook())
				require.Equal(t, notificationprovider.ProviderRoutingModeExplicit, registry.Webhook().RoutingMode())

				// Changing construction inputs must not replace stored dependencies.
				config.Webhook = nil
				config.Logger = nil

				require.Same(t, provider, registry.Webhook())
			}
		})
	}
}
