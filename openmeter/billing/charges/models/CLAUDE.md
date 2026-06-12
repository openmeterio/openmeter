# models

<!-- archie:ai-start -->

> Structural parent for the charges value-type/Ent-mixin layer. It owns no source files itself; its sub-packages carry the persisted domain shapes shared across charge types — chargemeta (shared charge metadata mixin), creditrealization (allocation/correction + lineage state machine), invoicedusage (accrued-usage totals), ledgertransaction (group references), and payment (two-phase settlements).

## Patterns

**One value-type + Ent-mixin package per concern** — Each child pairs domain value types with a RecursiveMixin and generic Create/Update/Map helpers that concrete charge schemas (flatfee/usagebased/creditpurchase) embed, so persistence shape and domain shape stay co-located (`chargemeta/mixin.go, creditrealization/mixin.go, payment/mixin.go`)
**Generic builder helpers over concrete Ent types** — Mapping/creation goes through generic Create/Update/Map functions (Creator/Getter interfaces) rather than calling Set* on a specific *entdb.XCreate, which would bypass normalization/validation (`creditrealization generic Create over allocation/correction creators`)
**UTC + currency-precision normalization at the boundary** — All timestamps coerced to UTC and decimal amounts rounded via currencyx.Calculator before persist; sub-microsecond/sub-window drift is not allowed (`Intent normalize in chargemeta; rounding in creditrealization`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `chargemeta/` | Shared charge-meta Ent mixin (periods, status, currency, managed-by, subscription/tax refs) embedded by every charge type | Go through generic Create/Update helpers; only TaxCodeID FK + Behavior belong on charge tables |
| `creditrealization/` | Allocation (positive) vs correction (non-positive) realizations plus the lineage origin/state machine | Corrections are separate rows referencing CorrectsRealizationID; revert draining is reverse-order; ledger_transaction_group_id is immutable |
| `invoicedusage/` | AccruedUsage snapshot built on the shared totals.Mixin, optionally linked to a ledger group | Use totals.Mixin not ad-hoc fields; don't reconstruct LedgerTransaction when group id is nil |
| `ledgertransaction/` | GroupReference/TimedGroupReference leaf value types with nil-safe accessors | Keep minimal — broadly imported; use GetIDOrNull guards instead of dereferencing |
| `payment/` | Two-phase (authorized->settled) payment settlements with External and Invoiced variants sharing a Base | StatusSettled requires both authorized+settled ledger data; line_id/invoice_id immutable; use predefined ValidationIssues |

## Anti-Patterns

- Adding business orchestration here — these are value-type/mixin packages; logic lives in charge services
- Calling Set* on a concrete *entdb.XCreate instead of the generic Create/Update helpers (bypasses normalization)
- Persisting non-UTC timestamps or unrounded decimals
- Putting charge-type-specific fields in the broadly imported ledgertransaction leaf package

## Decisions

- **Schema mixin and domain mapping co-located per concern** — Keeps the persisted Ent shape and the validated domain shape in one place so each charge type embeds a single source of truth
- **Corrections and settlements are append-only/immutable rows** — Credit realizations and payments form an audit trail; in-place edits would lose lifecycle history

<!-- archie:ai-end -->
