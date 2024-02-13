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
