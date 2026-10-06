package httperrors_test

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/status"

	"github.com/openmeterio/openmeter/api/v3/apierrors"
	"github.com/openmeterio/openmeter/pkg/framework/commonhttp"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport"
	"github.com/openmeterio/openmeter/pkg/models"
)

type healthService struct {
	grpc_health_v1.UnimplementedHealthServer
	check func(context.Context) error
}

func (h healthService) Check(ctx context.Context, _ *grpc_health_v1.HealthCheckRequest) (*grpc_health_v1.HealthCheckResponse, error) {
	return nil, h.check(ctx)
}

func grpcDependency(t *testing.T, check func(context.Context) error) grpc_health_v1.HealthClient {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	server := grpc.NewServer()
	grpc_health_v1.RegisterHealthServer(server, healthService{check: check})
	go func() {
		if err := server.Serve(listener); err != nil && !errors.Is(err, grpc.ErrServerStopped) {
			t.Errorf("serve gRPC dependency: %v", err)
		}
	}()
	t.Cleanup(server.Stop)
	conn, err := grpc.NewClient(listener.Addr().String(), grpc.WithTransportCredentials(insecure.NewCredentials()))
	require.NoError(t, err)
	t.Cleanup(func() { _ = conn.Close() })

	return grpc_health_v1.NewHealthClient(conn)
}

func TestCallerCancellation(t *testing.T) {
	for _, http2 := range []bool{false, true} {
		for _, test := range []struct {
			name   string
			grpc   bool
			decode bool
			joined bool
			api    bool
			self   bool
			direct string
		}{
			{name: "Go operation"},
			{name: "Go decoder", decode: true},
			{name: "gRPC operation", grpc: true},
			{name: "gRPC decoder", grpc: true, decode: true},
			{name: "joined gRPC causes", grpc: true, joined: true},
			{name: "wrapped v3 operation error", grpc: true, api: true},
			{name: "self-encoded Go cancellation", self: true},
			{name: "self-encoded gRPC cancellation", grpc: true, self: true},
			{name: "legacy Go problem", direct: "legacy"},
			{name: "legacy gRPC problem", grpc: true, direct: "legacy"},
			{name: "direct v3 Go error", direct: "v3"},
			{name: "direct v3 gRPC error", grpc: true, direct: "v3"},
		} {
			t.Run(fmt.Sprintf("http2=%t/%s", http2, test.name), func(t *testing.T) {
				// given a real request blocked in decoding, an operation, or a gRPC dependency
				started := make(chan struct{})
				var dependency grpc_health_v1.HealthClient
				if test.grpc {
					dependency = grpcDependency(t, func(ctx context.Context) error {
						close(started)
						<-ctx.Done()

						return status.FromContextError(ctx.Err()).Err()
					})
				}

				failure := func(ctx context.Context) error {
					var err error
					if test.grpc {
						_, err = dependency.Check(ctx, &grpc_health_v1.HealthCheckRequest{})
					} else {
						close(started)
						<-ctx.Done()
						err = ctx.Err()
					}

					if test.joined {
						err = errors.Join(status.Error(codes.Internal, "cleanup failed"), err)
					}

					if test.api {
						err = apierrors.NewInternalError(ctx, err)
					}

					return fmt.Errorf("dependency: %w", err)
				}
				server, completed := serveErrors(t, http2, func(diagnostics httptransport.ErrorHandler) http.Handler {
					if test.direct != "" {
						return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
							err := failure(r.Context())
							if test.direct == "legacy" {
								models.NewStatusProblem(r.Context(), err, http.StatusInternalServerError).Respond(w)
							} else {
								apierrors.NewInternalError(r.Context(), err).HandleAPIError(w, r)
							}
						})
					}

					return httptransport.NewHandler(
						func(ctx context.Context, _ *http.Request) (struct{}, error) {
							if test.decode {
								return struct{}{}, failure(ctx)
							}

							return struct{}{}, nil
						},
						func(ctx context.Context, _ struct{}) (struct{}, error) {
							err := failure(ctx)
							if test.self {
								return struct{}{}, commonhttp.NewHTTPError(http.StatusServiceUnavailable, err)
							}

							return struct{}{}, err
						},
						commonhttp.JSONResponseEncoder[struct{}],
						httptransport.WithErrorHandler(diagnostics),
						httptransport.WithErrorEncoder(apierrors.GenericErrorEncoder()),
					)
				})

				// when the client closes its sending direction or resets the HTTP/2 stream
				response := cancelRequest(t, server, http2, started, false)

				// then both response paths record an empty 499, and transport diagnostics warn
				observed := receive(t, completed)
				require.Equal(t, 499, observed.status)
				require.Empty(t, observed.body)
				require.Empty(t, observed.header.Get("Content-Type"))
				if response != nil {
					require.Equal(t, 499, response.status)
					require.Empty(t, response.body)
				}

				if test.direct == "" && !test.self {
					requireLogLevel(t, observed, "WARN")
					if test.grpc && !test.joined {
						require.Equal(t, codes.Canceled, status.Code(observed.diagnostic))
						require.False(t, errors.Is(observed.diagnostic, context.Canceled))
					}
				}
			})
		}
	}
}
