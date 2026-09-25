package entutils

import (
	"fmt"

	"entgo.io/ent/dialect/sql"

	"github.com/openmeterio/openmeter/pkg/filter"
	"github.com/openmeterio/openmeter/pkg/slicesx"
)

// JSONBIn returns a function that filters the given JSONB field by the given key and value
// Caveats:
// - PostgreSQL only
// - The field must be a JSONB field
// - The value must be a string (no support for other types, ->> converts all values to string)
// - This might not work if there's a join involved in the query, so add unit tests
func JSONBIn(field string, key string, values []string) func(*sql.Selector) {
	return func(s *sql.Selector) {
		// This is just a safeguard, it should never happen, but if it's not in place, then if
		// len(values) == 0, then generated SQL query will be field->>'key' IN (), which is invalid in SQL
		if len(values) == 0 {
			s.Where(sql.P(func(b *sql.Builder) {
				b.WriteString("false")
			}))
			return
		}
		s.Where(sql.P(func(b *sql.Builder) {
			b.WriteString("(")
			b.WriteString(field)
			b.WriteString("->>'")
			b.WriteString(key)
			b.WriteString("' IN (")
			b.Args(slicesx.Map(values, func(f string) any {
				return f
			})...)
			b.WriteString(")")
			b.WriteString(")")
		}))
	}
}

// JSONBKeyExistsInObject returns a function that filters the given JSONB field by mandating that a key exists in
// a specifc object.
//
// Example:
// given the field value of
//
//	{"failed": false, "immutable": false, "availableActions": {"delete": {"resultingState": "deleted"}, "advance": {"resultingState": "draft.waiting_auto_approval"}}}
//
// JSONBKeyExistsInObject("status_details_cache", "availableActions", "advance")
//
//	filters for such records that have the advance as an available action.
//
// Resulting condition:
//
//	status_details_cache -> 'availableActions' ? 'advance'
func JSONBKeyExistsInObject(field string, member string, expectedKey string) func(*sql.Selector) {
	return func(s *sql.Selector) {
		s.Where(sql.P(func(b *sql.Builder) {
			b.WriteString("(")
			b.WriteString(field)
			b.WriteString("->'")
			b.WriteString(member)
			b.WriteString("' ? '")
			b.WriteString(expectedKey)
			b.WriteString("')")
		}))
	}
}

// JSONBFilterString applies a string filter to the text value stored under key of a
// JSONB field. Only Eq, Ne, In, and And/Or of those are supported; the annotation
// filters that use this never carry other operators. A missing key behaves like a
// NULL column, so Ne does not match rows without the key. An empty filter yields a
// nil predicate. PostgreSQL only.
func JSONBFilterString(field string, key string, f filter.FilterString) (func(*sql.Selector), error) {
	accessor := func(s *sql.Selector, b *sql.Builder) {
		b.WriteString("(").Ident(s.C(field)).WriteString("->>").Arg(key).WriteString(")")
	}

	pred := func(write func(*sql.Selector, *sql.Builder)) func(*sql.Selector) {
		return func(s *sql.Selector) {
			s.Where(sql.P(func(b *sql.Builder) { write(s, b) }))
		}
	}

	switch {
	case f.IsEmpty():
		return nil, nil
	case f.And != nil:
		return jsonbCombine(field, key, *f.And, sql.AndPredicates)
	case f.Or != nil:
		return jsonbCombine(field, key, *f.Or, sql.OrPredicates)
	case f.Eq != nil:
		return pred(func(s *sql.Selector, b *sql.Builder) {
			accessor(s, b)
			b.WriteString(" = ").Arg(*f.Eq)
		}), nil
	case f.Ne != nil:
		return pred(func(s *sql.Selector, b *sql.Builder) {
			accessor(s, b)
			b.WriteString(" <> ").Arg(*f.Ne)
		}), nil
	case f.In != nil:
		if len(*f.In) == 0 {
			return pred(func(_ *sql.Selector, b *sql.Builder) { b.WriteString("FALSE") }), nil
		}
		return pred(func(s *sql.Selector, b *sql.Builder) {
			accessor(s, b)
			b.WriteString(" IN (").Args(slicesx.Map(*f.In, func(v string) any { return v })...).WriteString(")")
		}), nil
	default:
		return nil, fmt.Errorf("unsupported operator in filter on jsonb key %q", key)
	}
}

// JSONBFilterULID is JSONBFilterString for ULID filters, which nest their own And/Or.
func JSONBFilterULID(field string, key string, f filter.FilterULID) (func(*sql.Selector), error) {
	switch {
	case f.And != nil:
		return jsonbCombine(field, key, *f.And, sql.AndPredicates)
	case f.Or != nil:
		return jsonbCombine(field, key, *f.Or, sql.OrPredicates)
	default:
		return JSONBFilterString(field, key, f.FilterString)
	}
}

func jsonbCombine[F filter.FilterString | filter.FilterULID](
	field string,
	key string,
	children []F,
	combine func(...func(*sql.Selector)) func(*sql.Selector),
) (func(*sql.Selector), error) {
	preds := make([]func(*sql.Selector), 0, len(children))
	for _, child := range children {
		var (
			p   func(*sql.Selector)
			err error
		)
		switch c := any(child).(type) {
		case filter.FilterString:
			p, err = JSONBFilterString(field, key, c)
		case filter.FilterULID:
			p, err = JSONBFilterULID(field, key, c)
		}
		if err != nil {
			return nil, err
		}
		if p != nil {
			preds = append(preds, p)
		}
	}
	if len(preds) == 0 {
		return nil, nil
	}
	return combine(preds...), nil
}
