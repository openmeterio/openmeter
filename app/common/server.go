package common

import (
	"github.com/google/wire"

	"github.com/openmeterio/openmeter/openmeter/server"
)

var Server = wire.NewSet(
	NewTelemetryRouterHook,
	NewRouterHooks,
)

func NewRouterHooks(
	telemetry TelemetryMiddlewareHook,
) *server.RouterHooks {
	return &server.RouterHooks{
		Middlewares: []server.MiddlewareHook{
			telemetry,
		},
	}
}
