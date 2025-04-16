package testutils

import (
	"context"
	"log/slog"
	"testing"
)

func NewLogger(t testing.TB) *slog.Logger {
	t.Helper()
	return slog.Default()
}

func NewDiscardLogger(t testing.TB) *slog.Logger {
	t.Helper()

	return slog.New(discardHandler{})
}

// TODO: remove discardHandler as soon as the project is bumped to go1.24 as minimum version
// where the discard handler has been introduced.
// This is the exact copy from slog package: https://cs.opensource.google/go/go/+/refs/tags/go1.24.2:src/log/slog/handler.go;l=608-615
type discardHandler struct{}

func (dh discardHandler) Enabled(context.Context, slog.Level) bool  { return false }
func (dh discardHandler) Handle(context.Context, slog.Record) error { return nil }
func (dh discardHandler) WithAttrs(attrs []slog.Attr) slog.Handler  { return dh }
func (dh discardHandler) WithGroup(name string) slog.Handler        { return dh }
