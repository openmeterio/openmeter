package filters

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testFilter is a target exercising every filter type Parse recognizes.
type testFilter struct {
	Field     *FilterString      `json:"field,omitempty"`
	Name      *FilterString      `json:"name,omitempty"`
	Email     *FilterString      `json:"email,omitempty"`
	Labels    *FilterString      `json:"labels,omitempty"`
	Status    *FilterStringExact `json:"status,omitempty"`
	Count     *FilterNumeric     `json:"count,omitempty"`
	CreatedAt *FilterDateTime    `json:"created_at,omitempty"`
	Enabled   *FilterBoolean     `json:"enabled,omitempty"`
	Currency  *string            `json:"currency,omitempty"`
}

func TestParse_FilterString(t *testing.T) {
	t.Run("eq shorthand", func(t *testing.T) {
		var f testFilter
		require.NoError(t, Parse(url.Values{"filter[field]": {"my-value"}}, &f))
		require.NotNil(t, f.Field)
		assert.Equal(t, lo.ToPtr("my-value"), f.Field.Eq)
	})

	t.Run("explicit eq", func(t *testing.T) {
		var f testFilter
		require.NoError(t, Parse(url.Values{"filter[field][eq]": {"my-value"}}, &f))
		require.NotNil(t, f.Field)
		assert.Equal(t, lo.ToPtr("my-value"), f.Field.Eq)
	})

	t.Run("neq", func(t *testing.T) {
		var f testFilter
		require.NoError(t, Parse(url.Values{"filter[field][neq]": {"excluded"}}, &f))
		require.NotNil(t, f.Field)
		assert.Equal(t, lo.ToPtr("excluded"), f.Field.Neq)
	})

	t.Run("contains", func(t *testing.T) {
		var f testFilter
		require.NoError(t, Parse(url.Values{"filter[field][contains]": {"partial"}}, &f))
		require.NotNil(t, f.Field)
		assert.Equal(t, lo.ToPtr("partial"), f.Field.Contains)
	})

	t.Run("oeq comma separated", func(t *testing.T) {
		var f testFilter
		require.NoError(t, Parse(url.Values{"filter[field][oeq]": {"a,b,c"}}, &f))
		require.NotNil(t, f.Field)
		assert.Equal(t, []string{"a", "b", "c"}, f.Field.Oeq)
	})

	t.Run("oeq trims whitespace and URL-encoded spaces", func(t *testing.T) {
		// parseCommaSeparated must trim leading/trailing whitespace (including
		// decoded %20) so each value is stored without padding.
		var f testFilter
		require.NoError(t, Parse(url.Values{"filter[field][oeq]": {"foo , bar,duck"}}, &f))
		require.NotNil(t, f.Field)
		assert.Equal(t, []string{"foo", "bar", "duck"}, f.Field.Oeq)
	})

	t.Run("ocontains comma separated", func(t *testing.T) {
		var f testFilter
		require.NoError(t, Parse(url.Values{"filter[field][ocontains]": {"foo, bar"}}, &f))
		require.NotNil(t, f.Field)
		assert.Equal(t, []string{"foo", "bar"}, f.Field.Ocontains)
	})

	t.Run("gt/gte/lt/lte individually", func(t *testing.T) {
		cases := map[string]func(*FilterString) *string{
			"gt":  func(f *FilterString) *string { return f.Gt },
			"gte": func(f *FilterString) *string { return f.Gte },
			"lt":  func(f *FilterString) *string { return f.Lt },
			"lte": func(f *FilterString) *string { return f.Lte },
		}
		for op, get := range cases {
			t.Run(op, func(t *testing.T) {
				var f testFilter
				qs := url.Values{"filter[field][" + op + "]": {"abc"}}
				require.NoError(t, Parse(qs, &f))
				require.NotNil(t, f.Field)
				assert.Equal(t, lo.ToPtr("abc"), get(f.Field))
			})
		}
	})

	t.Run("combined gte and lte on one field", func(t *testing.T) {
		var f testFilter
		qs := url.Values{
			"filter[field][gte]": {"666"},
			"filter[field][lte]": {"999"},
		}
		require.NoError(t, Parse(qs, &f))
		require.NotNil(t, f.Field)
		assert.Equal(t, lo.ToPtr("666"), f.Field.Gte)
		assert.Equal(t, lo.ToPtr("999"), f.Field.Lte)
	})

	t.Run("exists", func(t *testing.T) {
		var f testFilter
		require.NoError(t, Parse(url.Values{"filter[field][exists]": {""}}, &f))
		require.NotNil(t, f.Field)
		assert.Equal(t, lo.ToPtr(true), f.Field.Exists)
	})

	t.Run("nexists", func(t *testing.T) {
		var f testFilter
		require.NoError(t, Parse(url.Values{"filter[field][nexists]": {""}}, &f))
		require.NotNil(t, f.Field)
		assert.Equal(t, lo.ToPtr(false), f.Field.Exists)
	})

	t.Run("bare key existence", func(t *testing.T) {
		var f testFilter
		require.NoError(t, Parse(url.Values{"filter[field]": {""}}, &f))
		require.NotNil(t, f.Field)
		assert.Equal(t, lo.ToPtr(true), f.Field.Exists)
	})

	t.Run("no filter keys leaves nil", func(t *testing.T) {
		var f testFilter
		require.NoError(t, Parse(url.Values{"page[size]": {"10"}}, &f))
		assert.Nil(t, f.Field)
	})

	t.Run("unknown operator returns error", func(t *testing.T) {
		var f testFilter
		err := Parse(url.Values{"filter[field][like]": {"value"}}, &f)
		assert.ErrorIs(t, err, ErrUnsupportedOperator)
	})

	t.Run("empty operator filter[field][] is rejected", func(t *testing.T) {
		var f testFilter
		err := Parse(url.Values{"filter[field][]": {"rekt"}}, &f)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "empty operator")
	})

	t.Run("corrupted key filter[field[ is ignored", func(t *testing.T) {
		require.NotPanics(t, func() {
			var f testFilter
			err := Parse(url.Values{"filter[field[": {"boom"}}, &f)
			require.NoError(t, err)
			assert.Nil(t, f.Field)
		})
	})
}

