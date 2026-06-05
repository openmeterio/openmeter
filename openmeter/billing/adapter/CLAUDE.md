# adapter

<!-- archie:ai-start -->

> Ent-backed persistence layer implementing billing.Adapter and its sub-interfaces (CustomerOverrideAdapter, GatheringInvoiceAdapter, SequenceAdapter, etc.) for invoices, lines, profiles, customer overrides, split-line groups, sequence numbers, schema-level tracking, and validation issues. All mutation flows are transaction-aware and namespace-scoped.

## Patterns

**Self-rebinding tx adapter** — adapter implements Tx/WithTx/Self; every read+write method body wraps in entutils.TransactingRepo or TransactingRepoWithNoValue so it rebinds to the ctx-carried transaction. New methods MUST do the same, never touch a.db directly inside multi-step mutations. (`return entutils.TransactingRepo(ctx, a, func(ctx context.Context, tx *adapter) (T, error) { ... tx.db.BillingInvoice.Query()... })`)
**Interface assertion per file** — Each file asserts the sub-interface it satisfies via a blank var, e.g. `var _ billing.GatheringInvoiceAdapter = (*adapter)(nil)`. Add the matching assertion when implementing a new adapter sub-interface. (`var _ billing.SequenceAdapter = (*adapter)(nil)`)
**Diff-then-upsert for line sets** — Line collections are persisted via entitydiff.DiffByID producing create/update/delete buckets, then upsertWithOptions (upsert.go) drives a single CreateBulk + OnConflict. Deletes are modeled as soft-delete updates (MarkDeleted sets DeletedAt) and applied FIRST to satisfy constraints. (`upsertWithOptions(ctx, a.db, diff.Line, upsertInput[*billing.GatheringLine, *db.BillingInvoiceLineCreate]{Create: ..., UpsertItems: ..., MarkDeleted: ...})`)
**mapXFromDB / mapXToDB conversions** — DB->domain mapping uses free funcs named mapCustomerOverrideFromDB, mapGatheringInvoiceFromDB, etc.; time fields are normalized to UTC via .In(time.UTC) and convert.TimePtrIn; ISO-duration strings parse via ParsePtrOrNil. Follow these names and UTC normalization. (`CreatedAt: invoice.CreatedAt.In(time.UTC), DeletedAt: convert.TimePtrIn(invoice.DeletedAt, time.UTC)`)
**Pessimistic locking & sequencing** — Per-customer serialization uses BillingCustomerLock upsert (DoNothing) + ForUpdate row lock in LockCustomerForUpdate; sequence numbers use ForUpdate + create-on-missing in NextSequenceNumber. New cross-invoice mutations should take the customer lock first. (`tx.db.BillingCustomerLock.Query().Where(...).ForUpdate().First(ctx)`)
**NotFound/Validation domain errors** — Map db.IsNotFound to billing.NotFoundError{Entity, Err} and reject illegal mutations with billing.ValidationError; immutable fields (currency, type, customerID) are re-checked against the existing row before update. (`return billing.NotFoundError{ID: input.CustomerID, Entity: billing.EntityCustomerOverride, Err: billing.ErrCustomerOverrideNotFound}`)
**Bulk batching to dodge param limits** — Bulk inserts chunk by defaultBulkAssignCustomersToProfileBatchSize (derived from 65535 / column count) before CreateBulk+OnConflict, because PostgreSQL caps parameters at 64k. (`for _, chunk := range lo.Chunk(creates, defaultBulkAssignCustomersToProfileBatchSize) { ...CreateBulk(chunk...).OnConflict(...).Exec(ctx) }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `adapter.go` | Config/New constructor and Tx/WithTx/Self via db.HijackTx + entutils.NewTxDriver | Config.Validate requires non-nil Client and Logger; WithTx must rebuild via NewTxClientFromRawConfig, never reuse a.db |
| `upsert.go` | Generic upsertWithOptions[T,CreateBulkType] diff applier | Delete bucket processed first as soft-delete; only runs UpsertItems when buckets non-empty |
| `gatheringlines.go` | Gathering invoice line diff/upsert + DB<->domain mapping | Only InvoiceLineAdapterTypeUsageBased + ParentLineIDIsNil lines are gathering lines; ULID assigned when ID/UBPConfigID empty |
| `invoicelinesplitgroup.go` | Split-line group create/get and SplitLineHierarchy resolution | Hierarchy expansion is a separate query path; keep namespace filters on every edge load |
| `seq.go` | NextSequenceNumber with ForUpdate row lock | Create-on-missing uses OnConflict().DoNothing() then re-reads; returns alpacadecimal.Decimal |
| `lock.go` | UpsertCustomerLock + LockCustomerForUpdate | Still triggers in-band invoice schema migration (shouldInvoicesBeMigrated/migrateCustomerInvoices) — temporary, do not remove blindly |
| `validationissue.go` | persistValidationIssues via sha256 dedupe-hash upsert | Hash covers severity+code+message+component+path; IntrospectValidationIssues is test-only, not on the interface |
| `stdinvoicelinemapper.go` | Standard-invoice line <-> DB mapping (detailed + UBP lines) | Companion to stdinvoicelinediff.go; keep mapper and diff in sync when adding line fields |

## Anti-Patterns

- Calling a.db.* directly for multi-step mutations instead of wrapping in entutils.TransactingRepo(WithNoValue) and using tx.db
- Returning raw db errors for not-found/invalid states instead of billing.NotFoundError / billing.ValidationError
- Persisting line collections with ad-hoc loops instead of entitydiff.DiffByID + upsertWithOptions
- Storing/returning non-UTC times — every timestamp must be .In(time.UTC) on map-in and map-out
- Single CreateBulk of unbounded customer/line slices without lo.Chunk batching (hits the 64k param ceiling)

## Decisions

- **Transaction hijacking (HijackTx/WithTx) instead of passing tx clients** — Lets service-layer orchestration share one Ent transaction across many adapter calls via context, keeping invoice mutations atomic
- **Validation issues deduplicated by content hash with soft-delete** — Re-running calculation re-asserts the same issues; hashing avoids duplicate rows while pruning issues no longer present
- **Gathering invoices reuse the BillingInvoice table with UNSET supplier/customer hacks** — Tables not yet split; UpdateGatheringInvoice clears supplier/customer columns as an interim until a dedicated gathering table exists

## Example: Transaction-aware adapter method returning a domain value

```
func (a *adapter) GetCustomerOverride(ctx context.Context, input billing.GetCustomerOverrideAdapterInput) (*billing.CustomerOverride, error) {
	return entutils.TransactingRepo(ctx, a, func(ctx context.Context, tx *adapter) (*billing.CustomerOverride, error) {
		dbCO, err := tx.db.BillingCustomerOverride.Query().
			Where(billingcustomeroverride.Namespace(input.Customer.Namespace)).
			Where(billingcustomeroverride.CustomerID(input.Customer.ID)).
			WithTaxCode().First(ctx)
		if err != nil {
			if db.IsNotFound(err) {
				return nil, nil
			}
			return nil, err
		}
		return mapCustomerOverrideFromDB(dbCO)
	})
}
```

<!-- archie:ai-end -->
