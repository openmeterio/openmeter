package test_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/getkin/kin-openapi/openapi3filter"
	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/require"

	api "github.com/openmeterio/openmeter/api/v3"
	"github.com/openmeterio/openmeter/api/v3/apierrors"
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

func TestCreateCreditGrantNumericValidation(t *testing.T) {
	// given: the production OpenAPI request validator and grant create schema
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
	router.Post("/openmeter/customers/{customerId}/credits/grants", noopHandler)

	const bodyTemplate = `{"name":"grant","amount":%q,"priority":10,"currency":"KCOIN","funding_method":"invoice","purchase":{"per_unit_cost_basis":%q,"availability_policy":"on_creation","currency":"USD"}}`

	for _, tc := range []struct {
		name      string
		amount    string
		costBasis string
		status    int
		field     string
	}{
		{name: "integer amount", amount: "100", costBasis: "1", status: http.StatusNoContent},
		{name: "decimal amount", amount: "100.00", costBasis: "0.5", status: http.StatusNoContent},
		{name: "amount with trailing decimal point", amount: "100.", costBasis: "1", status: http.StatusBadRequest, field: "amount"},
		{name: "nested cost basis with trailing decimal point", amount: "100", costBasis: "1.", status: http.StatusBadRequest, field: "purchase.per_unit_cost_basis"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			// when: creating a grant with a well-formed or malformed numeric
			request := httptest.NewRequest(http.MethodPost, "/openmeter/customers/01K3Q8QG4B9YJYJVQ3MZ3V8KT6/credits/grants", strings.NewReader(fmt.Sprintf(bodyTemplate, tc.amount, tc.costBasis)))
			request.Header.Set("Content-Type", "application/json")
			response := httptest.NewRecorder()
			router.ServeHTTP(response, request)

			// then: malformed numerics are rejected naming the violated pattern, not the allOf wrapper
			require.Equal(t, tc.status, response.Code, response.Body.String())
			if tc.status != http.StatusBadRequest {
				return
			}

			var body struct {
				InvalidParameters []apierrors.InvalidParameter `json:"invalid_parameters"`
			}
			require.NoError(t, json.Unmarshal(response.Body.Bytes(), &body))
			require.Len(t, body.InvalidParameters, 1, response.Body.String())
			require.Equal(t, tc.field, body.InvalidParameters[0].Field)
			require.Equal(t, "pattern", body.InvalidParameters[0].Rule)
			require.Contains(t, body.InvalidParameters[0].Reason, "regular expression")
		})
	}
}