func TestParse_FilterStringExact(t *testing.T) {
	t.Run("eq", func(t *testing.T) {
		var f testFilter
		require.NoError(t, Parse(url.Values{"filter[status][eq]": {"active"}}, &f))
		require.NotNil(t, f.Status)
		assert.Equal(t, lo.ToPtr("active"), f.Status.Eq)
	})

	t.Run("neq", func(t *testing.T) {
		var f testFilter
		require.NoError(t, Parse(url.Values{"filter[status][neq]": {"deleted"}}, &f))
		require.NotNil(t, f.Status)
		assert.Equal(t, lo.ToPtr("deleted"), f.Status.Neq)
	})

	t.Run("oeq", func(t *testing.T) {
		var f testFilter
		require.NoError(t, Parse(url.Values{"filter[status][oeq]": {"active,pending"}}, &f))
		require.NotNil(t, f.Status)
		assert.Equal(t, []string{"active", "pending"}, f.Status.Oeq)
	})

	t.Run("contains is rejected", func(t *testing.T) {
		var f testFilter
		err := Parse(url.Values{"filter[status][contains]": {"act"}}, &f)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "unsupported operator")
	})
}

func TestParse_FilterNumeric(t *testing.T) {
	t.Run("eq", func(t *testing.T) {
		var f testFilter
		require.NoError(t, Parse(url.Values{"filter[count][eq]": {"3.14"}}, &f))
		require.NotNil(t, f.Count)
		assert.Equal(t, lo.ToPtr(3.14), f.Count.Eq)
	})

	t.Run("oeq", func(t *testing.T) {
		var f testFilter
		require.NoError(t, Parse(url.Values{"filter[count][oeq]": {"1,2,3"}}, &f))
		require.NotNil(t, f.Count)
		assert.Equal(t, []float64{1, 2, 3}, f.Count.Oeq)
	})

	t.Run("gte/lte range", func(t *testing.T) {
		var f testFilter
		qs := url.Values{
			"filter[count][gte]": {"1"},
			"filter[count][lte]": {"10"},
		}
		require.NoError(t, Parse(qs, &f))
		require.NotNil(t, f.Count)
		assert.Equal(t, lo.ToPtr(1.0), f.Count.Gte)
		assert.Equal(t, lo.ToPtr(10.0), f.Count.Lte)
	})

	t.Run("non-numeric value is rejected", func(t *testing.T) {
		var f testFilter
		err := Parse(url.Values{"filter[count][eq]": {"not-a-number"}}, &f)
		require.Error(t, err)
	})
}

