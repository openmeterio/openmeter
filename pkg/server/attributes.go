package server

import (
	"net"
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
		// There is no semantic convention for request ID, so we use our own
		// Alternatively, we could use rpc.message.id
		attrs["req_id"] = reqID
	}

	return attrs
}
