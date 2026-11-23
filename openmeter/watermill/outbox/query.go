package outbox

import (
	"entgo.io/ent/dialect/sql"
	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/ent/db/eventoutbox"
)

// pendingTransactionQuery makes the oldest pending row the claim for its entire
// source transaction. A locked head keeps siblings ineligible to other workers.
type pendingTransactionQuery struct {
	Topics               []string
	MaxAttempts          int
	ExcludedTransactions []string
	AppRole              string
}

func (q pendingTransactionQuery) Apply(s *sql.Selector) {
	earlier := sql.Table(eventoutbox.Table).As("earlier_outbox")
	s.Where(sql.And(
		sql.In(s.C(eventoutbox.FieldTopic), lo.ToAnySlice(q.Topics)...),
		sql.EQ(s.C(eventoutbox.FieldAppRole), q.AppRole),
		sql.LT(s.C(eventoutbox.FieldAttempts), q.MaxAttempts),
		sql.NotExists(sql.Select(earlier.C(eventoutbox.FieldID)).From(earlier).Where(sql.And(
			sql.ColumnsEQ(earlier.C(eventoutbox.FieldAppRole), s.C(eventoutbox.FieldAppRole)),
			sql.LT(earlier.C(eventoutbox.FieldAttempts), q.MaxAttempts),
			sql.In(earlier.C(eventoutbox.FieldTopic), lo.ToAnySlice(q.Topics)...),
			sql.ColumnsEQ(earlier.C(eventoutbox.FieldTransactionID), s.C(eventoutbox.FieldTransactionID)),
			sql.ColumnsLT(earlier.C(eventoutbox.FieldID), s.C(eventoutbox.FieldID)),
		))),
	))

	if len(q.ExcludedTransactions) > 0 {
		s.Where(sql.NotIn(s.C(eventoutbox.FieldTransactionID), lo.ToAnySlice(q.ExcludedTransactions)...))
	}
}
