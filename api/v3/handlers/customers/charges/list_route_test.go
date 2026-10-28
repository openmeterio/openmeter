package charges

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/samber/lo"
	"github.com/stretchr/testify/require"

	api "github.com/openmeterio/openmeter/api/v3"
	billingcharges "github.com/openmeterio/openmeter/openmeter/billing/charges"
)

type customerChargesListService struct {
	billingcharges.Service
	input *billingcharges.ListCustomerChargesInput
}

func (s *customerChargesListService) ListCustomerCharges(_ context.Context, input billingcharges.ListCustomerChargesInput) (billingcharges.ListCustomerChargesResult, error) {
	s.input = &input
	return billingcharges.ListCustomerChargesResult{}, nil
}

type customerChargesListServer struct {
	api.ServerInterface
	handler Handler
}

func (s customerChargesListServer) ListCustomerCharges(w http.ResponseWriter, r *http.Request, customerID api.ULID, params api.ListCustomerChargesParams) {
	s.handler.ListCustomerCharges().With(ListCustomerChargesParams{CustomerID: customerID, Params: params}).ServeHTTP(w, r)
}

func TestListCustomerChargesForwardsUpdatedAt(t *testing.T) {
	// given: the generated HTTP binding and the real customer-scoped charge handler.
	service := &customerChargesListService{}
	h := New(func(context.Context) (string, error) { return "ns", nil }, service, false)
	router := api.Handler(customerChargesListServer{handler: h})

	// when: the customer-scoped listing is filtered by update time.
	req := httptest.NewRequest(http.MethodGet, "/openmeter/customers/01ARZ3NDEKTSV4RRFFQ69G5FAV/charges?filter[updated_at][gte]=2026-10-01T00:00:00Z", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	// then: the filter reaches the service scoped to that customer.
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	require.NotNil(t, service.input)
	require.Equal(t, []string{"01ARZ3NDEKTSV4RRFFQ69G5FAV"}, service.input.CustomerIDs)
	require.NotNil(t, service.input.UpdatedAt)
	require.Equal(t, time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC), lo.FromPtr(service.input.UpdatedAt.Gte).UTC())
}
