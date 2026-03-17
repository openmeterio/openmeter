package request_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/api/v3/filters"
	"github.com/openmeterio/openmeter/api/v3/request"
)

func TestGetAipAttributes_Pagination(t *testing.T) {
	t.Run("defaults applied when no params", func(t *testing.T) {
		r := httptest.NewRequest(http.MethodGet, "/", nil)
		attrs, err := request.GetAipAttributes(r)
		require.NoError(t, err)
		assert.Equal(t, request.DefaultPaginationSize, attrs.Pagination.Size)
		assert.Equal(t, request.DefaultPaginationNumber, attrs.Pagination.Number)
	})

	t.Run("page size and number parsed", func(t *testing.T) {
		r := httptest.NewRequest(http.MethodGet, "/?page%5Bsize%5D=5&page%5Bnumber%5D=3", nil)
		attrs, err := request.GetAipAttributes(r)
		require.NoError(t, err)
		assert.Equal(t, 5, attrs.Pagination.Size)
		assert.Equal(t, 3, attrs.Pagination.Number)
	})

	t.Run("page size capped at max", func(t *testing.T) {
		r := httptest.NewRequest(http.MethodGet, "/?page%5Bsize%5D=999", nil)
		attrs, err := request.GetAipAttributes(r, request.WithMaxPageSize(50))
		require.NoError(t, err)
		assert.Equal(t, 50, attrs.Pagination.Size)
	})
}

func TestGetAipAttributes_Combined(t *testing.T) {
	t.Run("pagination filter and sort all parsed together", func(t *testing.T) {
		r := httptest.NewRequest(http.MethodGet, "/", nil)
		q := r.URL.Query()
		q.Set("page[size]", "5")
		q.Set("page[number]", "2")
		q.Set("filter[name][eq]", "foo")
		q.Set("sort", "name desc")
		r.URL.RawQuery = q.Encode()

		attrs, err := request.GetAipAttributes(r)
		require.NoError(t, err)
		assert.Equal(t, 5, attrs.Pagination.Size)
		assert.Equal(t, 2, attrs.Pagination.Number)
		require.Len(t, attrs.Filters, 1)
		assert.Equal(t, "name", attrs.Filters[0].Name)
		require.Len(t, attrs.Sorts, 1)
		assert.Equal(t, "name", attrs.Sorts[0].Field)
		assert.Equal(t, request.SortOrderDesc, attrs.Sorts[0].Order)
	})
}

func TestRemapAipAttributes(t *testing.T) {
	t.Run("remaps a filter field name", func(t *testing.T) {
		attrs := &request.AipAttributes{
			Filters: []request.QueryFilter{
				{Name: "api_name", Value: "foo", Filter: request.QueryFilterEQ},
			},
		}
		request.RemapAipAttributes(attrs, map[string]string{"api_name": "db_name"})
		assert.Equal(t, "db_name", attrs.Filters[0].Name)
	})

	t.Run("unmapped filter field is unchanged", func(t *testing.T) {
		attrs := &request.AipAttributes{
			Filters: []request.QueryFilter{
				{Name: "other", Value: "foo", Filter: request.QueryFilterEQ},
			},
		}
		request.RemapAipAttributes(attrs, map[string]string{"api_name": "db_name"})
		assert.Equal(t, "other", attrs.Filters[0].Name)
	})

	t.Run("remaps dot filter preserving sub-attribute", func(t *testing.T) {
		attrs := &request.AipAttributes{
			Filters: []request.QueryFilter{
				{Name: "labels.env", Value: "prod", Filter: request.QueryFilterEQ},
			},
		}
		request.RemapAipAttributes(attrs, map[string]string{"labels": "metadata"})
		assert.Equal(t, "metadata.env", attrs.Filters[0].Name)
	})

	t.Run("remaps a sort field name", func(t *testing.T) {
		attrs := &request.AipAttributes{
			Sorts: []request.SortBy{
				{Field: "api_name", Order: request.SortOrderAsc},
			},
		}
		request.RemapAipAttributes(attrs, map[string]string{"api_name": "db_name"})
		assert.Equal(t, "db_name", attrs.Sorts[0].Field)
	})

	t.Run("unmapped sort field is unchanged", func(t *testing.T) {
		attrs := &request.AipAttributes{
			Sorts: []request.SortBy{
				{Field: "other", Order: request.SortOrderAsc},
			},
		}
		request.RemapAipAttributes(attrs, map[string]string{"api_name": "db_name"})
		assert.Equal(t, "other", attrs.Sorts[0].Field)
	})

	t.Run("remaps dot sort preserving sub-attribute", func(t *testing.T) {
		attrs := &request.AipAttributes{
			Sorts: []request.SortBy{
				{Field: "labels.env", Order: request.SortOrderDesc},
			},
		}
		request.RemapAipAttributes(attrs, map[string]string{"labels": "metadata"})
		assert.Equal(t, "metadata.env", attrs.Sorts[0].Field)
	})

	t.Run("nil filters and sorts are no-ops", func(t *testing.T) {
		attrs := &request.AipAttributes{}
		request.RemapAipAttributes(attrs, map[string]string{"api_name": "db_name"})
		assert.Nil(t, attrs.Filters)
		assert.Nil(t, attrs.Sorts)
	})
}

