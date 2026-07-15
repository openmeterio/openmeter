package apierrors

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/meter"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/feature"
	"github.com/openmeterio/openmeter/pkg/framework/commonhttp"
	"github.com/openmeterio/openmeter/pkg/models"
)

func encodeError(t *testing.T, err error) (bool, int, map[string]any) {
	t.Helper()

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/", nil)

	handled := GenericErrorEncoder()(r.Context(), err, w, r)
	if !handled {
		return false, 0, nil
	}

	var body map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))

	return true, w.Code, body
}

func TestGenericErrorEncoder(t *testing.T) {
	t.Run("FeatureNotFoundError returns 404", func(t *testing.T) {
		handled, code, body := encodeError(t, &feature.FeatureNotFoundError{ID: "feat-123"})

		require.True(t, handled)
		assert.Equal(t, http.StatusNotFound, code)
		assert.Equal(t, NotFoundType, body["type"])
		assert.Contains(t, body["detail"], "feature not found: feat-123")
	})

	t.Run("MeterNotFoundError returns 404", func(t *testing.T) {
		handled, code, body := encodeError(t, meter.NewMeterNotFoundError("meter-456"))

		require.True(t, handled)
		assert.Equal(t, http.StatusNotFound, code)
		assert.Equal(t, NotFoundType, body["type"])
		assert.Contains(t, body["detail"], "meter not found: meter-456")
	})

	t.Run("GenericConflictError returns 409 with the v3 conflict shape", func(t *testing.T) {
		handled, code, body := encodeError(t, fmt.Errorf("create: %w", models.NewGenericConflictError(
			errors.New("credit grant with key \"k-1\" already exists"),
		)))

		require.True(t, handled)
		assert.Equal(t, http.StatusConflict, code)
		assert.Equal(t, ConflictType, body["type"])
		assert.Contains(t, body["detail"], "credit grant with key \"k-1\" already exists")
		assert.NotContains(t, body, "conflicting_resource")
	})

	t.Run("GenericConflictError with resource exposes conflicting_resource", func(t *testing.T) {
		handled, code, body := encodeError(t, models.NewGenericConflictErrorWithResource(
			errors.New("credit grant with key \"k-1\" already exists"),
			models.ConflictingResource{
				Type:       "credit_grant",
				ID:         "grant-1",
				CustomerID: "customer-1",
			},
		))

		require.True(t, handled)
		assert.Equal(t, http.StatusConflict, code)
		assert.Equal(t, ConflictType, body["type"])

		resource, ok := body["conflicting_resource"].(map[string]any)
		require.True(t, ok, "conflicting_resource should be present, body: %v", body)
		assert.Equal(t, "credit_grant", resource["type"])
		assert.Equal(t, "grant-1", resource["id"])
		assert.Equal(t, "customer-1", resource["customer_id"])
	})

	t.Run("GenericNotFoundError returns 404 with detail", func(t *testing.T) {
		handled, code, body := encodeError(t, models.NewGenericNotFoundError(errors.New("credit grant grant-1 not found")))

		require.True(t, handled)
		assert.Equal(t, http.StatusNotFound, code)
		assert.Equal(t, NotFoundType, body["type"])
		assert.Contains(t, body["detail"], "credit grant grant-1 not found")
	})

	t.Run("GenericValidationError returns 400 with detail", func(t *testing.T) {
		handled, code, body := encodeError(t, models.NewGenericValidationError(errors.New("amount must be positive")))

		require.True(t, handled)
		assert.Equal(t, http.StatusBadRequest, code)
		assert.Equal(t, BadRequestType, body["type"])
		assert.Contains(t, body["detail"], "amount must be positive")
	})

	t.Run("GenericPreConditionFailedError returns 412", func(t *testing.T) {
		handled, code, body := encodeError(t, models.NewGenericPreConditionFailedError(errors.New("invoice is not in draft state")))

		require.True(t, handled)
		assert.Equal(t, http.StatusPreconditionFailed, code)
		assert.Equal(t, PreconditionFailedType, body["type"])
	})

	t.Run("GenericForbiddenError returns 403 with the fixed detail", func(t *testing.T) {
		handled, code, body := encodeError(t, models.NewGenericForbiddenError(errors.New("secret reason")))

		require.True(t, handled)
		assert.Equal(t, http.StatusForbidden, code)
		assert.Equal(t, ForbiddenType, body["type"])
		assert.Equal(t, ForbiddenDetail, body["detail"])
	})

	t.Run("GenericUnauthorizedError returns 401 with the fixed detail", func(t *testing.T) {
		handled, code, body := encodeError(t, models.NewGenericUnauthorizedError(errors.New("secret reason")))

		require.True(t, handled)
		assert.Equal(t, http.StatusUnauthorized, code)
		assert.Equal(t, UnauthenticatedType, body["type"])
		assert.Equal(t, UnauthenticatedDetail, body["detail"])
	})

	t.Run("GenericNotImplementedError returns 501", func(t *testing.T) {
		handled, code, body := encodeError(t, models.NewGenericNotImplementedError(errors.New("not yet")))

		require.True(t, handled)
		assert.Equal(t, http.StatusNotImplemented, code)
		assert.Equal(t, NotImplementedType, body["type"])
	})

	t.Run("validation issue with a 400 status maps to invalid_parameters", func(t *testing.T) {
		issue := models.NewValidationIssue(
			"credit_grant_amount_invalid",
			"amount must be positive",
			models.WithFieldString("amount"),
			models.WithCriticalSeverity(),
			commonhttp.WithHTTPStatusCodeAttribute(http.StatusBadRequest),
		)

		handled, code, body := encodeError(t, issue)

		require.True(t, handled)
		assert.Equal(t, http.StatusBadRequest, code)
		assert.Equal(t, BadRequestType, body["type"])

		params, ok := body["invalid_parameters"].([]any)
		require.True(t, ok, "invalid_parameters should be present, body: %v", body)
		require.Len(t, params, 1)

		param, ok := params[0].(map[string]any)
		require.True(t, ok)
		assert.Contains(t, param["field"], "amount")
		assert.Equal(t, "amount must be positive", param["reason"])
	})

	t.Run("unrecognized error is not handled", func(t *testing.T) {
		handled, _, _ := encodeError(t, errors.New("some transient failure"))

		require.False(t, handled)
	})
}
