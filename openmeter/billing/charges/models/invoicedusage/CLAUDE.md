# invoicedusage

<!-- archie:ai-start -->

> Models accrued usage snapshots that link a charge's computed totals to an optional standard invoice line. Provides an Ent mixin (invoicedusage.Mixin) and the AccruedUsage domain type capturing service period, billing totals, and an optional ledger transaction reference.

## Patterns

**Mutable flag semantics enforced in AccruedUsage.Validate()** — When mutable=false LineID must be non-nil (usage locked to an invoice line). When mutable=true LineID may be nil. Never persist AccruedUsage with mutable=false and LineID=nil. (`if !r.Mutable {
    if r.LineID == nil {
        errs = append(errs, fmt.Errorf("line ID is required when mutable is false"))
    }
}`)
**totals.Mixin composition for billing totals** — The Ent mixin embeds totals.Mixin{} to store billing totals alongside usage data. Use totals.Set(creator, invoicedUsage.Totals) in Create and totals.FromDB(dbEntity) in MapAccruedUsageFromDB. (`creator = totals.Set(creator, invoicedUsage.Totals)
// reading back:
Totals: totals.FromDB(dbEntity)`)
**Nil-safe ledger transaction reference** — LedgerTransaction is optional (*ledgertransaction.GroupReference). Extract group ID pointer safely for SetNillableLedgerTransactionGroupID; MapAccruedUsageFromDB reconstructs it only when non-nil. (`var trnsGroupID *string
if invoicedUsage.LedgerTransaction != nil {
    trnsGroupID = &invoicedUsage.LedgerTransaction.TransactionGroupID
}`)
**Mixin as type alias (not struct)** — type Mixin = entutils.RecursiveMixin[mixin] is a type alias, not a struct. It cannot carry instance fields. Use a standalone struct with embedded RecursiveMixin only when instance fields are needed (see creditrealization.Mixin). (`type Mixin = entutils.RecursiveMixin[mixin]`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `stdinvoice.go` | AccruedUsage domain struct with Validate(). The mutable/LineID invariant and Totals validation live here. | TODO comment notes LineID on AccruedUsage will be removed once flat-fee stores line assignment on its own realization state — don't add new dependencies on LineID being always present. |
| `mixin.go` | Ent mixin + Creator/Getter interfaces + Create/MapAccruedUsageFromDB generic helpers. | Mixin is a type alias (= entutils.RecursiveMixin[mixin]) — it cannot carry instance fields unlike creditrealization.Mixin which is a struct. |

## Anti-Patterns

- Persisting AccruedUsage with mutable=false and LineID=nil
- Bypassing totals.Set/totals.FromDB and setting totals fields manually on the Ent creator
- Adding a non-nil LedgerTransaction with an empty TransactionGroupID

## Decisions

- **Separate AccruedUsage entity instead of embedding usage data into the charge entity** — A charge may accrue usage across multiple billing periods and each accrual snapshot can independently transition from mutable to immutable as invoice lines are assigned.

<!-- archie:ai-end -->
