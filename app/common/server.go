package common

import (
	"fmt"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/google/wire"
	"go.opentelemetry.io/otel/metric"

	"github.com/openmeterio/openmeter/app/config"
	"github.com/openmeterio/openmeter/openmeter/server"
	"github.com/openmeterio/openmeter/pkg/framework/transaction/readonly"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport"
	pkgserver "github.com/openmeterio/openmeter/pkg/server"
)

var Server = wire.NewSet(
	NewTelemetryRouterHook,
	NewFFXConfigContextMiddleware,
	NewRouterHooks,
	NewPostAuthMiddlewares,
	NewClientIPMiddleware,
	NewReadOnlyRequests,
)

// NewRouterHooks depends on ReadOnlyRequests so that every binary that serves the
// API marks its read-only requests, without its main having to remember a call.
func NewRouterHooks(
	telemetry TelemetryMiddlewareHook,
	_ ReadOnlyRequests,
) *server.RouterHooks {
	return &server.RouterHooks{
		Middlewares: []server.MiddlewareHook{
			server.MiddlewareHook(telemetry),
		},
	}
}

func NewPostAuthMiddlewares(
	ffx FFXConfigContextMiddleware,
) server.PostAuthMiddlewares {
	return server.PostAuthMiddlewares{
		func(h http.Handler) http.Handler {
			return ffx(h)
		},
	}
}

// ReadOnlyRequests stands for the process-wide marking of read-only requests being
// set up, see NewReadOnlyRequests.
type ReadOnlyRequests struct{}

// NewReadOnlyRequests makes GET and HEAD requests report the database writes they
// attempt, unless the config turns it off. The handlers are built in many places
// with their own options, so the marker is installed process-wide.
func NewReadOnlyRequests(conf config.ReadOnlyRequestsConfig, logger *slog.Logger, meter metric.Meter) (ReadOnlyRequests, error) {
	if !conf.Mode.Enabled() {
		return ReadOnlyRequests{}, nil
	}

	observer, err := readonly.NewObserver(readonly.ObserverConfig{
		Logger: logger,
		Meter:  meter,
	})
	if err != nil {
		return ReadOnlyRequests{}, fmt.Errorf("failed to create read-only request observer: %w", err)
	}

	httptransport.SetReadOnlyMarker(observer.MarkReadOnly)

	return ReadOnlyRequests{}, nil
}

// ClientIPMiddleware is a defined type (not an alias) so the wire graph does not
// provide the ubiquitous pkgserver.MiddlewareFunc type directly.
type ClientIPMiddleware pkgserver.MiddlewareFunc

func NewClientIPMiddleware(cfg config.ClientIPMiddlewareConfig) (ClientIPMiddleware, error) {
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid client ip middleware config: %w", err)
	}

	switch cfg.Source {
	case config.ClientIPSourceRemoteAddr:
		return middleware.ClientIPFromRemoteAddr, nil
	case config.ClientIPSourceHeader:
		return middleware.ClientIPFromHeader(cfg.Header), nil
	case config.ClientIPSourceXFF:
		if len(cfg.TrustedIPPrefixes) > 0 {
			return middleware.ClientIPFromXFF(cfg.TrustedIPPrefixes...), nil
		}

		return middleware.ClientIPFromXFFTrustedProxies(cfg.TrustedProxies), nil
	default:
		return nil, fmt.Errorf("invalid client ip middleware source: %s", cfg.Source)
	}
}