func TestParse_FilterDateTime(t *testing.T) {
	t.Run("bare key implies exists", func(t *testing.T) {
		var f testFilter
		require.NoError(t, Parse(url.Values{"filter[created_at]": {""}}, &f))
		require.NotNil(t, f.CreatedAt)
		assert.Equal(t, lo.ToPtr(true), f.CreatedAt.Exists)
	})

	t.Run("eq", func(t *testing.T) {
		var f testFilter
		require.NoError(t, Parse(url.Values{"filter[created_at][eq]": {"2024-01-02T03:04:05Z"}}, &f))
		require.NotNil(t, f.CreatedAt)
		require.NotNil(t, f.CreatedAt.Eq)
		assert.Equal(t, time.Date(2024, 1, 2, 3, 4, 5, 0, time.UTC), *f.CreatedAt.Eq)
	})

	t.Run("gt and lte range", func(t *testing.T) {
		var f testFilter
		qs := url.Values{
			"filter[created_at][gt]":  {"2024-01-01T00:00:00Z"},
			"filter[created_at][lte]": {"2024-12-31T23:59:59Z"},
		}
		require.NoError(t, Parse(qs, &f))
		require.NotNil(t, f.CreatedAt)
		require.NotNil(t, f.CreatedAt.Gt)
		require.NotNil(t, f.CreatedAt.Lte)
		assert.Equal(t, time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), *f.CreatedAt.Gt)
		assert.Equal(t, time.Date(2024, 12, 31, 23, 59, 59, 0, time.UTC), *f.CreatedAt.Lte)
	})

	t.Run("invalid datetime is rejected", func(t *testing.T) {
		var f testFilter
		err := Parse(url.Values{"filter[created_at][eq]": {"not-a-date"}}, &f)
		assert.ErrorIs(t, err, ErrInvalidDateTime)
	})

	t.Run("neq is rejected", func(t *testing.T) {
		var f testFilter
		err := Parse(url.Values{"filter[created_at][neq]": {"2024-01-01T00:00:00Z"}}, &f)
		assert.ErrorIs(t, err, ErrUnsupportedOperator)
	})

	t.Run("oeq is rejected", func(t *testing.T) {
		var f testFilter
		err := Parse(url.Values{"filter[created_at][oeq]": {"2024-01-01T00:00:00Z"}}, &f)
		assert.ErrorIs(t, err, ErrUnsupportedOperator)
	})
}

func TestParse_FilterBoolean(t *testing.T) {
	t.Run("eq true", func(t *testing.T) {
		var f testFilter
		require.NoError(t, Parse(url.Values{"filter[enabled][eq]": {"true"}}, &f))
		require.NotNil(t, f.Enabled)
		assert.Equal(t, lo.ToPtr(true), f.Enabled.Eq)
	})

	t.Run("eq false", func(t *testing.T) {
		var f testFilter
		require.NoError(t, Parse(url.Values{"filter[enabled][eq]": {"false"}}, &f))
		require.NotNil(t, f.Enabled)
		assert.Equal(t, lo.ToPtr(false), f.Enabled.Eq)
	})

	t.Run("non-boolean value is rejected", func(t *testing.T) {
		var f testFilter
		err := Parse(url.Values{"filter[enabled][eq]": {"yes"}}, &f)
		require.Error(t, err)
	})

	t.Run("unsupported operator is rejected", func(t *testing.T) {
		var f testFilter
		err := Parse(url.Values{"filter[enabled][gt]": {"true"}}, &f)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "unsupported operator")
	})
}

