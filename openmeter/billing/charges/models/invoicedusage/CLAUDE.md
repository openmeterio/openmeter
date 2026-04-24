# invoicedusage

<!-- archie:ai-start -->

> Models accrued usage snapshots that link a charge's computed totals to an optional standard invoice line. Provides an Ent mixin (invoicedusage.Mixin) and the AccruedUsage domain type, which captures service period, totals, mutability flag, and optional ledger transaction reference.

## Patterns

**Mutable flag semantics** — When mutable=false the LineID must be non-nil (usage has been locked to an invoice line). When mutable=true LineID may be nil (usage is still gathering and can be reallocated as credits). AccruedUsage.Validate() enforces this. (`if !r.Mutable {
    if r.LineID == nil {
        errs = append(errs, fmt.Errorf("line ID is required when mutable is false"))
    }
}`)
**totals.Mixin composition** — The Ent mixin embeds totals.Mixin{} to store billing totals alongside usage data. Use totals.Set(creator, invoicedUsage.Totals) in Create and totals.FromDB(dbEntity) in MapAccruedUsageFromDB. (`creator = totals.Set(creator, invoicedUsage.Totals)
// ...
Totals: totals.FromDB(dbEntity)`)
**Nil-safe ledger transaction reference** — LedgerTransaction is optional (*ledgertransaction.GroupReference). The Create function extracts the group ID pointer safely and passes it via SetNillableLedgerTransactionGroupID. MapAccruedUsageFromDB reconstructs it only when non-nil. (`var trnsGroupID *string
if invoicedUsage.LedgerTransaction != nil {
    trnsGroupID = &invoicedUsage.LedgerTransaction.TransactionGroupID
}`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `stdinvoice.go` | AccruedUsage domain struct with Validate(). The mutable/LineID invariant and Totals validation live here. | TODO comment notes that LineID on AccruedUsage will be removed once flat-fee stores line assignment on its own realization state — don't add new dependencies on LineID being always present. |
| `mixin.go` | Ent mixin + Creator/Getter interfaces + Create/MapAccruedUsageFromDB generic helpers. | The Mixin type is a type alias (= entutils.RecursiveMixin[mixin]) not a struct — it cannot carry instance fields. |

## Anti-Patterns

- Persisting AccruedUsage with mutable=false and LineID=nil
- Bypassing totals.Set/totals.FromDB and setting totals fields manually
- Adding a non-nil LedgerTransaction with an empty TransactionGroupID

## Decisions

- **Separate AccruedUsage entity instead of embedding usage data into the charge entity** — A charge may accrue usage across multiple billing periods and each accrual snapshot can independently transition from mutable to immutable as invoice lines are assigned.

<!-- archie:ai-end -->
