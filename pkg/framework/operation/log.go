package operation

import (
	"context"
	"log/slog"
)

// NewLogHandler returns a new [slog.Handler]
func NewLogHandler(handler slog.Handler) slog.Handler {
	return Handler{handler}
}

type Handler struct {
	handler slog.Handler
}

func (h Handler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.handler.Enabled(ctx, level)
}

func (h Handler) Handle(ctx context.Context, record slog.Record) error {
	name, ok := Name(ctx)

	if ok {
		record.AddAttrs(slog.String("operation", name))
	}

	return h.handler.Handle(ctx, record)
}

func (h Handler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return Handler{h.handler.WithAttrs(attrs)}
}

func (h Handler) WithGroup(name string) slog.Handler {
	return Handler{h.handler.WithGroup(name)}
}
