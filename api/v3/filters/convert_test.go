package filters

import (
	"testing"
	"time"

	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/pkg/filter"
)

func TestConvertFilterString(t *testing.T) {
	t.Run("eq", func(t *testing.T) {
		out, err := FromAPIFilterString(&FilterString{Eq: lo.ToPtr("val")})
		require.NoError(t, err)
		assert.Equal(t, lo.ToPtr("val"), out.Eq)
	})

	t.Run("neq maps to Ne", func(t *testing.T) {
		out, err := FromAPIFilterString(&FilterString{Neq: lo.ToPtr("val")})
		require.NoError(t, err)
		assert.Equal(t, lo.ToPtr("val"), out.Ne)
	})

	t.Run("contains maps to Contains", func(t *testing.T) {
		out, err := FromAPIFilterString(&FilterString{Contains: lo.ToPtr("part")})
		require.NoError(t, err)
		assert.Equal(t, lo.ToPtr("part"), out.Contains)
	})

	t.Run("oeq maps to In", func(t *testing.T) {
		out, err := FromAPIFilterString(&FilterString{Oeq: []string{"a", "b"}})
		require.NoError(t, err)
		require.NotNil(t, out.In)
		assert.Equal(t, []string{"a", "b"}, *out.In)
	})

	t.Run("ocontains maps to Or of Contains", func(t *testing.T) {
		out, err := FromAPIFilterString(&FilterString{Ocontains: []string{"x", "y"}})
		require.NoError(t, err)
		require.NotNil(t, out.Or)
		require.Len(t, *out.Or, 2)
		assert.Equal(t, lo.ToPtr("x"), (*out.Or)[0].Contains)
		assert.Equal(t, lo.ToPtr("y"), (*out.Or)[1].Contains)
	})

	t.Run("exists", func(t *testing.T) {
		out, err := FromAPIFilterString(&FilterString{Exists: lo.ToPtr(true)})
		require.NoError(t, err)
		assert.Equal(t, lo.ToPtr(true), out.Exists)
	})

	t.Run("single-bound gt maps directly", func(t *testing.T) {
		out, err := FromAPIFilterString(&FilterString{Gt: lo.ToPtr("a")})
		require.NoError(t, err)
		assert.Equal(t, lo.ToPtr("a"), out.Gt)
		assert.Nil(t, out.And, "single-bound should not produce an And")
	})

	t.Run("gt and lte are split into an And of two single-op predicates", func(t *testing.T) {
		out, err := FromAPIFilterString(&FilterString{Gt: lo.ToPtr("a"), Lte: lo.ToPtr("z")})
		require.NoError(t, err)
		assert.Nil(t, out.Gt, "top-level Gt must be empty when range is split")
		assert.Nil(t, out.Lte, "top-level Lte must be empty when range is split")
		require.NotNil(t, out.And)
		require.Len(t, *out.And, 2)
		assert.Equal(t, lo.ToPtr("a"), (*out.And)[0].Gt)
		assert.Equal(t, lo.ToPtr("z"), (*out.And)[1].Lte)
	})

	t.Run("gte and lt are split into an And", func(t *testing.T) {
		out, err := FromAPIFilterString(&FilterString{Gte: lo.ToPtr("a"), Lt: lo.ToPtr("z")})
		require.NoError(t, err)
		require.NotNil(t, out.And)
		require.Len(t, *out.And, 2)
		assert.Equal(t, lo.ToPtr("a"), (*out.And)[0].Gte)
		assert.Equal(t, lo.ToPtr("z"), (*out.And)[1].Lt)
	})

	t.Run("non-range plus range mixes into an And of single-op leaves", func(t *testing.T) {
		// The converter should normalize any combination of set operators with AND..
		out, err := FromAPIFilterString(&FilterString{
			Neq: lo.ToPtr("excluded"),
			Gte: lo.ToPtr("a"),
			Lte: lo.ToPtr("z"),
		})
		require.NoError(t, err)
		assert.Nil(t, out.Ne, "top-level Ne must be empty when mixed ops land in And")
		assert.Nil(t, out.Gte, "top-level Gte must be empty when mixed ops land in And")
		assert.Nil(t, out.Lte, "top-level Lte must be empty when mixed ops land in And")

		require.NotNil(t, out.And)
		require.Len(t, *out.And, 3)

		children := *out.And
		assert.Equal(t, lo.ToPtr("excluded"), children[0].Ne)
		assert.Equal(t, lo.ToPtr("a"), children[1].Gte)
		assert.Equal(t, lo.ToPtr("z"), children[2].Lte)
	})

	t.Run("eq plus contains produces an And of two leaves", func(t *testing.T) {
		out, err := FromAPIFilterString(&FilterString{
			Eq:       lo.ToPtr("foo"),
			Contains: lo.ToPtr("oo"),
		})
		require.NoError(t, err)
		require.NotNil(t, out.And)
		require.Len(t, *out.And, 2)
		assert.Equal(t, lo.ToPtr("foo"), (*out.And)[0].Eq)
		assert.Equal(t, lo.ToPtr("oo"), (*out.And)[1].Contains)
	})

	t.Run("ocontains plus gte produces an And carrying the Or leaf", func(t *testing.T) {
		out, err := FromAPIFilterString(&FilterString{
			Ocontains: []string{"x", "y"},
			Gte:       lo.ToPtr("a"),
		})
		require.NoError(t, err)
		require.NotNil(t, out.And)
		require.Len(t, *out.And, 2)

		assert.Equal(t, lo.ToPtr("a"), (*out.And)[0].Gte)
		require.NotNil(t, (*out.And)[1].Or)
		require.Len(t, *(*out.And)[1].Or, 2)
		assert.Equal(t, lo.ToPtr("x"), (*(*out.And)[1].Or)[0].Contains)
		assert.Equal(t, lo.ToPtr("y"), (*(*out.And)[1].Or)[1].Contains)
	})

	t.Run("empty filter returns nil", func(t *testing.T) {
		out, err := FromAPIFilterString(&FilterString{})
		require.NoError(t, err)
		assert.Nil(t, out, "empty input must return nil, not an And of nothing")
	})

	t.Run("result validates against pkg/filter single-operator rule", func(t *testing.T) {
		cases := map[string]FilterString{
			"eq":                    {Eq: lo.ToPtr("a")},
			"neq":                   {Neq: lo.ToPtr("a")},
			"contains":              {Contains: lo.ToPtr("a")},
			"single gt":             {Gt: lo.ToPtr("a")},
			"closed range":          {Gte: lo.ToPtr("a"), Lte: lo.ToPtr("z")},
			"oeq":                   {Oeq: []string{"a", "b"}},
			"ocontains":             {Ocontains: []string{"a", "b"}},
			"eq plus contains":      {Eq: lo.ToPtr("foo"), Contains: lo.ToPtr("oo")},
			"neq plus closed range": {Neq: lo.ToPtr("x"), Gte: lo.ToPtr("a"), Lte: lo.ToPtr("z")},
			"ocontains plus gte":    {Ocontains: []string{"x", "y"}, Gte: lo.ToPtr("a")},
		}
		for name, in := range cases {
			t.Run(name, func(t *testing.T) {
				out, err := FromAPIFilterString(&in)
				require.NoError(t, err)
				require.NoError(t, out.Validate(), "converter output must satisfy pkg/filter.FilterString.Validate")
			})
		}
	})
}

