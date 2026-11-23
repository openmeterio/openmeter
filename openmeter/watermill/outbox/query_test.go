package outbox

import (
	"strings"
	"testing"

	"entgo.io/ent/dialect"
	"entgo.io/ent/dialect/sql"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/ent/db/eventoutbox"
)

func TestDeploymentTransactionClaimQuery(t *testing.T) {
	const expected = `
		SELECT "event_outboxes"."id" FROM "event_outboxes"
		WHERE "event_outboxes"."topic" IN ($1) AND "event_outboxes"."attempts" < $2
			AND NOT EXISTS (SELECT "earlier_outbox"."id" FROM "event_outboxes" AS "earlier_outbox"
				WHERE "earlier_outbox"."app_role" = "event_outboxes"."app_role"
				AND "earlier_outbox"."attempts" < $3 AND "earlier_outbox"."topic" IN ($4)
				AND "earlier_outbox"."transaction_id" = "event_outboxes"."transaction_id"
				AND "earlier_outbox"."id" < "event_outboxes"."id")`

	for _, tc := range []struct {
		name          string
		drainAllRoles bool
		wantSQL       string
		wantArgs      []any
	}{
		{"own role", false, expected + ` AND "event_outboxes"."app_role" = $5`, []any{"system", 10, 10, "system", "worker"}},
		{"all roles", true, expected, []any{"system", 10, 10, "system"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			builder := sql.Dialect(dialect.Postgres)
			table := builder.Table(eventoutbox.Table)
			selector := builder.Select(table.C(eventoutbox.FieldID)).From(table)
			pendingTransactionQuery{
				Topics: []string{"system"}, MaxAttempts: 10, AppRole: "worker", DrainAllRoles: tc.drainAllRoles,
			}.Apply(selector)

			query, args := selector.Query()
			require.Equal(t, strings.Join(strings.Fields(tc.wantSQL), " "), query)
			require.Equal(t, tc.wantArgs, args)
		})
	}
}
