# models

<!-- archie:ai-start -->

> Structural grouping for shared data-model sub-packages used by all charge types (flatfee, usagebased, creditpurchase). Sub-packages provide: chargemeta (Ent mixin + generic Create/Update/MapFromDB), creditrealization (allocation/correction ledger records), invoicedusage (accrued usage snapshots), ledgertransaction (GroupReference/TimedGroupReference), and payment (External/Invoiced payment models with Ent mixins). No direct source files live here.

## Patterns

**chargemeta.Mixin embedding in all charge Ent schemas** — All charge Ent schemas must embed chargemeta.Mixin via Mixin() to get standard charge fields (customer_id, periods, status, currency, managed_by, subscription refs, advance_after). Never re-define these fields per charge type. (`func (CreditPurchaseCharge) Mixin() []ent.Mixin { return []ent.Mixin{chargemeta.Mixin{}, ...} }`)
**creditrealization.Mixin + SelfReferenceType** — Charge entities that hold credit realizations must use creditrealization.Mixin with SelfReferenceType set to the entity's own type for self-referencing correction edges. (`creditrealization.Mixin{SelfReferenceType: entschema.CreditPurchaseRealization{}}`)
**payment.Base hierarchy for new payment variants** — Payment entities share a common Base (status, currency, amount, authorized/settled refs) via payment.Mixin. Invoiced adds Immutable line_id/invoice_id. Never embed Base fields directly in a new payment variant. (`type ExternalPayment struct { payment.Base; ... }`)
**ValidationIssue sentinels in payment/errors.go** — All payment domain errors are defined as package-level var ValidationIssue with HTTP status codes, not fmt.Errorf. Use errors.Is to match them. (`var ErrAlreadyAuthorized = models.NewValidationIssue(ErrCodeAlreadyAuthorized, "...", commonhttp.WithHTTPStatusCodeAttribute(http.StatusConflict))`)
**ledgertransaction.GroupReference nil-safe helpers** — Use GetIDOrNull() for SetNillable Ent field methods; call Validate() before accessing TransactionGroupID directly. (`creator.SetNillableTransactionGroupID(ref.GetIDOrNull())`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `chargemeta/mixin.go` | Ent mixin providing all standard charge columns and generic Create/Update/MapFromDB type-parameterized helpers. | intent.Normalized() and meta.NormalizeOptionalTimestamp are called at the top of Create/Update — never skip these; SortHint must be unique per batch. |
| `creditrealization/models.go` | Realization, Realizations, InitialLineageSpec types; allocation/correction duality enforced in CreateInput.Validate(). | SortHint must be non-zero unique values per batch for deterministic correction ordering. |
| `creditrealization/realizations.go` | Realizations.Correct() correction planning — must be used instead of manually constructing correction records. | Round amounts via currencyx.Calculator.RoundToPrecision before calling Correct(); never manually set CorrectsRealizationID. |
| `ledgertransaction/ledger.go` | GroupReference and TimedGroupReference value types. GetIDOrNull() for nil-safe Ent SetNillable methods. | Constructing TimedGroupReference with a zero Time value — Validate() will reject it. |
| `payment/invoiced.go` | Invoiced type with Immutable LineID/InvoiceID fields. | LineID and InvoiceID must not be mutated after creation; UpdateInvoicedPayment only for status/transaction group fields. |
| `payment/errors.go` | Domain error sentinels as ValidationIssue with HTTP status codes. | Use errors.Is; do not create new errors for conditions already covered by these sentinels. |

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
