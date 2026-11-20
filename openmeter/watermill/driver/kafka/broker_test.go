package kafka

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"log/slog"
	"math/big"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/IBM/sarama"
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

func TestCreateKafkaConfigProtocolVersion(t *testing.T) {
	for _, role := range []string{"publisher", "subscriber"} {
		t.Run(role, func(t *testing.T) {
			opts := BrokerOptions{
				KafkaConfig: config.KafkaConfiguration{
					Broker: "localhost:9092",
				},
				ClientID:    "test",
				Logger:      slog.New(slog.DiscardHandler),
				MetricMeter: noop.NewMeterProvider().Meter("test"),
			}

			cfg, err := opts.createKafkaConfig(role)
			require.NoError(t, err)
			require.Equal(t, sarama.V2_6_2_0, cfg.Version)
			require.False(t, cfg.ApiVersionsRequest)
		})
	}
}

func newCAFile(t *testing.T) (path string, cert *x509.Certificate) {
	t.Helper()

	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)

	tmpl := &x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{CommonName: "unit-test-ca"},
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              time.Now().Add(time.Hour),
		IsCA:                  true,
		BasicConstraintsValid: true,
		KeyUsage:              x509.KeyUsageCertSign,
	}

	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	require.NoError(t, err)

	cert, err = x509.ParseCertificate(der)
	require.NoError(t, err)

	path = filepath.Join(t.TempDir(), "ca.pem")
	require.NoError(t, os.WriteFile(path, pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der}), 0o600))

	return path, cert
}

func TestCreateKafkaConfigSslCaLocation(t *testing.T) {
	caPath, caCert := newCAFile(t)

	notPEM := filepath.Join(t.TempDir(), "garbage.pem")
	require.NoError(t, os.WriteFile(notPEM, []byte("this is not a certificate"), 0o600))

	build := func(protocol, caLocation string) (*sarama.Config, error) {
		opts := BrokerOptions{
			KafkaConfig: config.KafkaConfiguration{
				Broker:           "localhost:9092",
				SecurityProtocol: protocol,
				SaslMechanisms:   "PLAIN",
				SaslUsername:     "user",
				SaslPassword:     "pass",
				SslCaLocation:    caLocation,
			},
			ClientID:    "test",
			Logger:      slog.New(slog.DiscardHandler),
			MetricMeter: noop.NewMeterProvider().Meter("test"),
		}

		return opts.createKafkaConfig("publisher")
	}

	for _, protocol := range []string{"SSL", "SASL_SSL"} {
		t.Run(protocol+" with CA", func(t *testing.T) {
			cfg, err := build(protocol, caPath)
			require.NoError(t, err)

			require.True(t, cfg.Net.TLS.Enable)
			require.NotNil(t, cfg.Net.TLS.Config)
			require.False(t, cfg.Net.TLS.Config.InsecureSkipVerify)
			require.NotNil(t, cfg.Net.TLS.Config.RootCAs)

			// The pool contains exactly the configured CA (it replaces the system roots).
			subjects := cfg.Net.TLS.Config.RootCAs.Subjects()
			require.Equal(t, [][]byte{caCert.RawSubject}, subjects)
		})

		t.Run(protocol+" without CA", func(t *testing.T) {
			cfg, err := build(protocol, "")
			require.NoError(t, err)
			require.NotNil(t, cfg.Net.TLS.Config)
			require.Nil(t, cfg.Net.TLS.Config.RootCAs)
		})

		t.Run(protocol+" missing CA file", func(t *testing.T) {
			_, err := build(protocol, filepath.Join(t.TempDir(), "missing.pem"))
			require.ErrorContains(t, err, "failed to read kafka ssl ca location")
		})

		t.Run(protocol+" CA file without certificates", func(t *testing.T) {
			_, err := build(protocol, notPEM)
			require.ErrorContains(t, err, "does not contain any valid PEM certificates")
		})
	}

	t.Run("plaintext ignores CA", func(t *testing.T) {
		cfg, err := build("", caPath)
		require.NoError(t, err)
		require.False(t, cfg.Net.TLS.Enable)
		require.Nil(t, cfg.Net.TLS.Config)
	})
}