func TestConvertFilterStringPtr(t *testing.T) {
	t.Run("nil returns nil", func(t *testing.T) {
		out, err := FromAPIFilterString(nil)
		require.NoError(t, err)
		assert.Nil(t, out)
	})

	t.Run("empty returns nil", func(t *testing.T) {
		out, err := FromAPIFilterString(&FilterString{})
		require.NoError(t, err)
		assert.Nil(t, out)
	})

	t.Run("non-empty returns value", func(t *testing.T) {
		out, err := FromAPIFilterString(&FilterString{Eq: lo.ToPtr("v")})
		require.NoError(t, err)
		require.NotNil(t, out)
		assert.Equal(t, lo.ToPtr("v"), out.Eq)
	})
}

func TestConvertFilterLabel(t *testing.T) {
	t.Run("nil returns nil", func(t *testing.T) {
		out, err := FromAPIFilterLabel(nil)
		require.NoError(t, err)
		assert.Nil(t, out)
	})

	t.Run("empty returns nil", func(t *testing.T) {
		out, err := FromAPIFilterLabel(&FilterLabel{})
		require.NoError(t, err)
		assert.Nil(t, out)
	})

	t.Run("eq", func(t *testing.T) {
		out, err := FromAPIFilterLabel(&FilterLabel{Eq: lo.ToPtr("prod")})
		require.NoError(t, err)
		require.NotNil(t, out)
		assert.Equal(t, lo.ToPtr("prod"), out.Eq)
	})

	t.Run("neq maps to Ne", func(t *testing.T) {
		out, err := FromAPIFilterLabel(&FilterLabel{Neq: lo.ToPtr("dev")})
		require.NoError(t, err)
		require.NotNil(t, out)
		assert.Equal(t, lo.ToPtr("dev"), out.Ne)
	})

	t.Run("contains", func(t *testing.T) {
		out, err := FromAPIFilterLabel(&FilterLabel{Contains: lo.ToPtr("pro")})
		require.NoError(t, err)
		require.NotNil(t, out)
		assert.Equal(t, lo.ToPtr("pro"), out.Contains)
	})

	t.Run("oeq maps to In", func(t *testing.T) {
		out, err := FromAPIFilterLabel(&FilterLabel{Oeq: []string{"a", "b"}})
		require.NoError(t, err)
		require.NotNil(t, out)
		require.NotNil(t, out.In)
		assert.Equal(t, []string{"a", "b"}, *out.In)
	})

	t.Run("ocontains maps to Or of Contains", func(t *testing.T) {
		out, err := FromAPIFilterLabel(&FilterLabel{Ocontains: []string{"x", "y"}})
		require.NoError(t, err)
		require.NotNil(t, out)
		require.NotNil(t, out.Or)
		require.Len(t, *out.Or, 2)
		assert.Equal(t, lo.ToPtr("x"), (*out.Or)[0].Contains)
		assert.Equal(t, lo.ToPtr("y"), (*out.Or)[1].Contains)
	})

	t.Run("multiple ops produce And", func(t *testing.T) {
		out, err := FromAPIFilterLabel(&FilterLabel{
			Eq:  lo.ToPtr("prod"),
			Neq: lo.ToPtr("dev"),
		})
		require.NoError(t, err)
		require.NotNil(t, out)
		require.NotNil(t, out.And)
		require.Len(t, *out.And, 2)
	})
}

