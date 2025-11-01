package server

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/pkg/framework/commonhttp"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport"
	"github.com/openmeterio/openmeter/pkg/models"
)

const ErrCodeSomething models.ErrorCode = "something"

func TestIssueIfHTTPStatusKnownErrorResponses(t *testing.T) {
	t.Run("Should map http.StatusBadRequest to errors", func(t *testing.T) {
		testHandler := httptransport.NewHandler(
			func(ctx context.Context, r *http.Request) (any, error) {
				return nil, nil
			},
			func(ctx context.Context, req any) (any, error) {
				// We'll just return a validation issue to see it get mapped
				return nil, models.NewValidationIssue(ErrCodeSomething, "something went wrong", commonhttp.WithHTTPStatusCodeAttribute(http.StatusBadRequest), models.WithAttribute("expectedKey", "expectedValue"))
			},
			commonhttp.JSONResponseEncoderWithStatus[any](http.StatusOK),
		)

		r := chi.NewRouter()
		r.Handle("/", testHandler)

		req := httptest.NewRequest(http.MethodGet, "/", bytes.NewReader(nil))

		// Make request
		writer := httptest.NewRecorder()
		r.ServeHTTP(writer, req)
		res := writer.Result()

		defer res.Body.Close()

		// status
		require.Equal(t, http.StatusBadRequest, res.StatusCode, writer.Body.String())

		// Let's parse the body to json
		var body map[string]interface{}
		err := json.Unmarshal(writer.Body.Bytes(), &body)
		require.NoError(t, err)

		// extensions
		extensions, ok := body["extensions"].(map[string]interface{})
		require.True(t, ok)

		issues, ok := extensions["validationErrors"].([]interface{})
		require.True(t, ok, "got body: %+v", body)
		require.Len(t, issues, 1)

		require.Equal(t,
			map[string]interface{}{
				"code": "something",
				// See how this is removed
				// "commonhttp.httpAttributeKey:openmeter.http.status_code": 400.0,
				"expectedKey": "expectedValue",
				"message":     "something went wrong",
				"severity":    "critical",
			}, issues[0])
	})

	t.Run("Should map other statuses to errors", func(t *testing.T) {
		testHandler := httptransport.NewHandler(
			func(ctx context.Context, r *http.Request) (any, error) {
				return nil, nil
			},
			func(ctx context.Context, req any) (any, error) {
				// We'll just return a validation issue to see it get mapped
				return nil, models.NewValidationIssue(ErrCodeSomething, "something went wrong", commonhttp.WithHTTPStatusCodeAttribute(http.StatusConflict), models.WithAttribute("expectedKey", "expectedValue"))
			},
			commonhttp.JSONResponseEncoderWithStatus[any](http.StatusOK),
		)

		r := chi.NewRouter()
		r.Handle("/", testHandler)

		req := httptest.NewRequest(http.MethodGet, "/", bytes.NewReader(nil))

		// Make request
		writer := httptest.NewRecorder()
		r.ServeHTTP(writer, req)
		res := writer.Result()

		defer res.Body.Close()

		// status
		require.Equal(t, http.StatusConflict, res.StatusCode, writer.Body.String())

		// Let's parse the body to json
		var body map[string]interface{}
		err := json.Unmarshal(writer.Body.Bytes(), &body)
		require.NoError(t, err)

		// extensions
		extensions, ok := body["extensions"].(map[string]interface{})
		require.True(t, ok)

		_, ok = extensions["409"].([]interface{})
		require.False(t, ok)

		issues, ok := extensions["validationErrors"].([]interface{})
		require.True(t, ok, "got body: %+v", body)
		require.Len(t, issues, 1)

		require.Equal(t,
			map[string]interface{}{
				"code": "something",
				// See how this is removed
				// "commonhttp.httpAttributeKey:openmeter.http.status_code": 400.0,
				"expectedKey": "expectedValue",
				"message":     "something went wrong",
				"severity":    "critical",
			}, issues[0])
	})
}
