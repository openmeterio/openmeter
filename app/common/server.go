package common

import (
	"net/http"

	"github.com/google/wire"

	"github.com/openmeterio/openmeter/openmeter/server"
)

var Server = wire.NewSet(
	NewTelemetryRouterHook,
	NewFFXConfigContextMiddleware,
	NewRouterHooks,
	NewPostAuthMiddlewares,
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
