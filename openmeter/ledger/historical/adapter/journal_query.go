package adapter

import (
	"time"

	"entgo.io/ent/dialect"
	"entgo.io/ent/dialect/sql"

	"github.com/openmeterio/openmeter/openmeter/ent/db/ledgerentry"
	"github.com/openmeterio/openmeter/openmeter/ent/db/ledgertransaction"
	"github.com/openmeterio/openmeter/openmeter/ledger"
)

type accountBookingCoverageQuery struct {
	input ledger.AccountBookingCoverageInput
}

// Verify all candidate prefixes in one pass over the selected accounts. The
// earliest remaining booking is the next time their projection could change.
func (q accountBookingCoverageQuery) SQL() (string, []any) {
	d := sql.Dialect(dialect.Postgres)
	entries := d.Table(ledgerentry.Table)
	tx := d.Table(ledgertransaction.Table).As("tx")
	bookedAt := tx.C(ledgertransaction.FieldBookedAt)
	query := d.Select().From(entries).Join(tx).On(entries.C(ledgerentry.FieldTransactionID), tx.C(ledgertransaction.FieldID))
	for _, filter := range accountEntries(q.input.AccountEntryScope) {
		filter(query)
	}

	query.AppendSelect(sql.Count("*"))
	query.AppendSelectExpr(sql.ExprFunc(func(b *sql.Builder) {
		b.WriteString("MIN(").Ident(bookedAt).WriteString(") FILTER (WHERE ").Join(sql.GT(bookedAt, q.input.AsOf)).WriteByte(')')
	}))
	for _, at := range append([]time.Time{q.input.AsOf}, q.input.Earlier...) {
		query.AppendSelectExpr(sql.ExprFunc(func(b *sql.Builder) {
			b.WriteString("COUNT(*) FILTER (WHERE ").Join(sql.LTE(bookedAt, at)).WriteByte(')')
		}))
	}

	return query.Query()
}
