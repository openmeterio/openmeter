package httptransport_test

import (
	"bytes"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/openmeterio/openmeter/api/v3/apierrors"
	"github.com/openmeterio/openmeter/pkg/errorsx"
	"github.com/openmeterio/openmeter/pkg/framework/commonhttp"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport"
	"github.com/openmeterio/openmeter/pkg/models"
)

func TestCancellationBeforeErrorEncoders(t *testing.T) {
	for _, test := range []struct {
		name     string
		err      error
		canceled bool
		decode   bool
		want     int
	}{
		{"gRPC operation", fmt.Errorf("authorize: %w", status.Error(codes.Canceled, "request canceled")), true, false, models.StatusClientClosedRequest},
		{"gRPC decoder", status.Error(codes.Canceled, "request canceled"), true, true, models.StatusClientClosedRequest},
		{"API error", apierrors.NewInternalError(t.Context(), status.Error(codes.Canceled, "request canceled")), true, false, models.StatusClientClosedRequest},
		{"transaction cleanup", errors.Join(context.Canceled, sql.ErrTxDone), true, false, models.StatusClientClosedRequest},
		{"internal gRPC cancellation", status.Error(codes.Canceled, "request canceled"), false, false, 500},
		{"deadline", status.Error(codes.DeadlineExceeded, "deadline exceeded"), true, false, 500},
		{"unrelated error", errors.New("database unavailable"), true, false, 500},
	} {
		t.Run(test.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(t.Context())
			defer cancel()
			if test.canceled {
				cancel()
			}
			var output bytes.Buffer
			handler := httptransport.NewHandler(
				func(context.Context, *http.Request) (struct{}, error) {
					if test.decode {
						return struct{}{}, test.err
					}
					return struct{}{}, nil
				},
				func(context.Context, struct{}) (struct{}, error) { return struct{}{}, test.err },
				commonhttp.EmptyResponseEncoder[struct{}](http.StatusOK),
				httptransport.WithErrorHandler(errorsx.NewSlogHandler(slog.New(slog.NewTextHandler(&output, nil)))),
				httptransport.WithErrorEncoder(apierrors.GenericErrorEncoder()),
			)
			writer := httptest.NewRecorder()
			handler.ServeHTTP(writer, httptest.NewRequestWithContext(ctx, http.MethodGet, "/", nil))
			require.Equal(t, test.want, writer.Code)
			if test.want == models.StatusClientClosedRequest {
				require.Empty(t, writer.Body.Bytes())
				require.Contains(t, output.String(), "level=WARN")
			} else {
				require.Contains(t, output.String(), "level=ERROR")
			}
		})
	}
}

type recordingStatusWriter struct {
	http.ResponseWriter
	status int
	count  int
}

func (w *recordingStatusWriter) WriteHeader(status int) {
	w.status = status
	w.count++
	w.ResponseWriter.WriteHeader(status)
}

func TestClientDisconnect(t *testing.T) {
	for _, http2 := range []bool{false, true} {
		t.Run(fmt.Sprintf("http2=%t", http2), func(t *testing.T) {
			// given a handler waiting on a downstream operation
			started := make(chan struct{})
			completed := make(chan int, 1)
			handler := httptransport.NewHandler(
				func(context.Context, *http.Request) (struct{}, error) { return struct{}{}, nil },
				func(ctx context.Context, _ struct{}) (struct{}, error) {
					close(started)
					<-ctx.Done()
					return struct{}{}, fmt.Errorf("query: %w", status.FromContextError(ctx.Err()).Err())
				},
				commonhttp.EmptyResponseEncoder[struct{}](http.StatusOK),
			)
			server := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if (r.ProtoMajor == 2) != http2 {
					t.Errorf("unexpected protocol %s", r.Proto)
				}
				writer := &recordingStatusWriter{ResponseWriter: w}
				handler.ServeHTTP(writer, r)
				completed <- writer.status
			}))
			server.EnableHTTP2 = http2
			server.StartTLS()
			defer server.Close()
			ctx, cancel := context.WithCancel(t.Context())
			defer cancel()
			request, err := http.NewRequestWithContext(ctx, http.MethodGet, server.URL, nil)
			require.NoError(t, err)
			clientDone := make(chan error, 1)
			go func() {
				response, err := server.Client().Do(request)
				if response != nil {
					_ = response.Body.Close()
				}
				clientDone <- err
			}()
			select {
			case <-started:
			case <-time.After(5 * time.Second):
				t.Fatal("handler did not start")
			}

			// when the client cancels its HTTP/1 or HTTP/2 request
			cancel()

			// then the transport records cancellation rather than a server failure
			select {
			case err := <-clientDone:
				require.ErrorIs(t, err, context.Canceled)
			case <-time.After(5 * time.Second):
				t.Fatal("client did not finish")
			}
			select {
			case code := <-completed:
				require.Equal(t, models.StatusClientClosedRequest, code)
			case <-time.After(5 * time.Second):
				t.Fatal("handler did not finish")
			}
		})
	}
}

