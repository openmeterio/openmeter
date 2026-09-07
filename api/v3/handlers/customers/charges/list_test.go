package charges

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/samber/lo"
	"github.com/stretchr/testify/require"

	api "github.com/openmeterio/openmeter/api/v3"
	"github.com/openmeterio/openmeter/api/v3/filters"
	billingcharges "github.com/openmeterio/openmeter/openmeter/billing/charges"
	"github.com/openmeterio/openmeter/pkg/pagination"
)

type listCustomerChargesCaptureService struct {
	billingcharges.Service
	input billingcharges.ListCustomerChargesInput
}

func (s *listCustomerChargesCaptureService) ListCustomerCharges(_ context.Context, input billingcharges.ListCustomerChargesInput) (billingcharges.ListCustomerChargesResult, error) {
	s.input = input

	return billingcharges.ListCustomerChargesResult{
		Charges: pagination.Result[billingcharges.CustomerCharge]{
			Page: input.Page,
		},
	}, nil
}

func TestListCustomerChargesValidationIssuePresenceFilter(t *testing.T) {
	tests := []struct {
		name   string
		exists bool
	}{
		{
			name:   "with validation issues",
			exists: true,
		},
		{
			name:   "without validation issues",
			exists: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := &listCustomerChargesCaptureService{}
			handler := New(
				func(context.Context) (string, error) { return "namespace", nil },
				service,
				false,
			)
			request := httptest.NewRequest(http.MethodGet, "/api/v3/customers/customer-id/charges", nil)
			response := httptest.NewRecorder()

			handler.ListCustomerCharges().With(ListCustomerChargesParams{
				CustomerID: "01ARZ3NDEKTSV4RRFFQ69G5FAV",
				Params: api.ListCustomerChargesParams{
					Filter: &api.ListChargesParamsFilter{
						ValidationIssues: &filters.FilterPresence{Exists: lo.ToPtr(tt.exists)},
					},
				},
			}).ServeHTTP(response, request)

			require.Equal(t, http.StatusOK, response.Code)
			require.NotNil(t, service.input.HasValidationIssues)
			require.Equal(t, lo.ToPtr(tt.exists), service.input.HasValidationIssues.Eq)
		})
	}
}
