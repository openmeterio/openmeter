package softdelete

import (
	"time"

	"entgo.io/ent/dialect/sql"
)

// FieldName is the column used by all soft-delete-bearing schemas. It must
// match `entutils.TimeMixin` and any hand-rolled `field.Time("deleted_at")`
// declarations.
const FieldName = "deleted_at"

// ActivePredicate returns a storage-level predicate matching rows that are
// active relative to `now`: either `deleted_at IS NULL` or `deleted_at >
// now`. The time-windowed semantics (rather than plain IS NULL) match
// existing OpenMeter usage where `DeletedAtGT(now)` is widespread.
func ActivePredicate(now time.Time) func(*sql.Selector) {
	return func(s *sql.Selector) {
		s.Where(sql.Or(
			sql.IsNull(s.C(FieldName)),
			sql.GT(s.C(FieldName), now),
		))
	}
}
