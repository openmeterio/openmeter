package adapter

import (
	"reflect"
	"slices"
	"time"

	"entgo.io/ent/dialect"
	"entgo.io/ent/dialect/sql"

	ledgerentrydb "github.com/openmeterio/openmeter/openmeter/ent/db/ledgerentry"
	ledgertransactiondb "github.com/openmeterio/openmeter/openmeter/ent/db/ledgertransaction"
	"github.com/openmeterio/openmeter/openmeter/ledger"
)

type balancesAtBoundariesQuery struct {
	input ledger.GetBalancesAtBoundariesInput
}

type balanceBoundaryScope struct {
	query    ledger.Query
	latestAt time.Time
}

func (q balancesAtBoundariesQuery) SQL() (string, []any, error) {
	if err := q.input.Validate(); err != nil {
		return "", nil, err
	}

	// History rows usually share just two scopes per currency: FBO and advance
	// receivable. Read each scope once instead of summing its history per row.
	scopes, scopeIndexes := q.scopes()
	if len(scopes) == len(q.input.Queries) {
		// With no shared scopes, a plain SUM avoids sorting the entire history.
		return q.independentSums()
	}

	var movements *sql.Selector
	for idx, scope := range scopes {
		entries, err := scope.movements(idx)
		if err != nil {
			return "", nil, err
		}

		if movements == nil {
			movements = entries
		} else {
			movements.UnionAll(entries)
		}
	}

	// Put zero-amount markers at the requested boundaries. The running total
	// at each marker is its balance, even when that scope has no entries.
	movements.UnionAll(q.boundaries(scopeIndexes))
	d := sql.Dialect(dialect.Postgres)
	ctes := d.With("movements").As(movements).
		With("running_balances").As(d.Select("boundary_index").
		AppendSelectExprAs(sql.Window(func(b *sql.Builder) {
			b.Ident(sql.Sum("amount"))
		}).PartitionBy("scope_index").OrderExpr(sql.Raw(
			"booked_at, created_at NULLS LAST, transaction_id NULLS LAST, boundary_index NULLS FIRST",
		)), "balance").From(d.Table("movements")))
	query, args := d.Select("balance", "boundary_index").From(d.Table("running_balances")).
		Where(sql.NotNull("boundary_index")).Prefix(ctes).Query()

	return query, args, nil
}

func (q balancesAtBoundariesQuery) scopes() ([]balanceBoundaryScope, []int) {
	var scopes []balanceBoundaryScope
	indexes := make([]int, len(q.input.Queries))
	for idx, query := range q.input.Queries {
		var at time.Time
		if query.Filters.After != nil {
			at = query.Filters.After.BookedAt
		} else {
			at = *query.Filters.AsOf
		}

		query.Filters.After = nil
		query.Filters.AsOf = nil
		scopeIdx := slices.IndexFunc(scopes, func(scope balanceBoundaryScope) bool {
			return reflect.DeepEqual(scope.query, query)
		})
		if scopeIdx == -1 {
			scopeIdx = len(scopes)
			scopes = append(scopes, balanceBoundaryScope{query: query, latestAt: at})
		} else if at.After(scopes[scopeIdx].latestAt) {
			scopes[scopeIdx].latestAt = at
		}

		indexes[idx] = scopeIdx
	}

	return scopes, indexes
}

func (s balanceBoundaryScope) movements(index int) (*sql.Selector, error) {
	predicates, err := (&sumEntriesQuery{query: s.query}).entryPredicates()
	if err != nil {
		return nil, err
	}

	d := sql.Dialect(dialect.Postgres)
	entries := d.Table(ledgerentrydb.Table)
	txs := d.Table(ledgertransactiondb.Table).As("transactions")
	selector := d.Select().AppendSelectExprAs(sql.ExprFunc(func(b *sql.Builder) {
		b.Arg(index).WriteString("::integer")
	}), "scope_index").AppendSelect(
		entries.C(ledgerentrydb.FieldAmount),
		txs.C(ledgertransactiondb.FieldBookedAt),
		txs.C(ledgertransactiondb.FieldCreatedAt),
		sql.As(txs.C(ledgertransactiondb.FieldID), "transaction_id"),
		"NULL::integer AS boundary_index",
	).From(entries).Join(txs).On(entries.C(ledgerentrydb.FieldTransactionID), txs.C(ledgertransactiondb.FieldID))
	for _, predicate := range predicates {
		predicate(selector)
	}

	selector.Where(sql.LTE(txs.C(ledgertransactiondb.FieldBookedAt), s.latestAt))

	return selector, nil
}

// A cursor marker sorts after all entries of that transaction. A timestamp
// marker has no created_at or ID, so it sorts after every transaction at that
// booked time. This keeps same-time rows distinct without inventing an ID.
func (q balancesAtBoundariesQuery) boundaries(scopeIndexes []int) *sql.Selector {
	d := sql.Dialect(dialect.Postgres)

	return d.Select("scope_index", "0::numeric AS amount", "booked_at", "created_at", "transaction_id", "boundary_index").
		FromExpr(sql.ExprFunc(func(b *sql.Builder) {
			b.WriteString("(VALUES ")
			for idx, query := range q.input.Queries {
				if idx > 0 {
					b.Comma()
				}

				b.WriteByte('(').Arg(scopeIndexes[idx]).WriteString("::integer, ")
				if cursor := query.Filters.After; cursor != nil {
					b.Arg(cursor.BookedAt).WriteString("::timestamptz, ").
						Arg(cursor.CreatedAt).WriteString("::timestamptz, ").Arg(cursor.ID.ID).WriteString("::bpchar, ")
				} else {
					b.Arg(*query.Filters.AsOf).WriteString("::timestamptz, NULL::timestamptz, NULL::bpchar, ")
				}

				b.Arg(idx).WriteString("::integer)")
			}

			b.WriteString(") AS boundaries(scope_index, booked_at, created_at, transaction_id, boundary_index)")
		}))
}

func (q balancesAtBoundariesQuery) independentSums() (string, []any, error) {
	var combined *sql.Selector
	for idx, query := range q.input.Queries {
		selector, err := (&sumEntriesQuery{query: query}).selector()
		if err != nil {
			return "", nil, err
		}

		selector.AppendSelectExpr(sql.ExprFunc(func(b *sql.Builder) {
			b.Arg(idx).WriteString("::integer AS boundary_index")
		}))
		if combined == nil {
			combined = selector
		} else {
			combined.UnionAll(selector)
		}
	}

	query, args := combined.Query()

	return query, args, nil
}
