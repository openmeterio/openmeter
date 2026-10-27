package customerscredits

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/samber/mo"
	"github.com/stretchr/testify/require"

	api "github.com/openmeterio/openmeter/api/v3"
	"github.com/openmeterio/openmeter/api/v3/apierrors"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/creditpurchase"
	"github.com/openmeterio/openmeter/openmeter/billing/creditgrant"
	"github.com/openmeterio/openmeter/openmeter/ledger"
	"github.com/openmeterio/openmeter/pkg/pagination"
	"github.com/openmeterio/openmeter/pkg/sortx"
)

type grantListService struct {
	creditgrant.Service
	input *creditgrant.ListInput
}

func (s *grantListService) List(_ context.Context, input creditgrant.ListInput) (pagination.Result[creditpurchase.Charge], error) {
	s.input = &input
	return pagination.Result[creditpurchase.Charge]{}, input.Validate()
}

type grantListServer struct {
	api.ServerInterface
	handler *handler
}

func (s grantListServer) ListCreditGrants(w http.ResponseWriter, r *http.Request, customerID api.ULID, params api.ListCreditGrantsParams) {
	s.handler.ListCreditGrants().With(ListCreditGrantsParams{CustomerID: customerID, Params: params}).ServeHTTP(w, r)
}

func TestListCreditGrantsSpendabilityQuery(t *testing.T) {
	for _, tc := range []struct {
		query    string
		features mo.Option[ledger.FeatureFilters]
		invalid  bool
	}{
		{},
		{query: "filter[feature_key]=a", features: mo.Some(ledger.FeatureFilters{"a"})},
		{query: "filter[feature_key][eq]=a", features: mo.Some(ledger.FeatureFilters{"a"})},
		{query: "filter[feature_key][oeq]=b,a,b", features: mo.Some(ledger.FeatureFilters{"a", "b"})},
		{query: "filter[feature_key][exists]=false", features: mo.Some[ledger.FeatureFilters](nil)},
		{query: "filter[unknown]=a", invalid: true},
		{query: "filter[feature_key][eq]=", invalid: true},
		{query: "filter[feature_key][contains]=a", invalid: true},
		{query: "filter[feature_key][neq]=a", invalid: true},
		{query: "filter[feature_key][exists]=true", invalid: true},
		{query: "filter[feature_key][eq]=a&filter[feature_key][oeq]=b", invalid: true},
		{query: "filter[feature_key][eq]=a&filter[feature_key][exists]=false", invalid: true},
	} {
		t.Run(tc.query, func(t *testing.T) {
			// Given the generated HTTP binding and the real grant list handler.
			service := &grantListService{}
			h := &handler{resolveNamespace: func(context.Context) (string, error) { return "ns", nil }, creditGrantService: service}
			router := api.Handler(grantListServer{handler: h})
			// When a deep-object query reaches the endpoint.
			req := httptest.NewRequest(http.MethodGet, "/openmeter/customers/01ARZ3NDEKTSV4RRFFQ69G5FAV/credits/grants?"+tc.query, nil)
			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, req)
			// Then invalid queries fail before listing and valid filters reach the service.
			if tc.invalid {
				require.Equal(t, http.StatusBadRequest, rec.Code, rec.Body.String())
				require.Nil(t, service.input)

				return
			}

			require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
			require.NotNil(t, service.input)
			require.Equal(t, tc.features, service.input.FeatureFilters)
		})
	}
}

func TestListCreditGrantsSortQuery(t *testing.T) {
	for _, tc := range []struct {
		name    string
		orderBy creditpurchase.OrderBy
		order   sortx.Order
		invalid bool
	}{
		{name: "default", orderBy: creditpurchase.OrderByCreatedAt, order: sortx.OrderDesc},
		{name: "created_at", orderBy: creditpurchase.OrderByCreatedAt, order: sortx.OrderAsc},
		{name: "created_at asc", orderBy: creditpurchase.OrderByCreatedAt, order: sortx.OrderAsc},
		{name: "created_at desc", orderBy: creditpurchase.OrderByCreatedAt, order: sortx.OrderDesc},
		{name: "updated_at desc", orderBy: creditpurchase.OrderByUpdatedAt, order: sortx.OrderDesc},
		{name: "name", orderBy: creditpurchase.OrderByName, order: sortx.OrderAsc},
		{name: "id desc", orderBy: creditpurchase.OrderByID, order: sortx.OrderDesc},
		{name: "", invalid: true},
		{name: "createdAt", invalid: true},
		{name: "unknown", invalid: true},
		{name: "created_at invalid", invalid: true},
		{name: "created_at desc extra", invalid: true},
		{name: "created_at,name", invalid: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			// given: the generated HTTP binding and the real grant list handler.
			service := &grantListService{}
			h := &handler{resolveNamespace: func(context.Context) (string, error) { return "ns", nil }, creditGrantService: service}
			router := api.Handler(grantListServer{handler: h})

			// when: a sort query reaches the endpoint.
			query := url.Values{}
			if tc.name != "default" {
				query.Set("sort", tc.name)
			}

			req := httptest.NewRequest(http.MethodGet, "/openmeter/customers/01ARZ3NDEKTSV4RRFFQ69G5FAV/credits/grants?"+query.Encode(), nil)
			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, req)

			// then: invalid sorts fail before listing; valid sorts reach the service.
			if tc.invalid {
				require.Equal(t, http.StatusBadRequest, rec.Code, rec.Body.String())
				require.Nil(t, service.input)
				var problem struct {
					InvalidParameters apierrors.InvalidParameters `json:"invalid_parameters"`
				}
				require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &problem))
				require.Len(t, problem.InvalidParameters, 1)
				require.Equal(t, "sort", problem.InvalidParameters[0].Field)
				require.Equal(t, apierrors.InvalidParamSourceQuery, problem.InvalidParameters[0].Source)

				return
			}

			require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
			require.NotNil(t, service.input)
			require.Equal(t, tc.orderBy, service.input.OrderBy)
			require.Equal(t, tc.order, service.input.Order)
		})
	}
}
