package common

import (
	"context"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/mock"
)

func TestLevelHandler(t *testing.T) {
	mockHandler := &MockHandler{}
	logger := slog.New(NewLevelHandler(mockHandler, slog.LevelInfo))

	mockHandler.On("Enabled", mock.Anything, slog.LevelInfo).Return(true)
	mockHandler.On("Enabled", mock.Anything, slog.LevelWarn).Return(true)
	mockHandler.On("Enabled", mock.Anything, slog.LevelError).Return(true)

	logger.Debug("debug")
	logger.Info("info")
	logger.Warn("warn")
	logger.Error("error")

	mockHandler.AssertExpectations(t)
}

func TestLevelHandlerWith(t *testing.T) {
	mockHandler := &MockHandler{}
	logger := slog.New(NewLevelHandler(mockHandler, slog.LevelInfo))

	logger = logger.With(slog.String("key", "value"))

	mockHandler.On("Enabled", mock.Anything, slog.LevelInfo).Return(true)
	mockHandler.On("Enabled", mock.Anything, slog.LevelWarn).Return(true)
	mockHandler.On("Enabled", mock.Anything, slog.LevelError).Return(true)

	logger.Debug("debug")
	logger.Info("info")
	logger.Warn("warn")
	logger.Error("error")

	mockHandler.AssertExpectations(t)
}

type MockHandler struct {
	mock.Mock
}

func (h *MockHandler) Enabled(ctx context.Context, level slog.Level) bool {
	args := h.Called(ctx, level)
	return args.Bool(0)
}

func (h *MockHandler) WithGroup(name string) slog.Handler {
	return h
}

func (h *MockHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return h
}

func (h *MockHandler) Handle(ctx context.Context, record slog.Record) error {
	return nil
}