type failedResponseWriter struct {
	*recordingStatusWriter
	err error
}

func (w failedResponseWriter) Write([]byte) (int, error) { return 0, w.err }

func TestResponseWriteCancellationPreservesCommittedStatus(t *testing.T) {
	for _, test := range []struct {
		name     string
		err      error
		canceled bool
		level    string
	}{
		{"broken pipe", &net.OpError{Op: "write", Net: "tcp", Err: syscall.EPIPE}, true, "WARN"},
		{"connection reset", fmt.Errorf("write: %w", syscall.ECONNRESET), true, "WARN"},
		{"closed connection", net.ErrClosed, true, "WARN"},
		{"opaque protocol error", errors.New("stream closed"), true, "WARN"},
		{"active request write failure", syscall.EPIPE, false, "ERROR"},
		{"server write deadline", os.ErrDeadlineExceeded, true, "ERROR"},
		{"dependency write deadline", context.DeadlineExceeded, true, "ERROR"},
	} {
		t.Run(test.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(t.Context())
			defer cancel()
			if test.canceled {
				cancel()
			}
			var output bytes.Buffer
			handler := httptransport.NewHandler(
				func(context.Context, *http.Request) (struct{}, error) { return struct{}{}, nil },
				func(context.Context, struct{}) (struct{}, error) { return struct{}{}, nil },
				commonhttp.JSONResponseEncoderWithStatus[struct{}](http.StatusOK),
				httptransport.WithErrorHandler(errorsx.NewSlogHandler(slog.New(slog.NewTextHandler(&output, nil)))),
			)
			writer := &recordingStatusWriter{ResponseWriter: httptest.NewRecorder()}
			handler.ServeHTTP(failedResponseWriter{writer, test.err}, httptest.NewRequestWithContext(ctx, http.MethodGet, "/", nil))
			require.Equal(t, http.StatusOK, writer.status)
			require.Equal(t, 1, writer.count)
			require.Contains(t, output.String(), "level="+test.level)
		})
	}
}

func TestSerializationFailureWithCanceledRequest(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	var output bytes.Buffer
	handler := httptransport.NewHandler(
		func(context.Context, *http.Request) (struct{}, error) { return struct{}{}, nil },
		func(context.Context, struct{}) (any, error) { return make(chan struct{}), nil },
		commonhttp.JSONResponseEncoderWithStatus[any](http.StatusOK),
		httptransport.WithErrorHandler(errorsx.NewSlogHandler(slog.New(slog.NewTextHandler(&output, nil)))),
	)
	writer := &recordingStatusWriter{ResponseWriter: httptest.NewRecorder()}
	handler.ServeHTTP(writer, httptest.NewRequestWithContext(ctx, http.MethodGet, "/", nil))
	require.Zero(t, writer.count)
	require.Contains(t, output.String(), "level=ERROR")
}

func TestHTTP2DisconnectDuringResponseWrite(t *testing.T) {
	// given a handler ready to write a response over HTTP/2
	started := make(chan struct{})
	completed := make(chan struct{})
	var output bytes.Buffer
	handler := httptransport.NewHandler(
		func(context.Context, *http.Request) (struct{}, error) { return struct{}{}, nil },
		func(context.Context, struct{}) (string, error) { return strings.Repeat("a", 64<<10), nil },
		func(ctx context.Context, w http.ResponseWriter, r *http.Request, response string) error {
			close(started)
			<-ctx.Done()
			return commonhttp.PlainTextResponseEncoder(ctx, w, r, response)
		},
		httptransport.WithErrorHandler(errorsx.NewSlogHandler(slog.New(slog.NewTextHandler(&output, nil)))),
	)
	server := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.ProtoMajor != 2 {
			t.Errorf("unexpected protocol %s", r.Proto)
		}
		handler.ServeHTTP(w, r)
		close(completed)
	}))
	server.EnableHTTP2 = true
	server.StartTLS()
	defer server.Close()
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, server.URL, nil)
	require.NoError(t, err)
	clientDone := make(chan error, 1)
	go func() {
		response, err := server.Client().Do(request)
		if response != nil {
			_ = response.Body.Close()
		}
		clientDone <- err
	}()
	select {
	case <-started:
	case <-time.After(5 * time.Second):
		t.Fatal("handler did not start")
	}

	// when the client resets the stream before the encoder writes
	cancel()

	// then the untyped HTTP/2 write failure is logged as cancellation
	select {
	case err := <-clientDone:
		require.ErrorIs(t, err, context.Canceled)
	case <-time.After(5 * time.Second):
		t.Fatal("client did not finish")
	}
	select {
	case <-completed:
		require.Contains(t, output.String(), "level=WARN")
		require.NotContains(t, output.String(), "level=ERROR")
	case <-time.After(5 * time.Second):
		t.Fatal("handler did not finish")
	}
}
