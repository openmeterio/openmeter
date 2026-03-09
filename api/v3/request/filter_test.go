package request_test

import (
	"testing"

	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	api "github.com/openmeterio/openmeter/api/v3"
	"github.com/openmeterio/openmeter/api/v3/request"
	"github.com/openmeterio/openmeter/pkg/filter"
)

func TestConvertQueryFilterString(t *testing.T) {
	t.Run("maps direct operators", func(t *testing.T) {
		in := api.QueryFilterString{
			Eq:  lo.ToPtr("eq"),
			Neq: lo.ToPtr("ne"),
			In:  &[]string{"a", "b"},
			Nin: &[]string{"c", "d"},
		}

		got := request.ConvertQueryFilterString(in)

		assert.Equal(t, filter.FilterString{
			Eq:  lo.ToPtr("eq"),
			Ne:  lo.ToPtr("ne"),
			In:  &[]string{"a", "b"},
			Nin: &[]string{"c", "d"},
		}, got)
	})

	t.Run("maps contains operators", func(t *testing.T) {
		in := api.QueryFilterString{
			Contains:  lo.ToPtr("abc"),
			Ncontains: lo.ToPtr("xyz"),
		}

		got := request.ConvertQueryFilterString(in)

		assert.Equal(t, filter.FilterString{
			Like:  lo.ToPtr("%abc%"),
			Nlike: lo.ToPtr("%xyz%"),
		}, got)
	})

	t.Run("escapes like metacharacters in contains operators", func(t *testing.T) {
		in := api.QueryFilterString{
			Contains:  lo.ToPtr(`100%\path_name`),
			Ncontains: lo.ToPtr(`a_b\c%d`),
		}

		got := request.ConvertQueryFilterString(in)

		assert.Equal(t, filter.FilterString{
			Like:  lo.ToPtr(`%100\%\\path\_name%`),
			Nlike: lo.ToPtr(`%a\_b\\c\%d%`),
		}, got)
	})

	t.Run("maps and or recursively", func(t *testing.T) {
		in := api.QueryFilterString{
			And: &[]api.QueryFilterString{
				{Eq: lo.ToPtr("a")},
				{Contains: lo.ToPtr("b")},
			},
			Or: &[]api.QueryFilterString{
				{Neq: lo.ToPtr("c")},
				{Ncontains: lo.ToPtr("d")},
			},
		}

		got := request.ConvertQueryFilterString(in)

		assert.Equal(t, filter.FilterString{
			And: &[]filter.FilterString{
				{Eq: lo.ToPtr("a")},
				{Like: lo.ToPtr("%b%")},
			},
			Or: &[]filter.FilterString{
				{Ne: lo.ToPtr("c")},
				{Nlike: lo.ToPtr("%d%")},
			},
		}, got)
	})

	t.Run("maps deeply nested and or", func(t *testing.T) {
		in := api.QueryFilterString{
			And: &[]api.QueryFilterString{
				{
					Or: &[]api.QueryFilterString{
						{Eq: lo.ToPtr("nested")},
					},
				},
			},
		}

		got := request.ConvertQueryFilterString(in)

		assert.Equal(t, filter.FilterString{
			And: &[]filter.FilterString{
				{
					Or: &[]filter.FilterString{
						{Eq: lo.ToPtr("nested")},
					},
				},
			},
		}, got)
	})
}

func TestConvertQueryFilterStringPtr(t *testing.T) {
	t.Run("nil pointer", func(t *testing.T) {
		assert.Nil(t, request.ConvertQueryFilterStringPtr(nil))
	})

	t.Run("non nil pointer", func(t *testing.T) {
		in := &api.QueryFilterString{Contains: lo.ToPtr(`abc\_%`)}

		got := request.ConvertQueryFilterStringPtr(in)
		require.NotNil(t, got)
		assert.Equal(t, &filter.FilterString{
			Like: lo.ToPtr(`%abc\\\_\%%`),
		}, got)
	})
}
