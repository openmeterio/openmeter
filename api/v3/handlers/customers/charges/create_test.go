package charges

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCreateCustomerChargeRejectsUnitConfigWhenDisabled(t *testing.T) {
	handler := New(
		func(context.Context) (string, error) { return "namespace", nil },
		nil,
		false,
	)

	body := []byte(`{
		"type":"usage_based",
		"name":"usage charge",
		"currency":"USD",
		"feature_id":"feature-id",
		"invoice_at":"2026-07-06T11:00:00Z",
		"service_period":{"from":"2026-07-06T10:00:00Z","to":"2026-07-06T11:00:00Z"},
		"settlement_mode":"credit_then_invoice",
		"price":{"type":"unit","amount":"0.25"},
		"unit_config":{"operation":"divide","conversion_factor":"1000"}
	}`)
	request := httptest.NewRequest(http.MethodPost, "/api/v3/openmeter/customers/customer-id/charges", bytes.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()

	handler.CreateCustomerCharge().With(CreateCustomerChargesParams{
		CustomerID: "customer-id",
	}).ServeHTTP(response, request)

	require.Equal(t, http.StatusBadRequest, response.Code)
	require.Contains(t, response.Body.String(), "unit_config is not enabled")
}
