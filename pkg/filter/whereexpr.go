package filter

import (
	"strings"

	"github.com/huandu/go-sqlbuilder"
	"github.com/samber/lo"
)

const (
	whereExprAndSeparator = " AND "
	whereExprOrSeparator  = " OR "
)

// SelectWhereExprBuilder builds a WHERE expression around a field that may carry its
// own bound parameters, unlike SelectWhereExpr which takes a plain string. Returns nil
// when no operator is set.
func (f FilterString) SelectWhereExprBuilder(field sqlbuilder.Builder) sqlbuilder.Builder {
	switch {
	case f.Eq != nil:
		return sqlbuilder.Buildf("%v = %v", field, *f.Eq)
	case f.Ne != nil:
		return sqlbuilder.Buildf("%v <> %v", field, *f.Ne)
	case f.Exists != nil:
		if *f.Exists {
			return sqlbuilder.Buildf("%v IS NOT NULL", field)
		}

		return sqlbuilder.Buildf("%v IS NULL", field)
	case f.In != nil:
		return sqlbuilder.Buildf("%v IN (%v)", field, *f.In)
	case f.Nin != nil:
		return sqlbuilder.Buildf("%v NOT IN (%v)", field, *f.Nin)
	case f.Like != nil:
		return sqlbuilder.Buildf("%v LIKE %v", field, *f.Like)
	case f.Nlike != nil:
		return sqlbuilder.Buildf("%v NOT LIKE %v", field, *f.Nlike)
	case f.Ilike != nil:
		return caseInsensitiveLike{field: field, value: *f.Ilike}
	case f.Nilike != nil:
		return caseInsensitiveLike{field: field, value: *f.Nilike, negated: true}
	case f.Contains != nil:
		return caseInsensitiveLike{field: field, value: ContainsPattern(*f.Contains)}
	case f.Ncontains != nil:
		return caseInsensitiveLike{field: field, value: ContainsPattern(*f.Ncontains), negated: true}
	case f.Gt != nil:
		return sqlbuilder.Buildf("%v > %v", field, *f.Gt)
	case f.Gte != nil:
		return sqlbuilder.Buildf("%v >= %v", field, *f.Gte)
	case f.Lt != nil:
		return sqlbuilder.Buildf("%v < %v", field, *f.Lt)
	case f.Lte != nil:
		return sqlbuilder.Buildf("%v <= %v", field, *f.Lte)
	case f.And != nil:
		return joinWhereExprBuilders(field, *f.And, whereExprAndSeparator)
	case f.Or != nil:
		return joinWhereExprBuilders(field, *f.Or, whereExprOrSeparator)
	default:
		return nil
	}
}

// joinWhereExprBuilders combines child expressions, dropping empty operands so an
// all-empty group yields nil instead of "()".
func joinWhereExprBuilders(field sqlbuilder.Builder, filters []FilterString, separator string) sqlbuilder.Builder {
	exprs := make([]any, 0, len(filters))

	for _, child := range filters {
		if expr := child.SelectWhereExprBuilder(field); expr != nil {
			exprs = append(exprs, expr)
		}
	}

	if len(exprs) == 0 {
		return nil
	}

	format := "(" + strings.Join(lo.RepeatBy(len(exprs), func(_ int) string { return "%v" }), separator) + ")"

	return sqlbuilder.Buildf(format, exprs...)
}

// caseInsensitiveLike renders a case-insensitive LIKE: native ILIKE where the flavor
// has it, LOWER(...) LIKE LOWER(...) otherwise. It duplicates go-sqlbuilder's Cond.ILike
// switch (which only takes a plain field string) because ClickHouse lower() folds ASCII
// only, so collapsing to the fallback form would drop non-ASCII matches.
type caseInsensitiveLike struct {
	field   sqlbuilder.Builder
	value   string
	negated bool
}

var _ sqlbuilder.Builder = caseInsensitiveLike{}

func (e caseInsensitiveLike) Flavor() sqlbuilder.Flavor {
	return e.field.Flavor()
}

func (e caseInsensitiveLike) Build() (string, []any) {
	return e.BuildWithFlavor(e.Flavor())
}

func (e caseInsensitiveLike) BuildWithFlavor(flavor sqlbuilder.Flavor, initialArg ...any) (string, []any) {
	var format string

	switch flavor {
	case sqlbuilder.PostgreSQL, sqlbuilder.ClickHouse:
		format = "%v ILIKE %v"
		if e.negated {
			format = "%v NOT ILIKE %v"
		}
	default:
		format = "LOWER(%v) LIKE LOWER(%v)"
		if e.negated {
			format = "LOWER(%v) NOT LIKE LOWER(%v)"
		}
	}

	return sqlbuilder.Buildf(format, e.field, e.value).BuildWithFlavor(flavor, initialArg...)
}
