package outbox

import (
	"entgo.io/ent/dialect/sql"
	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/ent/db/eventoutbox"
)

// pendingTransactionQuery makes the oldest pending row the claim for its entire
// source transaction. A locked head keeps siblings ineligible to other workers.
type pendingTransactionQuery struct {
	Topic                string
	MaxAttempts          int
	ExcludedTransactions []string
}

func (q pendingTransactionQuery) Apply(s *sql.Selector) {
	earlier := sql.Table(eventoutbox.Table).As("earlier_outbox")
	s.Where(sql.And(
		sql.EQ(s.C(eventoutbox.FieldTopic), q.Topic),
		sql.LT(s.C(eventoutbox.FieldAttempts), q.MaxAttempts),
		sql.NotExists(sql.Select(earlier.C(eventoutbox.FieldID)).From(earlier).Where(sql.And(
			sql.LT(earlier.C(eventoutbox.FieldAttempts), q.MaxAttempts),
			sql.ColumnsEQ(earlier.C(eventoutbox.FieldTopic), s.C(eventoutbox.FieldTopic)),
			sql.ColumnsEQ(earlier.C(eventoutbox.FieldTransactionID), s.C(eventoutbox.FieldTransactionID)),
			sql.ColumnsLT(earlier.C(eventoutbox.FieldID), s.C(eventoutbox.FieldID)),
		))),
	))
	if len(q.ExcludedTransactions) > 0 {
		s.Where(sql.NotIn(s.C(eventoutbox.FieldTransactionID), lo.ToAnySlice(q.ExcludedTransactions)...))
	}
}