func TestConvertFilterLabels(t *testing.T) {
	t.Run("nil returns nil", func(t *testing.T) {
		out, err := FromAPIFilterLabels(nil)
		require.NoError(t, err)
		assert.Nil(t, out)
	})

	t.Run("single key", func(t *testing.T) {
		labels := FilterLabels{"env": FilterLabel{Eq: lo.ToPtr("prod")}}
		out, err := FromAPIFilterLabels(&labels)
		require.NoError(t, err)
		require.Len(t, out, 1)
		assert.Equal(t, lo.ToPtr("prod"), out["env"].Eq)
	})

	t.Run("multiple keys", func(t *testing.T) {
		labels := FilterLabels{
			"env":    FilterLabel{Eq: lo.ToPtr("prod")},
			"region": FilterLabel{Contains: lo.ToPtr("us")},
		}
		out, err := FromAPIFilterLabels(&labels)
		require.NoError(t, err)
		require.Len(t, out, 2)
		assert.Equal(t, lo.ToPtr("prod"), out["env"].Eq)
		assert.Equal(t, lo.ToPtr("us"), out["region"].Contains)
	})
}

func TestConvertFilterStringExact(t *testing.T) {
	t.Run("eq and neq", func(t *testing.T) {
		out, err := FromAPIFilterStringExact(&FilterStringExact{Eq: lo.ToPtr("v")})
		require.NoError(t, err)
		assert.Equal(t, lo.ToPtr("v"), out.Eq)
	})

	t.Run("oeq maps to In", func(t *testing.T) {
		out, err := FromAPIFilterStringExact(&FilterStringExact{Oeq: []string{"a", "b"}})
		require.NoError(t, err)
		require.NotNil(t, out.In)
		assert.Equal(t, []string{"a", "b"}, *out.In)
	})
}

