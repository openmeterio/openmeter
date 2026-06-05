# adapter

<!-- archie:ai-start -->

> Ent/Postgres repository for the ledger's historical (append-only) double-entry model: books transactions and entries, reconstructs domain Transaction/TransactionGroup aggregates from rows, and runs sum/list queries over ledger entries. Implements ledgerhistorical.Repo and is the only place that reads/writes the ledger_transaction(_group), ledger_entry, ledger_sub_account(_route) tables for this domain.

## Patterns

**Implement ledgerhistorical.Repo on a private repo struct** — All persistence methods hang off `type repo struct { db *db.Client }` with a compile-time assertion `var _ ledgerhistorical.Repo = (*repo)(nil)`. Construct only via NewRepo(dbClient) which returns the interface, not the struct. (`func NewRepo(dbClient *db.Client) ledgerhistorical.Repo { return &repo{db: dbClient} }`)
**Wrap every method body in entutils.TransactingRepo** — BookTransaction, CreateTransactionGroup, GetTransactionGroup, SumEntries, ListTransactions all execute inside entutils.TransactingRepo(ctx, r, func(ctx, tx *repo)...) so they rebind to any tx already in ctx. Use tx.db (not r.db) inside the closure. (`return entutils.TransactingRepo(ctx, r, func(ctx context.Context, tx *repo) (*ledgerhistorical.Transaction, error) { entity, err := tx.db.LedgerTransaction.Create()... })`)
**Reconstruct aggregates only via NewTransactionFromData / NewTransactionGroupFromData** — DB rows are never exposed as domain objects directly. Map ent edges into ledgerhistorical.EntryData/TransactionData and call the domain constructors (hydrateHistoricalTransaction does this for reads). Never hand-build a Transaction struct. (`ledgerhistorical.NewTransactionFromData(ledgerhistorical.TransactionData{ID: tx.ID, ...}, entryData)`)
**Eager-load the full edge chain and assert presence with *OrErr** — Reads must WithEntries -> WithSubAccount -> WithAccount + WithRoute, ordered ByCreatedAt then ByID. Hydration accesses edges through SubAccountOrErr/AccountOrErr/RouteOrErr and returns a wrapped error if any edge is missing rather than nil-panicking. (`subAccount, err := entry.Edges.SubAccountOrErr(); if err != nil { return EntryData{}, fmt.Errorf("entry %s missing sub-account edge: %w", entry.ID, err) }`)
**Keyset (cursor) pagination via raw sql.Selector predicates** — ListTransactions orders by (BookedAt, CreatedAt, ID) and builds after/before cursor predicates as predicate.LedgerTransaction closures over *sql.Selector. Fetch Limit+1 to detect hasMore; Before pages are reversed with slices.Reverse and NextCursor is taken from the last returned item to avoid overlap on resume. (`query = query.Limit(input.Limit + 1); hasMore := len(dbItems) > input.Limit; if hasMore { dbItems = dbItems[:input.Limit] }`)
**Build sum/filter predicates through sumEntriesQuery, normalizing Route first** — SumEntries delegates to sumEntriesQuery{query}.Build/entryPredicates/subAccountPredicates. Route filters call Filters.Route.Normalize() (returning ErrLedgerQueryInvalid on failure) and translate each optional field to Eq vs IsNil predicates (TaxCode, Features, CostBasis, TaxBehavior). Features arrays use pq.StringArray. (`normalizedRoute, err := b.query.Filters.Route.Normalize(); if err != nil { return nil, ledger.ErrLedgerQueryInvalid.WithAttrs(...) }`)
**Sum aggregation via NullString scan, not Decimal directly** — SumEntries aggregates db.Sum(FieldAmount) AS sum_amount into a stdsql.NullString, returns alpacadecimal.NewFromInt(0) when no/invalid rows, otherwise alpacadecimal.NewFromString. Do not Scan a sum straight into a Decimal. (`var rows []struct{ SumAmount stdsql.NullString `json:"sum_amount,omitempty"` }; ...Aggregate(db.As(db.Sum(ledgerentrydb.FieldAmount), "sum_amount")).Scan(ctx, &rows)`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `repo.go` | Defines the repo struct, NewRepo constructor, and the entutils.TxUser plumbing (Tx via HijackTx, WithTx via NewTxClientFromRawConfig, Self). | Tx uses TxOptions{ReadOnly: false}; WithTx must rebuild db from the raw tx config or transactions silently won't propagate. Keep the `var _ entutils.TxUser[*repo]` and `var _ ledgerhistorical.Repo` assertions intact. |
| `ledger.go` | All Repo methods (BookTransaction, CreateTransactionGroup, GetTransactionGroup, SumEntries, ListTransactions) plus hydrateHistoricalTransaction and the cursor predicate helpers. | BookTransaction precomputes route/account-type maps keyed by SubAccountID because CreateBulk-returned entries lack edges; if you add entry fields, populate them in BOTH the post-insert mapping and hydrateHistoricalTransaction. nil input must return ledger.ErrTransactionInputRequired. |
| `sumentries_query.go` | sumEntriesQuery builder translating ledger.Query.Filters into LedgerEntry predicates (Build for ent query, SQL for raw shape). | BookedAtPeriod uses GTE/LT (half-open), AsOf uses BookedAtLTE, and the After cursor predicate uses LTE on ID (inclusive tail) — different from ListTransactions' strict-LT cursor. Optional Route fields must emit *IsNil predicates when the option is present-but-empty. |
| `ledger_test.go` | Integration tests (NewTestEnv/DBSchemaMigrate against real Postgres) covering booking, group hydration, tax-behavior preservation, and forward/Before cursor pagination without overlap. | Tests use transactionstestutils.AnyEntryInput and env.createSubAccount/createSubAccountOfType helpers; sleeps separate transactions so CreatedAt ordering is deterministic. Requires POSTGRES_HOST set or the suite skips. |

## Anti-Patterns

- Using r.db inside a method body instead of the tx-bound tx.db from the TransactingRepo closure — bypasses the active transaction.
- Returning ent rows or constructing ledgerhistorical.Transaction by hand instead of going through NewTransactionFromData / NewTransactionGroupFromData.
- Accessing entry/sub-account/account/route edges without eager-loading them or without the *OrErr presence checks, causing nil-pointer panics.
- Adding a new persisted entry/route field without updating both BookTransaction's per-SubAccountID maps and hydrateHistoricalTransaction.
- Skipping Route.Normalize() (and its ErrLedgerQueryInvalid path) or omitting *IsNil predicates when filtering optional route fields, silently widening result sets.

## Decisions

- **Reconstruct domain aggregates from a separate *Data DTO layer rather than mapping ent rows directly.** — Keeps the historical/append-only invariants (entry balancing, route identity) enforced by the domain constructors in one place, independent of how rows were loaded or just-inserted.
- **BookTransaction caches AccountType/Route/RouteKey by SubAccountID before CreateBulk.** — CreateBulk returns entries without their sub-account/account/route edges, so the view must be assembled from inputs to avoid an extra eager-load round trip.
- **Cursor pagination over (BookedAt, CreatedAt, ID) with Limit+1 and Before-page reversal.** — Gives stable keyset ordering across booking-time ties and lets NextCursor from a Before page resume forward paging without re-emitting the page tail (verified by TestRepo_ListTransactions_BeforeNextCursorResumesWithoutOverlap).

## Example: Transaction-aware read that eager-loads the edge chain and rebuilds the domain aggregate

```
func (r *repo) GetTransactionGroup(ctx context.Context, id models.NamespacedID) (*ledgerhistorical.TransactionGroup, error) {
  return entutils.TransactingRepo(ctx, r, func(ctx context.Context, tx *repo) (*ledgerhistorical.TransactionGroup, error) {
    entity, err := tx.db.LedgerTransactionGroup.Query().
      Where(ledgertransactiongroupdb.Namespace(id.Namespace), ledgertransactiongroupdb.ID(id.ID)).
      WithTransactions(func(q *db.LedgerTransactionQuery) {
        q.Order(ledgertransactiondb.ByCreatedAt(), ledgertransactiondb.ByID())
        q.WithEntries(func(eq *db.LedgerEntryQuery) {
          eq.Order(ledgerentrydb.ByCreatedAt(), ledgerentrydb.ByID())
          eq.WithSubAccount(func(sq *db.LedgerSubAccountQuery) { sq.WithAccount(); sq.WithRoute() })
        })
      }).Only(ctx)
    if err != nil {
      return nil, fmt.Errorf("failed to query transaction group: %w", err)
    }
    transactions, err := slicesx.MapWithErr(entity.Edges.Transactions, hydrateHistoricalTransaction)
// ...
```

<!-- archie:ai-end -->
