package apierrors

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/meter"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/feature"
)

func TestGenericErrorEncoder(t *testing.T) {
	encoder := GenericErrorEncoder()

	t.Run("FeatureNotFoundError returns 404", func(t *testing.T) {
		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/", nil)
		err := &feature.FeatureNotFoundError{ID: "feat-123"}

		handled := encoder(r.Context(), err, w, r)

		require.True(t, handled)
		assert.Equal(t, http.StatusNotFound, w.Code)

		var body map[string]interface{}
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
		assert.Contains(t, body["detail"], "feature not found: feat-123")
	})

	t.Run("MeterNotFoundError returns 404", func(t *testing.T) {
		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/", nil)
		err := meter.NewMeterNotFoundError("meter-456")

		handled := encoder(r.Context(), err, w, r)

		require.True(t, handled)
		assert.Equal(t, http.StatusNotFound, w.Code)

		var body map[string]interface{}
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
		assert.Contains(t, body["detail"], "meter not found: meter-456")
	})
}
