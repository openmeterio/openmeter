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
	"github.com/openmeterio/openmeter/openmeter/credit/grant"
	"github.com/openmeterio/openmeter/openmeter/entitlement"
	pkgfilter "github.com/openmeterio/openmeter/pkg/filter"
	"github.com/openmeterio/openmeter/pkg/models"
	pagination "github.com/openmeterio/openmeter/pkg/pagination/v2"
	"github.com/openmeterio/openmeter/pkg/sortx"
)

func serveListGrants(t *testing.T, svc fakeService, params api.ListGrantsParams) *httptest.ResponseRecorder {
	t.Helper()

	response := httptest.NewRecorder()
	newTestHandler(svc).ListGrants().With(params).ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/v3/openmeter/grants", nil))

	return response
}

func TestListGrantsHandler(t *testing.T) {
	cursor := pagination.NewCursor(time.Date(2025, time.January, 1, 0, 0, 0, 123000, time.UTC), "01K4WAQ0J99ZZ0MD75HXR112HA")

	t.Run("applies the defaults", func(t *testing.T) {
		var received entitlement.ListNamespaceGrantsInput

		response := serveListGrants(t, fakeService{
			listGrants: func(_ context.Context, input entitlement.ListNamespaceGrantsInput) (pagination.Result[grant.Grant], error) {
				received = input
				return pagination.Result[grant.Grant]{}, nil
			},
		}, api.ListGrantsParams{})

		require.Equal(t, http.StatusOK, response.Code, response.Body.String())
		require.Equal(t, entitlement.ListNamespaceGrantsInput{
			Namespace: testNamespace,
			OrderBy:   grant.OrderByCreatedAt,
			Order:     sortx.OrderAsc,
			PageSize:  defaultListGrantsPageSize,
		}, received)

		var body api.EntitlementGrantPaginatedResponse
		require.NoError(t, json.Unmarshal(response.Body.Bytes(), &body))
		require.Empty(t, body.Data)
		require.True(t, body.Meta.Page.Next.IsNull())
	})

	t.Run("maps the query and the next cursor", func(t *testing.T) {
		var received entitlement.ListNamespaceGrantsInput

		response := serveListGrants(t, fakeService{
			listGrants: func(_ context.Context, input entitlement.ListNamespaceGrantsInput) (pagination.Result[grant.Grant], error) {
				received = input
				return pagination.Result[grant.Grant]{
					Items: []grant.Grant{{
						ID:          "01K4WAQ0J99ZZ0MD75HXR112HB",
						OwnerID:     "01K4WAQ0J99ZZ0MD75HXR112HC",
						Amount:      10,
						EffectiveAt: cursor.Time,
					}},
					NextCursor: &cursor,
				}, nil
			},
		}, api.ListGrantsParams{
			Page: &api.CursorPaginationQuery{
				Size:  lo.ToPtr(5),
				After: lo.ToPtr(cursor.Encode()),
			},
			Sort: lo.ToPtr("effective_at desc"),
			Filter: &api.ListGrantsParamsFilter{
				CustomerId: &filters.FilterULID{Eq: lo.ToPtr("01K4WAQ0J99ZZ0MD75HXR112H8")},
				FeatureKey: &filters.FilterStringExact{Oeq: []string{"a", "b"}},
			},
			IncludeDeleted: lo.ToPtr(true),
		})

		require.Equal(t, http.StatusOK, response.Code, response.Body.String())
		require.Equal(t, 5, received.PageSize)
		require.Equal(t, &cursor, received.Cursor)
		require.Equal(t, grant.OrderByEffectiveAt, received.OrderBy)
		require.Equal(t, sortx.OrderDesc, received.Order)
		require.True(t, received.IncludeDeleted)
		require.Equal(t, &pkgfilter.FilterULID{FilterString: pkgfilter.FilterString{Eq: lo.ToPtr("01K4WAQ0J99ZZ0MD75HXR112H8")}}, received.CustomerID)
		require.Equal(t, &pkgfilter.FilterString{In: &[]string{"a", "b"}}, received.FeatureKey)
		require.Nil(t, received.FeatureID)

		var body api.EntitlementGrantPaginatedResponse
		require.NoError(t, json.Unmarshal(response.Body.Bytes(), &body))
		require.Len(t, body.Data, 1)
		require.Equal(t, "01K4WAQ0J99ZZ0MD75HXR112HB", body.Data[0].Id)
		require.Equal(t, "01K4WAQ0J99ZZ0MD75HXR112HC", body.Data[0].EntitlementId)
		require.Equal(t, cursor.Encode(), body.Meta.Page.Next.MustGet())
		require.Equal(t, float32(5), body.Meta.Page.Size)
	})

	for _, tc := range []struct {
		name   string
		params api.ListGrantsParams
		field  string
	}{
		{name: "rejects backward pagination", params: api.ListGrantsParams{Page: &api.CursorPaginationQuery{Before: lo.ToPtr(cursor.Encode())}}, field: "page[before]"},
		{name: "rejects an invalid cursor", params: api.ListGrantsParams{Page: &api.CursorPaginationQuery{After: lo.ToPtr("not-a-cursor")}}, field: "page[after]"},
		{name: "rejects a page size out of range", params: api.ListGrantsParams{Page: &api.CursorPaginationQuery{Size: lo.ToPtr(entitlement.MaxGrantsPageSize + 1)}}, field: "page[size]"},
		{name: "rejects a sort field that cannot key the cursor", params: api.ListGrantsParams{Sort: lo.ToPtr("updated_at")}, field: "sort"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			response := serveListGrants(t, fakeService{
				listGrants: func(context.Context, entitlement.ListNamespaceGrantsInput) (pagination.Result[grant.Grant], error) {
					t.Fatal("service must not be called")
					return pagination.Result[grant.Grant]{}, nil
				},
			}, tc.params)

			require.Equal(t, http.StatusBadRequest, response.Code)
			require.Contains(t, response.Body.String(), tc.field)
		})
	}

	t.Run("maps a domain validation error to 400", func(t *testing.T) {
		response := serveListGrants(t, fakeService{
			listGrants: func(context.Context, entitlement.ListNamespaceGrantsInput) (pagination.Result[grant.Grant], error) {
				return pagination.Result[grant.Grant]{}, models.NewGenericValidationError(errors.New("invalid filter"))
			},
		}, api.ListGrantsParams{})

		require.Equal(t, http.StatusBadRequest, response.Code)
	})
}
