# adapter

<!-- archie:ai-start -->

> Ent/PostgreSQL implementation of breakage.Adapter — persists and queries LedgerBreakageRecord rows (credit expiration plan/release/reopen records) used by the ledger breakage flow. All access is namespace- and customer-scoped, soft-delete aware, and runs through context-propagated Ent transactions.

## Patterns

**TxCreator/TxUser triad** — adapter implements Tx (HijackTx + entutils.NewTxDriver), WithTx (NewTxClientFromRawConfig), and Self so entutils.TransactingRepo can rebind to a ctx-bound transaction or run on Self(). (`func (a *adapter) Tx(ctx context.Context) (context.Context, transaction.Driver, error) { txCtx, rawConfig, eDriver, err := a.db.HijackTx(ctx, &sql.TxOptions{ReadOnly: false}); return txCtx, entutils.NewTxDriver(eDriver, rawConfig), err }`)
**TransactingRepo wraps every method body** — Reads wrap with entutils.TransactingRepo and writes with TransactingRepoWithNoValue; never touch a.db directly outside the wrapper, which captures tx and degrades to Self() when no tx is in ctx. (`return entutils.TransactingRepoWithNoValue(ctx, a, func(ctx context.Context, tx *adapter) error { /* tx.db... */ })`)
**Config.Validate() before New** — New(Config) calls config.Validate() (Client must be non-nil) and returns (breakage.Adapter, error); never construct &adapter{} directly. (`func New(config Config) (breakage.Adapter, error) { if err := config.Validate(); err != nil { return nil, err }; return &adapter{db: config.Client}, nil }`)
**Input validation before any query** — Each method validates input (input.Validate(), CustomerID.Validate(), Currency.Validate(), AsOf non-zero) before opening a transaction, and short-circuits to (nil, nil) when there is nothing to query (e.g. no source filters in ListReleaseRecords). (`if len(input.SourceEntryID) == 0 && len(input.SourceTransactionGroupID) == 0 { return nil, nil }`)
**ForUpdate row locking on read-modify queries** — ListReleaseRecords and ListCandidateRecords append .ForUpdate() so concurrent breakage processing serializes on the selected rows; ListExpiredRecords omits it (read-only scan). (`tx.db.LedgerBreakageRecord.Query().Where(...).Order(...).ForUpdate().All(ctx)`)
**Predicate-only filtering, deterministic ordering, mapRecordFromDB conversion** — Queries always filter NamespaceEQ + CustomerIDEQ + DeletedAtIsNil, apply explicit multi-key Order(...) for deterministic results, and convert every db row to a domain breakage.Record via mapRecordFromDB. (`dbledgerbreakagerecord.DeletedAtIsNil(); out = append(out, mapRecordFromDB(row))`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `adapter.go` | Config/Validate/New constructor and the Tx/WithTx/Self transaction triad. | Tx hijacks a writable transaction (ReadOnly: false); do not bypass entutils.NewTxDriver or store a raw *entdb.Tx as a field. |
| `record.go` | All query/write methods (CreateRecords, ListReleaseRecords, ListExpiredRecords, ListCandidateRecords) plus mapRecordFromDB / mapRecords converters. | ListReleaseRecords also fetches matching BreakageKindReopen rows by ReleaseID and appends them; ListExpiredRecords filters ExpiresAtLTE(AsOf) while ListCandidateRecords filters ExpiresAtGT(AsOf) — the comparison direction is load-bearing. CreateRecords uses CreateBulk with SetNillable* for optional source/plan/release fields. |

## Anti-Patterns

- Calling a.db.* directly in a method body without wrapping in entutils.TransactingRepo / TransactingRepoWithNoValue (drops the ctx-bound transaction, risking partial writes)
- Storing a raw *entdb.Tx as a struct field instead of rebinding via WithTx
- Omitting NamespaceEQ + CustomerIDEQ + DeletedAtIsNil predicates (breaks multi-tenancy and soft-delete invariants)
- Removing .ForUpdate() from ListReleaseRecords / ListCandidateRecords (loses row-lock serialization for concurrent breakage processing)
- Adding domain/business logic here instead of in the breakage service — this layer is pure persistence

## Decisions

- **Read methods lock candidate/release rows with ForUpdate, but the expired-records scan is left unlocked.** — Release and candidate listings feed read-modify-write breakage processing that must serialize per customer; the expired scan is a non-mutating snapshot and does not need a lock.
- **Optional source/plan/release identifiers are written via SetNillable* and ordering is always multi-key.** — Breakage records carry optional provenance pointers; SetNillable* keeps NULLs intact, and deterministic ordering (priority/expiry/id) makes downstream allocation reproducible.

## Example: Adapter query method wrapped in TransactingRepo with namespace/customer/soft-delete predicates

```
func (a *adapter) ListExpiredRecords(ctx context.Context, input breakage.ListExpiredRecordsInput) ([]breakage.Record, error) {
    if err := input.CustomerID.Validate(); err != nil { return nil, fmt.Errorf("customer id: %w", err) }
    if input.AsOf.IsZero() { return nil, fmt.Errorf("as of is required") }
    return entutils.TransactingRepo(ctx, a, func(ctx context.Context, tx *adapter) ([]breakage.Record, error) {
        rows, err := tx.db.LedgerBreakageRecord.Query().
            Where(
                dbledgerbreakagerecord.NamespaceEQ(input.CustomerID.Namespace),
                dbledgerbreakagerecord.CustomerIDEQ(input.CustomerID.ID),
                dbledgerbreakagerecord.DeletedAtIsNil(),
                dbledgerbreakagerecord.ExpiresAtLTE(input.AsOf),
            ).
            Order(dbledgerbreakagerecord.ByExpiresAt(sql.OrderDesc()), dbledgerbreakagerecord.ByID(sql.OrderDesc())).
            All(ctx)
        if err != nil { return nil, fmt.Errorf("list expired breakage records: %w", err) }
        return mapRecords(rows), nil
// ...
```

<!-- archie:ai-end -->
