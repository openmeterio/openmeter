package customerscredits

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/stretchr/testify/require"

	api "github.com/openmeterio/openmeter/api/v3"
	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/openmeter/ledger"
	"github.com/openmeterio/openmeter/openmeter/ledger/customerbalance"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/pagination"
	"github.com/openmeterio/openmeter/pkg/sortx"
)

type listAllServer struct {
	api.ServerInterface
	handler *handler
}

func (s listAllServer) ListAllCreditGrants(w http.ResponseWriter, r *http.Request, params api.ListAllCreditGrantsParams) {
	s.handler.ListAllCreditGrants().With(params).ServeHTTP(w, r)
}

func (s listAllServer) ListAllCreditBalances(w http.ResponseWriter, r *http.Request, params api.ListAllCreditBalancesParams) {
	s.handler.ListAllCreditBalances().With(params).ServeHTTP(w, r)
}

func TestListAllCreditGrantsCustomerScope(t *testing.T) {
	for _, tc := range []struct {
		query       string
		customerIDs []string
		invalid     bool
	}{
		{query: "", customerIDs: nil},
		{query: "filter[customer_id][eq]=01ARZ3NDEKTSV4RRFFQ69G5FAV", customerIDs: []string{"01ARZ3NDEKTSV4RRFFQ69G5FAV"}},
		{query: "filter[customer_id][oeq]=01ARZ3NDEKTSV4RRFFQ69G5FAV,01ARZ3NDEKTSV4RRFFQ69G5FAW", customerIDs: []string{"01ARZ3NDEKTSV4RRFFQ69G5FAV", "01ARZ3NDEKTSV4RRFFQ69G5FAW"}},
		{query: "filter[customer_id][neq]=01ARZ3NDEKTSV4RRFFQ69G5FAV", invalid: true},
	} {
		t.Run(tc.query, func(t *testing.T) {
			// given: the generated HTTP binding and the real namespace-wide grant handler.
			service := &grantListService{}
			h := &handler{resolveNamespace: func(context.Context) (string, error) { return "ns", nil }, creditGrantService: service}
			router := api.Handler(listAllServer{handler: h})

			// when: the namespace-wide grant listing is requested.
			req := httptest.NewRequest(http.MethodGet, "/openmeter/credits/grants?"+tc.query, nil)
			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, req)

			// then: the customer filter selects the customer scope; no filter lists every customer.
			if tc.invalid {
				require.Equal(t, http.StatusBadRequest, rec.Code, rec.Body.String())
				require.Nil(t, service.input)

				return
			}

			require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
			require.NotNil(t, service.input)
			require.Equal(t, "ns", service.input.Namespace)
			require.Equal(t, tc.customerIDs, service.input.CustomerIDs)
		})
	}
}

type balanceListCustomerService struct {
	customer.Service
	customers []customer.Customer
	input     *customer.ListCustomersInput
}

func (s *balanceListCustomerService) ListCustomers(_ context.Context, input customer.ListCustomersInput) (pagination.Result[customer.Customer], error) {
	s.input = &input
	return pagination.Result[customer.Customer]{Page: input.Page, Items: s.customers, TotalCount: len(s.customers)}, nil
}

type balanceListFacade struct {
	customerBalanceFacade
	mu       sync.Mutex
	inputs   []customerbalance.GetBalancesInput
	balances map[string][]customerbalance.BalanceByCurrency
	errs     map[string]error
}

func (f *balanceListFacade) GetBalances(_ context.Context, input customerbalance.GetBalancesInput) ([]customerbalance.BalanceByCurrency, error) {
	f.mu.Lock()
	f.inputs = append(f.inputs, input)
	f.mu.Unlock()

	return f.balances[input.CustomerID.ID], f.errs[input.CustomerID.ID]
}

func TestListAllCreditBalances(t *testing.T) {
	const (
		funded   = "01ARZ3NDEKTSV4RRFFQ69G5FA1"
		noLedger = "01ARZ3NDEKTSV4RRFFQ69G5FA2"
	)

	newHarness := func() (*balanceListCustomerService, *balanceListFacade, http.Handler) {
		customers := &balanceListCustomerService{customers: []customer.Customer{
			{ManagedResource: models.ManagedResource{NamespacedModel: models.NamespacedModel{Namespace: "ns"}, ID: funded}},
			{ManagedResource: models.ManagedResource{NamespacedModel: models.NamespacedModel{Namespace: "ns"}, ID: noLedger}},
		}}
		facade := &balanceListFacade{
			balances: map[string][]customerbalance.BalanceByCurrency{
				funded: {{Currency: currencyx.Code("USD"), Balance: staticCreditBalance{booked: alpacadecimal.NewFromInt(150), spendable: alpacadecimal.NewFromInt(120)}}},
			},
			errs: map[string]error{
				noLedger: ledger.ErrCustomerAccountMissing,
			},
		}
		h := &handler{
			resolveNamespace: func(context.Context) (string, error) { return "ns", nil },
			customerService:  customers,
			balanceFacade:    facade,
		}

		return customers, facade, api.Handler(listAllServer{handler: h})
	}

	t.Run("historical page keeps customer order and lists customers without ledger accounts as empty", func(t *testing.T) {
		// given: one funded customer and one customer that predates credits.
		customers, facade, router := newHarness()

		// when: a close job reads balances at the end of a day.
		req := httptest.NewRequest(http.MethodGet, "/openmeter/credits/balances?timestamp=2026-09-30T23:59:59Z&page[size]=100", nil)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		// then: customers are walked in ID order, the timestamp is a historical read, and
		// the customer without ledger accounts reads as having no balances.
		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
		require.Equal(t, "id", customers.input.OrderBy)
		require.Equal(t, sortx.OrderAsc, customers.input.Order)

		asOf := time.Date(2026, 9, 30, 23, 59, 59, 0, time.UTC)
		require.Len(t, facade.inputs, 2)
		for _, input := range facade.inputs {
			require.Equal(t, &asOf, input.AsOf)
		}

		var body struct {
			Data []api.BillingCustomerCreditBalances `json:"data"`
		}
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
		require.Len(t, body.Data, 2)

		require.Equal(t, funded, body.Data[0].Customer.Id)
		require.Equal(t, asOf, body.Data[0].RetrievedAt)
		require.Len(t, body.Data[0].Balances, 1)
		require.Equal(t, "150", body.Data[0].Balances[0].Booked)

		require.Equal(t, noLedger, body.Data[1].Customer.Id)
		require.Empty(t, body.Data[1].Balances)
	})

	t.Run("other balance failures fail the page", func(t *testing.T) {
		// given: a balance read that fails for a reason other than missing accounts.
		_, facade, router := newHarness()
		facade.errs[funded] = errors.New("ledger unavailable")

		// when: the page is read.
		req := httptest.NewRequest(http.MethodGet, "/openmeter/credits/balances", nil)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		// then: the page is not returned with silently missing balances.
		require.Equal(t, http.StatusInternalServerError, rec.Code, rec.Body.String())
	})

	t.Run("page size above the cap is rejected", func(t *testing.T) {
		// given: the real handler.
		customers, _, router := newHarness()

		// when: a page larger than the per-request computation cap is requested.
		req := httptest.NewRequest(http.MethodGet, "/openmeter/credits/balances?page[size]=101", nil)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		// then: the request fails before any customer is read.
		require.Equal(t, http.StatusBadRequest, rec.Code, rec.Body.String())
		require.Nil(t, customers.input)
	})
}
