package filter

import (
	"entgo.io/ent/dialect/sql"

	"github.com/openmeterio/openmeter/pkg/slicesx"
)

// JSONBFilter is implemented by filters that can build predicates over the text
// value stored under a key of a PostgreSQL JSONB column.
type JSONBFilter interface {
	Filter
	// SelectJSONB converts the filter to an Ent selector predicate applied to the
	// text value stored under the given key of a JSONB column. PostgreSQL only.
	SelectJSONB(field string, key string) func(*sql.Selector)
}

var (
	_ JSONBFilter = (*FilterString)(nil)
	_ JSONBFilter = (*FilterULID)(nil)
)

// SelectJSONB converts the filter to an Ent selector predicate applied to the text
// value stored under the given key of a JSONB column.
//
// The value is compared as text (through the ->> accessor), so ordering operators use
// lexicographic comparison and a missing key behaves like a NULL column: Ne, Nin and
// the negated pattern operators do not match rows without the key, mirroring the
// column semantics of Select. PostgreSQL only.
func (f FilterString) SelectJSONB(field string, key string) func(*sql.Selector) {
	if f.IsEmpty() {
		return nil
	}

	// accessor writes the parenthesized `field->>'key'` expression, table-qualifying
	// the column so the predicate stays unambiguous under joins.
	accessor := func(s *sql.Selector, b *sql.Builder) {
		b.WriteString("(").Ident(s.C(field)).WriteString("->>").Arg(key).WriteString(")")
	}

	pred := func(write func(s *sql.Selector, b *sql.Builder)) func(*sql.Selector) {
		return func(s *sql.Selector) {
			s.Where(sql.P(func(b *sql.Builder) {
				write(s, b)
			}))
		}
	}

	binary := func(op string, arg any) func(*sql.Selector) {
		return pred(func(s *sql.Selector, b *sql.Builder) {
			accessor(s, b)
			b.WriteString(" ").WriteString(op).WriteString(" ").Arg(arg)
		})
	}

	membership := func(op string, values []string) func(*sql.Selector) {
		return pred(func(s *sql.Selector, b *sql.Builder) {
			accessor(s, b)
			b.WriteString(" ").WriteString(op).WriteString(" (")
			b.Args(slicesx.Map(values, func(v string) any { return v })...)
			b.WriteString(")")
		})
	}

	switch {
	case f.Eq != nil:
		return binary("=", *f.Eq)
	case f.Ne != nil:
		return binary("<>", *f.Ne)
	case f.Exists != nil:
		if *f.Exists {
			return pred(func(s *sql.Selector, b *sql.Builder) {
				accessor(s, b)
				b.WriteString(" IS NOT NULL")
			})
		}
		return pred(func(s *sql.Selector, b *sql.Builder) {
			accessor(s, b)
			b.WriteString(" IS NULL")
		})
	case f.In != nil:
		if len(*f.In) == 0 {
			// `IN ()` is invalid SQL; an empty membership list matches nothing.
			return pred(func(_ *sql.Selector, b *sql.Builder) {
				b.WriteString("FALSE")
			})
		}
		return membership("IN", *f.In)
	case f.Nin != nil:
		if len(*f.Nin) == 0 {
			// `NOT IN ()` would be invalid SQL; an empty exclusion list matches everything.
			return nil
		}
		return membership("NOT IN", *f.Nin)
	case f.Like != nil:
		return binary("LIKE", *f.Like)
	case f.Nlike != nil:
		return binary("NOT LIKE", *f.Nlike)
	case f.Ilike != nil:
		return binary("ILIKE", *f.Ilike)
	case f.Nilike != nil:
		return binary("NOT ILIKE", *f.Nilike)
	case f.Contains != nil:
		return binary("ILIKE", ContainsPattern(*f.Contains))
	case f.Ncontains != nil:
		return binary("NOT ILIKE", ContainsPattern(*f.Ncontains))
	case f.Gt != nil:
		return binary(">", *f.Gt)
	case f.Gte != nil:
		return binary(">=", *f.Gte)
	case f.Lt != nil:
		return binary("<", *f.Lt)
	case f.Lte != nil:
		return binary("<=", *f.Lte)
	case f.And != nil:
		preds := make([]func(*sql.Selector), 0, len(*f.And))
		for _, child := range *f.And {
			if p := child.SelectJSONB(field, key); p != nil {
				preds = append(preds, p)
			}
		}
		return sql.AndPredicates(preds...)
	case f.Or != nil:
		preds := make([]func(*sql.Selector), 0, len(*f.Or))
		for _, child := range *f.Or {
			if p := child.SelectJSONB(field, key); p != nil {
				preds = append(preds, p)
			}
		}
		return sql.OrPredicates(preds...)
	default:
		return nil
	}
}

// SelectJSONB converts the filter to an Ent selector predicate applied to the text
// value stored under the given key of a JSONB column. See FilterString.SelectJSONB.
func (f FilterULID) SelectJSONB(field string, key string) func(*sql.Selector) {
	switch {
	case f.And != nil:
		preds := make([]func(*sql.Selector), 0, len(*f.And))
		for _, child := range *f.And {
			if p := child.SelectJSONB(field, key); p != nil {
				preds = append(preds, p)
			}
		}
		return sql.AndPredicates(preds...)
	case f.Or != nil:
		preds := make([]func(*sql.Selector), 0, len(*f.Or))
		for _, child := range *f.Or {
			if p := child.SelectJSONB(field, key); p != nil {
				preds = append(preds, p)
			}
		}
		return sql.OrPredicates(preds...)
	default:
		return f.FilterString.SelectJSONB(field, key)
	}
}

// ApplyToQueryJSONB applies a filter to the text value stored under the given key of
// a JSONB column if the filter is non-nil and non-empty. PostgreSQL only.
func ApplyToQueryJSONB[F JSONBFilter, Q EntQuery[Q, P], P Predicate](q Q, f *F, field string, key string) Q {
	if f == nil {
		return q
	}

	if s := (*f).SelectJSONB(field, key); s != nil {
		return q.Where(P(s))
	}

	return q
}
