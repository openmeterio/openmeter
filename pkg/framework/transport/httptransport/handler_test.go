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
)

type recordingErrorHandler struct {
	err error
}

func (h *recordingErrorHandler) HandleContext(_ context.Context, err error) {
	h.err = err
}

func decodeRequest(_ context.Context, _ *http.Request) (struct{}, error) {
	return struct{}{}, nil
}

func decodeCanceledRequest(_ context.Context, _ *http.Request) (struct{}, error) {
	return struct{}{}, fmt.Errorf("decode request: %w", context.Canceled)
}

func executeRequest(_ context.Context, _ struct{}) (struct{}, error) {
	return struct{}{}, nil
}

func executeCanceledRequest(_ context.Context, _ struct{}) (struct{}, error) {
	return struct{}{}, fmt.Errorf("execute request: %w", context.Canceled)
}

func encodeResponse(_ context.Context, w http.ResponseWriter, _ *http.Request, _ struct{}) error {
	w.WriteHeader(http.StatusNoContent)
	return nil
}

func TestHandlerRecordsClientClosedRequest(t *testing.T) {
	tests := []struct {
		name           string
		requestDecoder RequestDecoder[struct{}]
		operation      operation.Operation[struct{}, struct{}]
	}{
		{
			name:           "request decoder cancellation",
			requestDecoder: decodeCanceledRequest,
			operation:      executeRequest,
		},
		{
			name:           "operation cancellation",
			requestDecoder: decodeRequest,
			operation:      executeCanceledRequest,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			errorHandler := &recordingErrorHandler{}
			handler := NewHandler(
				test.requestDecoder,
				test.operation,
				encodeResponse,
				WithErrorHandler(errorHandler),
				WithErrorEncoder(commonhttp.DummyErrorEncoder()),
			)

			writer := httptest.NewRecorder()
			handler.ServeHTTP(writer, httptest.NewRequest(http.MethodGet, "/", nil))

			require.Equal(t, statusClientClosedRequest, writer.Code)
			require.Empty(t, writer.Body.Bytes())
			require.ErrorIs(t, errorHandler.err, context.Canceled)
		})
	}
}
