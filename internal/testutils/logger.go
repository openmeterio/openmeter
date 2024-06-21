package testutils

import (
	"log/slog"
	"testing"
)

func NewLogger(t testing.TB) *slog.Logger {
	t.Helper()
	return slog.Default()
}
