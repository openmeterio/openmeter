package request_test

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/api/v3/request"
)

func newSortRequest(t *testing.T, sortValue string) *http.Request {
	t.Helper()
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.URL.RawQuery = url.Values{"sort": {sortValue}}.Encode()
	return r
}

func TestExtractSort_NoParam(t *testing.T) {
	t.Run("no sort param returns nil", func(t *testing.T) {
		r := httptest.NewRequest(http.MethodGet, "/", nil)
		attrs, err := request.GetAipAttributes(r)
		require.NoError(t, err)
		assert.Nil(t, attrs.Sorts)
	})

	t.Run("no sort param with default sort returns default", func(t *testing.T) {
		r := httptest.NewRequest(http.MethodGet, "/", nil)
		attrs, err := request.GetAipAttributes(r, request.WithDefaultSort("created_at", request.SortOrderDesc))
		require.NoError(t, err)
		require.Len(t, attrs.Sorts, 1)
		assert.Equal(t, "created_at", attrs.Sorts[0].Field)
		assert.Equal(t, request.SortOrderDesc, attrs.Sorts[0].Order)
	})

	t.Run("explicit sort param overrides default", func(t *testing.T) {
		attrs, err := request.GetAipAttributes(
			newSortRequest(t, "name"),
			request.WithDefaultSort("created_at", request.SortOrderDesc),
		)
		require.NoError(t, err)
		require.Len(t, attrs.Sorts, 1)
		assert.Equal(t, "name", attrs.Sorts[0].Field)
	})
}

func TestExtractSort_SingleField(t *testing.T) {
	t.Run("field only defaults to asc", func(t *testing.T) {
		attrs, err := request.GetAipAttributes(newSortRequest(t, "name"))
		require.NoError(t, err)
		require.Len(t, attrs.Sorts, 1)
		assert.Equal(t, "name", attrs.Sorts[0].Field)
		assert.Equal(t, request.SortOrderAsc, attrs.Sorts[0].Order)
	})

	t.Run("field with asc order", func(t *testing.T) {
		attrs, err := request.GetAipAttributes(newSortRequest(t, "name asc"))
		require.NoError(t, err)
		require.Len(t, attrs.Sorts, 1)
		assert.Equal(t, request.SortOrderAsc, attrs.Sorts[0].Order)
	})

	t.Run("field with desc order", func(t *testing.T) {
		attrs, err := request.GetAipAttributes(newSortRequest(t, "name desc"))
		require.NoError(t, err)
		require.Len(t, attrs.Sorts, 1)
		assert.Equal(t, "name", attrs.Sorts[0].Field)
		assert.Equal(t, request.SortOrderDesc, attrs.Sorts[0].Order)
	})

	t.Run("unknown order string defaults to asc", func(t *testing.T) {
		attrs, err := request.GetAipAttributes(newSortRequest(t, "name invalid_order"))
		require.NoError(t, err)
		require.Len(t, attrs.Sorts, 1)
		assert.Equal(t, request.SortOrderAsc, attrs.Sorts[0].Order)
	})
}

func TestExtractSort_MultipleFields(t *testing.T) {
	t.Run("two comma-separated fields", func(t *testing.T) {
		attrs, err := request.GetAipAttributes(newSortRequest(t, "name,created_at"))
		require.NoError(t, err)
		require.Len(t, attrs.Sorts, 2)
		assert.Equal(t, "name", attrs.Sorts[0].Field)
		assert.Equal(t, request.SortOrderAsc, attrs.Sorts[0].Order)
		assert.Equal(t, "created_at", attrs.Sorts[1].Field)
		assert.Equal(t, request.SortOrderAsc, attrs.Sorts[1].Order)
	})

	t.Run("mixed order on multiple fields", func(t *testing.T) {
		attrs, err := request.GetAipAttributes(newSortRequest(t, "name asc,created_at desc"))
		require.NoError(t, err)
		require.Len(t, attrs.Sorts, 2)
		assert.Equal(t, request.SortOrderAsc, attrs.Sorts[0].Order)
		assert.Equal(t, request.SortOrderDesc, attrs.Sorts[1].Order)
	})

	t.Run("empty segment in comma list is ignored", func(t *testing.T) {
		attrs, err := request.GetAipAttributes(newSortRequest(t, "name,,created_at"))
		require.NoError(t, err)
		assert.Len(t, attrs.Sorts, 2)
	})
}

func TestExtractSort_AuthorizedSorts(t *testing.T) {
	t.Run("authorized field passes through", func(t *testing.T) {
		attrs, err := request.GetAipAttributes(
			newSortRequest(t, "name"),
			request.WithAuthorizedSorts([]string{"name", "created_at"}),
		)
		require.NoError(t, err)
		require.Len(t, attrs.Sorts, 1)
		assert.Equal(t, "name", attrs.Sorts[0].Field)
	})

	t.Run("unauthorized field is silently ignored", func(t *testing.T) {
		attrs, err := request.GetAipAttributes(
			newSortRequest(t, "unknown_field"),
			request.WithAuthorizedSorts([]string{"name"}),
		)
		require.NoError(t, err)
		assert.Empty(t, attrs.Sorts)
	})

	t.Run("only authorized fields pass through from multi-field sort", func(t *testing.T) {
		attrs, err := request.GetAipAttributes(
			newSortRequest(t, "name,unknown_field"),
			request.WithAuthorizedSorts([]string{"name"}),
		)
		require.NoError(t, err)
		require.Len(t, attrs.Sorts, 1)
		assert.Equal(t, "name", attrs.Sorts[0].Field)
	})
}

func TestExtractSort_AuthorizedDotSorts(t *testing.T) {
	t.Run("dot sub-attribute authorized by prefix", func(t *testing.T) {
		attrs, err := request.GetAipAttributes(
			newSortRequest(t, "labels.env"),
			request.WithAuthorizedDotSorts([]string{"labels"}),
		)
		require.NoError(t, err)
		require.Len(t, attrs.Sorts, 1)
		assert.Equal(t, "labels.env", attrs.Sorts[0].Field)
	})

	t.Run("exact dot field authorized", func(t *testing.T) {
		attrs, err := request.GetAipAttributes(
			newSortRequest(t, "labels.env"),
			request.WithAuthorizedDotSorts([]string{"labels.env"}),
		)
		require.NoError(t, err)
		require.Len(t, attrs.Sorts, 1)
	})

	t.Run("unauthorized dot prefix is ignored", func(t *testing.T) {
		attrs, err := request.GetAipAttributes(
			newSortRequest(t, "other.key"),
			request.WithAuthorizedDotSorts([]string{"labels"}),
		)
		require.NoError(t, err)
		assert.Empty(t, attrs.Sorts)
	})
}
