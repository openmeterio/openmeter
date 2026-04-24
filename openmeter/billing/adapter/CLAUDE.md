# adapter

<!-- archie:ai-start -->

> Ent/PostgreSQL adapter implementing billing.Adapter — the single persistence boundary for all billing domain entities (profiles, customer overrides, invoices, invoice lines, split line groups, sequences, locks, schema migration). Every method wraps Ent queries inside entutils.TransactingRepo so operations compose correctly with caller-supplied transactions carried in ctx.

## Patterns

**TransactingRepo on every mutating method** — All adapter methods that write or need read-consistency must be wrapped with entutils.TransactingRepo (returns value) or entutils.TransactingRepoWithNoValue. The pattern rebinds a.db to the ctx-carried transaction via WithTx, so callers can compose multiple adapter calls inside one transaction without partial writes. (`return entutils.TransactingRepo(ctx, a, func(ctx context.Context, tx *adapter) (*billing.CustomerOverride, error) { ... tx.db.BillingCustomerOverride.Create()... })`)
**Interface compliance assertions at file top** — Every file that implements an adapter sub-interface must declare a compile-time assertion: var _ billing.XxxAdapter = (*adapter)(nil). This catches missing methods immediately rather than at call sites. (`var _ billing.GatheringInvoiceAdapter = (*adapter)(nil)`)
**workflowConfigWithTaxCode eager-load option** — All queries that return a billing profile or workflow config must pass WithWorkflowConfig(workflowConfigWithTaxCode) so that the TaxCode edge is populated and BackfillTaxConfig works correctly on the read path. Omitting it silently produces nil TaxCode in mapped domain objects. (`tx.db.BillingProfile.Query().Where(...).WithWorkflowConfig(workflowConfigWithTaxCode).First(ctx)`)
**Manually attach edges after Save** — Ent's Save() never populates edges on the returned node. After creating or updating a WorkflowConfig or BillingProfile that has a TaxCodeID, manually fetch and assign the edge (e.g. saved.Edges.TaxCode = tc) before returning, or downstream mappers panic on nil pointer dereferences. (`if saved.TaxCodeID != nil { tc, _ := a.db.TaxCode.Get(ctx, *saved.TaxCodeID); saved.Edges.TaxCode = tc }`)
**entitydiff-based upsert for line hierarchies** — Invoice lines and their children (detailed lines, discounts) are updated via a two-pass diff: diffInvoiceLines / diffGatheringInvoiceLines computes Create/Update/Delete sets from DBState vs expected state, then upsertWithOptions executes bulk CREATE ON CONFLICT operations. Never do item-by-item inserts/updates for line hierarchies — use the diff pipeline. (`lineDiffs, _ := diffInvoiceLines(input.Lines); tx.upsertFeeLineConfig(ctx, lineDiffs.DetailedLine)`)
**DBState snapshot on read** — After mapping a line from DB (mapStandardInvoiceLinesFromDB, mapGatheringInvoiceLineFromDB), call line.SaveDBSnapshot() to record the persisted state. The diff engine uses line.DBState to detect changes; omitting the snapshot causes every field to look changed on the next write. (`if err := line.SaveDBSnapshot(); err != nil { return nil, fmt.Errorf("saving DB snapshot [id=%s]: %w", line.GetID(), err) }`)
**Soft-delete pattern with DeletedAt** — Billing entities use soft-delete (SetDeletedAt(clock.Now())) not hard DELETE. Queries must add Where(billinginvoice.DeletedAtIsNil()) unless explicitly fetching deleted records via IncludeDeleted. Sequences and lock rows are never soft-deleted. (`tx.db.BillingInvoice.Update().Where(billinginvoice.ID(id)).SetDeletedAt(clock.Now()).Save(ctx)`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `adapter.go` | Defines Config, the adapter struct (db *entdb.Client, logger), constructor New, and the Tx/WithTx/Self trio required by entutils.TransactingRepo. All other files add methods to this struct. | Tx() uses HijackTx — do not replace with db.BeginTx as it breaks the transaction driver contract. |
| `stdinvoicelines.go` | Implements billing.InvoiceLineAdapter.UpsertInvoiceLines — the most complex write path. Multi-table upsert across BillingInvoiceLine, BillingInvoiceFlatFeeLineConfig, BillingInvoiceUsageBasedLineConfig, discounts, and the schema-level-2 detailed line tables. | SchemaLevel gates which detail-line table is written (schema 1 uses FlatFeeLine table; schema 2 uses BillingStandardInvoiceDetailedLine). Mixing them corrupts the read path. |
| `stdinvoicelinediff.go` | Pure diffing logic (no DB access) for invoice line hierarchies. Produces invoiceLineDiff with separate entitydiff.Diff sets for lines, usage discounts, detailed lines, and detailed line discounts. | Line equality checks call StandardLineBase.Equal and UsageBased.Equal — if you add new fields to those structs you must update the Equal methods or diffs will be missed. |
| `gatheringlines.go` | CRUD and diff/upsert for gathering invoice lines (usage-based only, stored in BillingInvoiceLine + BillingInvoiceUsageBasedLineConfig). Also implements expandSplitLineHierarchy for progressive billing. | Hard-delete (not soft-delete) is used for gathering lines via HardDeleteGatheringInvoiceLines — must also delete the associated BillingInvoiceUsageBasedLineConfig rows atomically. |
| `profile.go` | Profile and WorkflowConfig CRUD. createWorkflowConfig and updateWorkflowConfig both manually fetch the TaxCode edge after Save since Ent never populates edges on mutation results. | workflowConfigWithTaxCode closure must be used on every BillingProfile query that will be mapped — silently omitting it produces nil TaxCode in BackfillTaxConfig. |
| `schemamigration.go` | On-demand per-customer invoice schema migration triggered inside LockCustomerForUpdate. Calls a raw SQL stored-procedure (om_func_migrate_customer_invoices_to_schema_level_2) for schema level 1→2. | Migration runs inside the advisory lock transaction — never call migrateCustomerInvoices outside a LockCustomerForUpdate context. |
| `seq.go` | Sequence number allocation using FOR UPDATE pessimistic locking on BillingSequenceNumbers. Uses upsert-then-reselect to handle the first-use race condition. | NextSequenceNumber must always run inside a transaction (uses ForUpdate). Calling it outside a transaction will succeed but without lock safety. |
| `stdinvoicelinemapper.go` | Pure DB→domain mapping for standard invoice lines (no writes). backfillTaxConfigReferences handles the legacy stripe-code/TaxCodeID precedence logic. | mapStandardInvoiceDetailedLineFromDB is schema-level-1 path; mapStandardInvoiceDetailedLineV2FromDB is schema-level-2 path. The caller in mapStandardInvoiceLinesFromDB selects the correct path via schemaLevelByInvoiceID. |

## Anti-Patterns

- Using a.db directly inside a helper function that could be called from within a transaction — always call entutils.TransactingRepo to rebind to the ctx-carried transaction.
- Querying BillingProfile or BillingWorkflowConfig without WithWorkflowConfig(workflowConfigWithTaxCode) — omitting the eager load produces silent nil in BackfillTaxConfig.
- Calling Save() on a WorkflowConfig or BillingProfile and returning the result without manually attaching the TaxCode edge — downstream mappers dereference the nil edge.
- Bypassing the entitydiff pipeline to write invoice lines one-by-one — the diff pipeline tracks AffectedLineIDs and handles UpdatedAt propagation; direct writes miss these.
- Calling migrateCustomerInvoices outside of a LockCustomerForUpdate context — the schema migration must run under the advisory lock to be safe against concurrent invoice creation.

## Decisions

- **entutils.TransactingRepo is called at the outermost adapter method boundary, not once at the service layer.** — Adapter methods compose with each other (e.g., GetCustomerOverride called inside UpdateCustomerOverride) and the TransactingRepo pattern safely nests by reusing the tx already in ctx. This keeps the service layer free of transaction management.
- **Invoice lines use an entity-diff + bulk-upsert pipeline rather than load-compare-update.** — Invoice line sets can be large; generating a diff first then executing bulk CREATE ON CONFLICT operations minimises round trips and correctly propagates UpdatedAt only to actually changed lines/discounts.
- **Schema migration is lazy (triggered inside LockCustomerForUpdate) rather than a global offline migration job.** — Migrating all invoices at once would require a long-running exclusive lock. Per-customer lazy migration under the billing advisory lock amortises the cost without blocking unrelated customers.

## Example: Implementing a new adapter method that reads and writes inside a transaction

```
func (a *adapter) CreateFoo(ctx context.Context, input billing.CreateFooInput) (*billing.Foo, error) {
    if err := input.Validate(); err != nil {
        return nil, billing.ValidationError{Err: err}
    }
    return entutils.TransactingRepo(ctx, a, func(ctx context.Context, tx *adapter) (*billing.Foo, error) {
        row, err := tx.db.BillingFoo.Create().
            SetNamespace(input.Namespace).
            SetName(input.Name).
            Save(ctx)
        if err != nil {
            return nil, err
        }
        return mapFooFromDB(row), nil
    })
}
```

<!-- archie:ai-end -->
