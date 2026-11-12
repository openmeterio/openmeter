package server

import (
	"net"
	"net/http"

	"github.com/go-chi/chi/v5/middleware"
	semconv "go.opentelemetry.io/otel/semconv/v1.27.0"

	"github.com/openmeterio/openmeter/pkg/contextx"
)

// NewRequestAttributesMiddleware adds request attributes to the context.
// Use it after middleware.RequestID and the client IP middleware so their values
// are available to downstream handlers and loggers.
func NewRequestAttributesMiddleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := contextx.WithAttrs(r.Context(), GetRequestAttributes(r))
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// GetRequestAttributes returns a map of attributes for a request.
// Requires `middleware.RequestID` middleware to be used.
func GetRequestAttributes(r *http.Request) map[string]string {
	ctx := r.Context()

	attrs := map[string]string{
		// HTTP attributes
		string(semconv.HTTPRequestMethodKey): r.Method,
		string(semconv.URLFullKey):           r.URL.String(),
		string(semconv.URLPathKey):           r.URL.Path,
		string(semconv.URLQueryKey):          r.URL.RawQuery,
		// TODO: Match the route template; endpoint matching happens after this middleware.
		string(semconv.URLTemplateKey): r.URL.Path,

		"request_id": middleware.GetReqID(ctx),

		// Net attributes
		string(semconv.NetworkProtocolNameKey): r.Proto,

		// User attributes
		string(semconv.UserAgentOriginalKey): r.UserAgent(),
	}

	peerAddr, peerPort, err := net.SplitHostPort(r.RemoteAddr)
	if err == nil {
		attrs[string(semconv.NetworkPeerAddressKey)] = peerAddr
		attrs[string(semconv.NetworkPeerPortKey)] = peerPort
	}

	if clientAddr := middleware.GetClientIPAddr(ctx); clientAddr.IsValid() {
		attrs[string(semconv.ClientAddressKey)] = clientAddr.String()
	}

	if reqID := middleware.GetReqID(ctx); reqID != "" {
		// Preserve the legacy key for existing log consumers.
		attrs["req_id"] = reqID
	}

	return attrs
}
