package server_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/peterbourgon/ctxdata/v4"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/pkg/contextx"
	"github.com/openmeterio/openmeter/pkg/server"
)

func TestGetRequestAttributes(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "https://example.com/items/item%3A123?tag=a%2Fb&tag=c", nil).WithContext(t.Context())
	req.RemoteAddr = "[2001:db8::1]:1234"
	req.Header.Set("User-Agent", "test-client")
	req.Header.Set("X-Request-ID", "test-request-id")

	handler := middleware.RequestID(middleware.ClientIPFromRemoteAddr(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, map[string]string{
			"http.request.method":   "GET",
			"url.full":              "https://example.com/items/item%3A123?tag=a%2Fb&tag=c",
			"url.path":              "/items/item:123",
			"url.query":             "tag=a%2Fb&tag=c",
			"url.template":          "/items/item:123",
			"network.protocol.name": "HTTP/1.1",
			"network.peer.address":  "2001:db8::1",
			"network.peer.port":     "1234",
			"client.address":        "2001:db8::1",
			"user_agent.original":   "test-client",
			"req_id":                "test-request-id",
			"request_id":            "test-request-id",
		}, server.GetRequestAttributes(r))
		w.WriteHeader(http.StatusNoContent)
	})))
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, req)

	require.Equal(t, http.StatusNoContent, response.Code)
}

func TestGetRequestAttributesWithoutMiddleware(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil).WithContext(t.Context())
	req.RemoteAddr = "invalid-address"

	attrs := server.GetRequestAttributes(req)

	require.Contains(t, attrs, "request_id")
	require.Empty(t, attrs["request_id"])
	require.NotContains(t, attrs, "req_id")
	require.NotContains(t, attrs, "client.address")
	require.NotContains(t, attrs, "network.peer.address")
	require.NotContains(t, attrs, "network.peer.port")
	require.Contains(t, attrs, "url.query")
	require.Empty(t, attrs["url.query"])
}

func TestNewRequestAttributesMiddleware(t *testing.T) {
	for _, tc := range []struct {
		name  string
		attrs map[string]string
	}{
		{name: "new context"},
		{
			name:  "existing attributes",
			attrs: map[string]string{"existing": "value", "request_id": "old-id"},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			// Given request identity middleware and an endpoint with a path parameter.
			router := chi.NewRouter()
			router.Use(middleware.RequestID)
			router.Use(middleware.ClientIPFromRemoteAddr)
			router.Use(server.NewRequestAttributesMiddleware())
			router.Get("/items/{itemID}", func(w http.ResponseWriter, r *http.Request) {
				attrs := ctxdata.From(r.Context()).GetAllMap()
				require.Equal(t, "test-request-id", attrs["request_id"])
				require.Equal(t, "test-request-id", attrs["req_id"])
				require.Equal(t, "192.0.2.1", attrs["client.address"])
				require.Equal(t, "/items/123", attrs["url.path"])
				require.Equal(t, "limit=10", attrs["url.query"])
				// Attribute collection precedes endpoint matching, so the template is the path for now.
				require.Equal(t, "/items/123", attrs["url.template"])
				require.Equal(t, "/items/{itemID}", chi.RouteContext(r.Context()).RoutePattern())
				if tc.attrs != nil {
					require.Equal(t, "value", attrs["existing"])
				}

				w.WriteHeader(http.StatusUnauthorized)
			})
			req := httptest.NewRequest(http.MethodGet, "/items/123?limit=10", nil).WithContext(t.Context())
			req.RemoteAddr = "192.0.2.1:1234"
			req.Header.Set("X-Request-ID", "test-request-id")
			if tc.attrs != nil {
				req = req.WithContext(contextx.WithAttrs(req.Context(), tc.attrs))
			}

			// When a downstream handler rejects the request.
			response := httptest.NewRecorder()
			router.ServeHTTP(response, req)

			// Then attributes were available to the handler and its response is preserved.
			require.Equal(t, http.StatusUnauthorized, response.Code)
		})
	}
}
