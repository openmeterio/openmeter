# adapter

<!-- archie:ai-start -->

> Ent/PostgreSQL implementation of billing.Adapter — the single persistence boundary for all billing domain entities (profiles, customer overrides, invoices, invoice lines, split line groups, sequences, locks, schema migration). Primary constraint: every method must compose with caller-supplied ctx transactions.

## Patterns

**TransactingRepo on every method body** — Wrap write/read-consistent methods in entutils.TransactingRepo or TransactingRepoWithNoValue so a.db rebinds to the ctx-carried tx via WithTx; nesting is safe. (`return entutils.TransactingRepo(ctx, a, func(ctx context.Context, tx *adapter) (*billing.CustomerOverride, error) { row, err := tx.db.BillingCustomerOverride.Create()....Save(ctx); return mapCustomerOverrideFromDB(row) })`)
**Compile-time interface assertions at file top** — Each file implementing a sub-interface declares var _ billing.XxxAdapter = (*adapter)(nil) to catch missing methods at compile time. (`var _ billing.GatheringInvoiceAdapter = (*adapter)(nil)`)
**workflowConfigWithTaxCode eager-load option** — Every BillingProfile/WorkflowConfig query that will be mapped must pass WithWorkflowConfig(workflowConfigWithTaxCode); omitting it yields silent nil TaxCode in BackfillTaxConfig. (`tx.db.BillingProfile.Query().Where(...).WithWorkflowConfig(workflowConfigWithTaxCode).First(ctx)`)
**Manually attach edges after Save()** — Ent never populates edges on mutation results; after Save with a TaxCodeID, fetch and assign the edge before returning. (`if saved.TaxCodeID != nil { tc, _ := a.db.TaxCode.Get(ctx, *saved.TaxCodeID); saved.Edges.TaxCode = tc }`)
**entitydiff-based upsert for line hierarchies** — Update invoice lines via diffInvoiceLines/diffGatheringInvoiceLines (Create/Update/Delete sets) then bulk CREATE ON CONFLICT; never item-by-item. (`lineDiffs, _ := diffInvoiceLines(input.Lines); tx.upsertFeeLineConfig(ctx, lineDiffs.DetailedLine)`)
**DBState snapshot after read** — After mapping a line from DB call line.SaveDBSnapshot(); the diff engine uses DBState, so omitting makes every field look changed on next write. (`if err := line.SaveDBSnapshot(); err != nil { return nil, fmt.Errorf("saving DB snapshot [id=%s]: %w", line.GetID(), err) }`)
**Soft-delete with DeletedAt** — Billing entities soft-delete via SetDeletedAt(clock.Now()); queries must add Where(...DeletedAtIsNil()) unless IncludeDeleted. Sequences and lock rows are never soft-deleted. (`tx.db.BillingInvoice.Update().Where(billinginvoice.ID(id)).SetDeletedAt(clock.Now()).Save(ctx)`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `adapter.go` | Config, adapter struct (db *entdb.Client, logger), New, and the Tx/WithTx/Self triad required by entutils.TransactingRepo. | Tx() uses HijackTx — do not replace with db.BeginTx as it breaks the transaction driver contract. |
| `stdinvoicelines.go` | Implements UpsertInvoiceLines — multi-table upsert across BillingInvoiceLine, flat-fee/usage-based configs, discounts, schema-level-2 detailed lines. | SchemaLevel gates which detail table is written (schema 1 FlatFeeLine vs schema 2 DetailedLine); mixing corrupts reads. |
| `stdinvoicelinediff.go` | Pure diffing logic (no DB) for line hierarchies; produces invoiceLineDiff with separate entitydiff.Diff sets. | Equality uses StandardLineBase.Equal / UsageBased.Equal — new fields require updating Equal or diffs are missed. |
| `gatheringlines.go` | CRUD/diff/upsert for gathering lines (usage-based) plus expandSplitLineHierarchy for progressive billing. | Gathering lines hard-delete via HardDeleteGatheringInvoiceLines — must atomically delete associated UsageBasedLineConfig rows. |
| `schemamigration.go` | Lazy per-customer invoice schema migration (level 1→2) via raw SQL proc om_func_migrate_customer_invoices_to_schema_level_2. | Never call migrateCustomerInvoices outside a LockCustomerForUpdate context. |
| `profile.go` | Profile/WorkflowConfig CRUD; manually fetches TaxCode edge after Save. | workflowConfigWithTaxCode must be used on every mapped BillingProfile query. |
| `seq.go` | Sequence allocation with FOR UPDATE pessimistic locking, upsert-then-reselect for first-use race. | NextSequenceNumber must run inside a transaction (uses ForUpdate). |
| `stdinvoicelinemapper.go` | Pure DB→domain mapping; backfillTaxConfigReferences handles legacy stripe-code/TaxCodeID precedence. | mapStandardInvoiceDetailedLineFromDB is schema-1; mapStandardInvoiceDetailedLineV2FromDB is schema-2 — caller selects via schemaLevelByInvoiceID. |

## Anti-Patterns

- Using a.db directly inside a helper without entutils.TransactingRepo — ignores the ctx-carried tx and produces partial writes.
- Querying BillingProfile/WorkflowConfig without WithWorkflowConfig(workflowConfigWithTaxCode) — silent nil TaxCode.
- Returning a Save() result without manually attaching the TaxCode edge — mappers dereference nil.
- Bypassing the entitydiff pipeline to write lines one-by-one — misses AffectedLineIDs/UpdatedAt propagation.
- Calling migrateCustomerInvoices outside a LockCustomerForUpdate context.

## Decisions

- **TransactingRepo at each adapter method boundary, not once in the service layer.** — Methods compose (GetCustomerOverride inside UpdateCustomerOverride); TransactingRepo safely nests by reusing the ctx tx, keeping service free of tx management.
- **Entity-diff + bulk-upsert pipeline for invoice lines instead of load-compare-update.** — Line sets can be large; diffing then bulk CREATE ON CONFLICT minimises round trips and propagates UpdatedAt only to changed lines.
- **Schema migration is lazy under LockCustomerForUpdate, not a global offline job.** — A global migration needs a long exclusive lock; per-customer lazy migration amortises cost without blocking unrelated customers.

## Example: Adding a new adapter method that reads/writes inside a transaction

```
func (a *adapter) CreateFoo(ctx context.Context, input billing.CreateFooInput) (*billing.Foo, error) {
    if err := input.Validate(); err != nil { return nil, billing.ValidationError{Err: err} }
    return entutils.TransactingRepo(ctx, a, func(ctx context.Context, tx *adapter) (*billing.Foo, error) {
        row, err := tx.db.BillingFoo.Create().SetNamespace(input.Namespace).SetName(input.Name).Save(ctx)
        if err != nil { return nil, err }
        return mapFooFromDB(row), nil
    })
}
```

<!-- archie:ai-end -->
