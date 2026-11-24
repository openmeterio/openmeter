package events

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
	"github.com/openmeterio/openmeter/pkg/models"
)

func TestErrorEncoder(t *testing.T) {
	t.Run("not found", func(t *testing.T) {
		ctx := context.WithValue(t.Context(), middleware.RequestIDKey, "test-request-id")
		r := httptest.NewRequestWithContext(ctx, http.MethodGet, "/", nil)
		w := httptest.NewRecorder()
		err := fmt.Errorf("failed to get event: %w", notification.NotFoundError{NamespacedID: models.NamespacedID{Namespace: "ns", ID: "id"}})

		require.True(t, errorEncoder()(ctx, err, w, r))

		assert.Equal(t, http.StatusNotFound, w.Code)
		assert.Equal(t, apierrors.ContentTypeProblemValue, w.Header().Get("Content-Type"))

		var body apierrors.BaseAPIError
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
		assert.Equal(t, apierrors.NotFoundType, body.Type)
		assert.Equal(t, "The requested notification event was not found", body.Detail)
		assert.Equal(t, "urn:request:test-request-id", body.Instance)
	})

	t.Run("other errors are left to the next encoder", func(t *testing.T) {
		r := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/", nil)
		w := httptest.NewRecorder()

		assert.False(t, errorEncoder()(t.Context(), errors.New("boom"), w, r))
		assert.Zero(t, w.Body.Len())
	})
}
