package testutils

import (
	"log/slog"
	"testing"
)

func NewLogger(t testing.TB) *slog.Logger {
	t.Helper()
	return slog.Default()
}

func NewDiscardLogger(t testing.TB) *slog.Logger {
	t.Helper()

	return slog.New(slog.DiscardHandler)
}
