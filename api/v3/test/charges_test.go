package test_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/getkin/kin-openapi/openapi3filter"
	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/require"

	api "github.com/openmeterio/openmeter/api/v3"
	"github.com/openmeterio/openmeter/api/v3/oasmiddleware"
)

func TestCreateFlatFeeChargeRejectsUsageBasedPrice(t *testing.T) {
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
	router.Post("/openmeter/customers/{customerId}/charges", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})

	request := httptest.NewRequest(http.MethodPost, "/openmeter/customers/01K3Q8QG4B9YJYJVQ3MZ3V8KT6/charges", strings.NewReader(`{
		"type":"flat_fee",
		"name":"flat fee charge",
		"currency":"USD",
		"invoice_at":"2026-07-06T11:00:00Z",
		"service_period":{"from":"2026-07-06T10:00:00Z","to":"2026-07-06T11:00:00Z"},
		"settlement_mode":"credit_then_invoice",
		"payment_term":"in_advance",
		"proration_configuration":{"mode":"no_proration"},
		"amount_before_proration":{"currency":"USD","amount":"10"},
		"price":{"type":"unit","amount":"0.25"}
	}`))
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()

	router.ServeHTTP(response, request)

	require.Equal(t, http.StatusBadRequest, response.Code)
	require.Contains(t, response.Body.String(), "price")
}
