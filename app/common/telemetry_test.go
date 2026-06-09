package common

import (
	"context"
	"log/slog"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestLowCardinalityPath(t *testing.T) {
	tests := []struct {
		name string
		path string
		want string
	}{
		{
			name: "fixed route is unchanged",
			path: "/api/v3/openmeter/customers",
			want: "/api/v3/openmeter/customers",
		},
		{
			name: "ULID segment masked",
			path: "/api/v3/openmeter/customers/01ARZ3NDEKTSV4RRFFQ69G5FAV/billing",
			want: "/api/v3/openmeter/customers/:id/billing",
		},
		{
			name: "UUID segment masked",
			path: "/api/v3/openmeter/customers/550e8400-e29b-41d4-a716-446655440000",
			want: "/api/v3/openmeter/customers/:id",
		},
		{
			name: "numeric segment masked",
			path: "/api/v1/things/12345",
			want: "/api/v1/things/:id",
		},
		{
			name: "over-long opaque token masked",
			path: "/api/v1/things/" + strings.Repeat("x", 40),
			want: "/api/v1/things/:id",
		},
		{
			name: "short legit segments preserved",
			path: "/api/v3/openmeter/customers/01ARZ3NDEKTSV4RRFFQ69G5FAV/entitlement-access",
			want: "/api/v3/openmeter/customers/:id/entitlement-access",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, lowCardinalityPath(tc.path))
		})
	}
}

func TestLowCardinalityPath_TruncatesDeepPaths(t *testing.T) {
	// A pathologically deep path (scanner traversal) is truncated and suffixed.
	deep := "/" + strings.Repeat("a/", 30)
	got := lowCardinalityPath(deep)

	assert.True(t, strings.HasSuffix(got, "/..."), "deep path should be truncated, got %q", got)
	assert.LessOrEqual(t, strings.Count(got, "/"), maxRouteSegments+1)
}

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
