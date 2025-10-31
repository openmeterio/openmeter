package datadog

import (
	"context"
	"log/slog"

	"go.opentelemetry.io/otel/trace"
)

const (
	traceIDKey      = "trace_id"
	spanIDKey       = "span_id"
	dataDogGroupKey = "dd"
)

// DatadogTraceAttributesHandler is a slog.Handler that adds Datadog trace attributes to the log record.
// DataDog requires trace attributes to be called "dd.trace_id" and "dd.span_id".
type DatadogTraceAttributesHandler struct {
	slog.Handler
}

func NewDatadogTraceAttributesHandler(handler slog.Handler) DatadogTraceAttributesHandler {
	return DatadogTraceAttributesHandler{Handler: handler}
}

func TraceDatadogAttributesMiddleware() func(slog.Handler) slog.Handler {
	return func(handler slog.Handler) slog.Handler {
		return NewDatadogTraceAttributesHandler(handler)
	}
}

func (h DatadogTraceAttributesHandler) Handle(ctx context.Context, record slog.Record) error {
	spanCtx := trace.SpanContextFromContext(ctx)

	attrs := make([]slog.Attr, 0, 2)
	if spanCtx.HasTraceID() {
		attrs = append(attrs, slog.String(traceIDKey, spanCtx.TraceID().String()))
	}

	if spanCtx.HasSpanID() {
		attrs = append(attrs, slog.String(spanIDKey, spanCtx.SpanID().String()))
	}

	if len(attrs) > 0 {
		record.AddAttrs(slog.GroupAttrs(dataDogGroupKey, attrs...))
	}

	return h.Handler.Handle(ctx, record)
}

// WithAttrs implements [slog.Handler].
func (h DatadogTraceAttributesHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	if h.Handler == nil {
		return h
	}

	return DatadogTraceAttributesHandler{h.Handler.WithAttrs(attrs)}
}

// WithGroup implements [slog.Handler].
func (h DatadogTraceAttributesHandler) WithGroup(name string) slog.Handler {
	if h.Handler == nil {
		return h
	}

	return DatadogTraceAttributesHandler{h.Handler.WithGroup(name)}
}
