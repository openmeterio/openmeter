package config

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestClientIPMiddlewareConfigValidate(t *testing.T) {
	tests := []struct {
		name    string
		config  ClientIPMiddlewareConfig
		wantErr string
	}{
		{
			name:    "empty source",
			config:  ClientIPMiddlewareConfig{},
			wantErr: "invalid client IP source",
		},
		{
			name:    "unknown source",
			config:  ClientIPMiddlewareConfig{Source: "bogus"},
			wantErr: "invalid client IP source",
		},
		{
			name:   "remote address source",
			config: ClientIPMiddlewareConfig{Source: ClientIPSourceRemoteAddr},
		},
		{
			name:    "header source without header",
			config:  ClientIPMiddlewareConfig{Source: ClientIPSourceHeader},
			wantErr: "missing client IP header",
		},
		{
			name:   "header source with overwrite-style header",
			config: ClientIPMiddlewareConfig{Source: ClientIPSourceHeader, Header: "X-Real-IP"},
		},
		{
			name:    "header source with X-Forwarded-For",
			config:  ClientIPMiddlewareConfig{Source: ClientIPSourceHeader, Header: "X-Forwarded-For"},
			wantErr: "X-Forwarded-For cannot be used as client IP header",
		},
		{
			name:    "header source with non-canonical X-Forwarded-For",
			config:  ClientIPMiddlewareConfig{Source: ClientIPSourceHeader, Header: "x-forwarded-for"},
			wantErr: "X-Forwarded-For cannot be used as client IP header",
		},
		{
			name:   "xff source with valid prefixes",
			config: ClientIPMiddlewareConfig{Source: ClientIPSourceXFF, TrustedIPPrefixes: []string{"10.0.0.0/8", "2600:9000::/28"}},
		},
		{
			name:    "xff source with invalid prefix",
			config:  ClientIPMiddlewareConfig{Source: ClientIPSourceXFF, TrustedIPPrefixes: []string{"not-a-cidr"}},
			wantErr: "invalid trusted IP prefixes",
		},
		{
			// net.ParseCIDR accepts this but chi's netip.MustParsePrefix panics on it.
			name:    "xff source with leading zero prefix bits",
			config:  ClientIPMiddlewareConfig{Source: ClientIPSourceXFF, TrustedIPPrefixes: []string{"10.0.0.0/08"}},
			wantErr: "invalid trusted IP prefixes",
		},
		{
			name:   "xff source with trusted proxies",
			config: ClientIPMiddlewareConfig{Source: ClientIPSourceXFF, TrustedProxies: 2},
		},
		{
			name:    "xff source with neither prefixes nor proxies",
			config:  ClientIPMiddlewareConfig{Source: ClientIPSourceXFF},
			wantErr: "either trusted IP prefixes or a positive number of trusted proxies",
		},
		{
			// chi's ClientIPFromXFFTrustedProxies panics if the count is < 1.
			name:    "xff source with negative trusted proxies",
			config:  ClientIPMiddlewareConfig{Source: ClientIPSourceXFF, TrustedProxies: -1},
			wantErr: "either trusted IP prefixes or a positive number of trusted proxies",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if tt.wantErr == "" {
				require.NoError(t, err)
				return
			}

			require.ErrorContains(t, err, tt.wantErr)
		})
	}
}
