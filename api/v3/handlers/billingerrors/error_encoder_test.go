package billingerrors

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/billing"
)

func TestErrorEncoderWrappedValidationError(t *testing.T) {
	issue := billing.ValidationIssue{
		Severity:  billing.ValidationIssueSeverityWarning,
		Message:   "tax configuration needs attention",
		Code:      "tax_configuration_warning",
		Component: billing.ComponentName("app.stripe.invoicing.validate"),
		Path:      "/lines/line-1/taxConfig",
	}
	err := fmt.Errorf("creating charge: %w", billing.ValidationError{Err: issue})
	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/", nil)

	handled := ErrorEncoder()(t.Context(), err, response, request)

	require.True(t, handled)
	require.Equal(t, http.StatusBadRequest, response.Code)

	var problem struct {
		Status     int `json:"status"`
		Extensions struct {
			Severity  billing.ValidationIssueSeverity `json:"severity"`
			Code      string                          `json:"code"`
			Component billing.ComponentName           `json:"component"`
			Path      string                          `json:"path"`
		} `json:"extensions"`
	}
	require.NoError(t, json.Unmarshal(response.Body.Bytes(), &problem))
	require.Equal(t, http.StatusBadRequest, problem.Status)
	require.Equal(t, issue.Severity, problem.Extensions.Severity)
	require.Equal(t, issue.Code, problem.Extensions.Code)
	require.Equal(t, issue.Component, problem.Extensions.Component)
	require.Equal(t, issue.Path, problem.Extensions.Path)
}

func TestErrorEncoderRawValidationIssue(t *testing.T) {
	issue := billing.NewValidationError("invalid_invoice_line", "invoice line is invalid")
	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/", nil)

	handled := ErrorEncoder()(t.Context(), issue, response, request)

	require.True(t, handled)
	require.Equal(t, http.StatusBadRequest, response.Code)
	require.Contains(t, response.Body.String(), `"code":"invalid_invoice_line"`)
}

func TestErrorEncoderIgnoresUnrelatedErrors(t *testing.T) {
	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/", nil)

	handled := ErrorEncoder()(t.Context(), errors.New("database unavailable"), response, request)

	require.False(t, handled)
	require.Empty(t, response.Body.String())
}
