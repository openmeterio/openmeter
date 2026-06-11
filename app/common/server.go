package common

import (
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/google/wire"

	"github.com/openmeterio/openmeter/app/config"
	"github.com/openmeterio/openmeter/openmeter/server"
	pkgserver "github.com/openmeterio/openmeter/pkg/server"
)

var Server = wire.NewSet(
	NewTelemetryRouterHook,
	NewFFXConfigContextMiddleware,
	NewRouterHooks,
	NewPostAuthMiddlewares,
	NewClientIPMiddleware,
)

func NewRouterHooks(
	telemetry TelemetryMiddlewareHook,
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
