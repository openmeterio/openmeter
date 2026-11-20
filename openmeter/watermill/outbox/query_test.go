package outbox

import (
	"strings"
	"testing"

	"entgo.io/ent/dialect"
	"entgo.io/ent/dialect/sql"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/ent/db/eventoutbox"
)

func TestScopedTransactionClaimQuery(t *testing.T) {
	builder := sql.Dialect(dialect.Postgres)
	table := builder.Table(eventoutbox.Table)
	selector := builder.Select(table.C(eventoutbox.FieldID)).From(table)
	pendingTransactionQuery{Topics: []string{"system"}, MaxAttempts: 10, EventType: "balance.updated"}.Apply(selector)
	query, args := selector.Query()
	const expected = `
		SELECT "event_outboxes"."id" FROM "event_outboxes"
		WHERE ("event_outboxes"."topic" IN ($1) AND "event_outboxes"."attempts" < $2
			AND NOT EXISTS (SELECT "earlier_outbox"."id" FROM "event_outboxes" AS "earlier_outbox"
				WHERE "earlier_outbox"."attempts" < $3 AND "earlier_outbox"."topic" IN ($4)
				AND "earlier_outbox"."transaction_id" = "event_outboxes"."transaction_id"
				AND "earlier_outbox"."id" < "event_outboxes"."id"))
		AND ("event_outboxes"."metadata" @> $5 AND NOT EXISTS (SELECT "sibling_outbox"."id" FROM "event_outboxes" AS "sibling_outbox"
			WHERE "sibling_outbox"."transaction_id" = "event_outboxes"."transaction_id"
			AND "sibling_outbox"."topic" IN ($6) AND (NOT ("sibling_outbox"."metadata" @> $7))))`
	require.Equal(t, strings.Join(strings.Fields(expected), " "), query)

	const header = `{"ce_type":"balance.updated"}`
	require.Equal(t, []any{"system", 10, 10, "system", header, "system", header}, args)
}
