package httptransport

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/pkg/framework/transaction"
	transactiontestutils "github.com/openmeterio/openmeter/pkg/framework/transaction/testutils"
)

func TestReadOnlyScopeMarking(t *testing.T) {
	tests := []struct {
		name     string
		method   string
		options  []HandlerOption
		noMarker bool

		expectReadOnly  bool
		expectOperation []string
	}{
		{
			name:            "GET runs in a read-only scope",
			method:          http.MethodGet,
			options:         []HandlerOption{WithOperationName("get-thing")},
			expectReadOnly:  true,
			expectOperation: []string{"get-thing"},
		},
		{
			name:            "HEAD runs in a read-only scope",
			method:          http.MethodHead,
			options:         []HandlerOption{WithOperationName("head-thing")},
			expectReadOnly:  true,
			expectOperation: []string{"head-thing"},
		},
		{
			name:            "a handler without an operation name is still marked",
			method:          http.MethodGet,
			expectReadOnly:  true,
			expectOperation: []string{""},
		},
		{
			name:    "POST is left alone",
			method:  http.MethodPost,
			options: []HandlerOption{WithOperationName("create-thing")},
		},
		{
			name:    "a GET that opted out is left alone",
			method:  http.MethodGet,
			options: []HandlerOption{WithOperationName("get-balance"), WithPrimaryDatabase()},
		},
		{
			name:     "nothing is marked until a marker is set",
			method:   http.MethodGet,
			options:  []HandlerOption{WithOperationName("get-thing")},
			noMarker: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// given a marker that records the operations it marks
			var opened []string

			recorder := &transactiontestutils.WriteRecorder{}

			if !test.noMarker {
				SetReadOnlyMarker(func(ctx context.Context, operation string) context.Context {
					opened = append(opened, operation)

					ctx, err := transaction.WithWriteObserver(ctx, recorder)
					require.NoError(t, err)

					return ctx
				})
				t.Cleanup(func() { SetReadOnlyMarker(nil) })
			}

			// A context is read-only when a write reported with it reaches the observer.
			isReadOnly := func(ctx context.Context) bool {
				before := len(recorder.Attempts())
				require.NoError(t, transaction.ReportWrite(ctx, transaction.WriteAttempt{Kind: transaction.WriteKindMutation}))

				return len(recorder.Attempts()) > before
			}

			var decoderReadOnly, operationReadOnly bool

			// The args variant copies the handler, so it has to carry the opt-out too.
			handler := NewHandlerWithArgs(
				func(ctx context.Context, _ *http.Request, _ struct{}) (struct{}, error) {
					decoderReadOnly = isReadOnly(ctx)

					return struct{}{}, nil
				},
				func(ctx context.Context, _ struct{}) (struct{}, error) {
					operationReadOnly = isReadOnly(ctx)

					return struct{}{}, nil
				},
				func(_ context.Context, w http.ResponseWriter, _ *http.Request, _ struct{}) error {
					w.WriteHeader(http.StatusNoContent)

					return nil
				},
				test.options...,
			).With(struct{}{})

			// when the request is served
			handler.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(test.method, "/things", nil))

			// then the decoder and the operation see the same scope
			require.Equal(t, test.expectReadOnly, decoderReadOnly, "decoder")
			require.Equal(t, test.expectReadOnly, operationReadOnly, "operation")
			require.Equal(t, test.expectOperation, opened)
		})
	}
}
