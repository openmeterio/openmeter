# payment

<!-- archie:ai-start -->

> Models payment settlements for charges, supporting two concrete variants: External (direct external payment) and Invoiced (settled via a standard invoice line). Provides an Ent mixin with authorized/settled transaction group fields, typed Creator/Updater/Getter interfaces, and domain-level error sentinel variables with HTTP status codes.

## Patterns

**Base→Payment→Invoiced/External type hierarchy** — Base holds the mutable fields (status, amount, service period, authorized/settled references). Payment embeds Base + NamespacedID + ManagedModel. Invoiced and External embed Payment adding their specific fields. Create/Update/MapFromDB are layered accordingly. (`func CreateInvoiced[T InvoicedCreator[T]](creator InvoicedCreator[T], in InvoicedCreate) T {
    creator = Create(creator, in.Namespace, in.Base)
    creator = creator.SetInvoiceID(in.InvoiceID)
    return creator.SetLineID(in.LineID)
}`)
**Status invariants enforced in Base.Validate()** — StatusAuthorized requires Authorized != nil. StatusSettled requires both Authorized != nil and Settled != nil. Adding a new status must add a corresponding validation case here. (`case StatusSettled:
    if r.Settled == nil { errs = append(errs, ...) }
    if r.Authorized == nil { errs = append(errs, ...) }`)
**Domain error sentinels as ValidationIssue** — Package-level errors (ErrPaymentAlreadyAuthorized, ErrPaymentAlreadySettled, ErrCannotSettleNotAuthorizedPayment) are models.ValidationIssue values with HTTP status attached via commonhttp.WithHTTPStatusCodeAttribute. Use these sentinels directly rather than creating inline errors. (`var ErrPaymentAlreadyAuthorized = models.NewValidationIssue(
    ErrCodePaymentAlreadyAuthorized,
    "payment already authorized",
    models.WithCriticalSeverity(),
    commonhttp.WithHTTPStatusCodeAttribute(http.StatusBadRequest),
)`)
**InvoicedMixin for invoiced-payment-specific Ent schema** — Use InvoicedMixin (= entutils.RecursiveMixin[invoicedMixin]) when defining an Ent entity for invoiced payments. It composes the base Mixin{} and adds the line_id + invoice_id immutable fields. (`func (MyInvoicedPayment) Mixin() []ent.Mixin {
    return []ent.Mixin{
        payment.InvoicedMixin{},
    }
}`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `models.go` | Base, Payment, Status enum, attribute key constants. Status invariants and amount-must-be-positive validation live here. | PaymentSettlementTypeExternal / PaymentSettlementTypeStandardInvoice are plain string constants, not typed — TODO comment says make a single constant per type; don't add a third type as a bare string. |
| `mixin.go` | Ent mixin for the base payment fields, Creator/Updater/Getter interfaces, generic Create/Update/mapPaymentFromDB/mapTimedLedgerTransactionGroupReferenceFromDB. | convert.TimePtrIn(ptr, time.UTC) is used in Update to normalize timestamps to UTC before writing; raw pointers without timezone conversion will cause drift. |
| `external.go` | External type and ExternalCreateInput. ExternalMixin = Mixin (no extra fields needed). CreateExternal delegates to the generic Create. | ErrorAttributes() on External includes the type string PaymentSettlementTypeExternal — keep in sync with models.go constants. |
| `invoiced.go` | Invoiced type with line_id + invoice_id, InvoicedCreator interface, InvoicedMixin for Ent schema. | line_id and invoice_id are Immutable in the Ent schema — they cannot be updated after creation. |
| `errors.go` | Package-level error sentinel variables. All are ValidationIssue with critical severity and HTTP 400. | Each error has an ErrorCode constant (ErrCodePaymentAlreadyAuthorized etc.) that is the stable wire identifier — don't rename codes without a compatibility check. |

## Anti-Patterns

- Creating a StatusSettled payment without an Authorized reference
- Mutating line_id or invoice_id on an existing Invoiced entity (they are Immutable)
- Returning raw fmt.Errorf for domain payment errors instead of the package-level ValidationIssue sentinels
- Adding a new payment variant by copying Base fields instead of embedding Base via the mixin hierarchy

## Decisions

- **Separate Invoiced and External as distinct types sharing a Payment base** — Invoiced payments are settled through the OpenMeter invoice flow and must carry line_id + invoice_id as immutable references; External payments have no invoice counterpart. Keeping them separate prevents incorrect assumptions about which fields are present.
- **ValidationIssue sentinels with HTTP status attributes instead of generic errors** — The HTTP error encoder reads the httpStatusCodeErrorAttribute to produce the correct status code; domain errors must carry this attribute to avoid falling through to a 500.

## Example: Create an external payment record

```
import "github.com/openmeterio/openmeter/openmeter/billing/charges/models/payment"

in := payment.ExternalCreateInput{
    Namespace: ns,
    Base: payment.Base{
        Status:        payment.StatusAuthorized,
        Amount:        amount,
        ServicePeriod: period,
        Authorized: &ledgertransaction.TimedGroupReference{
            GroupReference: ledgertransaction.GroupReference{TransactionGroupID: txGroupID},
            Time:           authorizedAt,
        },
    },
}
// pass to adapter: payment.CreateExternal(creator, in)
```

<!-- archie:ai-end -->
