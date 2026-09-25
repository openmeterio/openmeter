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

// Equivalent claim SQL, including the ordering, limit, and lock added by drain:
//
//	SELECT e.* FROM event_outboxes e
//	WHERE e.topic = :topic
//	  AND e.attempts < :max_attempts
//	  AND NOT EXISTS (
//	    SELECT 1 FROM event_outboxes earlier
//	    WHERE earlier.topic = e.topic
//	      AND earlier.transaction_id = e.transaction_id
//	      AND earlier.attempts < :max_attempts
//	      AND earlier.id < e.id
//	  )
//	  AND e.transaction_id NOT IN (:excluded_transactions)
//	ORDER BY e.id LIMIT 1 FOR UPDATE SKIP LOCKED;
//
// The NOT IN clause is omitted when there are no excluded transactions.
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
