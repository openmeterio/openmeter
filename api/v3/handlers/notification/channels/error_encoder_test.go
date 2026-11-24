package channels

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/api/v3/apierrors"
	"github.com/openmeterio/openmeter/openmeter/notification"
	"github.com/openmeterio/openmeter/openmeter/notification/webhook"
	"github.com/openmeterio/openmeter/pkg/models"
)

func TestErrorEncoder(t *testing.T) {
	tests := map[string]struct {
		err        error
		wantStatus int
		wantType   string
		wantDetail string
	}{
		"webhook validation error hides provider text": {
			err:        fmt.Errorf("failed to create webhook: %w", webhook.NewValidationError(errors.New("provider secret detail"))),
			wantStatus: http.StatusBadRequest,
			wantType:   apierrors.BadRequestType,
			wantDetail: "Invalid webhook configuration",
		},
		"not found": {
			err:        fmt.Errorf("failed to get channel: %w", notification.NotFoundError{NamespacedID: models.NamespacedID{Namespace: "ns", ID: "id"}}),
			wantStatus: http.StatusNotFound,
			wantType:   apierrors.NotFoundType,
			wantDetail: "The requested notification channel was not found",
		},
		"update after delete": {
			err:        notification.UpdateAfterDeleteError{Err: errors.New("not allowed to update deleted channel")},
			wantStatus: http.StatusConflict,
			wantType:   apierrors.ConflictType,
			wantDetail: "not allowed to update deleted channel",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			ctx := context.WithValue(t.Context(), middleware.RequestIDKey, "test-request-id")
			r := httptest.NewRequestWithContext(ctx, http.MethodGet, "/", nil)
			w := httptest.NewRecorder()

			require.True(t, errorEncoder()(ctx, tt.err, w, r))

			assert.Equal(t, tt.wantStatus, w.Code)
			assert.Equal(t, apierrors.ContentTypeProblemValue, w.Header().Get("Content-Type"))
			assert.NotContains(t, w.Body.String(), "provider secret detail")
			assert.NotContains(t, w.Body.String(), "ns namespace")

			var body apierrors.BaseAPIError
			require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
			assert.Equal(t, tt.wantType, body.Type)
			assert.Equal(t, tt.wantStatus, body.Status)
			assert.Equal(t, tt.wantDetail, body.Detail)
			assert.Equal(t, "urn:request:test-request-id", body.Instance)
		})
	}

	t.Run("other errors are left to the next encoder", func(t *testing.T) {
		r := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/", nil)
		w := httptest.NewRecorder()

		assert.False(t, errorEncoder()(t.Context(), errors.New("boom"), w, r))
		assert.Zero(t, w.Body.Len())
	})
}
