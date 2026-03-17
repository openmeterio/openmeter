package request_test

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/api/v3/apierrors"
	"github.com/openmeterio/openmeter/api/v3/request"
)

// newFilterRequest builds an HTTP GET request with the given query params.
func newFilterRequest(t *testing.T, params url.Values) *http.Request {
	t.Helper()
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.URL.RawQuery = params.Encode()
	return r
}

func TestExtractFilter_NoParams(t *testing.T) {
	t.Run("no query params returns empty filters", func(t *testing.T) {
		r := httptest.NewRequest(http.MethodGet, "/", nil)
		attrs, err := request.GetAipAttributes(r)
		require.NoError(t, err)
		assert.Empty(t, attrs.Filters)
	})

	t.Run("non-filter params do not produce filters", func(t *testing.T) {
		params := url.Values{"page[size]": {"10"}, "sort": {"name"}}
		attrs, err := request.GetAipAttributes(newFilterRequest(t, params))
		require.NoError(t, err)
		assert.Empty(t, attrs.Filters)
	})

	t.Run("filter prefix without brackets is ignored", func(t *testing.T) {
		params := url.Values{"filtername": {"foo"}}
		attrs, err := request.GetAipAttributes(newFilterRequest(t, params))
		require.NoError(t, err)
		assert.Empty(t, attrs.Filters)
	})
}

func TestExtractFilter_Operations(t *testing.T) {
	cases := []struct {
		op         string
		param      string
		wantFilter request.QueryFilterOp
	}{
		{"eq", "filter[name][eq]", request.QueryFilterEQ},
		{"neq", "filter[name][neq]", request.QueryFilterNEQ},
		{"gt", "filter[name][gt]", request.QueryFilterGT},
		{"gte", "filter[name][gte]", request.QueryFilterGTE},
		{"lt", "filter[name][lt]", request.QueryFilterLT},
		{"lte", "filter[name][lte]", request.QueryFilterLTE},
		{"contains", "filter[name][contains]", request.QueryFilterContains},
		{"oeq", "filter[name][oeq]", request.QueryFilterOrEQ},
		{"ocontains", "filter[name][ocontains]", request.QueryFilterOrContains},
	}

	for _, tc := range cases {
		t.Run("parses "+tc.op+" operator", func(t *testing.T) {
			params := url.Values{tc.param: {"testvalue"}}
			attrs, err := request.GetAipAttributes(newFilterRequest(t, params))
			require.NoError(t, err)
			require.Len(t, attrs.Filters, 1)
			assert.Equal(t, "name", attrs.Filters[0].Name)
			assert.Equal(t, "testvalue", attrs.Filters[0].Value)
			assert.Equal(t, tc.wantFilter, attrs.Filters[0].Filter)
		})
	}

	t.Run("no op bracket defaults to eq", func(t *testing.T) {
		params := url.Values{"filter[name]": {"testvalue"}}
		attrs, err := request.GetAipAttributes(newFilterRequest(t, params))
		require.NoError(t, err)
		require.Len(t, attrs.Filters, 1)
		assert.Equal(t, request.QueryFilterEQ, attrs.Filters[0].Filter)
	})

	t.Run("empty value on plain filter is exists", func(t *testing.T) {
		params := url.Values{"filter[name]": {""}}
		attrs, err := request.GetAipAttributes(newFilterRequest(t, params))
		require.NoError(t, err)
		require.Len(t, attrs.Filters, 1)
		assert.Equal(t, request.QueryFilterExists, attrs.Filters[0].Filter)
	})

	t.Run("explicit exists operator", func(t *testing.T) {
		params := url.Values{"filter[name][exists]": {"1"}}
		attrs, err := request.GetAipAttributes(newFilterRequest(t, params))
		require.NoError(t, err)
		require.Len(t, attrs.Filters, 1)
		assert.Equal(t, request.QueryFilterExists, attrs.Filters[0].Filter)
	})

	t.Run("invalid operation returns error", func(t *testing.T) {
		params := url.Values{"filter[name][bogus]": {"foo"}}
		_, err := request.GetAipAttributes(newFilterRequest(t, params))
		require.Error(t, err)
	})
}

func TestExtractFilter_MultiValueOps(t *testing.T) {
	t.Run("oeq splits comma-separated values", func(t *testing.T) {
		params := url.Values{"filter[name][oeq]": {"a,b,c"}}
		attrs, err := request.GetAipAttributes(newFilterRequest(t, params))
		require.NoError(t, err)
		require.Len(t, attrs.Filters, 1)
		assert.Equal(t, []string{"a", "b", "c"}, attrs.Filters[0].Values)
	})

	t.Run("oeq trims whitespace from values", func(t *testing.T) {
		params := url.Values{"filter[name][oeq]": {"a, b , c"}}
		attrs, err := request.GetAipAttributes(newFilterRequest(t, params))
		require.NoError(t, err)
		require.Len(t, attrs.Filters, 1)
		assert.Equal(t, []string{"a", "b", "c"}, attrs.Filters[0].Values)
	})

	t.Run("ocontains splits comma-separated values", func(t *testing.T) {
		params := url.Values{"filter[name][ocontains]": {"foo,bar"}}
		attrs, err := request.GetAipAttributes(newFilterRequest(t, params))
		require.NoError(t, err)
		require.Len(t, attrs.Filters, 1)
		assert.Equal(t, []string{"foo", "bar"}, attrs.Filters[0].Values)
	})
}

func TestExtractFilter_MultipleFilters(t *testing.T) {
	t.Run("multiple different fields", func(t *testing.T) {
		params := url.Values{
			"filter[name][eq]":   {"foo"},
			"filter[status][eq]": {"active"},
		}
		attrs, err := request.GetAipAttributes(newFilterRequest(t, params))
		require.NoError(t, err)
		assert.Len(t, attrs.Filters, 2)
	})
}

