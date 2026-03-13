package aip160_test

import (
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/pkg/filter/aip160"
)

// ── Filter helpers ────────────────────────────────────────────────────────────

func TestFilter_IsEmpty(t *testing.T) {
	assert.True(t, aip160.Filter("").IsEmpty())
	assert.True(t, aip160.Filter("   ").IsEmpty())
	assert.False(t, aip160.Filter("name[eq]=foo").IsEmpty())
}

func TestFilter_UnmarshalText(t *testing.T) {
	var f aip160.Filter
	require.NoError(t, f.UnmarshalText([]byte("name[eq]=foo")))
	assert.Equal(t, aip160.Filter("name[eq]=foo"), f)
}

// ── Parse (simple format: field[op]=value) ────────────────────────────────────

func TestParse_Empty(t *testing.T) {
	conditions, err := aip160.Parse("")
	require.NoError(t, err)
	assert.Nil(t, conditions)
}

func TestParse_EqImplicit(t *testing.T) {
	conditions, err := aip160.Parse("name=Foo+User")
	require.NoError(t, err)
	require.Len(t, conditions, 1)
	assert.Equal(t, aip160.FieldFilter{Field: "name", Operator: aip160.OpEq, Value: "Foo User"}, conditions[0])
}

func TestParse_EqExplicit(t *testing.T) {
	conditions, err := aip160.Parse("name[eq]=Foo+User")
	require.NoError(t, err)
	require.Len(t, conditions, 1)
	assert.Equal(t, aip160.FieldFilter{Field: "name", Operator: aip160.OpEq, Value: "Foo User"}, conditions[0])
}

func TestParse_Neq(t *testing.T) {
	conditions, err := aip160.Parse("name[neq]=Batman")
	require.NoError(t, err)
	require.Len(t, conditions, 1)
	assert.Equal(t, aip160.FieldFilter{Field: "name", Operator: aip160.OpNeq, Value: "Batman"}, conditions[0])
}

func TestParse_OEq(t *testing.T) {
	conditions, err := aip160.Parse("city[oeq]=San+Francisco,London")
	require.NoError(t, err)
	require.Len(t, conditions, 1)
	assert.Equal(t, aip160.FieldFilter{Field: "city", Operator: aip160.OpOEq, Values: []string{"San Francisco", "London"}}, conditions[0])
}

func TestParse_Contains(t *testing.T) {
	conditions, err := aip160.Parse("email[contains]=@konghq.com")
	require.NoError(t, err)
	require.Len(t, conditions, 1)
	assert.Equal(t, aip160.FieldFilter{Field: "email", Operator: aip160.OpContains, Value: "@konghq.com"}, conditions[0])
}

func TestParse_OContains(t *testing.T) {
	conditions, err := aip160.Parse("email[ocontains]=smith,jones")
	require.NoError(t, err)
	require.Len(t, conditions, 1)
	assert.Equal(t, aip160.FieldFilter{Field: "email", Operator: aip160.OpOContains, Values: []string{"smith", "jones"}}, conditions[0])
}

func TestParse_Comparison(t *testing.T) {
	for _, tc := range []struct {
		input string
		op    aip160.Operator
	}{
		{"age[lt]=60", aip160.OpLt},
		{"age[lte]=60", aip160.OpLte},
		{"age[gt]=30", aip160.OpGt},
		{"age[gte]=30", aip160.OpGte},
	} {
		conditions, err := aip160.Parse(tc.input)
		require.NoError(t, err)
		require.Len(t, conditions, 1)
		assert.Equal(t, tc.op, conditions[0].Operator)
		assert.Equal(t, "age", conditions[0].Field)
	}
}

func TestParse_Exists(t *testing.T) {
	conditions, err := aip160.Parse("deleted_time")
	require.NoError(t, err)
	require.Len(t, conditions, 1)
	assert.Equal(t, aip160.FieldFilter{Field: "deleted_time", Operator: aip160.OpExists}, conditions[0])
}

func TestParse_Nexists(t *testing.T) {
	conditions, err := aip160.Parse("labels.activity[nexists]")
	require.NoError(t, err)
	require.Len(t, conditions, 1)
	assert.Equal(t, aip160.FieldFilter{Field: "labels.activity", Operator: aip160.OpNexists}, conditions[0])
}

func TestParse_DotNotation(t *testing.T) {
	conditions, err := aip160.Parse("labels.owner[eq]=alice")
	require.NoError(t, err)
	require.Len(t, conditions, 1)
	assert.Equal(t, aip160.FieldFilter{Field: "labels.owner", Operator: aip160.OpEq, Value: "alice"}, conditions[0])
}

func TestParse_MultipleConditions(t *testing.T) {
	conditions, err := aip160.Parse("name[contains]=Wayne&age[gt]=60")
	require.NoError(t, err)
	require.Len(t, conditions, 2)

	byField := indexByField(conditions)
	assert.Equal(t, aip160.FieldFilter{Field: "name", Operator: aip160.OpContains, Value: "Wayne"}, byField["name"])
	assert.Equal(t, aip160.FieldFilter{Field: "age", Operator: aip160.OpGt, Value: "60"}, byField["age"])
}

func TestParse_UnsupportedOperator(t *testing.T) {
	_, err := aip160.Parse("name[unknown]=foo")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported operator")
}

func TestParse_MissingValue(t *testing.T) {
	_, err := aip160.Parse("name[eq]=")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "requires a non-empty value")
}

// ── ParseFromValues (deep-object: ?filter[field][op]=value) ──────────────────

