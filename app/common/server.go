package common

import (
	"github.com/google/wire"

	"github.com/openmeterio/openmeter/openmeter/server"
)

var Server = wire.NewSet(
	NewTelemetryRouterHook,
	NewFFXConfigContextMiddlewareHook,
	NewRouterHooks,
)

func NewRouterHooks(
	telemetry TelemetryMiddlewareHook,
	ffx FFXConfigContextMiddlewareHook,
) *server.RouterHooks {
	return &server.RouterHooks{
		Middlewares: []server.MiddlewareHook{
			server.MiddlewareHook(telemetry),
			server.MiddlewareHook(ffx),
		},
	}
}
