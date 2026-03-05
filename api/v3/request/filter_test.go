package request_test

import (
	"testing"

	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	api "github.com/openmeterio/openmeter/api/v3"
	"github.com/openmeterio/openmeter/api/v3/request"
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

		require.NotNil(t, got.Eq)
		assert.Equal(t, "eq", *got.Eq)
		require.NotNil(t, got.Ne)
		assert.Equal(t, "ne", *got.Ne)
		require.NotNil(t, got.In)
		assert.Equal(t, []string{"a", "b"}, *got.In)
		require.NotNil(t, got.Nin)
		assert.Equal(t, []string{"c", "d"}, *got.Nin)
	})

	t.Run("maps contains operators", func(t *testing.T) {
		in := api.QueryFilterString{
			Contains:  lo.ToPtr("abc"),
			Ncontains: lo.ToPtr("xyz"),
		}

		got := request.ConvertQueryFilterString(in)

		require.NotNil(t, got.Like)
		assert.Equal(t, "%abc%", *got.Like)
		require.NotNil(t, got.Nlike)
		assert.Equal(t, "%xyz%", *got.Nlike)
	})

	t.Run("escapes like metacharacters in contains operators", func(t *testing.T) {
		in := api.QueryFilterString{
			Contains:  lo.ToPtr(`100%\path_name`),
			Ncontains: lo.ToPtr(`a_b\c%d`),
		}

		got := request.ConvertQueryFilterString(in)

		require.NotNil(t, got.Like)
		assert.Equal(t, `%100\%\\path\_name%`, *got.Like)
		require.NotNil(t, got.Nlike)
		assert.Equal(t, `%a\_b\\c\%d%`, *got.Nlike)
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

		require.NotNil(t, got.And)
		require.Len(t, *got.And, 2)
		require.NotNil(t, (*got.And)[0].Eq)
		assert.Equal(t, "a", *(*got.And)[0].Eq)
		require.NotNil(t, (*got.And)[1].Like)
		assert.Equal(t, "%b%", *(*got.And)[1].Like)

		require.NotNil(t, got.Or)
		require.Len(t, *got.Or, 2)
		require.NotNil(t, (*got.Or)[0].Ne)
		assert.Equal(t, "c", *(*got.Or)[0].Ne)
		require.NotNil(t, (*got.Or)[1].Nlike)
		assert.Equal(t, "%d%", *(*got.Or)[1].Nlike)
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
		require.NotNil(t, got.Like)
		assert.Equal(t, `%abc\\\_\%%`, *got.Like)
	})
}
