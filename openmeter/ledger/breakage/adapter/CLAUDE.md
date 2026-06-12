# adapter

<!-- archie:ai-start -->

> Ent-backed persistence adapter for the ledger breakage sub-domain: it stores and queries durable `ledger_breakage_record` rows (plan/release/reopen/expired) that back the breakage service's open-amount accounting. It implements the `breakage.Adapter` interface declared in the parent package's types.go.

## Patterns

**Constructor returns the parent interface, not the concrete type** — New(Config) returns breakage.Adapter; Config carries only *entdb.Client and Config.Validate() rejects a nil client before constructing the unexported adapter struct. (`func New(config Config) (breakage.Adapter, error) { if err := config.Validate(); err != nil { return nil, err }; return &adapter{db: config.Client}, nil }`)
**Transaction-aware repo wrapping on every method** — Each adapter method wraps its body in entutils.TransactingRepo (read/return) or TransactingRepoWithNoValue (CreateRecords), so the Ent client rebinds to any tx already carried in ctx. Tx/WithTx/Self implement the entutils transacting-repo contract via HijackTx and NewTxClientFromRawConfig. (`return entutils.TransactingRepo(ctx, a, func(ctx context.Context, tx *adapter) ([]breakage.Record, error) { ... tx.db.LedgerBreakageRecord.Query()... })`)
**Validate inputs before touching the DB** — Methods call input.Validate()/input.CustomerID.Validate()/input.Currency.Validate() and guard required fields (AsOf.IsZero(), empty source-id slices) up front, returning early (nil,nil for no-op selectors) before opening a transaction. (`if input.AsOf.IsZero() { return nil, fmt.Errorf("as of is required") }`)
**Always filter DeletedAtIsNil and Namespace + CustomerID** — Every query scopes by NamespaceEQ + CustomerIDEQ + DeletedAtIsNil; breakage rows are soft-deleted and multi-tenant, so omitting any of these leaks cross-tenant or stale rows. (`dbledgerbreakagerecord.NamespaceEQ(...), dbledgerbreakagerecord.CustomerIDEQ(...), dbledgerbreakagerecord.DeletedAtIsNil()`)
**ForUpdate() locking on contended selectors** — ListCandidateRecords, ListReleaseRecords (and its reopen follow-up query) call .ForUpdate() so concurrent collectors/corrections cannot double-release the same open amount; ListExpiredRecords intentionally does NOT lock. (`tx.db.LedgerBreakageRecord.Query().Where(...).Order(...).ForUpdate().All(ctx)`)
**Centralized DB->domain mapping via mapRecordFromDB** — All rows are converted through mapRecordFromDB (and the mapRecords slice helper); new persisted columns must be added there AND in the CreateRecords builder chain or fields silently drop. (`out = append(out, mapRecordFromDB(row))`)
**Bulk insert with SetNillable for optional FKs** — CreateRecords builds *LedgerBreakageRecordCreate per record and CreateBulk(...).Exec; optional source/plan/release pointers use SetNillable* setters, and empty input short-circuits to nil before Exec. (`create := tx.db.LedgerBreakageRecord.Create().SetID(...).SetNillableSourceEntryID(record.SourceEntryID)...`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `adapter.go` | Defines Config (just *entdb.Client), New constructor, the unexported adapter struct, and the entutils transacting-repo plumbing (Tx via HijackTx, WithTx via NewTxClientFromRawConfig, Self). | Tx hijacks a read-write tx (ReadOnly:false); WithTx rebuilds the client from the raw tx config — don't bypass these or TransactingRepo can't rebind to the caller's transaction. |
| `record.go` | All breakage.Adapter methods: CreateRecords (bulk insert), ListReleaseRecords (release+reopen rows, locked), ListExpiredRecords (expiry <= AsOf, unlocked), ListCandidateRecords (Plan/Release/Reopen kinds, expiry > AsOf, ordered by CreditPriority). Plus mapRecordFromDB/mapRecords. | Ordering matters: candidates order by CreditPriority asc then ExpiresAt asc; expired order DESC. ListReleaseRecords needs at least one of SourceEntryID/SourceTransactionGroupID or it returns (nil,nil). Reopen rows are fetched in a second query keyed by ReleaseIDIn(releaseIDs). |

## Anti-Patterns

- Returning the concrete *adapter from New instead of breakage.Adapter, or skipping Config.Validate().
- Running Ent queries directly on a.db instead of through entutils.TransactingRepo(WithNoValue) — breaks transaction propagation from ctx.
- Omitting DeletedAtIsNil / Namespace / CustomerID predicates, leaking soft-deleted or cross-tenant rows.
- Dropping .ForUpdate() on candidate/release queries, allowing concurrent collectors to release the same open amount twice.
- Adding a Record field without updating both the CreateRecords builder chain and mapRecordFromDB, silently losing the column.

## Decisions

- **Adapter persists only durable record rows; the ledger entries themselves live elsewhere.** — Per the parent service.go comment, breakage records are an accounting projection layered over the real ledger entries, so this adapter owns row CRUD/locking, not entry posting.
- **Locking is selective — write/contended selectors use ForUpdate, expiry read does not.** — Candidate and release computations must be serialized to avoid double-release, but expired-row reads are netted by the caller and don't need row locks.

## Example: A new Adapter list method scoped to a customer, transaction-aware and tenant-safe

```
func (a *adapter) ListExpiredRecords(ctx context.Context, input breakage.ListExpiredRecordsInput) ([]breakage.Record, error) {
	if err := input.CustomerID.Validate(); err != nil {
		return nil, fmt.Errorf("customer id: %w", err)
	}
	if input.AsOf.IsZero() {
		return nil, fmt.Errorf("as of is required")
	}
	return entutils.TransactingRepo(ctx, a, func(ctx context.Context, tx *adapter) ([]breakage.Record, error) {
		rows, err := tx.db.LedgerBreakageRecord.Query().
			Where(
				dbledgerbreakagerecord.NamespaceEQ(input.CustomerID.Namespace),
				dbledgerbreakagerecord.CustomerIDEQ(input.CustomerID.ID),
				dbledgerbreakagerecord.DeletedAtIsNil(),
				dbledgerbreakagerecord.ExpiresAtLTE(input.AsOf),
			).All(ctx)
// ...
```

<!-- archie:ai-end -->