func TestParseFromValues_Single(t *testing.T) {
	// Equivalent to: ?filter[labels.key_1][eq]=val_A
	values := url.Values{
		"filter[labels.key_1][eq]": {"val_A"},
	}
	conditions, err := aip160.ParseFromValues(values, "filter")
	require.NoError(t, err)
	require.Len(t, conditions, 1)
	assert.Equal(t, aip160.FieldFilter{Field: "labels.key_1", Operator: aip160.OpEq, Value: "val_A"}, conditions[0])
}

func TestParseFromValues_Multiple(t *testing.T) {
	// ?filter[name][contains]=Wayne&filter[age][gt]=60
	values := url.Values{
		"filter[name][contains]": {"Wayne"},
		"filter[age][gt]":        {"60"},
	}
	conditions, err := aip160.ParseFromValues(values, "filter")
	require.NoError(t, err)
	require.Len(t, conditions, 2)

	byField := indexByField(conditions)
	assert.Equal(t, aip160.FieldFilter{Field: "name", Operator: aip160.OpContains, Value: "Wayne"}, byField["name"])
	assert.Equal(t, aip160.FieldFilter{Field: "age", Operator: aip160.OpGt, Value: "60"}, byField["age"])
}

func TestParseFromValues_Existence(t *testing.T) {
	// ?filter[deleted_time]  (no value — field exists)
	values := url.Values{
		"filter[deleted_time]": {""},
	}
	conditions, err := aip160.ParseFromValues(values, "filter")
	require.NoError(t, err)
	require.Len(t, conditions, 1)
	assert.Equal(t, aip160.FieldFilter{Field: "deleted_time", Operator: aip160.OpExists}, conditions[0])
}

func TestParseFromValues_Nexists(t *testing.T) {
	// ?filter[labels.activity][nexists]
	values := url.Values{
		"filter[labels.activity][nexists]": {""},
	}
	conditions, err := aip160.ParseFromValues(values, "filter")
	require.NoError(t, err)
	require.Len(t, conditions, 1)
	assert.Equal(t, aip160.FieldFilter{Field: "labels.activity", Operator: aip160.OpNexists}, conditions[0])
}

func TestParseFromValues_OEq(t *testing.T) {
	// ?filter[city][oeq]=London,Paris
	values := url.Values{
		"filter[city][oeq]": {"London,Paris"},
	}
	conditions, err := aip160.ParseFromValues(values, "filter")
	require.NoError(t, err)
	require.Len(t, conditions, 1)
	assert.Equal(t, aip160.FieldFilter{Field: "city", Operator: aip160.OpOEq, Values: []string{"London", "Paris"}}, conditions[0])
}

func TestParseFromValues_IgnoresOtherParams(t *testing.T) {
	// Only "filter[...]" keys should be processed; others are ignored.
	values := url.Values{
		"filter[name][eq]": {"foo"},
		"sort":             {"name asc"},
		"page":             {"1"},
	}
	conditions, err := aip160.ParseFromValues(values, "filter")
	require.NoError(t, err)
	require.Len(t, conditions, 1)
	assert.Equal(t, "name", conditions[0].Field)
}

func TestParseFromValues_DotNotation(t *testing.T) {
	// ?filter[labels.key_1][eq]=val_A&filter[labels.key_2][contains]=val
	values := url.Values{
		"filter[labels.key_1][eq]":       {"val_A"},
		"filter[labels.key_2][contains]": {"val"},
	}
	conditions, err := aip160.ParseFromValues(values, "filter")
	require.NoError(t, err)
	require.Len(t, conditions, 2)

	byField := indexByField(conditions)
	assert.Equal(t, "val_A", byField["labels.key_1"].Value)
	assert.Equal(t, "val", byField["labels.key_2"].Value)
}

// ── Filter.Parse auto-detection ───────────────────────────────────────────────

func TestFilterParse_AutoDetectSimple(t *testing.T) {
	f := aip160.Filter("name[eq]=Bruce+Wayne&age[gt]=30")
	conditions, err := f.Parse()
	require.NoError(t, err)
	require.Len(t, conditions, 2)
}

func TestFilterParse_AutoDetectDeepObject(t *testing.T) {
	// Deep-object string: filter[name][eq]=foo&filter[age][gt]=30
	f := aip160.Filter("filter[name][eq]=Bruce+Wayne&filter[age][gt]=30")
	conditions, err := f.Parse()
	require.NoError(t, err)
	require.Len(t, conditions, 2)

	byField := indexByField(conditions)
	assert.Equal(t, aip160.FieldFilter{Field: "name", Operator: aip160.OpEq, Value: "Bruce Wayne"}, byField["name"])
	assert.Equal(t, aip160.FieldFilter{Field: "age", Operator: aip160.OpGt, Value: "30"}, byField["age"])
}

func TestFilterParse_AutoDetectDeepObject_Existence(t *testing.T) {
	f := aip160.Filter("filter[deleted_time]&filter[name][contains]=Wayne")
	conditions, err := f.Parse()
	require.NoError(t, err)
	require.Len(t, conditions, 2)

	byField := indexByField(conditions)
	assert.Equal(t, aip160.OpExists, byField["deleted_time"].Operator)
	assert.Equal(t, aip160.OpContains, byField["name"].Operator)
}

// ── helpers ───────────────────────────────────────────────────────────────────

func indexByField(conditions []aip160.FieldFilter) map[string]aip160.FieldFilter {
	m := make(map[string]aip160.FieldFilter, len(conditions))
	for _, c := range conditions {
		m[c.Field] = c
	}
	return m
}
