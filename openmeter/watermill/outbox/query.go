package outbox

import (
	"entgo.io/ent/dialect/sql"
	"entgo.io/ent/dialect/sql/sqljson"
	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/ent/db/eventoutbox"
	"github.com/openmeterio/openmeter/openmeter/watermill/marshaler"
)

// pendingTransactionQuery makes the oldest pending row the claim for its entire
// source transaction. A locked head keeps siblings ineligible to other workers.
type pendingTransactionQuery struct {
	Topics               []string
	MaxAttempts          int
	ExcludedTransactions []string
	EventType            string
}

// Equivalent claim SQL, including the ordering, limit, and lock added by drain:
//
//	SELECT e.* FROM event_outboxes e
//	WHERE e.topic IN (:topics)
//	  AND e.attempts < :max_attempts
//	  AND NOT EXISTS (
//	    SELECT 1 FROM event_outboxes earlier
//	    WHERE earlier.topic IN (:topics)
//	      AND earlier.transaction_id = e.transaction_id
//	      AND earlier.attempts < :max_attempts
//	      AND earlier.id < e.id
//	  )
//	  AND e.metadata @> :type_header
//	  AND NOT EXISTS (
//	    SELECT 1 FROM event_outboxes sibling
//	    WHERE sibling.transaction_id = e.transaction_id AND sibling.topic IN (:topics)
//	      AND NOT (sibling.metadata @> :type_header)
//	  )
//	  AND e.transaction_id NOT IN (:excluded_transactions)
//	ORDER BY e.id LIMIT 1 FOR UPDATE SKIP LOCKED;
//
// Both type-header predicates and the excluded transaction predicate are optional.
func (q pendingTransactionQuery) Apply(s *sql.Selector) {
	earlier := sql.Table(eventoutbox.Table).As("earlier_outbox")
	s.Where(sql.And(
		sql.In(s.C(eventoutbox.FieldTopic), lo.ToAnySlice(q.Topics)...),
		sql.LT(s.C(eventoutbox.FieldAttempts), q.MaxAttempts),
		sql.NotExists(sql.Select(earlier.C(eventoutbox.FieldID)).From(earlier).Where(sql.And(
			sql.LT(earlier.C(eventoutbox.FieldAttempts), q.MaxAttempts),
			sql.In(earlier.C(eventoutbox.FieldTopic), lo.ToAnySlice(q.Topics)...),
			sql.ColumnsEQ(earlier.C(eventoutbox.FieldTransactionID), s.C(eventoutbox.FieldTransactionID)),
			sql.ColumnsLT(earlier.C(eventoutbox.FieldID), s.C(eventoutbox.FieldID)),
		))),
	))
	if q.EventType != "" {
		// A scoped worker never decides when to skip an unrelated event, even
		// if it has a lower maxAttempts than the worker that owns that event.
		// Leave mixed transactions to a regular worker that can send them in order.
		header := map[string]string{marshaler.CloudEventsHeaderType: q.EventType}
		sibling := sql.Table(eventoutbox.Table).As("sibling_outbox")
		s.Where(sql.And(
			sqljson.ValueContains(s.C(eventoutbox.FieldMetadata), header),
			sql.NotExists(sql.Select(sibling.C(eventoutbox.FieldID)).From(sibling).Where(sql.And(
				sql.ColumnsEQ(sibling.C(eventoutbox.FieldTransactionID), s.C(eventoutbox.FieldTransactionID)),
				sql.In(sibling.C(eventoutbox.FieldTopic), lo.ToAnySlice(q.Topics)...),
				sql.Not(sqljson.ValueContains(sibling.C(eventoutbox.FieldMetadata), header)),
			))),
		))
	}

	if len(q.ExcludedTransactions) > 0 {
		s.Where(sql.NotIn(s.C(eventoutbox.FieldTransactionID), lo.ToAnySlice(q.ExcludedTransactions)...))
	}
}
