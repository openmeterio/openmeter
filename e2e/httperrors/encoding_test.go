package httperrors_test

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/pkg/framework/commonhttp"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport"
)

type csvResponse struct{ value string }

func (csvResponse) FileName() string      { return "response" }
func (c csvResponse) Records() [][]string { return [][]string{{"value"}, {c.value}} }

func TestDisconnectDuringResponseWrite(t *testing.T) {
	for _, http2 := range []bool{false, true} {
		for _, format := range []string{"json", "text", "csv"} {
			t.Run(fmt.Sprintf("http2=%t/%s", http2, format), func(t *testing.T) {
				// given a response whose headers have already reached the caller
				started := make(chan struct{})
				payload := strings.Repeat("a", 256<<10)
				server, completed := serveErrors(t, http2, func(diagnostics httptransport.ErrorHandler) http.Handler {
					return httptransport.NewHandler(
						func(context.Context, *http.Request) (struct{}, error) { return struct{}{}, nil },
						func(context.Context, struct{}) (string, error) { return payload, nil },
						func(ctx context.Context, w http.ResponseWriter, r *http.Request, response string) error {
							w.WriteHeader(http.StatusOK)
							if err := http.NewResponseController(w).Flush(); err != nil {
								return err
							}
							close(started)
							<-ctx.Done()
							switch format {
							case "json":
								return commonhttp.JSONResponseEncoder(ctx, w, r, map[string]string{"value": response})
							case "text":
								return commonhttp.PlainTextResponseEncoder(ctx, w, r, response)
							default:
								return commonhttp.CSVResponseEncoder(ctx, w, r, csvResponse{response})
							}
						}, httptransport.WithErrorHandler(diagnostics),
					)
				})

				// when the caller resets the connection or stream before the body write
				cancelRequest(t, server, http2, started, true)

				// then all common encoders report a canceled write without changing the 200
				observed := receive(t, completed)
				require.Equal(t, 200, observed.status)
				require.NotNil(t, observed.diagnostic)
				require.ErrorIs(t, observed.diagnostic, context.Canceled)
				requireLogLevel(t, observed, "WARN")
			})
		}
	}
}

func TestSerializationFailureAfterCallerCancellation(t *testing.T) {
	for _, http2 := range []bool{false, true} {
		t.Run(fmt.Sprintf("http2=%t", http2), func(t *testing.T) {
			// given an invalid response value waiting to be serialized
			started := make(chan struct{})
			server, completed := serveErrors(t, http2, func(diagnostics httptransport.ErrorHandler) http.Handler {
				return httptransport.NewHandler(
					func(context.Context, *http.Request) (struct{}, error) { return struct{}{}, nil },
					func(context.Context, struct{}) (chan struct{}, error) { return make(chan struct{}), nil },
					func(ctx context.Context, w http.ResponseWriter, r *http.Request, response chan struct{}) error {
						close(started)
						<-ctx.Done()
						return commonhttp.JSONResponseEncoder(ctx, w, r, response)
					}, httptransport.WithErrorHandler(diagnostics),
				)
			})

			// when caller cancellation coincides with a serialization failure
			cancelRequest(t, server, http2, started, false)

			// then the failure is still an ERROR, with no 499 or cancellation cause attached
			observed := receive(t, completed)
			require.NotEqual(t, 499, observed.status)
			require.Empty(t, observed.body)
			require.NotErrorIs(t, observed.diagnostic, context.Canceled)
			requireLogLevel(t, observed, "ERROR")
		})
	}
}

func TestServerWriteDeadlineRemainsFailure(t *testing.T) {
	for _, http2 := range []bool{false, true} {
		t.Run(fmt.Sprintf("http2=%t", http2), func(t *testing.T) {
			// given a live caller and a deadline imposed by the server itself
			server, completed := serveErrors(t, http2, func(diagnostics httptransport.ErrorHandler) http.Handler {
				return httptransport.NewHandler(
					func(context.Context, *http.Request) (struct{}, error) { return struct{}{}, nil },
					func(context.Context, struct{}) (string, error) { return strings.Repeat("a", 256<<10), nil },
					func(ctx context.Context, w http.ResponseWriter, r *http.Request, response string) error {
						w.WriteHeader(http.StatusOK)
						controller := http.NewResponseController(w)
						if err := controller.Flush(); err != nil {
							return err
						}
						if err := controller.SetWriteDeadline(time.Now().Add(-time.Second)); err != nil {
							return err
						}
						return commonhttp.PlainTextResponseEncoder(ctx, w, r, response)
					}, httptransport.WithErrorHandler(diagnostics),
				)
			})

			// when the connection write exceeds that deadline
			request, err := http.NewRequestWithContext(t.Context(), http.MethodGet, server.URL, nil)
			require.NoError(t, err)
			response, requestErr := server.Client().Do(request)
			if response != nil {
				_, requestErr = io.Copy(io.Discard, response.Body)
				_ = response.Body.Close()
			}

			// then it remains an ERROR, even if net/http also cancels the request context
			require.Error(t, requestErr)
			observed := receive(t, completed)
			var timeout net.Error
			require.ErrorAs(t, observed.diagnostic, &timeout)
			require.True(t, timeout.Timeout())
			require.NotErrorIs(t, observed.diagnostic, context.Canceled)
			requireLogLevel(t, observed, "ERROR")
		})
	}
}
