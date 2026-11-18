package oasmiddleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/getkin/kin-openapi/routers"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apiv3 "github.com/openmeterio/openmeter/api/v3"
	"github.com/openmeterio/openmeter/api/v3/oasmiddleware"
)

// TestValidateResponse_Violation proves that the ValidateResponse middleware fires its
// error hook when a handler returns a response body that violates the OpenAPI spec.
//
// GET /openmeter/addons requires a 200 body with both "data" and "meta" fields.
// Returning {} omits both required fields and must trigger a violation.
func TestValidateResponse_Violation(t *testing.T) {
	swagger, err := apiv3.GetSpec()
	require.NoError(t, err)

	swagger.Servers = nil

	router, err := oasmiddleware.NewValidationRouter(t.Context(), swagger, &oasmiddleware.ValidationRouterOpts{
		DeleteServers: true,
	})
	require.NoError(t, err)

	var gotErr error
	mw := oasmiddleware.ValidateResponse(router, oasmiddleware.ValidateResponseOption{
		ResponseValidationErrorHook: func(err error, r *http.Request) {
			gotErr = err
		},
	})

	badHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{}`))
	})

	req := httptest.NewRequest(http.MethodGet, "/openmeter/addons", nil)
	rec := httptest.NewRecorder()

	mw(badHandler).ServeHTTP(rec, req)

	assert.Error(t, gotErr, "expected a validation error for missing required fields")
	t.Logf("validation error (expected): %v", gotErr)
}

// TestValidateResponse_Clean proves that a well-formed response does not trigger the error hook.
func TestValidateResponse_Clean(t *testing.T) {
	swagger, err := apiv3.GetSpec()
	require.NoError(t, err)

	swagger.Servers = nil

	router, err := oasmiddleware.NewValidationRouter(t.Context(), swagger, &oasmiddleware.ValidationRouterOpts{
		DeleteServers: true,
	})
	require.NoError(t, err)

	var gotErr error
	mw := oasmiddleware.ValidateResponse(router, oasmiddleware.ValidateResponseOption{
		ResponseValidationErrorHook: func(err error, r *http.Request) {
			gotErr = err
		},
	})

	goodHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"data":[],"meta":{"page":{"number":0,"size":100,"total":0}}}`))
	})

	req := httptest.NewRequest(http.MethodGet, "/openmeter/addons", nil)
	rec := httptest.NewRecorder()

	mw(goodHandler).ServeHTTP(rec, req)

	assert.NoError(t, gotErr, "expected no validation error for a well-formed response")
}

// TestValidateResponse_RouteFilterSkipsValidation proves that when RouteFilterHook returns false,
// the response body is neither buffered nor validated — even if it would otherwise violate the spec.
// The filter is the per-route gate that lets callers (e.g. unstable-only mode) avoid the
// buffering overhead on routes they don't care about.
func TestValidateResponse_RouteFilterSkipsValidation(t *testing.T) {
	swagger, err := apiv3.GetSpec()
	require.NoError(t, err)

	swagger.Servers = nil

	router, err := oasmiddleware.NewValidationRouter(t.Context(), swagger, &oasmiddleware.ValidationRouterOpts{
		DeleteServers: true,
	})
	require.NoError(t, err)

	var (
		gotErr        error
		filteredRoute *routers.Route
	)
	mw := oasmiddleware.ValidateResponse(router, oasmiddleware.ValidateResponseOption{
		RouteFilterHook: func(route *routers.Route) bool {
			filteredRoute = route
			return false
		},
		ResponseValidationErrorHook: func(err error, r *http.Request) {
			gotErr = err
		},
	})

	// Body that would fail validation if the filter let it through.
	badHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{}`))
	})

	req := httptest.NewRequest(http.MethodGet, "/openmeter/addons", nil)
	rec := httptest.NewRecorder()

	mw(badHandler).ServeHTTP(rec, req)

	require.NotNil(t, filteredRoute, "RouteFilterHook should have been invoked with the matched route")
	assert.Equal(t, "/openmeter/addons", filteredRoute.Path)
	assert.NoError(t, gotErr, "validation error hook must not fire when the filter returns false")
	assert.Equal(t, http.StatusOK, rec.Code, "client response should still be served")
	assert.Equal(t, `{}`, rec.Body.String(), "client response body should still be served")
}

// TestValidateRequest_PaginationBounds proves that the request validator enforces
// the page bounds declared in the spec, so list handlers only see valid pages.
func TestValidateRequest_PaginationBounds(t *testing.T) {
	swagger, err := apiv3.GetSpec()
	require.NoError(t, err)

	router, err := oasmiddleware.NewValidationRouter(t.Context(), swagger, &oasmiddleware.ValidationRouterOpts{
		DeleteServers: true,
	})
	require.NoError(t, err)

	for _, tc := range []struct {
		method string
		target string
		valid  bool
	}{
		{method: http.MethodGet, target: "/openmeter/customers?page[size]=1000&page[number]=1", valid: true},
		{method: http.MethodGet, target: "/openmeter/customers?page[size]=1001"},
		{method: http.MethodGet, target: "/openmeter/customers?page[size]=0"},
		{method: http.MethodGet, target: "/openmeter/customers?page[number]=0"},
		{method: http.MethodGet, target: "/openmeter/customers?page[size]=9223372036854775807"},
		{method: http.MethodGet, target: "/openmeter/events?page[size]=1000", valid: true},
		{method: http.MethodGet, target: "/openmeter/events?page[size]=1001"},
		// Entitlement access has a smaller limit than the common one.
		{method: http.MethodPost, target: "/openmeter/entitlement-access/query?page[size]=101"},
	} {
		t.Run(tc.method+" "+tc.target, func(t *testing.T) {
			var gotErr error
			mw := oasmiddleware.ValidateRequest(router, oasmiddleware.ValidateRequestOption{
				RouteNotFoundHook: func(err error, w http.ResponseWriter, r *http.Request) bool {
					t.Fatalf("route not found: %v", err)
					return true
				},
				RouteValidationErrorHook: func(err error, w http.ResponseWriter, r *http.Request) bool {
					gotErr = err
					return true
				},
			})

			mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})).
				ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(tc.method, tc.target, nil))

			if !tc.valid {
				require.ErrorContains(t, gotErr, `parameter "page"`)
				return
			}

			require.NoError(t, gotErr)
		})
	}
}
