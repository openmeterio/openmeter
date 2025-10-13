package commonhttp_test

import (
	"encoding/json"
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
