package config

import "github.com/google/wire"

var Set = wire.NewSet(
	wire.FieldsOf(new(Configuration), "Telemetry"),
	wire.FieldsOf(new(TelemetryConfig), "Log"),
	wire.FieldsOf(new(TelemetryConfig), "Metrics"),
	wire.FieldsOf(new(TelemetryConfig), "Trace"),
)
