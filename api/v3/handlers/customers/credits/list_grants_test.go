package customerscredits

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/samber/mo"
	"github.com/stretchr/testify/require"

	api "github.com/openmeterio/openmeter/api/v3"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/creditpurchase"
	"github.com/openmeterio/openmeter/openmeter/billing/creditgrant"
	"github.com/openmeterio/openmeter/openmeter/ledger"
	"github.com/openmeterio/openmeter/pkg/pagination"
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
