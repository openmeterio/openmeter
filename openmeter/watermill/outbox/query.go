package outbox

import (
	"entgo.io/ent/dialect/sql"
	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/ent/db/eventoutbox"
)

// pendingQuery selects the oldest committed row for each delivery key. A locked
// row still blocks later visible rows of the same key.
type pendingQuery struct {
	Topic        string
	ExcludedKeys []string
}

func (q pendingQuery) Apply(s *sql.Selector) {
	earlier := sql.Table(eventoutbox.Table).As("earlier_outbox")
	earlierForKey := sql.Select(earlier.C(eventoutbox.FieldID)).
		From(earlier).
		Where(sql.And(
			sql.IsNull(earlier.C(eventoutbox.FieldDeletedAt)),
			sql.ColumnsEQ(earlier.C(eventoutbox.FieldTopic), s.C(eventoutbox.FieldTopic)),
			sql.ColumnsEQ(earlier.C(eventoutbox.FieldDeliveryKey), s.C(eventoutbox.FieldDeliveryKey)),
			sql.ColumnsLT(earlier.C(eventoutbox.FieldID), s.C(eventoutbox.FieldID)),
		))

	s.Where(sql.And(
		sql.EQ(s.C(eventoutbox.FieldTopic), q.Topic),
		sql.IsNull(s.C(eventoutbox.FieldDeletedAt)),
		sql.NotExists(earlierForKey),
	))
	if len(q.ExcludedKeys) > 0 {
		s.Where(sql.NotIn(s.C(eventoutbox.FieldDeliveryKey), lo.ToAnySlice(q.ExcludedKeys)...))
	}
}
