package httperrors_test

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/pkg/errorsx"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport"
)

type observation struct {
	status     int
	body       []byte
	header     http.Header
	diagnostic error
	logs       []byte
}

type wireResponse struct {
	status int
	header http.Header
	body   []byte
}

type diagnosticHandler struct {
	errorsx.SlogHandler
	err error
}

func (h *diagnosticHandler) HandleContext(ctx context.Context, err error) {
	h.err = err
	h.SlogHandler.HandleContext(ctx, err)
}

// serveErrors observes the same response status, body, and diagnostics that an
// access logger sees, including responses whose caller has disconnected.
func serveErrors(t *testing.T, http2 bool, factory func(httptransport.ErrorHandler) http.Handler) (*httptest.Server, <-chan observation) {
	t.Helper()
	var logs bytes.Buffer
	diagnostics := &diagnosticHandler{SlogHandler: errorsx.NewSlogHandler(slog.New(slog.NewJSONHandler(&logs, nil)))}
	handler := factory(diagnostics)
	completed := make(chan observation, 1)
	server := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if (r.ProtoMajor == 2) != http2 {
			t.Errorf("unexpected protocol %s", r.Proto)
		}

		writer := middleware.NewWrapResponseWriter(w, r.ProtoMajor)
		var body bytes.Buffer
		writer.Tee(&body)
		handler.ServeHTTP(writer, r)
		completed <- observation{
			status: writer.Status(), body: bytes.Clone(body.Bytes()), header: writer.Header().Clone(),
			diagnostic: diagnostics.err, logs: bytes.Clone(logs.Bytes()),
		}
	}))
	server.EnableHTTP2 = http2
	if http2 {
		server.StartTLS()
	} else {
		server.Start()
	}

	server.Client().Timeout = 5 * time.Second
	t.Cleanup(func() {
		server.CloseClientConnections()
		server.Close()
	})

	return server, completed
}

func receive[T any](t *testing.T, events <-chan T) T {
	t.Helper()
	select {
	case event := <-events:
		return event
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for the request lifecycle")
		var zero T

		return zero
	}
}

func requireLogLevel(t *testing.T, observed observation, level string) {
	t.Helper()
	var entry struct {
		Level string `json:"level"`
	}
	require.NoError(t, json.Unmarshal(observed.logs, &entry))
	require.Equal(t, level, entry.Level)
}

// cancelRequest waits until the failing dependency or encoder is running before
// canceling the real connection. A TCP half-close leaves HTTP/1 readable so its
// cancellation response can also be asserted on the wire.
func cancelRequest(t *testing.T, server *httptest.Server, http2 bool, started <-chan struct{}, reset bool) *wireResponse {
	t.Helper()
	if !http2 {
		conn, err := (&net.Dialer{}).DialContext(t.Context(), "tcp", server.Listener.Addr().String())
		require.NoError(t, err)
		t.Cleanup(func() { _ = conn.Close() })
		require.NoError(t, conn.SetDeadline(time.Now().Add(5*time.Second)))
		_, err = fmt.Fprint(conn, "GET / HTTP/1.1\r\nHost: localhost\r\n\r\n")
		require.NoError(t, err)
		receive(t, started)
		tcp := conn.(*net.TCPConn)
		if reset {
			require.NoError(t, tcp.SetLinger(0))
			require.NoError(t, tcp.Close())

			return nil
		}

		require.NoError(t, tcp.CloseWrite())
		response, err := http.ReadResponse(bufio.NewReader(conn), nil)
		require.NoError(t, err)
		defer response.Body.Close()
		body, err := io.ReadAll(response.Body)
		require.NoError(t, err)

		return &wireResponse{status: response.StatusCode, header: response.Header.Clone(), body: body}
	}

	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, server.URL, nil)
	require.NoError(t, err)
	clientDone := make(chan error, 1)
	go func() {
		response, err := server.Client().Do(request)
		if response != nil {
			if err == nil {
				_, err = io.Copy(io.Discard, response.Body)
			}

			_ = response.Body.Close()
		}

		clientDone <- err
	}()
	receive(t, started)
	cancel()
	require.ErrorIs(t, receive(t, clientDone), context.Canceled)

	return nil
}

func requireWireProblem(t *testing.T, response *http.Response, status int, detail string) {
	t.Helper()
	body, err := io.ReadAll(response.Body)
	require.NoError(t, err)
	requireProblemBody(t, wireResponse{status: response.StatusCode, header: response.Header, body: body}, status, detail)
}

func requireProblemBody(t *testing.T, response wireResponse, status int, detail string) {
	t.Helper()
	require.Equal(t, status, response.status)
	require.Equal(t, "application/problem+json", response.header.Get("Content-Type"))
	var problem struct {
		Status int    `json:"status"`
		Detail string `json:"detail"`
	}
	require.NoError(t, json.Unmarshal(response.body, &problem))
	require.Equal(t, status, problem.Status)
	require.Equal(t, detail, problem.Detail)
}
