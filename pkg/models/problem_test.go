package models

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestStatusProblemCancellation(t *testing.T) {
	for _, test := range []struct {
		name     string
		err      error
		canceled bool
		status   int
		want     int
	}{
		{"caller canceled", fmt.Errorf("query: %w", context.Canceled), true, 500, StatusClientClosedRequest},
		{"caller canceled gRPC", fmt.Errorf("query: %w", status.Error(codes.Canceled, "request canceled")), true, 500, StatusClientClosedRequest},
		{"internal cancellation", context.Canceled, false, 500, 500},
		{"internal gRPC cancellation", status.Error(codes.Canceled, "request canceled"), false, 503, 503},
		{"cancellation text", errors.New("dependency context canceled"), false, 503, 503},
		{"cancellation text and canceled caller", errors.New("dependency context canceled"), true, 500, 500},
		{"deadline", context.DeadlineExceeded, true, 500, 500},
		{"gRPC deadline", status.Error(codes.DeadlineExceeded, "deadline exceeded"), true, 503, 503},
		{"unrelated failure and canceled caller", errors.New("database unavailable"), true, 500, 500},
	} {
		t.Run(test.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(t.Context())
			defer cancel()
			if test.canceled {
				cancel()
			}
			problem := NewStatusProblem(ctx, test.err, test.status)
			writer := httptest.NewRecorder()
			problem.Respond(writer)
			require.Equal(t, test.want, problem.Status)
			require.Equal(t, test.want, writer.Code)
			switch test.want {
			case StatusClientClosedRequest:
				require.Equal(t, "Client Closed Request", problem.Title)
				require.Empty(t, writer.Body.Bytes())
				require.Empty(t, writer.Header().Get("Content-Type"))
			case http.StatusInternalServerError:
				require.Empty(t, problem.Detail)
				require.NotContains(t, writer.Body.String(), test.err.Error())
			}
		})
	}
}
