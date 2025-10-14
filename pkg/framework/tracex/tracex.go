package tracex

import (
	"context"
	"fmt"

	otelcodes "go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

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

type Span[T any] struct {
	ctx  context.Context
	span trace.Span
}

func Start[T any](ctx context.Context, tracer trace.Tracer, spanName string, opts ...trace.SpanStartOption) *Span[T] {
	ctx, span := tracer.Start(ctx, spanName, opts...)

	return &Span[T]{
		ctx:  ctx,
		span: span,
	}
}

func (s *Span[T]) Wrap(fn func(ctx context.Context) (T, error), opts ...Option) (T, error) {
	defer func() {
		if panicErr := recover(); panicErr != nil {
			s.span.RecordError(fmt.Errorf("panic: %v", panicErr))
			s.span.SetStatus(otelcodes.Error, "panic")
			s.span.End()

			panic(panicErr)
		}
	}()

	options := defaultOptions
	for _, opt := range opts {
		opt(&options)
	}

	result, err := fn(s.ctx)
	if err != nil {
		s.span.RecordError(err)
		s.span.SetStatus(otelcodes.Error, err.Error())
	} else {
		s.span.SetStatus(otelcodes.Ok, options.OkStatusDescription)
	}

	s.span.End()

	return result, err
}

type SpanNoValue struct {
	ctx  context.Context
	span trace.Span
}

func StartWithNoValue(ctx context.Context, tracer trace.Tracer, spanName string, opts ...trace.SpanStartOption) *SpanNoValue {
	ctx, span := tracer.Start(ctx, spanName, opts...)

	return &SpanNoValue{
		ctx:  ctx,
		span: span,
	}
}

func (s *SpanNoValue) Wrap(fn func(ctx context.Context) error, opts ...Option) error {
	span := &Span[any]{
		ctx:  s.ctx,
		span: s.span,
	}

	_, err := span.Wrap(func(ctx context.Context) (any, error) {
		return nil, fn(ctx)
	}, opts...)

	return err
}
