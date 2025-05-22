package tracex

import (
	"context"
	"fmt"

	otelcodes "go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

type Span[T any] struct {
	span trace.Span
}

func (s Span[T]) Wrap(ctx context.Context, fn func(ctx context.Context) (T, error), opts ...Option) (T, error) {
	o := defaultOptions

	for _, opt := range opts {
		opt(&o)
	}

	defer func() {
		if panicErr := recover(); panicErr != nil {
			s.span.RecordError(fmt.Errorf("panic: %v", panicErr))
			s.span.SetStatus(otelcodes.Error, "panic")
			s.span.End()

			panic(panicErr)
		}
	}()

	res, err := fn(ctx)

	if err != nil {
		s.span.RecordError(err)
		s.span.SetStatus(otelcodes.Error, err.Error())
	} else {
		s.span.SetStatus(otelcodes.Ok, o.OkStatusDescription)
	}

	s.span.End()

	return res, err
}

type Options struct {
	OkStatusDescription string
}

var defaultOptions = Options{
	OkStatusDescription: "success",
}

type Option func(*Options)

func WithOkStatusDescription(desc string) Option {
	return func(o *Options) {
		o.OkStatusDescription = desc
	}
}

func Start[T any](ctx context.Context, tracer trace.Tracer, spanName string, opts ...trace.SpanStartOption) (context.Context, Span[T]) {
	ctx, span := tracer.Start(ctx, spanName, opts...)

	return ctx, Span[T]{span: span}
}

type SpanNoValue struct {
	span Span[any]
}

func StartWithNoValue(ctx context.Context, tracer trace.Tracer, spanName string, opts ...trace.SpanStartOption) (context.Context, SpanNoValue) {
	ctx, span := Start[any](ctx, tracer, spanName, opts...)

	return ctx, SpanNoValue{span: span}
}

func (s SpanNoValue) Wrap(ctx context.Context, fn func(ctx context.Context) error, opts ...Option) error {
	_, err := s.span.Wrap(ctx, func(ctx context.Context) (any, error) {
		return nil, fn(ctx)
	}, opts...)

	return err
}
