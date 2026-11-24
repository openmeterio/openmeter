package config

import (
	"testing"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"
)

func validClickHouseConfig() ClickHouseAggregationConfiguration {
	return ClickHouseAggregationConfiguration{
		Address:         "127.0.0.1:9000",
		Protocol:        ClickHouseProtocolNative,
		Database:        "openmeter",
		DialTimeout:     10 * time.Second,
		MaxOpenConns:    5,
		MaxIdleConns:    5,
		ConnMaxLifetime: 10 * time.Minute,
		BlockBufferSize: 10,
	}
}

func TestClickHouseAggregationConfigurationValidateProtocol(t *testing.T) {
	tests := []struct {
		name     string
		protocol string
		wantErr  string
	}{
		{name: "native", protocol: ClickHouseProtocolNative},
		{name: "http", protocol: ClickHouseProtocolHTTP},
		{name: "empty", protocol: "", wantErr: "protocol must be"},
		{name: "unknown", protocol: "grpc", wantErr: `got "grpc"`},
		{name: "case sensitive", protocol: "HTTP", wantErr: "protocol must be"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := validClickHouseConfig()
			c.Protocol = tt.protocol

			err := c.Validate()
			if tt.wantErr == "" {
				require.NoError(t, err)
				return
			}

			require.ErrorContains(t, err, tt.wantErr)
		})
	}
}

func TestClickHouseAggregationConfigurationGetClientOptionsProtocol(t *testing.T) {
	t.Run("native is the driver default", func(t *testing.T) {
		c := validClickHouseConfig()

		require.Equal(t, clickhouse.Native, c.GetClientOptions().Protocol)
	})

	t.Run("http selects the HTTP protocol", func(t *testing.T) {
		c := validClickHouseConfig()
		c.Protocol = ClickHouseProtocolHTTP
		c.Address = "chproxy.example.com:443"
		c.TLS = true

		opts := c.GetClientOptions()

		require.Equal(t, clickhouse.HTTP, opts.Protocol)
		require.Equal(t, []string{"chproxy.example.com:443"}, opts.Addr)
		require.NotNil(t, opts.TLS)
		require.Equal(t, "openmeter", opts.Auth.Database)
	})

	t.Run("http without TLS has no TLS config", func(t *testing.T) {
		c := validClickHouseConfig()
		c.Protocol = ClickHouseProtocolHTTP

		require.Nil(t, c.GetClientOptions().TLS)
	})
}

func TestConfigureAggregationDefaultsToNativeProtocol(t *testing.T) {
	v := viper.New()
	ConfigureAggregation(v)

	require.Equal(t, ClickHouseProtocolNative, v.GetString("aggregation.clickhouse.protocol"))
}
