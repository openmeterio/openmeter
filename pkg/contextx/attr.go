package contextx

import (
	"context"

	"github.com/peterbourgon/ctxdata/v4"
)

// WithAttr adds a key-value pair to the context.
func WithAttr(ctx context.Context, key string, value any) context.Context {
	d := ctxdata.From(ctx)
	if d == nil {
		ctx, d = ctxdata.New(ctx)
	}

	_ = d.Set(key, value)

	return ctx
}

// ctxdata adds amultiple key-value pairs to the context.
func WithAttrs(ctx context.Context, data map[string]string) context.Context {
	d := ctxdata.From(ctx)
	if d == nil {
		ctx, d = ctxdata.New(ctx)
	}

	for key, value := range data {
		_ = d.Set(key, value)
	}

	return ctx
}
