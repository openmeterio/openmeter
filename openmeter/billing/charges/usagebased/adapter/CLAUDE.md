# adapter

<!-- archie:ai-start -->

> Ent-backed persistence adapter for usage-based charges and their realization runs. Implements the usagebased.Adapter interface (and its sub-interfaces) over the ChargeUsageBased / ChargeUsageBasedRuns / detailed-line / credit-allocation / invoiced-usage / payment Ent tables; it is the only place that translates usagebased domain types to and from DB rows.

## Patterns

**Transaction-aware repo body** — Every mutating or reading adapter method wraps its body in entutils.TransactingRepo / TransactingRepoWithNoValue, using the passed tx *adapter (tx.db) rather than a.db, so the call rebinds to any transaction already carried in ctx. (`return entutils.TransactingRepo(ctx, a, func(ctx context.Context, tx *adapter) (usagebased.Charge, error) { entities, err := tx.db.ChargeUsageBased.Query()... })`)
**Validate inputs before DB work** — Each method first calls Validate() on its domain inputs (charge, runID, input, in) and returns early on error before opening the transaction. (`if err := charge.Validate(); err != nil { return usagebased.ChargeBase{}, err }`)
**Interface conformance asserts per file** — Each file declares a compile-time assertion that *adapter satisfies the relevant sub-interface (ChargeAdapter, RealizationRunAdapter, RealizationRunPaymentAdapter, RealizationRunCreditAllocationAdapter, RealizationRunInvoiceUsageAdapter). (`var _ usagebased.RealizationRunPaymentAdapter = (*adapter)(nil)`)
**Exported Map*FromDB mappers** — DB->domain translation lives in mapper.go as exported MapChargeFromDB / MapChargeBaseFromDB / MapRealizationRun(s|Base)FromDB; sub-model mapping delegates to chargemeta, creditrealization, invoicedusage, payment, totals, stddetailedline package helpers (MapFromDB / FromDB). (`chargeMeta := chargemeta.MapFromDB(entity); ... Totals: totals.FromDB(dbRun)`)
**Timestamp normalization to UTC** — Times are passed through meta.NormalizeTimestamp / NormalizeOptionalTimestamp and .In(time.UTC) before persisting; mappers read DB times back with .UTC(). (`SetInvoiceAt(meta.NormalizeTimestamp(intent.InvoiceAt).In(time.UTC))`)
**Detailed-line soft-delete + upsert by ChildUniqueReferenceID** — UpsertRunDetailedLines soft-deletes existing rows (SetDeletedAt) for the run except ChildUniqueReferenceIDs being kept, sets the run's DetailedLinesPresent=true, then CreateBulk...OnConflict on (namespace, charge_id, run_id, child_unique_reference_id) preserving created_at/id. (`OnConflict(sql.ConflictColumns(FieldNamespace, FieldChargeID, FieldRunID, FieldChildUniqueReferenceID), sql.ConflictWhere(sql.IsNull(FieldDeletedAt)), sql.ResolveWithNewValues())`)
**Expand-gated edge loading** — Realization runs and their child edges (credit allocations, invoiced usage, payment) are only loaded when meta.Expands requests them; detailed lines are fetched in a second pass via FetchDetailedLines when ExpandDetailedLines is set. (`if input.Expands.Has(meta.ExpandRealizations) { query = expandRealizations(query, input.Expands) }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `adapter.go` | Config/New constructor, adapter struct (db, logger, metaAdapter), and the entutils transaction plumbing (Tx via HijackTx, WithTx via NewTxClientFromRawConfig, Self). | Config.Validate requires Client, Logger, and MetaAdapter all non-nil; New returns the usagebased.Adapter interface, not the concrete type. |
| `charge.go` | ChargeAdapter: CreateCharges, UpdateCharge, UpdateSubscriptionItemID, DeleteCharge (soft delete + StatusDeleted), GetByID(s); status is dual-written via SetStatus(metaStatus) and SetStatusDetailed(charge.Status). | GetByIDs intentionally skips a namespace EQ to allow multi-namespace expansion but relies on entutils.InIDOrder for namespace filtering/ordering; charge status must round-trip through ToMetaChargeStatus(). |
| `mapper.go` | All DB->domain mappers; MapRealizationRunFromDB requires CreditAllocations/InvoicedUsage/Payment edges loaded and errors (NotLoadedError) if not. | MapRealizationRunsFromDB forces empty runs to nil and sorts by ServicePeriodTo; calling it without WithRuns loaded returns a 'not loaded' error. |
| `detailedline.go` | FetchDetailedLines (gated by run.DetailedLinesPresent) and UpsertRunDetailedLines (soft-delete + bulk upsert); buildDetailedLineCreate and mapDetailedLineFromDB. | Detailed lines are only marked present (mo.Some) when the run's persisted DetailedLinesPresent flag is true; treating unknown lines as empty can overcharge late events. FetchDetailedLines never repairs the flag. |
| `realizationrun.go` | RealizationRunAdapter: CreateRealizationRun (sets Type and InitialType, DetailedLinesPresent=false) and UpdateRealizationRun (mo.Option-gated field updates, input.Normalized()). | Update only sets fields whose mo.Option IsPresent(); DeletedAt/LineID use SetOrClear semantics. Totals are written via totals.Set. |
| `creditallocation.go / invoicedusage.go / payment.go` | Per-edge create/update adapters for run credit allocations, invoiced (accrued) usage, and payments; each delegates field-setting to its model package (creditrealization.Create, invoicedusage.Create, payment.Create/UpdateInvoiced). | payment.go enforces runID.Namespace == in.Namespace; these write child rows keyed by RunID and must be created within the run's transaction. |
| `detailedline_test.go` | Suite-based integration test (DetailedLineAdapterSuite) over a real Postgres TestDB + migrations exercising upsert replace/soft-delete and the DetailedLinesPresent flag semantics. | Uses testutils.InitPostgresDB + migrate.OMMigrationsConfig; drives behavior through s.adapter (CreateCharges/CreateRealizationRun/UpsertRunDetailedLines), not raw Ent. |

## Anti-Patterns

- Using a.db directly inside a method body instead of the tx *adapter from TransactingRepo — breaks transaction propagation.
- Loading realization runs without their CreditAllocations/InvoicedUsage/Payment edges and passing to MapRealizationRunFromDB — returns 'not loaded' errors.
- Marking detailed lines as present (mo.Some) without honoring the run's DetailedLinesPresent flag — can cause late-event overcharging.
- Persisting timestamps without meta.NormalizeTimestamp + UTC, or reading them back without .UTC().
- Skipping the Validate() guard on inputs before opening a transaction.

## Decisions

- **Status is stored in two columns: a coarse meta status (SetStatus) plus a detailed status (SetStatusDetailed).** — meta-layer queries operate on the normalized ChargeStatus while the usagebased lifecycle needs its richer Status enum; both must stay in sync via ToMetaChargeStatus().
- **DetailedLinesPresent is a persisted boolean on the run rather than inferred from row count.** — An empty detailed-line set is meaningfully different from 'not yet materialized'; the flag prevents late-arriving usage from being treated as already-rated empty.
- **Detailed lines are replaced via soft-delete + conflict-resolving bulk upsert keyed by ChildUniqueReferenceID.** — Preserves stable row IDs/created_at across re-runs (corrections) while allowing rows to be retired without hard deletes.

## Example: A transaction-aware adapter method that validates, queries Ent gated by expands, and maps results back to domain.

```
func (a *adapter) GetByID(ctx context.Context, input usagebased.GetByIDInput) (usagebased.Charge, error) {
	if err := input.Validate(); err != nil {
		return usagebased.Charge{}, err
	}
	return entutils.TransactingRepo(ctx, a, func(ctx context.Context, tx *adapter) (usagebased.Charge, error) {
		query := tx.db.ChargeUsageBased.Query().
			Where(dbchargeusagebased.Namespace(input.ChargeID.Namespace)).
			Where(dbchargeusagebased.ID(input.ChargeID.ID))
		if input.Expands.Has(meta.ExpandRealizations) {
			query = expandRealizations(query, input.Expands)
		}
		entity, err := query.First(ctx)
		if err != nil {
			if db.IsNotFound(err) {
				return usagebased.Charge{}, models.NewGenericNotFoundError(fmt.Errorf("usage based charge [id=%s] not found", input.ChargeID))
// ...
```

<!-- archie:ai-end -->
