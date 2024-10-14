package common

import (
	"log/slog"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
)

// TODO: move this to framework?

// GlobalInitializer initializes global variables for the application.
type GlobalInitializer struct {
	Logger            *slog.Logger
	MeterProvider     metric.MeterProvider
	TracerProvider    trace.TracerProvider
	TextMapPropagator propagation.TextMapPropagator
}

// SetGlobals initializes global variables for the application.
//
// It is intended to be embedded into application structs and be called from func main.
func (i *GlobalInitializer) SetGlobals() {
	slog.SetDefault(i.Logger)
	otel.SetMeterProvider(i.MeterProvider)
	otel.SetTracerProvider(i.TracerProvider)
	otel.SetTextMapPropagator(i.TextMapPropagator)
}