func TestFilterStringFromAip(t *testing.T) {
	t.Run("returns nil when filter list is empty", func(t *testing.T) {
		assert.Nil(t, request.FilterStringFromAip(nil, "name"))
	})

	t.Run("returns nil when no filter matches the field", func(t *testing.T) {
		f := request.FilterStringFromAip([]request.QueryFilter{
			{Name: "other", Value: "foo", Filter: request.QueryFilterEQ},
		}, "name")
		assert.Nil(t, f)
	})

	t.Run("only processes the matching field", func(t *testing.T) {
		f := request.FilterStringFromAip([]request.QueryFilter{
			{Name: "name", Value: "foo", Filter: request.QueryFilterEQ},
			{Name: "other", Value: "bar", Filter: request.QueryFilterNEQ},
		}, "name")
		require.NotNil(t, f)
		require.NotNil(t, f.Eq)
		assert.Equal(t, "foo", *f.Eq)
		assert.Nil(t, f.Neq)
	})

	filterOpCases := []struct {
		name      string
		filterOp  request.QueryFilterOp
		value     string
		checkFunc func(t *testing.T, f *filters.StringFilter)
	}{
		{
			name: "eq", filterOp: request.QueryFilterEQ, value: "v",
			checkFunc: func(t *testing.T, f *filters.StringFilter) {
				t.Helper()
				require.NotNil(t, f.Eq)
				assert.Equal(t, "v", *f.Eq)
			},
		},
		{
			name: "neq", filterOp: request.QueryFilterNEQ, value: "v",
			checkFunc: func(t *testing.T, f *filters.StringFilter) {
				t.Helper()
				require.NotNil(t, f.Neq)
				assert.Equal(t, "v", *f.Neq)
			},
		},
		{
			name: "gt", filterOp: request.QueryFilterGT, value: "v",
			checkFunc: func(t *testing.T, f *filters.StringFilter) {
				t.Helper()
				require.NotNil(t, f.Gt)
				assert.Equal(t, "v", *f.Gt)
			},
		},
		{
			name: "gte", filterOp: request.QueryFilterGTE, value: "v",
			checkFunc: func(t *testing.T, f *filters.StringFilter) {
				t.Helper()
				require.NotNil(t, f.Gte)
				assert.Equal(t, "v", *f.Gte)
			},
		},
		{
			name: "lt", filterOp: request.QueryFilterLT, value: "v",
			checkFunc: func(t *testing.T, f *filters.StringFilter) {
				t.Helper()
				require.NotNil(t, f.Lt)
				assert.Equal(t, "v", *f.Lt)
			},
		},
		{
			name: "lte", filterOp: request.QueryFilterLTE, value: "v",
			checkFunc: func(t *testing.T, f *filters.StringFilter) {
				t.Helper()
				require.NotNil(t, f.Lte)
				assert.Equal(t, "v", *f.Lte)
			},
		},
		{
			name: "contains", filterOp: request.QueryFilterContains, value: "v",
			checkFunc: func(t *testing.T, f *filters.StringFilter) {
				t.Helper()
				require.NotNil(t, f.Contains)
				assert.Equal(t, "v", *f.Contains)
			},
		},
		{
			name: "oeq", filterOp: request.QueryFilterOrEQ, value: "v",
			checkFunc: func(t *testing.T, f *filters.StringFilter) {
				t.Helper()
				require.NotNil(t, f.Oeq)
				assert.Equal(t, "v", *f.Oeq)
			},
		},
		{
			name: "ocontains", filterOp: request.QueryFilterOrContains, value: "v",
			checkFunc: func(t *testing.T, f *filters.StringFilter) {
				t.Helper()
				require.NotNil(t, f.Ocontains)
				assert.Equal(t, "v", *f.Ocontains)
			},
		},
		{
			name: "exists", filterOp: request.QueryFilterExists,
			checkFunc: func(t *testing.T, f *filters.StringFilter) {
				t.Helper()
				require.NotNil(t, f.Exists)
				assert.True(t, *f.Exists)
			},
		},
	}

	for _, tc := range filterOpCases {
		t.Run("maps "+tc.name+" to StringFilter", func(t *testing.T) {
			f := request.FilterStringFromAip([]request.QueryFilter{
				{Name: "field", Value: tc.value, Filter: tc.filterOp},
			}, "field")
			require.NotNil(t, f)
			tc.checkFunc(t, f)
		})
	}
}
