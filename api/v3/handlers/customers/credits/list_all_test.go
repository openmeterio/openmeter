package customerscredits

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"reflect"
	"sync"
	"testing"

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
		{query: "filter[customer_id][eq]=", invalid: true},
		{query: "filter[customer_id]=", invalid: true},
		{query: "filter[customer_id][oeq]=", invalid: true},
		{query: "filter[updated_at][gte]=2026-10-01T00:00:00Z", invalid: true},
		{query: "page[size]=0", invalid: true},
		{query: "page[size]=1001", invalid: true},
		{query: "page[size]=1000", customerIDs: nil},
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
			require.Equal(t, len(tc.customerIDs) == 0, service.input.AllCustomers)
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
				noLedger: ledger.ErrCustomerWithoutAccounts,
			},
		}
		h := &handler{
			resolveNamespace: func(context.Context) (string, error) { return "ns", nil },
			customerService:  customers,
			balanceFacade:    facade,
		}

		return customers, facade, api.Handler(listAllServer{handler: h})
	}

	t.Run("current page keeps customer order and lists customers without ledger accounts as empty", func(t *testing.T) {
		// given: one funded customer and one customer that predates credits.
		customers, facade, router := newHarness()

		// when: current balances are requested.
		req := httptest.NewRequest(http.MethodGet, "/openmeter/credits/balances?page[size]=100", nil)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		// then: customers are walked in ID order with live balance reads, and
		// the customer without ledger accounts reads as having no balances.
		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
		require.Equal(t, "id", customers.input.OrderBy)
		require.Equal(t, sortx.OrderAsc, customers.input.Order)

		require.Len(t, facade.inputs, 2)
		for _, input := range facade.inputs {
			require.Nil(t, input.AsOf)
		}

		var body struct {
			Data []api.BillingCustomerCreditBalances `json:"data"`
		}
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
		require.Len(t, body.Data, 2)

		require.Equal(t, funded, body.Data[0].Customer.Id)
		require.False(t, body.Data[0].RetrievedAt.IsZero())
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

	t.Run("a partly provisioned customer fails the page", func(t *testing.T) {
		for _, missing := range []ledger.AccountType{ledger.AccountTypeCustomerFBO, ledger.AccountTypeCustomerAccrued} {
			// given: a customer with some ledger accounts but not this one.
			_, facade, router := newHarness()
			facade.errs[funded] = ledger.ErrCustomerAccountMissing.WithAttr("account_type", missing)

			// when: the page is read.
			req := httptest.NewRequest(http.MethodGet, "/openmeter/credits/balances", nil)
			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, req)

			// then: the customer's balance isn't silently reported as empty.
			require.Equal(t, http.StatusInternalServerError, rec.Code, rec.Body.String())
		}
	})

	t.Run("an empty currency filter is rejected instead of reading every currency", func(t *testing.T) {
		for _, query := range []string{"filter[currency][oeq]=", "filter[currency][eq]=", "filter[currency]="} {
			// given: the real handler.
			customers, _, router := newHarness()

			// when: the currency filter is present but selects no code.
			req := httptest.NewRequest(http.MethodGet, "/openmeter/credits/balances?"+query, nil)
			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, req)

			// then: the request fails before any customer is read.
			require.Equal(t, http.StatusBadRequest, rec.Code, query+": "+rec.Body.String())
			require.Nil(t, customers.input, query)
		}
	})

	t.Run("page sizes outside 1 to 100 are rejected", func(t *testing.T) {
		// A zero size would remove the query limit and read every customer.
		for _, size := range []string{"0", "101"} {
			// given: the real handler.
			customers, _, router := newHarness()

			// when: a page outside the per-request computation cap is requested.
			req := httptest.NewRequest(http.MethodGet, "/openmeter/credits/balances?page[size]="+size, nil)
			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, req)

			// then: the request fails before any customer is read.
			require.Equal(t, http.StatusBadRequest, rec.Code, size+": "+rec.Body.String())
			require.Nil(t, customers.input, size)
		}
	})
}

// The bulk filter spreads the per-customer one in TypeSpec but is copied field by
// field in the handler; a field added to one without the other would be dropped
// silently, returning unfiltered rows.
func TestListAllCreditGrantsFilterCopiesEveryField(t *testing.T) {
	fieldNames := func(v any) []string {
		typ := reflect.TypeOf(v)
		names := make([]string, 0, typ.NumField())
		for i := range typ.NumField() {
			names = append(names, typ.Field(i).Name)
		}

		return names
	}

	require.ElementsMatch(t,
		append(fieldNames(api.ListCreditGrantsParamsFilter{}), "CustomerId"),
		fieldNames(api.ListAllCreditGrantsParamsFilter{}),
		"update the filter copy in ListAllCreditGrants",
	)
}
