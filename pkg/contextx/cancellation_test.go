package contextx

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestIsCanceledError(t *testing.T) {
	for _, test := range []struct {
		name string
		err  error
		want bool
	}{
		{"nil", nil, false},
		{"Go cancellation", context.Canceled, true},
		{"wrapped Go cancellation", fmt.Errorf("query: %w", context.Canceled), true},
		{"gRPC cancellation", status.Error(codes.Canceled, "request canceled"), true},
		{"wrapped gRPC cancellation", fmt.Errorf("authorize: %w", status.Error(codes.Canceled, "request canceled")), true},
		{"joined gRPC cancellation", errors.Join(status.Error(codes.Internal, "cleanup failed"), status.Error(codes.Canceled, "request canceled")), true},
		{"wrapped joined gRPC cancellation", fmt.Errorf("query: %w", errors.Join(status.Error(codes.Internal, "cleanup failed"), status.Error(codes.Canceled, "request canceled"))), true},
		{"Go deadline", context.DeadlineExceeded, false},
		{"gRPC deadline", status.Error(codes.DeadlineExceeded, "deadline exceeded"), false},
		{"error text", errors.New("context canceled"), false},
	} {
		t.Run(test.name, func(t *testing.T) {
			require.Equal(t, test.want, IsCanceledError(test.err))
		})
	}
}
