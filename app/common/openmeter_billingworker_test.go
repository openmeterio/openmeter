package common

import (
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/app/config"
	watermillkafka "github.com/openmeterio/openmeter/openmeter/watermill/driver/kafka"
)

func TestWorkerSubscriptionEndpointDoesNotChangeHealth(t *testing.T) {
	tests := []struct {
		name     string
		register func(context.Context, TelemetryServer, *watermillkafka.SubscriptionStatus)
	}{
		{
			name: "billing",
			register: func(ctx context.Context, server TelemetryServer, status *watermillkafka.SubscriptionStatus) {
				BillingWorkerGroup(ctx, nil, nil, server, status)
			},
		},
		{
			name: "balance",
			register: func(ctx context.Context, server TelemetryServer, status *watermillkafka.SubscriptionStatus) {
				BalanceWorkerGroup(ctx, nil, server, status)
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// Given the normal telemetry endpoints and a subscriber that has not initialized.
			logger := slog.New(slog.DiscardHandler)
			original := NewTelemetryHandler(config.MetricsTelemetryConfig{}, NewHealthChecker(logger), logger)
			server, cleanup := NewTelemetryServer(config.TelemetryConfig{}, original)
			defer cleanup()
			test.register(t.Context(), server, watermillkafka.NewSubscriptionStatus([]string{"events"}))

			// When subscription readiness is failing, Kubernetes and legacy health stay healthy.
			for _, path := range []string{"/healthz", "/healthz/live", "/healthz/ready"} {
				t.Run(path, func(t *testing.T) {
					before := httptest.NewRecorder()
					original.ServeHTTP(before, httptest.NewRequest(http.MethodGet, path, nil))
					after := httptest.NewRecorder()
					server.Handler.ServeHTTP(after, httptest.NewRequest(http.MethodGet, path, nil))
					require.Equal(t, http.StatusOK, after.Code)
					require.Equal(t, before.Code, after.Code)
					require.Equal(t, before.Body.String(), after.Body.String())
				})
			}

			// Then the separate endpoint reports unready without changing the health registry.
			response := httptest.NewRecorder()
			server.Handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/healthz/subscriptions", nil))
			require.Equal(t, http.StatusServiceUnavailable, response.Code)
			require.JSONEq(t, `{"ready":false,"assigned":{},"initialized":{}}`, response.Body.String())
		})
	}
}
