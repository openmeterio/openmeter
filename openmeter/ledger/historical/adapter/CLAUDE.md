# adapter

<!-- archie:ai-start -->

> Ent/PostgreSQL implementation of ledgerhistorical.Repo: books double-entry transactions (LedgerTransaction + LedgerEntry rows), manages transaction groups, sums entries, and lists transactions with tri-tuple cursor pagination. All queries must eagerly load SubAccount→Account→Route edges or hydrateHistoricalTransaction errors.

## Patterns

**Full TxCreator+TxUser triad on repo** — repo implements Tx (HijackTx + entutils.NewTxDriver), WithTx (db.NewTxClientFromRawConfig), and Self(); every method body wraps with entutils.TransactingRepo so caller-supplied transactions are honored. (`return entutils.TransactingRepo(ctx, r, func(ctx context.Context, tx *repo) (*ledgerhistorical.Transaction, error) { ... tx.db.LedgerTransaction.Create()... })`)
**Eager edge loading on every query** — Every query returning a Transaction calls WithEntries -> WithSubAccount -> WithAccount + WithRoute. Missing any edge causes hydrateHistoricalTransaction to error on SubAccountOrErr/AccountOrErr/RouteOrErr. (`q.WithEntries(func(eq *db.LedgerEntryQuery) { eq.WithSubAccount(func(sq *db.LedgerSubAccountQuery) { sq.WithAccount(); sq.WithRoute() }) })`)
**hydrateHistoricalTransaction for all DB-to-domain conversion** — All DB rows convert to domain via hydrateHistoricalTransaction (calls ledgerhistorical.NewTransactionFromData); never map db.LedgerTransaction fields to domain structs inline. (`items, err := slicesx.MapWithErr(dbItems, func(tx *db.LedgerTransaction) (*ledgerhistorical.Transaction, error) { return hydrateHistoricalTransaction(tx) })`)
**Tri-tuple cursor pagination (bookedAt DESC, createdAt DESC, ID DESC)** — Pagination uses SQL predicate cursors over three fields; forward uses ledgerTransactionAfterCursorPredicate, backward uses ledgerTransactionBeforeCursorPredicate then slices.Reverse. NextCursor always points to the last returned item. (`if input.Before != nil { query = query.Where(ledgerTransactionBeforeCursorPredicate(*input.Before)) } ... if input.Before != nil { slices.Reverse(items) }`)
**sumEntriesQuery builder for all aggregation** — SumEntries delegates to sumEntriesQuery{query}.Build(tx.db). New filter dimensions go in entryPredicates() or subAccountPredicates() in sumentries_query.go, not inline. Route filters call Normalize() and propagate ledger.ErrLedgerQueryInvalid. (`q := sumEntriesQuery{query: query}; entryQuery, err := q.Build(tx.db); entryQuery.Aggregate(db.As(db.Sum(ledgerentrydb.FieldAmount), "sum_amount")).Scan(ctx, &rows)`)
**Bulk entry creation after transaction row** — BookTransaction saves the LedgerTransaction row first, then CreateBulk for all LedgerEntry rows in one batch — never insert entries one-by-one. (`createdEntries, err = tx.db.LedgerEntry.CreateBulk(createInputs...).Save(ctx)`)
**Compile-time interface compliance assertion** — repo.go declares var _ ledgerhistorical.Repo = (*repo)(nil) and var _ entutils.TxUser[*repo] = (*repo)(nil); every new adapter struct must keep both guards. (`var _ ledgerhistorical.Repo = (*repo)(nil)
var _ entutils.TxUser[*repo] = (*repo)(nil)`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `repo.go` | repo struct (only *db.Client), NewRepo constructor, Tx/WithTx/Self triad, and both compile-time interface assertions. | Do not store state on repo beyond *db.Client; inject deps via the constructor. Both interface assertions (Repo and TxUser) must stay. |
| `ledger.go` | BookTransaction, CreateTransactionGroup, GetTransactionGroup, SumEntries, ListTransactions, and cursor/ordering helpers; hydrateHistoricalTransaction. | hydrateHistoricalTransaction errors if SubAccount→Account/Route edges aren't loaded; the Before direction requires slices.Reverse after fetch with NextCursor pointing to the last item. |
| `sumentries_query.go` | sumEntriesQuery with entryPredicates()/subAccountPredicates() and SQL() (used by viewgen). All SumEntries filter dimensions live here. | Route filter normalization errors must wrap ledger.ErrLedgerQueryInvalid; SQL() uses dialect.Postgres explicitly and must stay in sync with entryPredicates(). |
| `ledger_test.go` | In-package DB integration tests via NewTestEnv(t) + DBSchemaMigrate(t); covers forward/backward pagination, resumption, currency/creditMovement/annotation filters. | Tests time.Sleep between transactions for distinct CreatedAt timestamps (deterministic ordering) — do not remove. Use t.Context(), not context.Background(). |

## Anti-Patterns

- Omitting WithSubAccount/WithAccount/WithRoute eager-load on any hydrated-transaction query
- Mapping db.LedgerTransaction fields to domain structs inline instead of hydrateHistoricalTransaction
- Inserting LedgerEntry rows one-by-one in a loop instead of CreateBulk
- Adding filter predicates directly inside ListTransactions/SumEntries instead of extending sumEntriesQuery
- Using context.Background() instead of the caller ctx — drops the Ent transaction and OTel spans

## Decisions

- **Tri-tuple cursor (bookedAt, createdAt, ID) instead of single-field offset** — bookedAt is user-visible and can repeat; tie-breaking on createdAt+ID makes pagination stable across concurrent inserts without a serial primary key.
- **sumEntriesQuery is a separate struct with Build() and SQL()** — Isolates filter predicate logic from repo methods and lets the viewgen CLI extract raw SQL without executing against a live DB.
- **Tx uses HijackTx + entutils.NewTxDriver instead of db.BeginTx** — Aligns with the project-wide entutils.TransactingRepo pattern so callers bind the transaction via ctx without leaking raw *sql.Tx across package boundaries.

## Example: Add a new filter dimension to SumEntries without touching repo methods

```
// In sumentries_query.go, extend subAccountPredicates():
if b.query.Filters.CostBasis != nil {
    routePredicates = append(routePredicates, ledgersubaccountroutedb.CostBasis(*b.query.Filters.CostBasis))
}
// SumEntries in ledger.go delegates unchanged:
func (r *repo) SumEntries(ctx context.Context, query ledger.Query) (alpacadecimal.Decimal, error) {
    return entutils.TransactingRepo(ctx, r, func(ctx context.Context, tx *repo) (alpacadecimal.Decimal, error) {
        q := sumEntriesQuery{query: query}
        entryQuery, err := q.Build(tx.db)
        if err != nil { return alpacadecimal.Decimal{}, err }
        // ... aggregate ...
        return alpacadecimal.NewFromInt(0), nil
    })
}
```

<!-- archie:ai-end -->
