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

// discardHandler is a slog.Handler implementation which does not emit log messages
// See: https://go-review.googlesource.com/c/go/+/548335/5/src/log/slog/example_discard_test.go#14
type discardHandler struct {
	slog.JSONHandler
}

func (d *discardHandler) Enabled(context.Context, slog.Level) bool { return false }

func NewDiscardLogger(t testing.TB) *slog.Logger {
	t.Helper()

	return slog.New(&discardHandler{})
}
