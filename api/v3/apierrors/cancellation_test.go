package apierrors

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

	"github.com/openmeterio/openmeter/pkg/models"
)

func TestHandleAPIErrorCancellation(t *testing.T) {
	for _, test := range []struct {
		name     string
		err      error
		canceled bool
		want     int
	}{
		{"caller cancellation", context.Canceled, true, models.StatusClientClosedRequest},
		{"caller gRPC cancellation", fmt.Errorf("authorize: %w", status.Error(codes.Canceled, "request canceled")), true, models.StatusClientClosedRequest},
		{"active request", context.Canceled, false, http.StatusInternalServerError},
		{"active request gRPC", status.Error(codes.Canceled, "request canceled"), false, http.StatusInternalServerError},
		{"deadline", context.DeadlineExceeded, true, http.StatusInternalServerError},
		{"unrelated failure", errors.New("database unavailable"), true, http.StatusInternalServerError},
	} {
		t.Run(test.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(t.Context())
			defer cancel()
			if test.canceled {
				cancel()
			}
			// The context stored in an API error can differ from the original request.
			errorCtx, cancelError := context.WithCancel(t.Context())
			cancelError()
			apiErr := NewInternalError(errorCtx, test.err)
			writer := httptest.NewRecorder()
			apiErr.HandleAPIError(writer, httptest.NewRequestWithContext(ctx, http.MethodGet, "/", nil))
			require.Equal(t, test.want, writer.Code)
			if test.want == models.StatusClientClosedRequest {
				require.Empty(t, writer.Body.Bytes())
			} else {
				require.NotEmpty(t, writer.Body.Bytes())
			}
		})
	}
}
