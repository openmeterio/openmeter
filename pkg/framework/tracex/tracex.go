package tracex

import (
	"context"
	"errors"
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

func WithSpan[T any](ctx context.Context, span trace.Span, fn func(ctx context.Context) (T, error), opts ...Option) (T, error) {
	o := defaultOptions

	for _, opt := range opts {
		opt(&o)
	}

	if span == nil {
		var empty T
		return empty, errors.New("span is nil")
	}

	defer func() {
		if panicErr := recover(); panicErr != nil {
			span.RecordError(fmt.Errorf("panic: %v", panicErr))
			span.SetStatus(otelcodes.Error, "panic")
			span.End()

			panic(panicErr)
		}
	}()

	res, err := fn(ctx)

	if err != nil {
		span.RecordError(err)
		span.SetStatus(otelcodes.Error, err.Error())
	} else {
		span.SetStatus(otelcodes.Ok, o.OkStatusDescription)
	}

	span.End()

	return res, err
}

func WithSpanNoValue(ctx context.Context, span trace.Span, fn func(ctx context.Context) error, opts ...Option) error {
	_, err := WithSpan(ctx, span, func(ctx context.Context) (any, error) {
		return nil, fn(ctx)
	}, opts...)

	return err
}