func TestExtractFilter_AuthorizedFilters(t *testing.T) {
	authorized := request.AuthorizedFilters{
		"name": {Filters: []request.QueryFilterOp{request.QueryFilterEQ, request.QueryFilterContains}},
	}

	t.Run("authorized field and op passes through", func(t *testing.T) {
		params := url.Values{"filter[name][eq]": {"foo"}}
		attrs, err := request.GetAipAttributes(newFilterRequest(t, params), request.WithAuthorizedFilters(authorized))
		require.NoError(t, err)
		require.Len(t, attrs.Filters, 1)
		assert.Equal(t, "name", attrs.Filters[0].Name)
	})

	t.Run("unauthorized field silently ignored in non-strict mode", func(t *testing.T) {
		params := url.Values{"filter[unknown][eq]": {"foo"}}
		attrs, err := request.GetAipAttributes(newFilterRequest(t, params), request.WithAuthorizedFilters(authorized))
		require.NoError(t, err)
		assert.Empty(t, attrs.Filters)
	})

	t.Run("unauthorized field returns 400 in strict mode", func(t *testing.T) {
		params := url.Values{"filter[unknown][eq]": {"foo"}}
		_, err := request.GetAipAttributes(newFilterRequest(t, params),
			request.WithAuthorizedFilters(authorized),
			request.WithAipStrictMode(),
		)
		require.Error(t, err)
		var apiErr *apierrors.BaseAPIError
		require.True(t, errors.As(err, &apiErr))
		assert.Equal(t, http.StatusBadRequest, apiErr.Status)
	})

	t.Run("unauthorized op silently ignored in non-strict mode", func(t *testing.T) {
		params := url.Values{"filter[name][gt]": {"foo"}} // gt not in authorized ops
		attrs, err := request.GetAipAttributes(newFilterRequest(t, params), request.WithAuthorizedFilters(authorized))
		require.NoError(t, err)
		assert.Empty(t, attrs.Filters)
	})

	t.Run("unauthorized op returns 400 in strict mode", func(t *testing.T) {
		params := url.Values{"filter[name][gt]": {"foo"}}
		_, err := request.GetAipAttributes(newFilterRequest(t, params),
			request.WithAuthorizedFilters(authorized),
			request.WithAipStrictMode(),
		)
		require.Error(t, err)
		var apiErr *apierrors.BaseAPIError
		require.True(t, errors.As(err, &apiErr))
		assert.Equal(t, http.StatusBadRequest, apiErr.Status)
	})
}

func TestExtractFilter_DotFilters(t *testing.T) {
	authorized := request.AuthorizedFilters{
		"labels": {
			Filters:   []request.QueryFilterOp{request.QueryFilterEQ},
			DotFilter: true,
		},
	}

	t.Run("dot sub-attribute passes for DotFilter field", func(t *testing.T) {
		params := url.Values{"filter[labels.env][eq]": {"prod"}}
		attrs, err := request.GetAipAttributes(newFilterRequest(t, params), request.WithAuthorizedFilters(authorized))
		require.NoError(t, err)
		require.Len(t, attrs.Filters, 1)
		assert.Equal(t, "labels.env", attrs.Filters[0].Name)
		assert.Equal(t, "prod", attrs.Filters[0].Value)
	})

	t.Run("bare field name rejected for DotFilter-only field", func(t *testing.T) {
		params := url.Values{"filter[labels][eq]": {"prod"}}
		attrs, err := request.GetAipAttributes(newFilterRequest(t, params), request.WithAuthorizedFilters(authorized))
		require.NoError(t, err)
		assert.Empty(t, attrs.Filters)
	})
}

func TestExtractFilter_ValidationFunc(t *testing.T) {
	errCustom := errors.New("custom validation error")
	authorized := request.AuthorizedFilters{
		"id": {
			Filters: []request.QueryFilterOp{request.QueryFilterEQ},
			ValidationFunc: func(_, value string) error {
				if value == "bad" {
					return errCustom
				}
				return nil
			},
		},
		"uuid_id": {
			Filters: []request.QueryFilterOp{request.QueryFilterEQ},
			ValidationFunc: func(_, value string) error {
				if value == "not-a-uuid" {
					return request.ErrReturnEmptySet
				}
				return nil
			},
		},
	}

	t.Run("valid value passes through", func(t *testing.T) {
		params := url.Values{"filter[id][eq]": {"good-value"}}
		attrs, err := request.GetAipAttributes(newFilterRequest(t, params), request.WithAuthorizedFilters(authorized))
		require.NoError(t, err)
		require.Len(t, attrs.Filters, 1)
	})

	t.Run("validation error returns 400", func(t *testing.T) {
		params := url.Values{"filter[id][eq]": {"bad"}}
		_, err := request.GetAipAttributes(newFilterRequest(t, params), request.WithAuthorizedFilters(authorized))
		require.Error(t, err)
		var apiErr *apierrors.BaseAPIError
		require.True(t, errors.As(err, &apiErr))
		assert.Equal(t, http.StatusBadRequest, apiErr.Status)
	})

	t.Run("ErrReturnEmptySet returns 200 empty-set response", func(t *testing.T) {
		params := url.Values{"filter[uuid_id][eq]": {"not-a-uuid"}}
		_, err := request.GetAipAttributes(newFilterRequest(t, params), request.WithAuthorizedFilters(authorized))
		require.Error(t, err)
		var apiErr *apierrors.BaseAPIError
		require.True(t, errors.As(err, &apiErr))
		assert.Equal(t, http.StatusOK, apiErr.Status)
	})
}
