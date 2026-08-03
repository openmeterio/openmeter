package commonhttp_test

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/pkg/framework/commonhttp"
	"github.com/openmeterio/openmeter/pkg/models"
)

func TestHandleIssueIfHTTPStatusKnown(t *testing.T) {
	test_err_code := models.ErrorCode("test_err_code")

	t.Run("Should hide http status code attribute", func(t *testing.T) {
		err := models.NewValidationIssue(test_err_code, "something went wrong", commonhttp.WithHTTPStatusCodeAttribute(http.StatusBadRequest))
		require.Error(t, err)

		writer := httptest.NewRecorder()
		require.True(t, commonhttp.HandleIssueIfHTTPStatusKnown(t.Context(), err, writer))

		res := writer.Result()
		defer res.Body.Close()

		require.Equal(t, http.StatusBadRequest, res.StatusCode)

		var body map[string]interface{}
		require.NoError(t, json.Unmarshal(writer.Body.Bytes(), &body))

		extensions, ok := body["extensions"].(map[string]interface{})
		require.True(t, ok)

		issues, ok := extensions["validationErrors"].([]interface{})
		require.True(t, ok)
		require.Len(t, issues, 1)

		issue, ok := issues[0].(map[string]interface{})
		require.True(t, ok)

		require.NotContains(t, issue, "commonhttp.httpAttributeKey:openmeter.http.status_code")

		require.Equal(t, string(test_err_code), issue["code"])
		require.Contains(t, issue, "message")
		require.Contains(t, issue, "severity")
		require.Len(t, issue, 3)
	})
}

func TestGenericErrorEncoderStatusMapping(t *testing.T) {
	tests := []struct {
		name           string
		err            error
		expectedStatus int
	}{
		{
			name:           "secret store failure is a failed dependency, not an internal error",
			err:            fmt.Errorf("failed to get webhook secret: %w", models.NewGenericStatusFailedDependencyError(errors.New("status-code=429 rate limit exceeded"))),
			expectedStatus: http.StatusFailedDependency,
		},
		{
			name:           "oversized payload is a request entity too large",
			err:            models.NewGenericRequestEntityTooLargeError(errors.New("http: request body too large")),
			expectedStatus: http.StatusRequestEntityTooLarge,
		},
		{
			name:           "a more specific error wrapped by a failed dependency keeps its own status",
			err:            models.NewGenericStatusFailedDependencyError(models.NewGenericNotFoundError(errors.New("secret is gone"))),
			expectedStatus: http.StatusNotFound,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			writer := httptest.NewRecorder()
			request := httptest.NewRequest(http.MethodPost, "/", nil)

			require.True(t, commonhttp.GenericErrorEncoder()(t.Context(), test.err, writer, request))
			require.Equal(t, test.expectedStatus, writer.Code)
		})
	}
}
