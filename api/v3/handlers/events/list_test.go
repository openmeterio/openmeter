package events

import (
	"errors"
	"testing"

	"github.com/samber/lo"
	"github.com/stretchr/testify/require"

	api "github.com/openmeterio/openmeter/api/v3"
	"github.com/openmeterio/openmeter/api/v3/apierrors"
	"github.com/openmeterio/openmeter/openmeter/streaming"
	"github.com/openmeterio/openmeter/pkg/sortx"
)

func TestFromAPICustomerIDFilter(t *testing.T) {
	ctx := t.Context()

	t.Run("nil filter returns nil", func(t *testing.T) {
		out, err := fromAPICustomerIDFilter(ctx, nil)
		require.NoError(t, err)
		require.Nil(t, out)
	})

	t.Run("eq maps to In with one element", func(t *testing.T) {
		out, err := fromAPICustomerIDFilter(ctx, &api.ULIDFieldFilter{Eq: lo.ToPtr("01G65Z755AFWAKHE12NY0CQ9FH")})
		require.NoError(t, err)
		require.NotNil(t, out)
		require.NotNil(t, out.In)
		require.Equal(t, []string{"01G65Z755AFWAKHE12NY0CQ9FH"}, *out.In)
	})

	t.Run("oeq maps to In", func(t *testing.T) {
		out, err := fromAPICustomerIDFilter(ctx, &api.ULIDFieldFilter{Oeq: []string{"a", "b"}})
		require.NoError(t, err)
		require.NotNil(t, out)
		require.Equal(t, []string{"a", "b"}, *out.In)
	})

	t.Run("neq is rejected", func(t *testing.T) {
		_, err := fromAPICustomerIDFilter(ctx, &api.ULIDFieldFilter{Neq: lo.ToPtr("x")})
		require.Error(t, err)
		assertBadRequestField(t, err, "filter[customer_id]")
	})

	t.Run("contains is rejected", func(t *testing.T) {
		_, err := fromAPICustomerIDFilter(ctx, &api.ULIDFieldFilter{Contains: lo.ToPtr("x")})
		require.Error(t, err)
		assertBadRequestField(t, err, "filter[customer_id]")
	})

	t.Run("ocontains is rejected", func(t *testing.T) {
		_, err := fromAPICustomerIDFilter(ctx, &api.ULIDFieldFilter{Ocontains: []string{"x"}})
		require.Error(t, err)
		assertBadRequestField(t, err, "filter[customer_id]")
	})

	t.Run("exists is rejected", func(t *testing.T) {
		_, err := fromAPICustomerIDFilter(ctx, &api.ULIDFieldFilter{Exists: lo.ToPtr(true)})
		require.Error(t, err)
		assertBadRequestField(t, err, "filter[customer_id]")
	})
}

func TestFromAPIEventSort(t *testing.T) {
	ctx := t.Context()

	t.Run("nil returns empty values", func(t *testing.T) {
		field, order, err := fromAPIEventSort(ctx, nil)
		require.NoError(t, err)
		require.Equal(t, streaming.EventSortField(""), field)
		require.Equal(t, sortx.Order(""), order)
	})

	t.Run("time defaults to desc when no suffix", func(t *testing.T) {
		sort := api.SortQuery("time")
		field, order, err := fromAPIEventSort(ctx, &sort)
		require.NoError(t, err)
		require.Equal(t, streaming.EventSortFieldTime, field)
		require.Equal(t, sortx.OrderDesc, order)
	})

	t.Run("ingested_at desc", func(t *testing.T) {
		sort := api.SortQuery("ingested_at desc")
		field, order, err := fromAPIEventSort(ctx, &sort)
		require.NoError(t, err)
		require.Equal(t, streaming.EventSortFieldIngestedAt, field)
		require.Equal(t, sortx.OrderDesc, order)
	})

	t.Run("stored_at defaults to desc when no suffix", func(t *testing.T) {
		sort := api.SortQuery("stored_at")
		field, order, err := fromAPIEventSort(ctx, &sort)
		require.NoError(t, err)
		require.Equal(t, streaming.EventSortFieldStoredAt, field)
		require.Equal(t, sortx.OrderDesc, order)
	})

	t.Run("time asc suffix is honored", func(t *testing.T) {
		sort := api.SortQuery("time asc")
		field, order, err := fromAPIEventSort(ctx, &sort)
		require.NoError(t, err)
		require.Equal(t, streaming.EventSortFieldTime, field)
		require.Equal(t, sortx.OrderAsc, order)
	})

	t.Run("unknown field is rejected", func(t *testing.T) {
		sort := api.SortQuery("created_at")
		_, _, err := fromAPIEventSort(ctx, &sort)
		require.Error(t, err)
		assertBadRequestField(t, err, "sort")
	})

	t.Run("malformed input is rejected", func(t *testing.T) {
		sort := api.SortQuery("time bogus extra")
		_, _, err := fromAPIEventSort(ctx, &sort)
		require.Error(t, err)
		assertBadRequestField(t, err, "sort")
	})
}

func assertBadRequestField(t *testing.T, err error, field string) {
	t.Helper()
	var apiErr *apierrors.BaseAPIError
	require.True(t, errors.As(err, &apiErr), "expected *apierrors.BaseAPIError, got %T", err)
	require.Len(t, apiErr.InvalidParameters, 1)
	require.Equal(t, field, apiErr.InvalidParameters[0].Field)
}
