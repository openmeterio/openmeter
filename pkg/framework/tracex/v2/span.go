package v2

import (
	"context"
	"fmt"
	"runtime/debug"

	"go.opentelemetry.io/otel/attribute"
	otelcodes "go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

type options struct {
	OkStatusDescription string
}

var defaultOptions = options{
	OkStatusDescription: "success",
}

// SpanOption configures the behavior of a reusable span definition.
type SpanOption interface {
	apply(*options)
}

type spanOptionFunc func(*options)

func (f spanOptionFunc) apply(o *options) {
	f(o)
}

// WithOkStatusDescription sets the description passed to SetStatus when a callback
// succeeds. The tracer implementation may discard descriptions for an OK status.
func WithOkStatusDescription(desc string) SpanOption {
	return spanOptionFunc(func(o *options) {
		o.OkStatusDescription = desc
	})
}

// Span is a reusable tracing configuration. Each Do or Do2 call starts and ends
// its own span using the caller's context.
type Span struct {
	tracer   trace.Tracer
	name     string
	spanOpts []trace.SpanStartOption
	opts     options
}

// WithOption returns a copy with the supplied options applied in order, preserving
// existing configuration without modifying the receiver or starting a span.
func (s *Span) WithOption(opts ...SpanOption) *Span {
	r := NewSpan(s.tracer, s.name, s.spanOpts...)
	r.opts = s.opts

	for _, o := range opts {
		o.apply(&r.opts)
	}

	return r
}

// Do runs f with a context containing a new span and returns its error unchanged.
// It uses the same status, panic recording, and cleanup behavior as Do2.
func (s *Span) Do(ctx context.Context, f func(context.Context) error) error {
	_, err := s.Do2[any](ctx, func(ctx context.Context) (any, error) {
		return nil, f(ctx)
	})

	return err
}

// Do2 runs f with a context containing a new span and returns its result and error
// unchanged. Callback errors are recorded with an error status; a nil error sets
// an OK status. Panics are recorded with a stack trace and rethrown unchanged.
// The span is ended even when the callback panics or calls runtime.Goexit.
func (s *Span) Do2[T any](ctx context.Context, f func(context.Context) (T, error)) (T, error) {
	sCtx, span := s.tracer.Start(ctx, s.name, s.spanOpts...)

	defer func() {
		span.End()
	}()

	defer func() {
		if r := recover(); r != nil {
			span.RecordError(
				fmt.Errorf("panic: %v", r),
				trace.WithStackTrace(true),
				trace.WithAttributes(
					attribute.String("panic.stacktrace", fmt.Sprintf("%v", string(debug.Stack()))),
				),
			)
			span.SetStatus(otelcodes.Error, "panic")

			panic(r)
		}
	}()

	result, err := f(sCtx)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(otelcodes.Error, err.Error())
	} else {
		span.SetStatus(otelcodes.Ok, s.opts.OkStatusDescription)
	}

	return result, err
}

// NewSpan creates a reusable tracing configuration without starting a span.
// The name and start options are used for every Do or Do2 invocation.
func NewSpan(tracer trace.Tracer, name string, opts ...trace.SpanStartOption) *Span {
	return &Span{tracer, name, append([]trace.SpanStartOption(nil), opts...), defaultOptions}
}
