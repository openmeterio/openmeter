package test_test

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/getkin/kin-openapi/openapi3filter"
	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/require"

	api "github.com/openmeterio/openmeter/api/v3"
	"github.com/openmeterio/openmeter/api/v3/oasmiddleware"
)

func TestListCreditGrantsCurrencyFilterValidation(t *testing.T) {
	// given: the production OpenAPI request validator and grant list schema
	doc, err := api.GetSpec()
	require.NoError(t, err)
	validationRouter, err := oasmiddleware.NewValidationRouter(t.Context(), doc, &oasmiddleware.ValidationRouterOpts{
		DeleteServers: true,
	})
	require.NoError(t, err)

	router := chi.NewRouter()
	router.Use(oasmiddleware.ValidateRequest(validationRouter, oasmiddleware.ValidateRequestOption{
		RouteNotFoundHook: oasmiddleware.OasRouteNotFoundErrorHook,
		RouteValidationErrorHook: func(err error, w http.ResponseWriter, request *http.Request) bool {
			return oasmiddleware.OasValidationErrorHook(request.Context(), err, w, request)
		},
		FilterOptions: &openapi3filter.Options{
			AuthenticationFunc: openapi3filter.NoopAuthenticationFunc,
			MultiError:         true,
		},
	}))
	router.Get("/openmeter/customers/{customerId}/credits/grants", noopHandler)

	for _, tc := range []struct {
		currency string
		status   int
	}{
		{currency: "USD", status: http.StatusNoContent},
		{currency: "TOKENS", status: http.StatusNoContent},
		{currency: "credits_2026", status: http.StatusNoContent},
		{currency: "AB", status: http.StatusBadRequest},
		{currency: "ABCDEFGHIJKLMNOPQRSTUVWXYZ", status: http.StatusBadRequest},
	} {
		t.Run(tc.currency, func(t *testing.T) {
			// when: filtering by a fiat, custom, or malformed currency code
			query := url.Values{"filter[currency]": {tc.currency}}
			request := httptest.NewRequest(http.MethodGet, "/openmeter/customers/01K3Q8QG4B9YJYJVQ3MZ3V8KT6/credits/grants?"+query.Encode(), nil)
			response := httptest.NewRecorder()
			router.ServeHTTP(response, request)

			// then: both currency types reach the handler, malformed codes do not
			require.Equal(t, tc.status, response.Code, response.Body.String())
		})
	}
}