func TestParse_StringPtr(t *testing.T) {
	t.Run("simple string value", func(t *testing.T) {
		var f testFilter
		require.NoError(t, Parse(url.Values{"filter[currency]": {"USD"}}, &f))
		assert.Equal(t, lo.ToPtr("USD"), f.Currency)
	})
}

func TestParse_PointerToPointer(t *testing.T) {
	t.Run("allocates pointer when filter keys exist", func(t *testing.T) {
		var f *testFilter
		require.NoError(t, Parse(url.Values{"filter[field][eq]": {"val"}}, &f))
		require.NotNil(t, f)
		require.NotNil(t, f.Field)
		assert.Equal(t, lo.ToPtr("val"), f.Field.Eq)
	})

	t.Run("stays nil when no filter keys", func(t *testing.T) {
		var f *testFilter
		require.NoError(t, Parse(url.Values{"sort": {"name"}}, &f))
		assert.Nil(t, f)
	})
}

func TestParse_UnknownFilterKey(t *testing.T) {
	t.Run("unknown field returns error", func(t *testing.T) {
		var f testFilter
		err := Parse(url.Values{"filter[unknown][eq]": {"x"}}, &f)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "unknown filter field")
		assert.Contains(t, err.Error(), "unknown")
	})

	t.Run("unknown field mixed with known field returns error", func(t *testing.T) {
		var f testFilter
		qs := url.Values{
			"filter[name][eq]":    {"ok"},
			"filter[unknown][eq]": {"x"},
		}
		err := Parse(qs, &f)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "unknown")
	})

	t.Run("unknown bare field returns error", func(t *testing.T) {
		var f testFilter
		err := Parse(url.Values{"filter[unknown]": {"x"}}, &f)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "unknown")
	})

	t.Run("multiple unknown fields listed sorted", func(t *testing.T) {
		var f testFilter
		qs := url.Values{
			"filter[zeta][eq]":  {"1"},
			"filter[alpha][eq]": {"2"},
		}
		err := Parse(qs, &f)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "alpha, zeta")
	})

	t.Run("unknown field on pointer-to-pointer path returns error", func(t *testing.T) {
		var f *testFilter
		err := Parse(url.Values{"filter[unknown][eq]": {"x"}}, &f)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "unknown")
	})

	t.Run("non-filter keys are not validated", func(t *testing.T) {
		var f testFilter
		qs := url.Values{
			"page[size]": {"10"},
			"sort":       {"name"},
		}
		require.NoError(t, Parse(qs, &f))
	})

	t.Run("known field with unknown operator is still rejected by parser", func(t *testing.T) {
		var f testFilter
		err := Parse(url.Values{"filter[field][wibble]": {"x"}}, &f)
		assert.ErrorIs(t, err, ErrUnsupportedOperator)
	})
}

func TestParse_DotNotation(t *testing.T) {
	t.Run("dot-notation with known base passes the unknown-field check", func(t *testing.T) {
		var f testFilter
		qs := url.Values{
			"filter[labels.env]":             {"prod"},
			"filter[labels.owner][neq]":      {"kong"},
			"filter[labels.tier][ocontains]": {"gold,platinum"},
		}
		require.NoError(t, Parse(qs, &f))
	})

	t.Run("multi-dot is delimited on the first dot only", func(t *testing.T) {
		var f testFilter
		err := Parse(url.Values{"filter[labels.a.b.c][eq]": {"foo"}}, &f)
		require.NoError(t, err)
	})

	t.Run("single dot filter[.] is rejected", func(t *testing.T) {
		var f testFilter
		err := Parse(url.Values{"filter[.][eq]": {"foo"}}, &f)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "unknown filter field")
	})

	t.Run("empty base filter[.key_1] is rejected", func(t *testing.T) {
		var f testFilter
		err := Parse(url.Values{"filter[.key_1][eq]": {"foo"}}, &f)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "unknown filter field")
	})

	t.Run("unknown base with dot-notation is rejected", func(t *testing.T) {
		var f testFilter
		err := Parse(url.Values{"filter[unknownbase.key][eq]": {"foo"}}, &f)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "unknownbase.key")
	})

	t.Run("dot-notation typo on a known field name is rejected", func(t *testing.T) {
		var f testFilter
		err := Parse(url.Values{"filter[nmae.env]": {"prod"}}, &f)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "nmae.env")
	})
}

