# models

<!-- archie:ai-start -->

> Structural grouping for shared data-model sub-packages used by all charge types (flatfee, usagebased, creditpurchase). Sub-packages provide: chargemeta (Ent mixin + generic Create/Update/MapFromDB), creditrealization (allocation/correction ledger records), invoicedusage (accrued usage snapshots), ledgertransaction (GroupReference/TimedGroupReference value types), and payment (External/Invoiced payment models with Ent mixin). No direct source files live here.

## Patterns

**chargemeta.Mixin embedding** — All charge Ent schemas must embed chargemeta.Mixin via ent.Mixin() to get standard charge fields (customer_id, periods, status, currency, managed_by, subscription refs, advance_after). Never re-define these fields per charge type. (`func (CreditPurchaseCharge) Mixin() []ent.Mixin { return []ent.Mixin{chargemeta.Mixin{}, ...} }`)
**creditrealization.Mixin + SelfReferenceType** — Charge entities that hold credit realizations must use creditrealization.Mixin with SelfReferenceType set to the entity's own type for self-referencing correction edges. (`creditrealization.Mixin{SelfReferenceType: entschema.CreditPurchaseRealization{}}`)
**totals.Mixin composition in invoicedusage** — AccruedUsage totals fields must be added via totals.Mixin, not inline field definitions, and read/written via totals.Set/totals.FromDB. (`type AccruedUsageMixin struct { ent.Schema }
func (AccruedUsageMixin) Mixin() []ent.Mixin { return []ent.Mixin{totals.Mixin{}} }`)
**payment.Base → External/Invoiced hierarchy** — Payment entities share a common Base (status, currency, amount, authorized/settled refs) via payment.Mixin. Invoiced adds line_id/invoice_id (Immutable). Never embed Base fields directly in a new payment variant. (`type ExternalPayment struct { payment.Base; ... }`)
**ValidationIssue sentinels in payment/errors.go** — All payment domain errors are defined as package-level var ValidationIssue with HTTP status codes, not fmt.Errorf. Use errors.Is to match them. (`var ErrAlreadyAuthorized = models.NewValidationIssue(..., commonhttp.WithHTTPStatusCodeAttribute(http.StatusConflict))`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `chargemeta/mixin.go` | Ent mixin providing all standard charge columns + generic Create/Update/MapFromDB. | intent.Normalized() + meta.NormalizeOptionalTimestamp are called at the top of Create/Update; never skip these. |
| `creditrealization/models.go` | Realization, Realizations, InitialLineageSpec types. | SortHint must be set to non-zero unique values per batch for deterministic correction ordering. |
| `creditrealization/realizations.go` | Realizations.Correct() correction planning; amounts must be rounded via currencyx.Calculator before calling. | Never manually set CorrectsRealizationID; use Realizations.Correct() which handles ID linking and remaining-amount calculation. |
| `ledgertransaction/ledger.go` | GroupReference (TransactionGroupID) and TimedGroupReference. Nil-safe GetIDOrNull(). | Validate() before accessing TransactionGroupID; use GetIDOrNull() for SetNillable Ent fields. |
| `payment/models.go` | Status enum, Base type, CreateInput/UpdateInput interfaces. | StatusSettled requires AuthorizedAt to be non-nil; enforced in Base.Validate(). |
| `payment/invoiced.go` | Invoiced type with Immutable LineID/InvoiceID. | LineID and InvoiceID must not be mutated after creation; use UpdateInvoicedPayment only for status/transaction group fields. |

## Anti-Patterns

- Defining per-charge-type copies of chargemeta fields instead of embedding chargemeta.Mixin.
- Setting SortHint=0 on all realizations in a batch — correction ordering becomes non-deterministic.
- Accessing ledgertransaction.TransactionGroupID without calling Validate() first.
- Mutating Invoiced.LineID or Invoiced.InvoiceID after creation — these are Immutable Ent fields.
- Creating a correction realization without going through Realizations.Correct() — bypasses CorrectsRealizationID assignment and amount validation.

## Decisions

- **chargemeta.Mixin + generic Create/Update/MapFromDB instead of per-type boilerplate** — Three charge types share identical persistence fields; type parameters eliminate duplication while keeping compile-time safety.
- **Two-type realization model (allocation + correction) instead of mutable balance rows** — Append-only model provides a full audit trail for credit reconciliation and avoids lost-update races.
- **Separate Invoiced and External payment types sharing a payment.Base** — Invoiced payments have an immutable invoice/line link not applicable to external payments; separate types prevent accidental mutation.

<!-- archie:ai-end -->
