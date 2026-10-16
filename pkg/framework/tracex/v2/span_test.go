package v2

import (
	"context"
	"errors"
	"runtime"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	"go.opentelemetry.io/otel/trace"
)

func newTestTracer(t *testing.T) (trace.Tracer, *tracetest.SpanRecorder) {
	t.Helper()

	recorder := tracetest.NewSpanRecorder()
	provider := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(recorder))
	t.Cleanup(func() {
		// The testing context is canceled before cleanup; shutdown needs its own deadline.
		ctx, cancel := context.WithTimeout(context.WithoutCancel(t.Context()), 5*time.Second)
		defer cancel()

		require.NoError(t, provider.Shutdown(ctx))
	})

	return provider.Tracer("tracex-v2-test"), recorder
}

func TestSpanDo2(t *testing.T) {
	tests := []struct {
		name string
		err  error
	}{
		{
			name: "success",
		},
		{
			name: "callback error",
			err:  errors.New("operation failed"),
		},
		{
			name: "cancellation",
			err:  context.Canceled,
		},
		{
			name: "deadline exceeded",
			err:  context.DeadlineExceeded,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given a parent context and configured span attributes.
			tracer, recorder := newTestTracer(t)
			parent := trace.NewSpanContext(trace.SpanContextConfig{
				TraceID:    trace.TraceID{1},
				SpanID:     trace.SpanID{2},
				TraceFlags: trace.FlagsSampled,
				Remote:     true,
			})
			type contextKey struct{}
			ctx := context.WithValue(t.Context(), contextKey{}, "preserved")
			ctx = trace.ContextWithRemoteSpanContext(ctx, parent)
			span := NewSpan(tracer, "operation", trace.WithAttributes(attribute.String("resource.id", "test-id")))
			var callbackSpan trace.SpanContext

			// When the callback returns a value, including on failure.
			result, err := span.Do2(ctx, func(ctx context.Context) (int, error) {
				require.Equal(t, "preserved", ctx.Value(contextKey{}))
				callbackSpan = trace.SpanContextFromContext(ctx)
				require.True(t, trace.SpanFromContext(ctx).IsRecording())

				return 42, tt.err
			})

			// Then the callback result and error survive and exactly one child span ends.
			require.Equal(t, 42, result)
			require.ErrorIs(t, err, tt.err)
			ended := recorder.Ended()
			require.Len(t, ended, 1)
			require.Len(t, recorder.Started(), 1)
			require.Equal(t, "operation", ended[0].Name())
			require.Equal(t, parent, ended[0].Parent())
			require.Equal(t, callbackSpan, ended[0].SpanContext())
			require.Equal(t, parent.TraceID(), callbackSpan.TraceID())
			require.NotEqual(t, parent.SpanID(), callbackSpan.SpanID())
			require.Contains(t, ended[0].Attributes(), attribute.String("resource.id", "test-id"))
			require.False(t, ended[0].EndTime().Before(ended[0].StartTime()))

			switch {
			case tt.err != nil:
				require.Equal(t, codes.Error, ended[0].Status().Code)
				require.Equal(t, tt.err.Error(), ended[0].Status().Description)
				require.Len(t, ended[0].Events(), 1)
				require.Equal(t, "exception", ended[0].Events()[0].Name)
				require.Contains(t, ended[0].Events()[0].Attributes, attribute.String("exception.message", tt.err.Error()))
			default:
				require.Equal(t, codes.Ok, ended[0].Status().Code)
				require.Empty(t, ended[0].Events())
			}
		})
	}
}

func TestSpanDo(t *testing.T) {
	tests := []struct {
		name string
		err  error
	}{
		{
			name: "success",
		},
		{
			name: "callback error",
			err:  errors.New("operation failed"),
		},
		{
			name: "cancellation",
			err:  context.Canceled,
		},
		{
			name: "deadline exceeded",
			err:  context.DeadlineExceeded,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given a span without a return value.
			tracer, recorder := newTestTracer(t)
			span := NewSpan(tracer, "operation")
			calls := 0

			// When the callback completes.
			err := span.Do(t.Context(), func(ctx context.Context) error {
				calls++
				require.True(t, trace.SpanContextFromContext(ctx).IsValid())

				return tt.err
			})

			// Then Do preserves the error and ends its span.
			require.Equal(t, 1, calls)
			require.ErrorIs(t, err, tt.err)
			require.Len(t, recorder.Ended(), 1)
		})
	}
}

