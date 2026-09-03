package customerscredits

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/creditpurchase"
	"github.com/openmeterio/openmeter/openmeter/billing/creditgrant"
)

type getCreditGrantErrorService struct {
	creditgrant.Service
}

func (getCreditGrantErrorService) Get(context.Context, creditgrant.GetInput) (creditpurchase.Charge, error) {
	return creditpurchase.Charge{}, fmt.Errorf("setting up billing: %w", billing.ValidationError{
		Err: billing.NewValidationError("profile_not_found", "profile not found"),
	})
}

func TestGetCreditGrantBillingErrorEncoder(t *testing.T) {
	handler := New(
		func(context.Context) (string, error) { return "namespace", nil },
		nil,
		nil,
		getCreditGrantErrorService{},
		nil,
		nil,
	)
	request := httptest.NewRequest(http.MethodGet, "/api/v3/customers/customer-id/credits/grants/grant-id", nil)
	response := httptest.NewRecorder()

	handler.GetCreditGrant().With(GetCreditGrantParams{
		CustomerID:    "customer-id",
		CreditGrantID: "grant-id",
	}).ServeHTTP(response, request)

	require.Equal(t, http.StatusBadRequest, response.Code)
	require.Contains(t, response.Body.String(), `"code":"profile_not_found"`)
}
