# adapter

<!-- archie:ai-start -->

> Ent/PostgreSQL implementation of ledgerhistorical.Repo: books double-entry transactions (LedgerTransaction + LedgerEntry rows), manages transaction groups, sums entries, and lists transactions with cursor-based pagination. All reads must eagerly load SubAccount→Account→Route edges via WithSubAccount(WithAccount, WithRoute) to hydrate domain objects without N+1 queries.

## Patterns

**Interface compliance assertion** — repo.go declares `var _ ledgerhistorical.Repo = (*repo)(nil)` at package level. Every new adapter struct must include this compile-time guard. (`var _ ledgerhistorical.Repo = (*repo)(nil)`)
**Eager edge loading on every query** — Every query that returns a hydrated Transaction must call WithEntries(func(eq){eq.WithSubAccount(func(sq){sq.WithAccount(); sq.WithRoute()})}) or hydrateHistoricalTransaction will error on missing edges. (`q.WithEntries(func(eq *db.LedgerEntryQuery) { eq.WithSubAccount(func(sq *db.LedgerSubAccountQuery) { sq.WithAccount(); sq.WithRoute() }) })`)
**hydrateHistoricalTransaction for domain reconstruction** — All DB rows are converted to domain objects via hydrateHistoricalTransaction, which calls ledgerhistorical.NewTransactionFromData. Never map DB structs to domain objects inline; always go through this function. (`items, err := slicesx.MapWithErr(dbItems, func(tx *db.LedgerTransaction) (*ledgerhistorical.Transaction, error) { return hydrateHistoricalTransaction(tx) })`)
**Tri-tuple cursor pagination (bookedAt, createdAt, ID)** — Pagination uses a three-field (BookedAt DESC, CreatedAt DESC, ID DESC) ordering with SQL predicate-based cursors. Both forward (Cursor) and backward (Before) directions are supported; Before results are slices.Reverse'd after fetch. NextCursor always points to the last returned item. (`ledgerTransactionAfterCursorPredicate / ledgerTransactionBeforeCursorPredicate in ledger.go`)
**sumEntriesQuery builder for aggregation** — SumEntries delegates to sumEntriesQuery.Build(client) which assembles entry predicates from ledger.Query. Add new filter dimensions by extending entryPredicates() and subAccountPredicates() in sumentries_query.go, not inline in repo methods. (`q := sumEntriesQuery{query: query}; entryQuery, err := q.Build(r.db)`)
**Tx via HijackTx + entutils.NewTxDriver** — repo.Tx hijacks the connection via r.db.HijackTx and wraps it with entutils.NewTxDriver. Do not use db.BeginTx or manual transaction management; callers use pkg/framework/transaction.Driver. (`txCtx, rawConfig, eDriver, err := r.db.HijackTx(ctx, &stdsql.TxOptions{ReadOnly: false}); return txCtx, entutils.NewTxDriver(eDriver, rawConfig), nil`)
**Bulk entry creation after transaction row** — BookTransaction first saves the LedgerTransaction row, then calls CreateBulk for all LedgerEntry rows in a single batch. Never insert entries one-by-one in a loop. (`createdEntries, err = r.db.LedgerEntry.CreateBulk(createInputs...).Save(ctx)`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `repo.go` | Defines the repo struct holding *db.Client, the NewRepo constructor, the Tx method, and the interface compliance assertion. Start here when adding new repo methods. | Do not store additional state on repo beyond *db.Client; inject dependencies via constructor. |
| `ledger.go` | Contains BookTransaction, CreateTransactionGroup, GetTransactionGroup, SumEntries, ListTransactions, and all cursor/ordering helpers. Core read/write operations live here. | hydrateHistoricalTransaction panics if SubAccount→Account or SubAccount→Route edges are not loaded; always eager-load them in every query. |
| `sumentries_query.go` | Encapsulates the sumEntriesQuery builder with entryPredicates() and subAccountPredicates(). Also exposes SQL() for raw SQL output (used in view generation tooling). | Route filters call b.query.Filters.Route.Normalize() — normalization errors must propagate as ledger.ErrLedgerQueryInvalid, not a raw error. |
| `ledger_test.go` | In-package DB integration tests using NewTestEnv(t) + env.DBSchemaMigrate(t). Tests cover pagination (forward, backward, no-overlap), filters (currency, creditMovement, annotations), and BookTransaction invariants. | Tests call time.Sleep between transactions to ensure distinct CreatedAt timestamps for ordering — do not remove these sleeps. |

## Anti-Patterns

- Loading edges lazily or omitting WithSubAccount/WithAccount/WithRoute — hydrateHistoricalTransaction will return an error on missing edges
- Inline DB-to-domain mapping instead of routing through hydrateHistoricalTransaction
- Using context.Background() instead of the caller-provided ctx in any repo method
- Inserting LedgerEntry rows one-by-one instead of using CreateBulk
- Adding filter logic directly inside ListTransactions or SumEntries instead of extending sumEntriesQuery or a dedicated predicate builder

## Decisions

- **Cursor uses three fields (bookedAt, createdAt, ID) rather than a single auto-increment offset** — BookedAt is the user-visible ordering field and can repeat; tie-breaking on createdAt+ID makes pagination stable across concurrent inserts without requiring a serial primary key.
- **sumEntriesQuery is a separate struct with a Build method and a SQL() method** — Isolates filter predicate logic from repo methods and enables the viewgen tool to extract raw SQL without executing a query.
- **Tx uses HijackTx + entutils.NewTxDriver instead of db.BeginTx** — Aligns with the project-wide entutils.TransactingRepo pattern so callers can bind the transaction via ctx; avoids leaking raw *sql.Tx across package boundaries.

## Example: Add a new filtered query method that sums entries scoped by a new dimension

```
// In sumentries_query.go, extend subAccountPredicates:
if b.query.Filters.SomeDimension != nil {
    routePredicates = append(routePredicates, ledgersubaccountroutedb.SomeDimension(*b.query.Filters.SomeDimension))
}
// In ledger.go, the repo method delegates unchanged:
func (r *repo) SumEntriesByDimension(ctx context.Context, query ledger.Query) (alpacadecimal.Decimal, error) {
    q := sumEntriesQuery{query: query}
    entryQuery, err := q.Build(r.db)
    if err != nil {
        return alpacadecimal.Decimal{}, err
    }
    var rows []struct{ SumAmount stdsql.NullString `json:"sum_amount,omitempty"` }
    err = entryQuery.Aggregate(db.As(db.Sum(ledgerentrydb.FieldAmount), "sum_amount")).Scan(ctx, &rows)
    // ... parse and return
}
```

<!-- archie:ai-end -->