func TestParse_Complex(t *testing.T) {
	t.Run("mixed eq, range, and ocontains on multiple fields", func(t *testing.T) {
		var f testFilter
		qs := url.Values{
			"filter[name][eq]":         {"foo"},
			"filter[field][gte]":       {"1"},
			"filter[field][lte]":       {"10"},
			"filter[email][ocontains]": {"a,b"},
		}
		require.NoError(t, Parse(qs, &f))

		require.NotNil(t, f.Name)
		assert.Equal(t, lo.ToPtr("foo"), f.Name.Eq)

		require.NotNil(t, f.Field)
		assert.Equal(t, lo.ToPtr("1"), f.Field.Gte)
		assert.Equal(t, lo.ToPtr("10"), f.Field.Lte)

		require.NotNil(t, f.Email)
		assert.Equal(t, []string{"a", "b"}, f.Email.Ocontains)
	})

	t.Run("unrelated page and sort params do not populate filters", func(t *testing.T) {
		var f testFilter
		qs := url.Values{
			"page[size]":   {"10"},
			"sort":         {"name desc"},
			"filter[name]": {"x"},
		}
		require.NoError(t, Parse(qs, &f))
		require.NotNil(t, f.Name)
		assert.Equal(t, lo.ToPtr("x"), f.Name.Eq)
		assert.Nil(t, f.Field)
	})
}

func TestParse_InvalidTarget(t *testing.T) {
	t.Run("nil target is rejected", func(t *testing.T) {
		err := Parse(url.Values{}, nil)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "non-nil pointer")
	})

	t.Run("non-pointer target is rejected", func(t *testing.T) {
		var f testFilter
		err := Parse(url.Values{}, f)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "non-nil pointer")
	})

	t.Run("typed nil pointer is rejected", func(t *testing.T) {
		var p *testFilter
		err := Parse(url.Values{}, p)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "non-nil pointer")
	})

	t.Run("pointer to non-struct is rejected", func(t *testing.T) {
		i := 42
		err := Parse(url.Values{"filter[field][eq]": {"foo"}}, &i)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "must point to a struct")
	})

	t.Run("pointer to pointer to non-struct is rejected", func(t *testing.T) {
		var mp *map[string]string
		err := Parse(url.Values{"filter[field][eq]": {"foo"}}, &mp)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "must point to a struct")
	})
}

func TestParse_MultiValueKeyRejected(t *testing.T) {
	t.Run("FilterString duplicate eq", func(t *testing.T) {
		var f testFilter
		err := Parse(url.Values{"filter[field][eq]": {"a", "b"}}, &f)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "repeated query parameter")
	})

	t.Run("FilterStringExact duplicate eq", func(t *testing.T) {
		var f testFilter
		err := Parse(url.Values{"filter[status][eq]": {"a", "b"}}, &f)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "repeated query parameter")
	})

	t.Run("FilterNumeric duplicate gte", func(t *testing.T) {
		var f testFilter
		err := Parse(url.Values{"filter[count][gte]": {"1", "2"}}, &f)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "repeated query parameter")
	})

	t.Run("FilterDateTime duplicate gt", func(t *testing.T) {
		var f testFilter
		err := Parse(url.Values{"filter[created_at][gt]": {"2024-01-01T00:00:00Z", "2024-06-01T00:00:00Z"}}, &f)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "repeated query parameter")
	})

	t.Run("FilterBoolean duplicate eq", func(t *testing.T) {
		var f testFilter
		err := Parse(url.Values{"filter[enabled][eq]": {"true", "false"}}, &f)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "repeated query parameter")
	})

	t.Run("stringPtr duplicate", func(t *testing.T) {
		var f testFilter
		err := Parse(url.Values{"filter[currency]": {"USD", "EUR"}}, &f)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "repeated query parameter")
	})
}

