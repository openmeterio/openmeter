package httptransport

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/pkg/framework/commonhttp"
	"github.com/openmeterio/openmeter/pkg/framework/operation"
	"github.com/openmeterio/openmeter/pkg/models"
)

type recordingErrorHandler struct {
	err error
}

func (h *recordingErrorHandler) HandleContext(_ context.Context, err error) {
	h.err = err
}

func TestHandlerRecordsClientClosedRequest(t *testing.T) {
	tests := []struct {
		name           string
		requestDecoder RequestDecoder[struct{}]
		operation      operation.Operation[struct{}, struct{}]
	}{
		{
			name: "request decoder cancellation",
			requestDecoder: func(_ context.Context, _ *http.Request) (struct{}, error) {
				return struct{}{}, fmt.Errorf("decode request: %w", context.Canceled)
			},
			operation: func(_ context.Context, _ struct{}) (struct{}, error) {
				return struct{}{}, nil
			},
		},
		{
			name: "operation cancellation",
			requestDecoder: func(_ context.Context, _ *http.Request) (struct{}, error) {
				return struct{}{}, nil
			},
			operation: func(_ context.Context, _ struct{}) (struct{}, error) {
				return struct{}{}, fmt.Errorf("execute request: %w", context.Canceled)
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			errorHandler := &recordingErrorHandler{}
			handler := NewHandler(
				test.requestDecoder,
				test.operation,
				func(_ context.Context, _ http.ResponseWriter, _ *http.Request, _ struct{}) error {
					require.FailNow(t, "response encoder must not be called")
					return nil
				},
				WithErrorHandler(errorHandler),
				WithErrorEncoder(commonhttp.DummyErrorEncoder()),
			)

			ctx, cancel := context.WithCancel(t.Context())
			cancel()

			writer := httptest.NewRecorder()
			request := httptest.NewRequest(http.MethodGet, "/", nil).WithContext(ctx)
			handler.ServeHTTP(writer, request)

			require.Equal(t, models.StatusClientClosedRequest, writer.Code)
			require.Empty(t, writer.Body.Bytes())
			require.ErrorIs(t, errorHandler.err, context.Canceled)
		})
	}
}

func TestHandlerDoesNotRecordInternalCancellationAsClientClosedRequest(t *testing.T) {
	errorHandler := &recordingErrorHandler{}
	handler := NewHandler(
		func(_ context.Context, _ *http.Request) (struct{}, error) {
			return struct{}{}, nil
		},
		func(_ context.Context, _ struct{}) (struct{}, error) {
			return struct{}{}, fmt.Errorf("execute request: %w", context.Canceled)
		},
		func(_ context.Context, _ http.ResponseWriter, _ *http.Request, _ struct{}) error {
			require.FailNow(t, "response encoder must not be called")
			return nil
		},
		WithErrorHandler(errorHandler),
		WithErrorEncoder(commonhttp.DummyErrorEncoder()),
	)

	writer := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/", nil)
	handler.ServeHTTP(writer, request)

	require.Equal(t, http.StatusBadRequest, writer.Code)
	require.NotEmpty(t, writer.Body.Bytes())
	require.Nil(t, errorHandler.err)
	require.NoError(t, request.Context().Err())
}
