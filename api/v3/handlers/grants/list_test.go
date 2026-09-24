package grants

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/samber/lo"
	"github.com/stretchr/testify/require"

	api "github.com/openmeterio/openmeter/api/v3"
	"github.com/openmeterio/openmeter/api/v3/filters"
	"github.com/openmeterio/openmeter/api/v3/response"
	"github.com/openmeterio/openmeter/openmeter/credit/grant"
	"github.com/openmeterio/openmeter/openmeter/entitlement"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/pagination"
	"github.com/openmeterio/openmeter/pkg/sortx"
)

func serveListGrants(t *testing.T, svc fakeService, params api.ListGrantsParams) *httptest.ResponseRecorder {
	t.Helper()

	res := httptest.NewRecorder()
	newTestHandler(svc).ListGrants().With(params).ServeHTTP(res, httptest.NewRequest(http.MethodGet, "/api/v3/openmeter/grants", nil))

	return res
}

func TestListGrantsHandler(t *testing.T) {
	t.Run("applies the defaults", func(t *testing.T) {
		var received entitlement.ListNamespaceGrantsInput

		res := serveListGrants(t, fakeService{
			listGrants: func(_ context.Context, input entitlement.ListNamespaceGrantsInput) (pagination.Result[grant.Grant], error) {
				received = input
				return pagination.Result[grant.Grant]{}, nil
			},
		}, api.ListGrantsParams{})

		require.Equal(t, http.StatusOK, res.Code, res.Body.String())
		require.Equal(t, entitlement.ListNamespaceGrantsInput{
			Namespace: testNamespace,
			OrderBy:   grant.OrderByCreatedAt,
			Order:     sortx.OrderAsc,
			Page:      pagination.NewPage(1, 20),
		}, received)

		var body response.PagePaginationResponse[api.BillingEntitlementGrant]
		require.NoError(t, json.Unmarshal(res.Body.Bytes(), &body))
		require.Empty(t, body.Data)
		require.Equal(t, 0, lo.FromPtr(body.Meta.Page.Total))
	})

	t.Run("maps the query and the page", func(t *testing.T) {
		var received entitlement.ListNamespaceGrantsInput

		effectiveAt := time.Date(2025, time.January, 1, 0, 0, 0, 0, time.UTC)

		res := serveListGrants(t, fakeService{
			listGrants: func(_ context.Context, input entitlement.ListNamespaceGrantsInput) (pagination.Result[grant.Grant], error) {
				received = input
				return pagination.Result[grant.Grant]{
					Items: []grant.Grant{{
						ID:          "01K4WAQ0J99ZZ0MD75HXR112HB",
						OwnerID:     "01K4WAQ0J99ZZ0MD75HXR112HC",
						Amount:      10,
						EffectiveAt: effectiveAt,
					}},
					TotalCount: 11,
				}, nil
			},
		}, api.ListGrantsParams{
			Page: &api.PagePaginationQuery{Number: lo.ToPtr(2), Size: lo.ToPtr(5)},
			Sort: lo.ToPtr("effective_at desc"),
			Filter: &api.ListGrantsParamsFilter{
				CustomerId: &filters.FilterULID{Eq: lo.ToPtr("01K4WAQ0J99ZZ0MD75HXR112H8")},
				Feature:    &filters.FilterStringExact{Oeq: []string{"a", "01K4WAQ0J99ZZ0MD75HXR112H9"}},
			},
			IncludeDeleted: lo.ToPtr(true),
		})

		require.Equal(t, http.StatusOK, res.Code, res.Body.String())
		require.Equal(t, entitlement.ListNamespaceGrantsInput{
			Namespace:        testNamespace,
			IncludeDeleted:   true,
			CustomerIDs:      []string{"01K4WAQ0J99ZZ0MD75HXR112H8"},
			FeatureIDsOrKeys: []string{"a", "01K4WAQ0J99ZZ0MD75HXR112H9"},
			OrderBy:          grant.OrderByEffectiveAt,
			Order:            sortx.OrderDesc,
			Page:             pagination.NewPage(2, 5),
		}, received)

		var body response.PagePaginationResponse[api.BillingEntitlementGrant]
		require.NoError(t, json.Unmarshal(res.Body.Bytes(), &body))
		require.Len(t, body.Data, 1)
		require.Equal(t, "01K4WAQ0J99ZZ0MD75HXR112HB", body.Data[0].Id)
		require.Equal(t, "01K4WAQ0J99ZZ0MD75HXR112HC", body.Data[0].EntitlementId)
		require.Equal(t, response.PageMetaPage{Size: 5, Number: 2, Total: lo.ToPtr(11)}, body.Meta.Page)
	})

	for _, tc := range []struct {
		name   string
		params api.ListGrantsParams
		field  string
	}{
		{name: "rejects an invalid page", params: api.ListGrantsParams{Page: &api.PagePaginationQuery{Number: lo.ToPtr(0)}}, field: "page"},
		{name: "rejects an unsupported sort field", params: api.ListGrantsParams{Sort: lo.ToPtr("owner_id")}, field: "sort"},
		{name: "rejects a negated customer filter", params: api.ListGrantsParams{Filter: &api.ListGrantsParamsFilter{CustomerId: &filters.FilterULID{Neq: lo.ToPtr("01K4WAQ0J99ZZ0MD75HXR112H8")}}}, field: "filter[customer_id]"},
		{name: "rejects a feature filter with both eq and oeq", params: api.ListGrantsParams{Filter: &api.ListGrantsParamsFilter{Feature: &filters.FilterStringExact{Eq: lo.ToPtr("a"), Oeq: []string{"b"}}}}, field: "filter[feature]"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			res := serveListGrants(t, fakeService{
				listGrants: func(context.Context, entitlement.ListNamespaceGrantsInput) (pagination.Result[grant.Grant], error) {
					t.Fatal("service must not be called")
					return pagination.Result[grant.Grant]{}, nil
				},
			}, tc.params)

			require.Equal(t, http.StatusBadRequest, res.Code)
			require.Contains(t, res.Body.String(), tc.field)
		})
	}

	t.Run("maps a domain validation error to 400", func(t *testing.T) {
		res := serveListGrants(t, fakeService{
			listGrants: func(context.Context, entitlement.ListNamespaceGrantsInput) (pagination.Result[grant.Grant], error) {
				return pagination.Result[grant.Grant]{}, models.NewGenericValidationError(errors.New("invalid order by"))
			},
		}, api.ListGrantsParams{})

		require.Equal(t, http.StatusBadRequest, res.Code)
	})
}
