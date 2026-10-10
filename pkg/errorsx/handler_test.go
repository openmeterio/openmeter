package errorsx

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestSlogHandlerCancellation(t *testing.T) {
	for _, test := range []struct {
		name     string
		err      error
		canceled bool
		level    string
	}{
		{"caller cancellation", fmt.Errorf("query: %w", context.Canceled), true, "WARN"},
		{"caller gRPC cancellation", fmt.Errorf("authorize: %w", status.Error(codes.Canceled, "request canceled")), true, "WARN"},
		{"internal cancellation", context.Canceled, false, "ERROR"},
		{"internal gRPC cancellation", status.Error(codes.Canceled, "request canceled"), false, "ERROR"},
		{"deadline", context.DeadlineExceeded, true, "ERROR"},
		{"gRPC deadline", status.Error(codes.DeadlineExceeded, "deadline exceeded"), true, "ERROR"},
		{"unrelated failure", errors.New("database unavailable"), true, "ERROR"},
		{"cancellation text", errors.New("context canceled"), true, "ERROR"},
	} {
		t.Run(test.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(t.Context())
			defer cancel()
			if test.canceled {
				cancel()
			}

			var output bytes.Buffer
			handler := NewSlogHandler(slog.New(slog.NewTextHandler(&output, nil)))
			handler.HandleContext(ctx, test.err)
			require.Contains(t, output.String(), "level="+test.level)
		})
	}
}

func TestSlogHandlerWithoutContextRecognizesGRPCCancellation(t *testing.T) {
	var output bytes.Buffer
	handler := NewSlogHandler(slog.New(slog.NewTextHandler(&output, nil)))
	handler.Handle(fmt.Errorf("authorize: %w", status.Error(codes.Canceled, "request canceled")))
	require.Contains(t, output.String(), "level=WARN")
}
