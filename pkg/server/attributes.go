package server

import (
	"net/http"

	"github.com/go-chi/chi/v5/middleware"
	semconv "go.opentelemetry.io/otel/semconv/v1.27.0"
)

// GetRequestAttributes returns a map of attributes for a request.
// Requires `middleware.RequestID` middleware to be used.
func GetRequestAttributes(r *http.Request) map[string]string {
	ctx := r.Context()

	attrs := map[string]string{
		// HTTP attributes
		string(semconv.HTTPRequestMethodKey): r.Method,
		string(semconv.URLFullKey):           r.URL.String(),

		// Net attributes
		string(semconv.NetworkProtocolNameKey): r.Proto,
		string(semconv.NetworkPeerAddressKey):  r.RemoteAddr,

		// User attributes
		string(semconv.UserAgentOriginalKey): r.UserAgent(),
	}

	if reqID := middleware.GetReqID(ctx); reqID != "" {
		// There is no semantic convention for request ID, so we use our own
		// Alternatively, we could use rpc.message.id
		attrs["req_id"] = reqID
	}

	return attrs
}
