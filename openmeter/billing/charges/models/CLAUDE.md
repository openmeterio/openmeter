# models

<!-- archie:ai-start -->

> Structural parent grouping shared data-model sub-packages used by all charge types: chargemeta (Ent mixin + generic Create/Update/MapFromDB), creditrealization (allocation/correction ledger records + lineage specs), invoicedusage (accrued usage snapshots), ledgertransaction (GroupReference/TimedGroupReference), and payment (External/Invoiced payment models). No direct source files live here.

## Patterns

**chargemeta.Mixin in all charge Ent schemas** — Every charge Ent schema embeds chargemeta.Mixin via Mixin() to inherit standard charge columns (customer_id, periods, status, currency, managed_by, subscription refs, advance_after). Never re-define these per charge type. (`func (CreditPurchaseCharge) Mixin() []ent.Mixin { return []ent.Mixin{chargemeta.Mixin{}, ...} }`)
**creditrealization.Mixin requires SelfReferenceType** — Entities holding credit realizations use creditrealization.Mixin with SelfReferenceType set to the entity's own type for self-referencing correction edges. (`creditrealization.Mixin{SelfReferenceType: entschema.CreditPurchaseRealization{}}`)
**payment.Base hierarchy for new payment variants** — Payment entities share Base (status, currency, amount, authorized/settled refs) via payment.Mixin; Invoiced adds Immutable line_id/invoice_id. Never embed Base fields directly in a new variant. (`type ExternalPayment struct { payment.Base; ... }`)
**ValidationIssue sentinels in payment/errors.go** — Payment domain errors are package-level var ValidationIssue with HTTP status codes, matched with errors.Is — not fmt.Errorf. (`var ErrAlreadyAuthorized = models.NewValidationIssue(ErrCodeAlreadyAuthorized, "...", commonhttp.WithHTTPStatusCodeAttribute(http.StatusConflict))`)
**ledgertransaction nil-safe helpers** — Use GroupReference.GetIDOrNull() for Ent SetNillable* field methods; call Validate() before reading TransactionGroupID directly. (`creator.SetNillableTransactionGroupID(ref.GetIDOrNull())`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `chargemeta/mixin.go` | Ent mixin with all standard charge columns plus generic Create/Update/MapFromDB helpers. | intent.Normalized() and meta.NormalizeOptionalTimestamp run at the top of Create/Update — never skip them; SortHint must be unique per batch. |
| `creditrealization/models.go` | Realization/Realizations/InitialLineageSpec; allocation/correction duality enforced in CreateInput.Validate(). | SortHint must be non-zero unique per batch for deterministic correction ordering. |
| `creditrealization/realizations.go` | Realizations.Correct() correction planning. | Round amounts via currencyx.Calculator.RoundToPrecision before Correct(); never set CorrectsRealizationID manually. |
| `ledgertransaction/ledger.go` | GroupReference and TimedGroupReference value types; GetIDOrNull() for nil-safe Ent writes. | Constructing TimedGroupReference with a zero Time fails Validate(). |
| `payment/invoiced.go` | Invoiced with Immutable LineID/InvoiceID. | LineID/InvoiceID must not be mutated after creation; UpdateInvoicedPayment is for status/transaction-group fields only. |
| `payment/errors.go` | Domain payment error sentinels as ValidationIssue with HTTP status codes. | Use errors.Is; do not create new errors for already-covered conditions. |

## Anti-Patterns

- Defining per-charge-type copies of chargemeta fields instead of embedding chargemeta.Mixin.
- Setting SortHint=0 on all realizations in a batch — correction ordering becomes non-deterministic.
- Accessing ledgertransaction.TransactionGroupID without calling Validate() first.
- Mutating Invoiced.LineID or Invoiced.InvoiceID after creation — these are Immutable Ent fields.
- Creating a correction realization without Realizations.Correct() — bypasses CorrectsRealizationID assignment and amount validation.

## Decisions

- **chargemeta.Mixin + generic Create/Update/MapFromDB instead of per-type boilerplate.** — Three charge types share identical persistence fields; type parameters eliminate duplication while keeping compile-time safety.
- **Two-type realization model (allocation + correction) instead of mutable balance rows.** — Append-only model provides a full audit trail for credit reconciliation and avoids lost-update races.
- **Separate Invoiced and External payment types sharing a payment.Base.** — Invoiced payments have an immutable invoice/line link not applicable to external payments; separate types prevent accidental mutation.

<!-- archie:ai-end -->
