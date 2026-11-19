package adapter

import (
	"time"

	"entgo.io/ent/dialect"
	"entgo.io/ent/dialect/sql"
	"github.com/lib/pq"

	demanddb "github.com/openmeterio/openmeter/openmeter/ent/db/rtedemand"
	"github.com/openmeterio/openmeter/openmeter/rte"
)

type renewPullQuery struct {
	identity  rte.TargetIdentity
	kind      rte.CalculationKind
	key       string
	at, until pullRenewalTime
}

func (q renewPullQuery) SQL() (string, []any) {
	pg := sql.Dialect(dialect.Postgres)
	d := pg.Table(demanddb.Table)

	lease := pg.Select(demanddb.FieldID).From(d).Where(sql.And(
		sql.EQ(demanddb.FieldNamespace, q.identity.Namespace),
		sql.EQ(demanddb.FieldKind, q.kind),
		sql.EQ(demanddb.FieldKeyHash, q.identity.KeyHash),
		sql.EQ(demanddb.FieldKey, q.key),
		sql.IsNull(demanddb.FieldDeletedAt),
		sql.IsFalse(demanddb.FieldPushEnabled),
		sql.GT(demanddb.FieldPullUntil, q.at),
	)).ForUpdate(sql.WithLockAction(sql.SkipLocked))

	update := pg.Update(demanddb.Table).
		Set(demanddb.FieldPullUntil, sql.ExprFunc(func(b *sql.Builder) {
			b.WriteString("GREATEST").Wrap(func(b *sql.Builder) {
				b.Ident(demanddb.FieldPullUntil).Comma().Arg(q.until)
			})
		})).
		Set(demanddb.FieldUpdatedAt, q.at).
		Where(sql.EQ(demanddb.FieldID, lease))

	clean := sql.And(
		sql.ColumnsEQ(demanddb.FieldCompletedVersion, demanddb.FieldDirtyVersion),
		sql.Or(sql.IsNull(demanddb.FieldSourceDeleteAt), sql.GT(demanddb.FieldSourceDeleteAt, q.at)),
	)

	// Ent's Returning accepts column names. Here we return the freshness check
	// against the row whose lease was renewed.
	var b sql.Builder
	b.SetDialect(dialect.Postgres)
	return b.Join(update).WriteString(" RETURNING ").Join(clean).Query()
}

type invalidateDemandQuery struct {
	selectors []rte.DemandSelector
	at        time.Time
}

func (q invalidateDemandQuery) SQL() (string, []any, error) {
	pg := sql.Dialect(dialect.Postgres)
	var selectors []*sql.Predicate
	for _, selector := range q.selectors {
		parts := []*sql.Predicate{
			sql.EQ(demanddb.FieldNamespace, selector.Namespace),
			sql.EQ(demanddb.FieldKind, selector.Kind),
		}
		if len(selector.OwnerIDs) > 0 {
			parts = append(parts, matchesAny(demanddb.FieldOwnerID, selector.OwnerIDs))
		}

		if len(selector.Targets) > 0 {
			hashes := make([]string, 0, len(selector.Targets))
			for _, target := range selector.Targets {
				identity, err := target.Identity()
				if err != nil {
					return "", nil, err
				}

				hashes = append(hashes, identity.KeyHash)
			}

			parts = append(parts, matchesAny(demanddb.FieldKeyHash, hashes))
		}

		selectors = append(selectors, sql.And(parts...))
	}

	matching := pg.Select().From(pg.Table(demanddb.Table))
	activeDemand(q.at)(matching)
	update := pg.Update(demanddb.Table).FromSelect(matching).Where(sql.Or(selectors...))
	markDemandDirty(update, q.at)

	query, args := update.Query()
	return query, args, nil
}

// One array argument keeps large usage batches below PostgreSQL's parameter limit.
func matchesAny(column string, values []string) *sql.Predicate {
	return sql.EQ(column, sql.ExprFunc(func(b *sql.Builder) {
		b.WriteString("ANY").Wrap(func(b *sql.Builder) { b.Arg(pq.Array(values)) })
	}))
}

// Failed keys accumulate new work but keep their existing retry deadline.
func markDemandDirty(update *sql.UpdateBuilder, now time.Time) {
	d := sql.Dialect(dialect.Postgres).Table(demanddb.Table)
	// Only a clean key joins the queue. Further events preserve its place.
	update.Add(demanddb.FieldDirtyVersion, 1).
		Set(demanddb.FieldUpdatedAt, now).
		Set(demanddb.FieldNextAttemptAt, sql.ExprFunc(func(b *sql.Builder) {
			b.WriteString("CASE WHEN ").Join(sql.And(sql.EQ(d.C(demanddb.FieldAttempts), 0), sql.ColumnsEQ(d.C(demanddb.FieldDirtyVersion), d.C(demanddb.FieldCompletedVersion)))).
				WriteString(" THEN ").Arg(now).
				WriteString(" ELSE ").Ident(d.C(demanddb.FieldNextAttemptAt)).
				WriteString(" END")
		}))
}