func TestParse_CommaSeparatedCapEnforced(t *testing.T) {
	// parseCommaSeparated enforces maxCommaSeparatedItems to bound DoS
	// amplification from filter[field][ocontains]=a,b,... where N items
	// would each compile to a leading-wildcard ILIKE term.
	t.Run("FilterString ocontains under cap succeeds", func(t *testing.T) {
		var items []string
		for i := range maxCommaSeparatedItems {
			items = append(items, fmt.Sprintf("v%d", i))
		}
		value := strings.Join(items, ",")

		var f testFilter
		require.NoError(t, Parse(url.Values{"filter[field][ocontains]": {value}}, &f))
		require.NotNil(t, f.Field)
		assert.Len(t, f.Field.Ocontains, maxCommaSeparatedItems)
	})

	t.Run("FilterString ocontains over cap is rejected", func(t *testing.T) {
		var items []string
		for i := range maxCommaSeparatedItems + 1 {
			items = append(items, fmt.Sprintf("v%d", i))
		}
		value := strings.Join(items, ",")

		var f testFilter
		err := Parse(url.Values{"filter[field][ocontains]": {value}}, &f)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "too many comma-separated items")
	})

	t.Run("FilterStringExact oeq over cap is rejected", func(t *testing.T) {
		var items []string
		for i := range maxCommaSeparatedItems + 1 {
			items = append(items, fmt.Sprintf("v%d", i))
		}
		value := strings.Join(items, ",")

		var f testFilter
		err := Parse(url.Values{"filter[status][oeq]": {value}}, &f)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "too many comma-separated items")
	})

	t.Run("FilterNumeric oeq over cap is rejected", func(t *testing.T) {
		var items []string
		for i := range maxCommaSeparatedItems + 1 {
			items = append(items, strconv.Itoa(i))
		}
		value := strings.Join(items, ",")

		var f testFilter
		err := Parse(url.Values{"filter[count][oeq]": {value}}, &f)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "too many comma-separated items")
	})
}

func TestParse_ValueLengthCapEnforced(t *testing.T) {
	t.Run("exactly at cap is accepted", func(t *testing.T) {
		value := strings.Repeat("a", maxFilterValueLength)
		var f testFilter
		require.NoError(t, Parse(url.Values{"filter[field][contains]": {value}}, &f))
	})

	t.Run("one over cap is rejected", func(t *testing.T) {
		value := strings.Repeat("a", maxFilterValueLength+1)
		var f testFilter
		err := Parse(url.Values{"filter[field][contains]": {value}}, &f)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "value too long")
	})
}

func TestParse_FilterNumericInvalidPerOp(t *testing.T) {
	for _, op := range []string{"eq", "neq", "gt", "gte", "lt", "lte", "oeq"} {
		t.Run(op, func(t *testing.T) {
			var f testFilter
			err := Parse(url.Values{"filter[count][" + op + "]": {"not-a-number"}}, &f)
			require.Error(t, err)
			assert.Contains(t, err.Error(), op, "error must name the offending operator")
		})
	}
}

func TestParse_EmptyTypedValueRejected(t *testing.T) {
	// Numeric bare filters carry no meaningful "exists" semantics and should be
	// rejected instead of being treated as unfiltered input.
	t.Run("numeric bare empty rejected", func(t *testing.T) {
		var f testFilter
		err := Parse(url.Values{"filter[count]": {""}}, &f)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "filter[count]")
		assert.Contains(t, err.Error(), "empty")
	})
}

func TestParse_AdversarialInputs(t *testing.T) {
	cases := map[string]string{
		"null bytes":         "abc\x00def",
		"invalid utf-8":      "\xff\xfe",
		"control characters": "\x01\x02\x03",
		"newlines":           "a\r\nb",
	}
	for name, value := range cases {
		t.Run(name, func(t *testing.T) {
			var f testFilter
			require.NotPanics(t, func() {
				_ = Parse(url.Values{"filter[field][eq]": {value}}, &f)
			})
		})
	}

	t.Run("oversized value rejected, no panic", func(t *testing.T) {
		huge := strings.Repeat("A", 1<<20) // 1 MiB
		var f testFilter
		require.NotPanics(t, func() {
			err := Parse(url.Values{"filter[field][eq]": {huge}}, &f)
			// Must be rejected by the value length cap; never accepted.
			require.Error(t, err)
			assert.Contains(t, err.Error(), "value too long")
		})
	})
}

