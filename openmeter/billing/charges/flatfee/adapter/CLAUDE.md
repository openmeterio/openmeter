# adapter

<!-- archie:ai-start -->

> Ent-backed persistence adapter for flat-fee charges and their realization runs. It implements the segmented flatfee.Adapter interfaces (ChargeAdapter, ChargeRunAdapter, ChargeDetailedLineAdapter, ChargeCreditAllocationAdapter, ChargePaymentAdapter, ChargeInvoicedUsageAdapter) over the ChargeFlatFee* Ent tables, and is the only place SQL-shaped operations for flat-fee charges live.

## Patterns

**Transaction-aware via entutils.TransactingRepo** — Every write/read method body is wrapped in entutils.TransactingRepo (value-returning) or entutils.TransactingRepoWithNoValue (no value), rebinding to the tx carried in ctx. The adapter implements Tx/WithTx/Self to participate in entutils transactions. (`return entutils.TransactingRepo(ctx, a, func(ctx context.Context, tx *adapter) (flatfee.ChargeBase, error) { ... tx.db.ChargeFlatFee.UpdateOneID(...) ... })`)
**Validate inputs before persisting** — Methods call input/charge Validate() (e.g. charge.ManagedModel.Validate(), in.Validate(), runID.Validate()) before touching the DB, returning early on error. (`if err := in.Validate(); err != nil { return nil, err }`)
**Namespace-scoped queries always** — Every Ent query/update filters by namespace (dbchargeflatfee.NamespaceEQ / .Namespace) in addition to ID — multi-tenant isolation is mandatory. (`tx.db.ChargeFlatFee.UpdateOneID(charge.ID).Where(dbchargeflatfee.NamespaceEQ(charge.Namespace))`)
**Map* functions for DB<->domain conversion** — All conversion lives in mapper.go as MapChargeFlatFeeFromDB / MapChargeBaseFromDB / mapRealizationRunFromDB. Edges are read via *OrErr() and a NotLoadedError is turned into an explicit 'not loaded' error rather than silently zero-valued. (`dbCreditsAllocated, err := dbRun.Edges.CreditAllocationsOrErr(); if _, ok := lo.ErrorsAs[*entdb.NotLoadedError](err); ok { return ..., fmt.Errorf("credits allocated not loaded ...") }`)
**Expands gate eager-loading** — Realization edges are loaded only when meta.Expands.Has(meta.ExpandRealizations) via expandRealizations(query); detailed lines are fetched separately when ExpandDetailedLines is set. Mapping respects the same flag so unloaded edges are never dereferenced. (`if input.Expands.Has(meta.ExpandRealizations) { query = expandRealizations(query) }`)
**Soft-delete + upsert for detailed lines** — UpsertDetailedLines soft-deletes (SetDeletedAt) rows whose ChildUniqueReferenceID is not in the new set, then CreateBulk with OnConflict on (namespace, run_id, child_unique_reference_id) WHERE deleted_at IS NULL, ignoring created_at/id. (`OnConflict(sql.ConflictColumns(FieldNamespace, FieldRunID, FieldChildUniqueReferenceID), sql.ConflictWhere(sql.IsNull(FieldDeletedAt)), sql.ResolveWithNewValues(), ...)`)
**Register/Deregister charge IDs with the meta adapter** — CreateCharges calls tx.metaAdapter.RegisterCharges and DeleteCharge calls tx.metaAdapter.DeleteRegisteredCharge inside the same tx, keeping the shared charge-meta registry in sync with flat-fee rows. (`err = tx.metaAdapter.RegisterCharges(ctx, meta.RegisterChargesInput{Namespace: in.Namespace, Type: meta.ChargeTypeFlatFee, Charges: ...})`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `adapter.go` | Config{MetaAdapter, Client, Logger} + New() returning flatfee.Adapter; implements Tx/WithTx/Self for entutils transactions. | Config.Validate() requires all three fields non-nil. WithTx rebuilds the adapter from raw tx config — do not bypass it by holding the original db client across a tx. |
| `charge.go` | ChargeAdapter: CreateCharges (bulk), UpdateCharge, UpdateSubscriptionItemID, DeleteCharge, GetByID(s); plus expandRealizations and buildCreateFlatFeeCharge. | Status must be converted via charge.Status.ToMetaChargeStatus() and persisted through chargemeta.Create/Update — do not write status fields directly. InvoiceAt is normalized via meta.NormalizeTimestamp(...).In(time.UTC). |
| `mapper.go` | DB->domain mappers and proRatingConfigFromDB/ToDB enum conversion; sortDetailedLines via stddetailedline.Compare. | Edge OrErr() NotLoadedError handling is load-bearing; CurrentRealizationRunID drives CurrentRun vs PriorRuns split — a current run id with no loaded run is an error. |
| `realizationrun.go` | ChargeRunAdapter: CreateCurrentRun (locks charge ForUpdate, rejects if a current run exists), UpdateRealizationRun (mo.Option-gated field sets), DetachCurrentRun. | CreateCurrentRun uses ForUpdate() to serialize; it errors with 'already has current run'. UpdateRealizationRun applies only Present() fields and calls input.Normalized() first. |
| `detailedline.go` | ChargeDetailedLineAdapter: FetchCurrentRunDetailedLines and UpsertDetailedLines (soft-delete + bulk upsert). | FetchCurrentRunDetailedLines requires charge.Realizations.CurrentRun != nil. UpsertDetailedLines clones each line, clears DeletedAt, and only deletes lines NOT in childRefsToKeep — an empty new set deletes all. |
| `credits.go` | ChargeCreditAllocationAdapter: CreateCreditAllocations bulk-inserts ChargeFlatFeeRunCreditAllocations after verifying the run exists. | Uses creditrealization.Create(create, namespace, idx, input) per row; run existence is checked with Only(ctx) before insert. |
| `payment.go` | ChargePaymentAdapter: CreatePayment / UpdatePayment over ChargeFlatFeeRunPayment using payment.CreateInvoiced/UpdateInvoiced builders. | Maps back via payment.MapInvoicedFromDB; update is namespace-scoped by UpdateOneID + Where(Namespace). |
| `usage.go` | ChargeInvoicedUsageAdapter: CreateInvoicedUsage sets the run's LineID/InvoiceID then inserts ChargeFlatFeeRunInvoicedUsage. | Two writes in one tx (run refs update + invoiced-usage create) must stay atomic; uses invoicedusage.Create builder helper. |

## Anti-Patterns

- Calling tx.db / a.db Ent builders outside an entutils.TransactingRepo(WithNoValue) wrapper, breaking transaction propagation carried in ctx.
- Querying or updating ChargeFlatFee* tables without a namespace predicate (cross-tenant leakage).
- Dereferencing realization/credit/usage/payment edges without checking *OrErr() NotLoadedError, or mapping realizations when ExpandRealizations was not requested.
- Writing StatusDetailed or meta fields directly instead of going through chargemeta.Create/Update and ToMetaChargeStatus().
- Creating a second current run without honoring the ForUpdate lock and CurrentRealizationRunID nil check in CreateCurrentRun.

## Decisions

- **Adapter split into many small interface implementations (Charge/Run/DetailedLine/CreditAllocation/Payment/InvoicedUsage) all backed by one *adapter struct.** — Lets the service layer depend on narrow capabilities while keeping a single transactional Ent client and meta-registry wiring.
- **Detailed lines are upserted by ChildUniqueReferenceID with soft-delete rather than delete-and-reinsert.** — Preserves row identity/created_at and external invoicing references across re-realizations while idempotently replacing the line set.
- **Charge IDs are reserved/released in the shared meta adapter inside the same transaction as the flat-fee row.** — Keeps the polymorphic charge registry consistent with the concrete flat-fee table so meta queries resolve correctly.

## Example: Standard transaction-aware, namespace-scoped, validate-first adapter method returning a mapped domain value.

```
func (a *adapter) GetByID(ctx context.Context, input flatfee.GetByIDInput) (flatfee.Charge, error) {
	if err := input.Validate(); err != nil {
		return flatfee.Charge{}, err
	}
	return entutils.TransactingRepo(ctx, a, func(ctx context.Context, tx *adapter) (flatfee.Charge, error) {
		query := tx.db.ChargeFlatFee.Query().
			Where(dbchargeflatfee.Namespace(input.ChargeID.Namespace)).
			Where(dbchargeflatfee.ID(input.ChargeID.ID))
		if input.Expands.Has(meta.ExpandRealizations) {
			query = expandRealizations(query)
		}
		entity, err := query.First(ctx)
		if err != nil {
			if db.IsNotFound(err) {
				return flatfee.Charge{}, models.NewGenericNotFoundError(fmt.Errorf("flat fee charge [id=%s] not found", input.ChargeID))
// ...
```

<!-- archie:ai-end -->
