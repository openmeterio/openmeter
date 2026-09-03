package charges

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/billing"
	billingcharges "github.com/openmeterio/openmeter/openmeter/billing/charges"
)

type listCustomerChargesErrorService struct {
	billingcharges.Service
	err error
}

func (s listCustomerChargesErrorService) ListCustomerCharges(context.Context, billingcharges.ListCustomerChargesInput) (billingcharges.ListCustomerChargesResult, error) {
	return billingcharges.ListCustomerChargesResult{}, s.err
}

func TestListCustomerChargesBillingErrorEncoder(t *testing.T) {
	tests := []struct {
		name       string
		err        error
		statusCode int
	}{
		{
			name: "billing validation error",
			err: billing.ValidationError{
				Err: billing.NewValidationError("namespace_locked", "namespace is locked"),
			},
			statusCode: http.StatusBadRequest,
		},
		{
			name:       "unrelated system error",
			err:        errors.New("database unavailable"),
			statusCode: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := New(
				func(context.Context) (string, error) { return "namespace", nil },
				listCustomerChargesErrorService{err: tt.err},
				false,
			)
			request := httptest.NewRequest(http.MethodGet, "/api/v3/customers/customer-id/charges", nil)
			response := httptest.NewRecorder()

			handler.ListCustomerCharges().With(ListCustomerChargesParams{
				CustomerID: "customer-id",
			}).ServeHTTP(response, request)

			require.Equal(t, tt.statusCode, response.Code)
		})
	}
}
