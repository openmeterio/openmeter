package pgdriver

import (
	"context"

	"github.com/jackc/pgx/v5"
)

// See discussion under https://github.com/jackc/pgx/issues/1935
// We hijack the pgx.ConnConfig.Tracer (aka pgx.QueryTracer) interface as a minimally functional interceptor
// We can do this as otelsql tracer uses a completely different mechanism.

type Interceptor interface {
	Before(context.Context) context.Context
	After(context.Context)
}

func interceptorAsTracer(i Interceptor) pgx.QueryTracer {
	return &interceptorTracer{i}
}

type interceptorTracer struct {
	Interceptor Interceptor
}

func (t *interceptorTracer) TraceQueryStart(ctx context.Context, conn *pgx.Conn, data pgx.TraceQueryStartData) context.Context {
	return t.Interceptor.Before(ctx)
}

func (t *interceptorTracer) TraceQueryEnd(ctx context.Context, conn *pgx.Conn, data pgx.TraceQueryEndData) {
	t.Interceptor.After(ctx)
}

type multiTracer struct {
	tracers []pgx.QueryTracer
}

var _ pgx.QueryTracer = &multiTracer{}

func (t *multiTracer) TraceQueryStart(ctx context.Context, conn *pgx.Conn, data pgx.TraceQueryStartData) context.Context {
	for _, tracer := range t.tracers {
		ctx = tracer.TraceQueryStart(ctx, conn, data)
	}
	return ctx
}

func (t *multiTracer) TraceQueryEnd(ctx context.Context, conn *pgx.Conn, data pgx.TraceQueryEndData) {
	for _, tracer := range t.tracers {
		tracer.TraceQueryEnd(ctx, conn, data)
	}
}
