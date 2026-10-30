package testutils

import (
	"bytes"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// WebhookRequest preserves the original body for payload and signature assertions.
type WebhookRequest struct {
	Header http.Header
	Body   []byte
}

// WebhookServer captures deliveries independently of the provider implementation.
type WebhookServer struct {
	URL string

	mu           sync.Mutex
	requests     []WebhookRequest
	responseCode int
	responseBody string
}

// NewWebhookServer starts a receiver on a random port and closes it during cleanup.
// advertisedHost is the hostname or IP reachable from Svix, without a scheme or port.
// The listener binds all IPv4 interfaces so containers can reach it.
func NewWebhookServer(t *testing.T, advertisedHost string) *WebhookServer {
	t.Helper()

	require.NotEmpty(t, advertisedHost, "advertised webhook host is required")

	receiver := &WebhookServer{
		responseCode: http.StatusNoContent,
	}

	server := httptest.NewUnstartedServer(http.HandlerFunc(receiver.serveHTTP))
	t.Cleanup(server.Close)

	// Replace httptest's loopback listener with a container-reachable listener.
	require.NoError(t, server.Listener.Close())

	listener, err := net.Listen("tcp4", "0.0.0.0:0")
	require.NoError(t, err, "starting webhook listener must succeed")

	server.Listener = listener
	server.Config.ReadHeaderTimeout = 5 * time.Second
	server.Config.ReadTimeout = 5 * time.Second
	server.Config.WriteTimeout = 5 * time.Second

	server.Start()

	_, port, err := net.SplitHostPort(listener.Addr().String())
	require.NoError(t, err)

	receiver.URL = "http://" + net.JoinHostPort(advertisedHost, port)

	return receiver
}

// SetResponse changes the response for subsequent captured deliveries.
// Requests already being handled may complete with the previous response.
func (s *WebhookServer) SetResponse(statusCode int, body string) error {
	if statusCode < 200 || statusCode > 599 {
		return fmt.Errorf("invalid response status code: %d", statusCode)
	}

	if (statusCode == http.StatusNoContent || statusCode == http.StatusNotModified) && body != "" {
		return fmt.Errorf("invalid response body: status %d does not allow a body", statusCode)
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	s.responseCode = statusCode
	s.responseBody = body

	return nil
}

func (s *WebhookServer) serveHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		w.WriteHeader(http.StatusMethodNotAllowed)

		return
	}

	// Bound memory use while retaining the exact body bytes.
	body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, 1<<20))
	if err != nil {
		http.Error(w, "failed to read webhook body", http.StatusBadRequest)

		return
	}

	s.mu.Lock()

	s.requests = append(s.requests, WebhookRequest{
		Header: r.Header.Clone(),
		Body:   body,
	})

	statusCode := s.responseCode
	responseBody := s.responseBody

	s.mu.Unlock()

	w.WriteHeader(statusCode)

	if responseBody != "" {
		_, _ = io.WriteString(w, responseBody)
	}
}

// Requests returns an independent snapshot safe to inspect while deliveries arrive.
func (s *WebhookServer) Requests() []WebhookRequest {
	s.mu.Lock()
	defer s.mu.Unlock()

	requests := make([]WebhookRequest, len(s.requests))

	for i, request := range s.requests {
		requests[i] = WebhookRequest{
			Header: request.Header.Clone(),
			Body:   bytes.Clone(request.Body),
		}
	}

	return requests
}
