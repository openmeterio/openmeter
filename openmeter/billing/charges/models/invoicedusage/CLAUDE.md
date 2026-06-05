# invoicedusage

<!-- archie:ai-start -->

> Value type + Ent mixin for AccruedUsage: the snapshot of usage totals over a service period that a usage-based charge run has accrued (optionally tied to a ledger transaction group). Bridges charge runs to invoice totals via the shared totals.Mixin.

## Patterns

**Totals composition via shared mixin** — The Ent mixin embeds totals.Mixin and the domain struct embeds totals.Totals; Create uses totals.Set and MapAccruedUsageFromDB uses totals.FromDB to round-trip the monetary totals. (`creator = totals.Set(creator, invoicedUsage.Totals)`)
**Optional ledger linkage as nillable group ref** — ledger_transaction_group_id is optional/nillable; the domain LedgerTransaction is a *ledgertransaction.GroupReference reconstructed only when the DB value is non-nil. (`if dbEntity.GetLedgerTransactionGroupID() != nil { ledgerTransaction = &ledgertransaction.GroupReference{...} }`)
**Generic Creator/Getter mapping** — Create[T Creator[T]] and MapAccruedUsageFromDB(Getter) compose entutils namespace/id/time/annotations mixin setters with totals setters, so embedding charge-run tables reuse them. (`func Create[T Creator[T]](creator T, ns string, invoicedUsage AccruedUsage) T`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `stdinvoice.go` | AccruedUsage domain struct (NamespacedID + ManagedModel + ServicePeriod + optional LedgerTransaction + Totals) and its Validate. | Validate checks ServicePeriod, Totals, and the LedgerTransaction only when present. |
| `mixin.go` | Ent mixin (service period, optional ledger_transaction_group_id, totals.Mixin) plus generic Create/MapAccruedUsageFromDB. | Service period boundaries are stored and read In(time.UTC); keep totals round-tripping through totals.Set/totals.FromDB. |

## Anti-Patterns

- Storing totals as ad-hoc fields instead of via the shared totals.Mixin/totals.Totals.
- Reconstructing LedgerTransaction when the DB group id is nil.
- Persisting non-UTC service period boundaries.

## Decisions

- **Usage accruals reuse the billing totals model** — Keeps charge-run usage snapshots consistent with invoice-line total semantics and avoids a parallel totals representation.

<!-- archie:ai-end -->
