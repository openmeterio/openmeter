package log

import (
	"context"
	"log/slog"

	"github.com/stretchr/testify/mock"
)

var _ slog.Handler = &MockHandler{}

func NewMockHandler() *MockHandler {
	return &MockHandler{}
}

type MockHandler struct {
	mock.Mock
}

func (m *MockHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return m.Called(ctx, level).Bool(0)
}

func (m *MockHandler) Handle(ctx context.Context, record slog.Record) error {
	return m.Called(ctx, record).Error(0)
}

func (m *MockHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return m.Called(attrs).Get(0).(slog.Handler)
}

func (m *MockHandler) WithGroup(name string) slog.Handler {
	return m.Called(name).Get(0).(slog.Handler)
}
