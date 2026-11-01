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
	"github.com/openmeterio/openmeter/openmeter/billing/charges/meta"
)

type asOfChargesService struct {
	billingcharges.Service
	input *billingcharges.ListCustomerChargesInput
}

func (s *asOfChargesService) ListCustomerCharges(_ context.Context, input billingcharges.ListCustomerChargesInput) (billingcharges.ListCustomerChargesResult, error) {
	s.input = &input
	return billingcharges.ListCustomerChargesResult{}, nil
}

type asOfChargesServer struct {
	api.ServerInterface
	handler Handler
}

func (s asOfChargesServer) ListCharges(w http.ResponseWriter, r *http.Request, params api.ListChargesParams) {
	s.handler.ListCharges().With(params).ServeHTTP(w, r)
}

func (s asOfChargesServer) ListCustomerCharges(w http.ResponseWriter, r *http.Request, customerID api.ULID, params api.ListCustomerChargesParams) {
	s.handler.ListCustomerCharges().With(ListCustomerChargesParams{CustomerID: customerID, Params: params}).ServeHTTP(w, r)
}

// Without this mapping a past-date read would silently return live usage.
func TestListChargesForwardsAsOf(t *testing.T) {
	asOf := time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)

	for _, path := range []string{
		"/openmeter/charges",
		"/openmeter/customers/01ARZ3NDEKTSV4RRFFQ69G5FAV/charges",
	} {
		t.Run(path, func(t *testing.T) {
			// given: the generated HTTP binding and the real charge handlers.
			service := &asOfChargesService{}
			h := New(func(context.Context) (string, error) { return "ns", nil }, service, false)
			router := api.Handler(asOfChargesServer{handler: h})

			// when: realtime usage is read as of a past time.
			req := httptest.NewRequest(http.MethodGet, path+"?expand=real_time_usage&as_of=2026-10-01T00:00:00Z", nil)
			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, req)

			// then: the cutoff reaches the service with the expand it requires.
			require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
			require.NotNil(t, service.input)
			require.Equal(t, asOf, lo.FromPtr(service.input.RealtimeUsageAsOf).UTC())
			require.True(t, service.input.Expands.Has(meta.ExpandRealtimeUsage))
		})
	}
}