func TestConvertFilterNumeric(t *testing.T) {
	t.Run("single-bound gt maps directly", func(t *testing.T) {
		out, err := FromAPIFilterNumeric(&FilterNumeric{Gt: lo.ToPtr(1.0)})
		require.NoError(t, err)
		assert.Equal(t, lo.ToPtr(1.0), out.Gt)
		assert.Nil(t, out.And)
	})

	t.Run("gt and lte are split into an And", func(t *testing.T) {
		out, err := FromAPIFilterNumeric(&FilterNumeric{
			Gt:  lo.ToPtr(1.0),
			Lte: lo.ToPtr(10.0),
		})
		require.NoError(t, err)
		assert.Nil(t, out.Gt, "top-level Gt must be empty when range is split")
		assert.Nil(t, out.Lte, "top-level Lte must be empty when range is split")
		require.NotNil(t, out.And)
		require.Len(t, *out.And, 2)
		assert.Equal(t, lo.ToPtr(1.0), (*out.And)[0].Gt)
		assert.Equal(t, lo.ToPtr(10.0), (*out.And)[1].Lte)
	})

	t.Run("oeq maps to Or of Eq", func(t *testing.T) {
		out, err := FromAPIFilterNumeric(&FilterNumeric{Oeq: []float64{1.0, 2.0}})
		require.NoError(t, err)
		require.NotNil(t, out.Or)
		require.Len(t, *out.Or, 2)
		assert.Equal(t, lo.ToPtr(1.0), (*out.Or)[0].Eq)
		assert.Equal(t, lo.ToPtr(2.0), (*out.Or)[1].Eq)
	})

	t.Run("neq", func(t *testing.T) {
		out, err := FromAPIFilterNumeric(&FilterNumeric{Neq: lo.ToPtr(5.0)})
		require.NoError(t, err)
		assert.Equal(t, lo.ToPtr(5.0), out.Ne)
	})
}

func TestConvertFilterDateTime(t *testing.T) {
	t.Run("eq maps to filter.FilterTime Eq", func(t *testing.T) {
		ts := time.Date(2024, 6, 15, 12, 0, 0, 0, time.UTC)
		out, err := FromAPIFilterDateTime(&FilterDateTime{Eq: &ts})
		require.NoError(t, err)
		require.NotNil(t, out.Eq)
		assert.Equal(t, ts, *out.Eq)
		assert.Nil(t, out.Gte, "eq must not fall back to Gte")
	})

	t.Run("single-bound gt maps directly", func(t *testing.T) {
		ts := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
		out, err := FromAPIFilterDateTime(&FilterDateTime{Gt: &ts})
		require.NoError(t, err)
		assert.Equal(t, ts, *out.Gt)
		assert.Nil(t, out.And)
	})

	t.Run("gt and lt are split into an And", func(t *testing.T) {
		gt := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
		lt := time.Date(2024, 12, 31, 23, 59, 59, 0, time.UTC)
		out, err := FromAPIFilterDateTime(&FilterDateTime{Gt: &gt, Lt: &lt})
		require.NoError(t, err)
		assert.Nil(t, out.Gt, "top-level Gt must be empty when range is split")
		assert.Nil(t, out.Lt, "top-level Lt must be empty when range is split")
		require.NotNil(t, out.And)
		require.Len(t, *out.And, 2)
		assert.Equal(t, gt, *(*out.And)[0].Gt)
		assert.Equal(t, lt, *(*out.And)[1].Lt)
	})

	t.Run("gte and lte are split into an And", func(t *testing.T) {
		gte := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
		lte := time.Date(2024, 12, 31, 23, 59, 59, 0, time.UTC)
		out, err := FromAPIFilterDateTime(&FilterDateTime{Gte: &gte, Lte: &lte})
		require.NoError(t, err)
		require.NotNil(t, out.And)
		require.Len(t, *out.And, 2)
		assert.Equal(t, gte, *(*out.And)[0].Gte)
		assert.Equal(t, lte, *(*out.And)[1].Lte)
	})

	t.Run("nil returns nil", func(t *testing.T) {
		out, err := FromAPIFilterDateTime(nil)
		require.NoError(t, err)
		assert.Nil(t, out)
	})
}

func TestConvertFilterBoolean(t *testing.T) {
	t.Run("eq true", func(t *testing.T) {
		out, err := FromAPIFilterBoolean(&FilterBoolean{Eq: lo.ToPtr(true)})
		require.NoError(t, err)
		assert.Equal(t, &filter.FilterBoolean{Eq: lo.ToPtr(true)}, out)
	})

	t.Run("nil ptr returns nil", func(t *testing.T) {
		out, err := FromAPIFilterBoolean(nil)
		require.NoError(t, err)
		assert.Nil(t, out)
	})
}
