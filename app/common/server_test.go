package common

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/app/config"
)

// resolveClientIP drives a request through the middleware built from cfg and
// returns the client IP chi resolved for it.
func resolveClientIP(t *testing.T, cfg config.ClientIPMiddlewareConfig, mutate func(r *http.Request)) string {
	t.Helper()

	mw, err := NewClientIPMiddleware(cfg)
	require.NoError(t, err)

	var got string

	h := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = middleware.GetClientIP(r.Context())
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	mutate(req)
	h.ServeHTTP(httptest.NewRecorder(), req)

	return got
}

func TestNewClientIPMiddleware(t *testing.T) {
	t.Run("invalid config returns error", func(t *testing.T) {
		_, err := NewClientIPMiddleware(config.ClientIPMiddlewareConfig{Source: "bogus"})
		require.ErrorContains(t, err, "invalid client ip middleware config")
	})

	t.Run("remote address source resolves the socket peer", func(t *testing.T) {
		got := resolveClientIP(t, config.ClientIPMiddlewareConfig{
			Source: config.ClientIPSourceRemoteAddr,
		}, func(r *http.Request) {
			r.RemoteAddr = "203.0.113.7:1234"
			r.Header.Set("X-Forwarded-For", "198.51.100.9") // must be ignored
		})

		require.Equal(t, "203.0.113.7", got)
	})

	t.Run("header source resolves the trusted header", func(t *testing.T) {
		got := resolveClientIP(t, config.ClientIPMiddlewareConfig{
			Source: config.ClientIPSourceHeader,
			Header: "X-Real-IP",
		}, func(r *http.Request) {
			r.RemoteAddr = "10.0.0.1:1234"
			r.Header.Set("X-Real-IP", "203.0.113.7")
		})

		require.Equal(t, "203.0.113.7", got)
	})

	t.Run("xff source with trusted prefixes skips trusted hops", func(t *testing.T) {
		got := resolveClientIP(t, config.ClientIPMiddlewareConfig{
			Source:            config.ClientIPSourceXFF,
			TrustedIPPrefixes: []string{"10.0.0.0/8"},
		}, func(r *http.Request) {
			r.Header.Set("X-Forwarded-For", "203.0.113.7, 10.0.0.1")
		})

		require.Equal(t, "203.0.113.7", got)
	})

	t.Run("xff source prefers trusted prefixes over trusted proxies when both are set", func(t *testing.T) {
		// With the prefixes branch the trusted 10.0.0.1 hop is skipped and the
		// client is 203.0.113.7; the trusted-proxies=3 branch would resolve no IP
		// at all for this two-entry chain. Pins the silent precedence.
		got := resolveClientIP(t, config.ClientIPMiddlewareConfig{
			Source:            config.ClientIPSourceXFF,
			TrustedIPPrefixes: []string{"10.0.0.0/8"},
			TrustedProxies:    3,
		}, func(r *http.Request) {
			r.Header.Set("X-Forwarded-For", "203.0.113.7, 10.0.0.1")
		})

		require.Equal(t, "203.0.113.7", got)
	})

	t.Run("xff source with trusted proxies resolves the proxy-added entry", func(t *testing.T) {
		got := resolveClientIP(t, config.ClientIPMiddlewareConfig{
			Source:         config.ClientIPSourceXFF,
			TrustedProxies: 1,
		}, func(r *http.Request) {
			r.Header.Set("X-Forwarded-For", "203.0.113.7")
		})

		require.Equal(t, "203.0.113.7", got)
	})
}
