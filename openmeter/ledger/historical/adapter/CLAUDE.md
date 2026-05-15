# adapter

<!-- archie:ai-start -->

> Ent/PostgreSQL implementation of ledgerhistorical.Repo: books double-entry transactions (LedgerTransaction + LedgerEntry rows), manages transaction groups, sums entries, and lists transactions with tri-tuple cursor pagination. All queries must eagerly load SubAccount→Account→Route edges or hydrateHistoricalTransaction will error.

## Patterns

**Full TxCreator+TxUser triad on repo** — repo implements Tx (via r.db.HijackTx + entutils.NewTxDriver), WithTx (via db.NewTxClientFromRawConfig), and Self(). Every method body wraps with entutils.TransactingRepo so caller-supplied transactions are honored. (`return entutils.TransactingRepo(ctx, r, func(ctx context.Context, tx *repo) (*ledgerhistorical.Transaction, error) { ... tx.db.LedgerTransaction.Create()... })`)
**Eager edge loading on every query** — Every query returning a Transaction must call WithEntries(func(eq){ eq.WithSubAccount(func(sq){ sq.WithAccount(); sq.WithRoute() }) }). Missing any edge causes hydrateHistoricalTransaction to return an error on SubAccountOrErr/AccountOrErr/RouteOrErr. (`q.WithEntries(func(eq *db.LedgerEntryQuery) { eq.WithSubAccount(func(sq *db.LedgerSubAccountQuery) { sq.WithAccount(); sq.WithRoute() }) })`)
**hydrateHistoricalTransaction for all DB-to-domain conversion** — All DB rows must be converted to domain objects via hydrateHistoricalTransaction, which calls ledgerhistorical.NewTransactionFromData. Never map db.LedgerTransaction fields to domain structs inline. (`items, err := slicesx.MapWithErr(dbItems, func(tx *db.LedgerTransaction) (*ledgerhistorical.Transaction, error) { return hydrateHistoricalTransaction(tx) })`)
**Tri-tuple cursor pagination (bookedAt DESC, createdAt DESC, ID DESC)** — Pagination uses SQL predicate-based cursors over three fields. Forward direction uses ledgerTransactionAfterCursorPredicate; backward uses ledgerTransactionBeforeCursorPredicate and then slices.Reverse on results. NextCursor always points to the last returned item. (`if input.Before != nil { query = query.Where(ledgerTransactionBeforeCursorPredicate(*input.Before)) } ... if input.Before != nil { slices.Reverse(items) }`)
**sumEntriesQuery builder for all aggregation** — SumEntries delegates to sumEntriesQuery{query: query}.Build(tx.db). New filter dimensions must be added to entryPredicates() or subAccountPredicates() in sumentries_query.go, not inline in repo methods. Route filters call b.query.Filters.Route.Normalize() and must propagate errors as ledger.ErrLedgerQueryInvalid. (`q := sumEntriesQuery{query: query}; entryQuery, err := q.Build(tx.db); entryQuery.Aggregate(db.As(db.Sum(ledgerentrydb.FieldAmount), "sum_amount")).Scan(ctx, &rows)`)
**Bulk entry creation after transaction row** — BookTransaction saves the LedgerTransaction row first, then calls CreateBulk for all LedgerEntry rows in a single batch. Never insert entries one-by-one in a loop. (`createdEntries, err = tx.db.LedgerEntry.CreateBulk(createInputs...).Save(ctx)`)
**Compile-time interface compliance assertion** — repo.go declares var _ ledgerhistorical.Repo = (*repo)(nil) and var _ entutils.TxUser[*repo] = (*repo)(nil) at package level. Every new adapter struct must include both guards. (`var _ ledgerhistorical.Repo = (*repo)(nil)
var _ entutils.TxUser[*repo] = (*repo)(nil)`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `repo.go` | Defines the repo struct (holds only *db.Client), NewRepo constructor, Tx/WithTx/Self triad, and both compile-time interface assertions. Entry point for adding new repo methods. | Do not store additional state on repo beyond *db.Client; inject all dependencies via the constructor. Both interface assertions (Repo and TxUser) must stay present. |
| `ledger.go` | Contains BookTransaction, CreateTransactionGroup, GetTransactionGroup, SumEntries, ListTransactions, and all cursor/ordering helpers (ledgerTransactionAfterCursorPredicate, ledgerTransactionBeforeCursorPredicate, listTransactionsOrdering, listTransactionsEntryPredicates). | hydrateHistoricalTransaction panics if SubAccount→Account or SubAccount→Route edges are not loaded; always eager-load them. The Before-direction requires slices.Reverse after fetch and produces NextCursor pointing to the last item. |
| `sumentries_query.go` | Encapsulates sumEntriesQuery with entryPredicates() and subAccountPredicates(). Also exposes SQL() for raw SQL extraction used by the viewgen tool. All filter dimensions for SumEntries live here. | Route filter normalization errors must be wrapped with ledger.ErrLedgerQueryInvalid, not returned as raw errors. SQL() uses dialect.Postgres explicitly and must stay in sync with entryPredicates(). |
| `ledger_test.go` | In-package DB integration tests using NewTestEnv(t) + env.DBSchemaMigrate(t). Covers forward/backward pagination, no-overlap resumption, currency filter, creditMovement filter, and annotation filter. | Tests call time.Sleep between transactions to ensure distinct CreatedAt timestamps for deterministic ordering — do not remove these sleeps. Tests use t.Context(), not context.Background(). |

## Anti-Patterns

- Omitting WithSubAccount/WithAccount/WithRoute eager-load on any query returning hydrated transactions — hydrateHistoricalTransaction returns an error on SubAccountOrErr/AccountOrErr/RouteOrErr
- Mapping db.LedgerTransaction fields to domain structs inline instead of routing through hydrateHistoricalTransaction
- Inserting LedgerEntry rows one-by-one in a loop instead of using CreateBulk
- Adding filter predicates directly inside ListTransactions or SumEntries instead of extending sumEntriesQuery.entryPredicates() / subAccountPredicates()
- Using context.Background() instead of the caller-provided ctx in any repo method, which drops the Ent transaction and OTel spans

## Decisions

- **Tri-tuple cursor (bookedAt, createdAt, ID) instead of single-field offset** — BookedAt is user-visible and can repeat; tie-breaking on createdAt+ID makes pagination stable across concurrent inserts without requiring a serial primary key.
- **sumEntriesQuery is a separate struct with Build() and SQL() methods** — Isolates filter predicate logic from repo methods and allows the viewgen CLI tool (tools/migrate/cmd/viewgen) to extract raw SQL without executing a query against a live database.
- **Tx uses HijackTx + entutils.NewTxDriver instead of db.BeginTx** — Aligns with the project-wide entutils.TransactingRepo pattern so callers can bind the transaction via ctx and participate in caller-supplied transactions without leaking raw *sql.Tx across package boundaries.

## Example: Add a new filter dimension (e.g. CostBasis) to SumEntries without touching repo methods

```
// In sumentries_query.go, extend subAccountPredicates():
if b.query.Filters.CostBasis != nil {
    routePredicates = append(routePredicates, ledgersubaccountroutedb.CostBasis(*b.query.Filters.CostBasis))
}
// No changes needed in ledger.go — SumEntries delegates unchanged:
func (r *repo) SumEntries(ctx context.Context, query ledger.Query) (alpacadecimal.Decimal, error) {
    return entutils.TransactingRepo(ctx, r, func(ctx context.Context, tx *repo) (alpacadecimal.Decimal, error) {
        q := sumEntriesQuery{query: query}
        entryQuery, err := q.Build(tx.db)
        if err != nil { return alpacadecimal.Decimal{}, err }
        var rows []struct{ SumAmount stdsql.NullString `json:"sum_amount,omitempty"` }
        if err := entryQuery.Aggregate(db.As(db.Sum(ledgerentrydb.FieldAmount), "sum_amount")).Scan(ctx, &rows); err != nil {
            return alpacadecimal.Decimal{}, fmt.Errorf("sum entries: %w", err)
        }
        if len(rows) == 0 || !rows[0].SumAmount.Valid { return alpacadecimal.NewFromInt(0), nil }
// ...
```

<!-- archie:ai-end -->
