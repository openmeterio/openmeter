package entutils

import (
	"entgo.io/ent/dialect/sql"

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
