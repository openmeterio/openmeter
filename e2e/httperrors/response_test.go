package httperrors_test

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/status"

	"github.com/openmeterio/openmeter/api/v3/apierrors"
	"github.com/openmeterio/openmeter/pkg/errorsx"
	"github.com/openmeterio/openmeter/pkg/framework/commonhttp"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport"
	"github.com/openmeterio/openmeter/pkg/models"
)

func TestActiveRequestFailures(t *testing.T) {
	for _, http2 := range []bool{false, true} {
		for _, test := range []struct {
			name   string
			cause  string
			render string
			status int
		}{
			{"Go cancellation", "cancel", "transport", 500},
			{"gRPC cancellation", "grpc-cancel", "transport", 500},
			{"Go deadline", "deadline", "transport", 500},
			{"gRPC deadline", "grpc-deadline", "transport", 500},
			{"ordinary failure", "failure", "transport", 500},
			{"cancellation text", "text", "transport", 500},
			{"legacy Go cancellation", "cancel", "legacy", 503},
			{"legacy gRPC cancellation", "grpc-cancel", "legacy", 503},
			{"legacy cancellation text", "text", "legacy", 503},
			{"legacy bad request", "cancel", "legacy", 400},
			{"legacy internal failure details", "cancel", "legacy", 500},
			{"self-encoded cancellation", "cancel", "self", 503},
			{"self-encoded gRPC cancellation", "grpc-cancel", "self", 503},
			{"direct v3 cancellation", "cancel", "v3", 500},
			{"direct v3 gRPC cancellation", "grpc-cancel", "v3", 500},
			{"direct v3 unavailable", "cancel", "v3", 503},
			{"context-free gRPC diagnostics", "grpc-cancel", "bare-log", 500},
		} {
			t.Run(fmt.Sprintf("http2=%t/%s", http2, test.name), func(t *testing.T) {
				// given a dependency failure while the HTTP caller remains connected
				var dependency grpc_health_v1.HealthClient
				if strings.HasPrefix(test.cause, "grpc-") {
					dependency = grpcDependency(t, func(ctx context.Context) error {
						if test.cause == "grpc-deadline" {
							expired, cancel := context.WithDeadline(ctx, time.Now().Add(-time.Second))
							defer cancel()
							return status.FromContextError(expired.Err()).Err()
						}
						canceled, cancel := context.WithCancel(ctx)
						cancel()
						return status.FromContextError(canceled.Err()).Err()
					})
				}
				var cause error
				server, completed := serveErrors(t, http2, func(diagnostics httptransport.ErrorHandler) http.Handler {
					failure := func(ctx context.Context) error {
						switch test.cause {
						case "cancel":
							canceled, cancel := context.WithCancel(ctx)
							cancel()
							return canceled.Err()
						case "deadline":
							expired, cancel := context.WithDeadline(ctx, time.Now().Add(-time.Second))
							defer cancel()
							return expired.Err()
						case "text":
							return errors.New("dependency context canceled")
						case "failure":
							return errors.New("dependency unavailable")
						default:
							_, err := dependency.Check(ctx, &grpc_health_v1.HealthCheckRequest{})
							return fmt.Errorf("dependency: %w", err)
						}
					}
					if test.render == "transport" || test.render == "self" {
						return httptransport.NewHandler(
							func(context.Context, *http.Request) (struct{}, error) { return struct{}{}, nil },
							func(ctx context.Context, _ struct{}) (struct{}, error) {
								cause = failure(ctx)
								if test.render == "self" {
									return struct{}{}, commonhttp.NewHTTPError(test.status, cause)
								}
								return struct{}{}, cause
							}, commonhttp.JSONResponseEncoder[struct{}],
							httptransport.WithErrorHandler(diagnostics),
						)
					}
					return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
						cause = failure(r.Context())
						if test.render == "legacy" {
							models.NewStatusProblem(r.Context(), cause, test.status).Respond(w)
							return
						}
						if test.render == "bare-log" {
							diagnostics.(errorsx.Handler).Handle(cause)
						}
						// A canceled context stored in the error must not change the original
						// request's status while that caller remains connected.
						stored, cancel := context.WithCancel(r.Context())
						cancel()
						if test.status == http.StatusServiceUnavailable {
							apierrors.NewServiceUnavailable(stored, cause).HandleAPIError(w, r)
						} else {
							apierrors.NewInternalError(stored, cause).HandleAPIError(w, r)
						}
					})
				})

				// when the request completes normally over HTTP/1 or HTTP/2
				request, err := http.NewRequestWithContext(t.Context(), http.MethodGet, server.URL, nil)
				require.NoError(t, err)
				response, err := server.Client().Do(request)
				require.NoError(t, err)
				defer response.Body.Close()
				observed := receive(t, completed)

				// then cancellation-like failures never become 408 or 499, and 500 stays sanitized
				detail := ""
				if test.status != http.StatusInternalServerError {
					if test.render == "v3" {
						detail = apierrors.UnavailableDetail
					} else {
						detail = cause.Error()
					}
				} else if test.render == "v3" || test.render == "bare-log" {
					detail = apierrors.InternalDetail
				}
				requireWireProblem(t, response, test.status, detail)
				require.Equal(t, test.status, observed.status)
				if test.status == http.StatusInternalServerError {
					require.NotContains(t, string(observed.body), cause.Error())
				}
				switch test.render {
				case "transport":
					requireLogLevel(t, observed, "ERROR")
				case "bare-log":
					requireLogLevel(t, observed, "WARN")
				}
			})
		}
	}
}

func TestUnrelatedFailureAfterCallerCancellation(t *testing.T) {
	for _, cause := range []error{
		errors.New("dependency context canceled"),
		errors.New("dependency unavailable"),
		context.DeadlineExceeded,
		status.Error(codes.DeadlineExceeded, "deadline exceeded"),
	} {
		for _, http2 := range []bool{false, true} {
			t.Run(fmt.Sprintf("http2=%t/%s", http2, cause), func(t *testing.T) {
				// given an unrelated failure arriving after the request is canceled
				started := make(chan struct{})
				server, completed := serveErrors(t, http2, func(diagnostics httptransport.ErrorHandler) http.Handler {
					return httptransport.NewHandler(
						func(context.Context, *http.Request) (struct{}, error) { return struct{}{}, nil },
						func(ctx context.Context, _ struct{}) (struct{}, error) {
							close(started)
							<-ctx.Done()
							return struct{}{}, cause
						}, commonhttp.JSONResponseEncoder[struct{}],
						httptransport.WithErrorHandler(diagnostics),
					)
				})

				// when the caller closes the connection
				response := cancelRequest(t, server, http2, started, false)

				// then the genuine failure remains a sanitized 500 and ERROR diagnostic
				observed := receive(t, completed)
				require.Equal(t, 500, observed.status)
				requireLogLevel(t, observed, "ERROR")
				require.NotContains(t, string(observed.body), cause.Error())
				if response != nil {
					requireProblemBody(t, *response, 500, "")
				}
			})
		}
	}
}