// testFilterWithLabels exercises the FilterLabels (map[string]FilterLabel) path.
type testFilterWithLabels struct {
	Name   *FilterString `json:"name,omitempty"`
	Labels FilterLabels  `json:"labels,omitempty"`
}

func TestParse_FilterLabels(t *testing.T) {
	t.Run("single label eq shorthand", func(t *testing.T) {
		var f testFilterWithLabels
		qs := url.Values{"filter[labels.env]": {"prod"}}
		require.NoError(t, Parse(qs, &f))
		require.NotNil(t, f.Labels)
		require.Contains(t, f.Labels, "env")
		assert.Equal(t, lo.ToPtr("prod"), f.Labels["env"].Eq)
	})

	t.Run("single label explicit eq", func(t *testing.T) {
		var f testFilterWithLabels
		qs := url.Values{"filter[labels.env][eq]": {"prod"}}
		require.NoError(t, Parse(qs, &f))
		require.Contains(t, f.Labels, "env")
		assert.Equal(t, lo.ToPtr("prod"), f.Labels["env"].Eq)
	})

	t.Run("single label neq", func(t *testing.T) {
		var f testFilterWithLabels
		qs := url.Values{"filter[labels.env][neq]": {"dev"}}
		require.NoError(t, Parse(qs, &f))
		require.Contains(t, f.Labels, "env")
		assert.Equal(t, lo.ToPtr("dev"), f.Labels["env"].Neq)
	})

	t.Run("single label contains", func(t *testing.T) {
		var f testFilterWithLabels
		qs := url.Values{"filter[labels.env][contains]": {"pro"}}
		require.NoError(t, Parse(qs, &f))
		require.Contains(t, f.Labels, "env")
		assert.Equal(t, lo.ToPtr("pro"), f.Labels["env"].Contains)
	})

	t.Run("single label oeq", func(t *testing.T) {
		var f testFilterWithLabels
		qs := url.Values{"filter[labels.env][oeq]": {"prod,staging"}}
		require.NoError(t, Parse(qs, &f))
		require.Contains(t, f.Labels, "env")
		assert.Equal(t, []string{"prod", "staging"}, f.Labels["env"].Oeq)
	})

	t.Run("single label ocontains", func(t *testing.T) {
		var f testFilterWithLabels
		qs := url.Values{"filter[labels.env][ocontains]": {"pro,stag"}}
		require.NoError(t, Parse(qs, &f))
		require.Contains(t, f.Labels, "env")
		assert.Equal(t, []string{"pro", "stag"}, f.Labels["env"].Ocontains)
	})

	t.Run("multiple label keys", func(t *testing.T) {
		var f testFilterWithLabels
		qs := url.Values{
			"filter[labels.env][eq]":    {"prod"},
			"filter[labels.region][eq]": {"us-east"},
		}
		require.NoError(t, Parse(qs, &f))
		require.Len(t, f.Labels, 2)
		assert.Equal(t, lo.ToPtr("prod"), f.Labels["env"].Eq)
		assert.Equal(t, lo.ToPtr("us-east"), f.Labels["region"].Eq)
	})

	t.Run("multiple ops on same label key", func(t *testing.T) {
		var f testFilterWithLabels
		qs := url.Values{
			"filter[labels.env][eq]":  {"prod"},
			"filter[labels.env][neq]": {"staging"},
		}
		require.NoError(t, Parse(qs, &f))
		require.Contains(t, f.Labels, "env")
		assert.Equal(t, lo.ToPtr("prod"), f.Labels["env"].Eq)
		assert.Equal(t, lo.ToPtr("staging"), f.Labels["env"].Neq)
	})

	t.Run("no label keys leaves nil", func(t *testing.T) {
		var f testFilterWithLabels
		qs := url.Values{"filter[name][eq]": {"foo"}}
		require.NoError(t, Parse(qs, &f))
		assert.Nil(t, f.Labels)
	})

	t.Run("mixed labels and other fields", func(t *testing.T) {
		var f testFilterWithLabels
		qs := url.Values{
			"filter[name][eq]":      {"foo"},
			"filter[labels.env]":    {"prod"},
			"filter[labels.region]": {"eu"},
		}
		require.NoError(t, Parse(qs, &f))
		require.NotNil(t, f.Name)
		assert.Equal(t, lo.ToPtr("foo"), f.Name.Eq)
		require.Len(t, f.Labels, 2)
		assert.Equal(t, lo.ToPtr("prod"), f.Labels["env"].Eq)
		assert.Equal(t, lo.ToPtr("eu"), f.Labels["region"].Eq)
	})

	t.Run("unsupported operator is rejected", func(t *testing.T) {
		var f testFilterWithLabels
		err := Parse(url.Values{"filter[labels.env][gt]": {"prod"}}, &f)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "unsupported operator")
	})

	t.Run("oeq comma cap enforced", func(t *testing.T) {
		var items []string
		for i := range maxCommaSeparatedItems + 1 {
			items = append(items, fmt.Sprintf("v%d", i))
		}
		var f testFilterWithLabels
		err := Parse(url.Values{"filter[labels.env][oeq]": {strings.Join(items, ",")}}, &f)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "too many comma-separated items")
	})
}

