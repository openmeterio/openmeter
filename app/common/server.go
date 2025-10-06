package common

import (
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
	"github.com/google/wire"

	"github.com/openmeterio/openmeter/openmeter/server"
	"github.com/openmeterio/openmeter/pkg/contextx"
	pkgserver "github.com/openmeterio/openmeter/pkg/server"
)

var Server = wire.NewSet(
	NewTelemetryRouterHook,
	NewFFXConfigContextMiddleware,
	NewRouterHooks,
)

// FIXME: ideally we would move Router + Serve instances in DI as a whole
func NewRouterHooks(
	telemetryHook TelemetryMiddlewareHook,
	ffx FFXConfigContextMiddleware,
) server.RouterHookManager {
	// It does not make sense to have reverse dependency direction,
	// e.g. package X constructor importing RouterHooks to register itself,
	// as package X might be functional without Router.
	// Example would be current FFX setup.
	hooks := server.NewRouterHookManager()

	hooks.RegisterMiddleware(100, middleware.RealIP)
	hooks.RegisterMiddleware(100, middleware.RequestID)
	hooks.RegisterMiddleware(100, func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()
			ctx = contextx.WithAttrs(ctx, pkgserver.GetRequestAttributes(r))

			h.ServeHTTP(w, r.WithContext(ctx))
		})
	})
	hooks.RegisterMiddleware(100, pkgserver.NewRequestLoggerMiddleware(slog.Default().Handler()))
	hooks.RegisterMiddleware(10, middleware.Recoverer)
	hooks.RegisterMiddleware(101, render.SetContentType(render.ContentTypeJSON))

	hooks.RegisterMiddleware(201, ffx)

	hooks.RegisterMiddlewareHook(201, func(m server.MiddlewareManager) {
		telemetryHook(m)
	})

	return hooks
}