func TestSpanWithOption(t *testing.T) {
	tests := []struct {
		name string
		with []SpanOption
		want string
	}{
		{
			name: "no options preserves previous options",
			want: "configured",
		},
		{
			name: "replacement",
			with: []SpanOption{WithOkStatusDescription("replacement")},
			want: "replacement",
		},
		{
			name: "no-op preserves previous options",
			with: []SpanOption{spanOptionFunc(func(*options) {})},
			want: "configured",
		},
		{
			name: "options compose",
			with: []SpanOption{
				spanOptionFunc(func(o *options) {
					o.OkStatusDescription += " suffix"
				}),
			},
			want: "configured suffix",
		},
		{
			name: "multiple options apply in order",
			with: []SpanOption{
				WithOkStatusDescription("replacement"),
				spanOptionFunc(func(o *options) {
					o.OkStatusDescription += " suffix"
				}),
			},
			want: "replacement suffix",
		},
		{
			name: "last replacement wins",
			with: []SpanOption{
				WithOkStatusDescription("first"),
				WithOkStatusDescription("last"),
			},
			want: "last",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given a configured reusable span.
			tracer, recorder := newTestTracer(t)
			base := NewSpan(tracer, "operation", trace.WithAttributes(attribute.Bool("configured", true)))
			configured := base.WithOption(WithOkStatusDescription("configured"))

			// When deriving another configuration.
			derived := configured.WithOption(tt.with...)

			// Then options compose without mutating either source configuration.
			require.NotSame(t, configured, derived)
			require.Equal(t, "success", base.opts.OkStatusDescription)
			require.Equal(t, "configured", configured.opts.OkStatusDescription)
			require.Equal(t, tt.want, derived.opts.OkStatusDescription)
			require.Empty(t, recorder.Started())
			require.NoError(t, derived.Do(t.Context(), func(context.Context) error { return nil }))
			ended := recorder.Ended()
			require.Len(t, ended, 1)
			require.Equal(t, "operation", ended[0].Name())
			require.Contains(t, ended[0].Attributes(), attribute.Bool("configured", true))
		})
	}
}

func TestSpanReuse(t *testing.T) {
	tests := []struct {
		name string
		run  func(*Span, context.Context) error
	}{
		{
			name: "Do",
			run: func(s *Span, ctx context.Context) error {
				return s.Do(ctx, func(context.Context) error { return nil })
			},
		},
		{
			name: "Do2",
			run: func(s *Span, ctx context.Context) error {
				_, err := s.Do2(ctx, func(context.Context) (int, error) { return 1, nil })

				return err
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given a span definition that has not started a span yet.
			tracer, recorder := newTestTracer(t)
			span := NewSpan(tracer, "operation")
			require.Empty(t, recorder.Started())

			// When the definition is executed twice.
			require.NoError(t, tt.run(span, t.Context()))
			require.NoError(t, tt.run(span, t.Context()))

			// Then each invocation owns a distinct completed span.
			ended := recorder.Ended()
			require.Len(t, ended, 2)
			require.NotEqual(t, ended[0].SpanContext().SpanID(), ended[1].SpanContext().SpanID())
		})
	}
}

func TestSpanPanic(t *testing.T) {
	panicValue := errors.New("callback panic")
	tests := []struct {
		name string
		run  func(*Span, context.Context)
	}{
		{
			name: "Do",
			run: func(s *Span, ctx context.Context) {
				_ = s.Do(ctx, func(context.Context) error { panic(panicValue) })
			},
		},
		{
			name: "Do2",
			run: func(s *Span, ctx context.Context) {
				_, _ = s.Do2(ctx, func(context.Context) (int, error) { panic(panicValue) })
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given a callback that panics.
			tracer, recorder := newTestTracer(t)
			span := NewSpan(tracer, "operation")

			// When the operation runs, the original panic must remain visible.
			require.PanicsWithValue(t, panicValue, func() { tt.run(span, t.Context()) })

			// Then the span is ended and records the panic and stack trace.
			ended := recorder.Ended()
			require.Len(t, ended, 1)
			require.Equal(t, codes.Error, ended[0].Status().Code)
			require.Equal(t, "panic", ended[0].Status().Description)
			require.Len(t, ended[0].Events(), 1)
			attrs := attribute.NewSet(ended[0].Events()[0].Attributes...)
			message, ok := attrs.Value("exception.message")
			require.True(t, ok)
			require.Equal(t, "panic: callback panic", message.AsString())
			stack, ok := attrs.Value("exception.stacktrace")
			require.True(t, ok)
			require.NotEmpty(t, stack.AsString())
		})
	}
}

func TestSpanGoexit(t *testing.T) {
	tests := []struct {
		name string
		run  func(*Span, context.Context)
	}{
		{
			name: "Do",
			run: func(s *Span, ctx context.Context) {
				_ = s.Do(ctx, func(context.Context) error {
					runtime.Goexit()

					return nil
				})
			},
		},
		{
			name: "Do2",
			run: func(s *Span, ctx context.Context) {
				_, _ = s.Do2(ctx, func(context.Context) (int, error) {
					runtime.Goexit()

					return 0, nil
				})
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given an operation that exits its goroutine without panicking.
			tracer, recorder := newTestTracer(t)
			span := NewSpan(tracer, "operation")
			done := make(chan struct{})

			// When the callback exits, deferred span cleanup must still run.
			go func() {
				defer close(done)
				tt.run(span, t.Context())
			}()

			select {
			case <-done:
			case <-time.After(5 * time.Second):
				t.Fatal("operation did not exit")
			}

			// Then the started span is not leaked.
			require.Len(t, recorder.Started(), 1)
			require.Len(t, recorder.Ended(), 1)
		})
	}
}