func TestParse_RangeValidation(t *testing.T) {
	t.Run("gte+lte range accepted", func(t *testing.T) {
		var f testFilter
		qs := url.Values{
			"filter[field][gte]": {"1"},
			"filter[field][lte]": {"10"},
		}
		require.NoError(t, Parse(qs, &f))
		require.NotNil(t, f.Field)
		assert.Equal(t, lo.ToPtr("1"), f.Field.Gte)
		assert.Equal(t, lo.ToPtr("10"), f.Field.Lte)
	})

	t.Run("gt+gte accepted (two lower bounds)", func(t *testing.T) {
		var f testFilter
		qs := url.Values{
			"filter[field][gt]":  {"1"},
			"filter[field][gte]": {"2"},
		}
		require.NoError(t, Parse(qs, &f))
		require.NotNil(t, f.Field)
		assert.Equal(t, lo.ToPtr("1"), f.Field.Gt)
		assert.Equal(t, lo.ToPtr("2"), f.Field.Gte)
	})

	t.Run("lt+lte accepted (two upper bounds)", func(t *testing.T) {
		var f testFilter
		qs := url.Values{
			"filter[field][lt]":  {"10"},
			"filter[field][lte]": {"20"},
		}
		require.NoError(t, Parse(qs, &f))
		require.NotNil(t, f.Field)
		assert.Equal(t, lo.ToPtr("10"), f.Field.Lt)
		assert.Equal(t, lo.ToPtr("20"), f.Field.Lte)
	})

	t.Run("eq+gte accepted (non-range + range)", func(t *testing.T) {
		var f testFilter
		qs := url.Values{
			"filter[field][eq]":  {"a"},
			"filter[field][gte]": {"b"},
		}
		require.NoError(t, Parse(qs, &f))
		require.NotNil(t, f.Field)
		assert.Equal(t, lo.ToPtr("a"), f.Field.Eq)
		assert.Equal(t, lo.ToPtr("b"), f.Field.Gte)
	})

	t.Run("eq+contains accepted (two non-range)", func(t *testing.T) {
		var f testFilter
		qs := url.Values{
			"filter[field][eq]":       {"a"},
			"filter[field][contains]": {"b"},
		}
		require.NoError(t, Parse(qs, &f))
		require.NotNil(t, f.Field)
		assert.Equal(t, lo.ToPtr("a"), f.Field.Eq)
		assert.Equal(t, lo.ToPtr("b"), f.Field.Contains)
	})
}
