package filter_test

import (
	"reflect"
	"testing"

	"github.com/huandu/go-sqlbuilder"
	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/pkg/filter"
)

// The field is an expression carrying its own bound argument, which SelectWhereExpr
// cannot express.
func TestFilterString_SelectWhereExprBuilder_ParameterisedField(t *testing.T) {
	field := sqlbuilder.Buildf("JSON_VALUE(%s, %v)", sqlbuilder.Raw("data"), "$.region")

	expr := filter.FilterString{Eq: lo.ToPtr("eu")}.SelectWhereExprBuilder(field)
	require.NotNil(t, expr)

	q := sqlbuilder.ClickHouse.NewSelectBuilder()
	q.Select("*").From("events")
	q.Where(q.Var(expr))

	sql, args := q.Build()

	assert.Equal(t, "SELECT * FROM events WHERE JSON_VALUE(data, ?) = ?", sql)
	assert.Equal(t, []any{"$.region", "eu"}, args, "the field's argument must precede the filter's, in SQL text order")
}

// ClickHouse lower() folds ASCII only, so the LOWER(...) fallback must not replace
// native ILIKE where the flavor has it.
func TestFilterString_SelectWhereExprBuilder_CaseInsensitiveLikeByFlavor(t *testing.T) {
	tests := []struct {
		name     string
		flavor   sqlbuilder.Flavor
		filter   filter.FilterString
		wantSQL  string
		wantArgs []any
	}{
		{
			name:     "clickhouse ilike uses the native operator",
			flavor:   sqlbuilder.ClickHouse,
			filter:   filter.FilterString{Ilike: lo.ToPtr("%eu%")},
			wantSQL:  "SELECT * FROM events WHERE col ILIKE ?",
			wantArgs: []any{"%eu%"},
		},
		{
			name:     "clickhouse nilike uses the native operator",
			flavor:   sqlbuilder.ClickHouse,
			filter:   filter.FilterString{Nilike: lo.ToPtr("%eu%")},
			wantSQL:  "SELECT * FROM events WHERE col NOT ILIKE ?",
			wantArgs: []any{"%eu%"},
		},
		{
			name:     "clickhouse contains uses the native operator",
			flavor:   sqlbuilder.ClickHouse,
			filter:   filter.FilterString{Contains: lo.ToPtr("eu")},
			wantSQL:  "SELECT * FROM events WHERE col ILIKE ?",
			wantArgs: []any{"%eu%"},
		},
		{
			name:     "clickhouse ncontains uses the native operator",
			flavor:   sqlbuilder.ClickHouse,
			filter:   filter.FilterString{Ncontains: lo.ToPtr("eu")},
			wantSQL:  "SELECT * FROM events WHERE col NOT ILIKE ?",
			wantArgs: []any{"%eu%"},
		},
		{
			name:     "flavors without ilike fall back to lower",
			flavor:   sqlbuilder.MySQL,
			filter:   filter.FilterString{Ilike: lo.ToPtr("%eu%")},
			wantSQL:  "SELECT * FROM events WHERE LOWER(col) LIKE LOWER(?)",
			wantArgs: []any{"%eu%"},
		},
		{
			name:     "flavors without ilike fall back to lower when negated",
			flavor:   sqlbuilder.MySQL,
			filter:   filter.FilterString{Nilike: lo.ToPtr("%eu%")},
			wantSQL:  "SELECT * FROM events WHERE LOWER(col) NOT LIKE LOWER(?)",
			wantArgs: []any{"%eu%"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			expr := tt.filter.SelectWhereExprBuilder(sqlbuilder.Buildf("%v", sqlbuilder.Raw("col")))
			require.NotNil(t, expr)

			q := tt.flavor.NewSelectBuilder()
			q.Select("*").From("events")
			q.Where(q.Var(expr))

			sql, args := q.Build()

			assert.Equal(t, tt.wantSQL, sql)
			assert.Equal(t, tt.wantArgs, args)
		})
	}
}

// An unwired operator returns nil, silently dropping the predicate and widening the
// result set; fail if any operator is unhandled.
func TestFilterString_SelectWhereExprBuilder_CoversEveryOperator(t *testing.T) {
	filterStringType := reflect.TypeFor[filter.FilterString]()

	for i := range filterStringType.NumField() {
		operator := filterStringType.Field(i)

		t.Run(operator.Name, func(t *testing.T) {
			f := filter.FilterString{}
			reflect.ValueOf(&f).Elem().Field(i).Set(sampleOperatorValue(t, operator.Type))

			expr := f.SelectWhereExprBuilder(sqlbuilder.Buildf("%v", sqlbuilder.Raw("col")))

			require.NotNil(t, expr, "operator %s yields no expression: wire it into SelectWhereExprBuilder", operator.Name)
		})
	}
}

// sampleOperatorValue returns a representative value for an operator field, failing
// loudly on an unhandled type so a new operator shape cannot skip the coverage check.
func sampleOperatorValue(t *testing.T, operatorType reflect.Type) reflect.Value {
	t.Helper()

	require.Equal(t, reflect.Pointer, operatorType.Kind(), "expected every operator to be a pointer")

	value := reflect.New(operatorType.Elem())

	switch elem := operatorType.Elem(); elem.Kind() {
	case reflect.String:
		value.Elem().SetString("sample")
	case reflect.Bool:
		value.Elem().SetBool(true)
	case reflect.Slice:
		switch elem.Elem().Kind() {
		case reflect.String:
			value.Elem().Set(reflect.ValueOf([]string{"sample"}))
		case reflect.Struct:
			value.Elem().Set(reflect.ValueOf([]filter.FilterString{{Eq: lo.ToPtr("sample")}}))
		default:
			t.Fatalf("unhandled operator slice element %s: extend sampleOperatorValue", elem.Elem())
		}
	default:
		t.Fatalf("unhandled operator type %s: extend sampleOperatorValue", operatorType)
	}

	return value
}
