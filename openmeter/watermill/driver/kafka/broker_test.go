package kafka

import (
	"crypto/tls"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/metric/noop"

	"github.com/openmeterio/openmeter/app/config"
)

func TestCreateKafkaConfigSecurityProtocol(t *testing.T) {
	tests := []struct {
		name        string
		protocol    string
		wantTLS     bool
		wantSASL    bool
		wantMinTLS  uint16
		wantNoSkipV bool
	}{
		{name: "plaintext", protocol: ""},
		{name: "ssl", protocol: "SSL", wantTLS: true, wantMinTLS: tls.VersionTLS12, wantNoSkipV: true},
		{name: "sasl ssl", protocol: "SASL_SSL", wantTLS: true, wantSASL: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := BrokerOptions{
				KafkaConfig: config.KafkaConfiguration{
					Broker:           "localhost:9092",
					SecurityProtocol: tt.protocol,
					SaslMechanisms:   "PLAIN",
					SaslUsername:     "user",
					SaslPassword:     "pass",
				},
				ClientID:    "test",
				Logger:      slog.New(slog.DiscardHandler),
				MetricMeter: noop.NewMeterProvider().Meter("test"),
			}

			cfg, err := opts.createKafkaConfig("publisher")
			require.NoError(t, err)

			require.Equal(t, tt.wantTLS, cfg.Net.TLS.Enable)
			require.Equal(t, tt.wantSASL, cfg.Net.SASL.Enable)

			if tt.wantMinTLS != 0 {
				require.NotNil(t, cfg.Net.TLS.Config)
				require.Equal(t, tt.wantMinTLS, cfg.Net.TLS.Config.MinVersion)
				require.False(t, cfg.Net.TLS.Config.InsecureSkipVerify)
			}
		})
	}
}
