# payment

<!-- archie:ai-start -->

> Models payment settlements for charges with two concrete variants — External (direct external payment) and Invoiced (settled via a standard invoice line) — sharing a typed Base struct. Provides Ent mixins, domain error sentinels as ValidationIssue values with HTTP status codes, and generic Create/Update/MapFromDB helpers.

## Patterns

**Base→Payment→Invoiced/External type hierarchy** — Base holds mutable fields. Payment embeds Base + NamespacedID + ManagedModel. Invoiced and External embed Payment adding type-specific fields. Create/Update/MapFromDB functions are layered accordingly. (`func CreateInvoiced[T InvoicedCreator[T]](creator InvoicedCreator[T], in InvoicedCreate) T {
    creator = Create(creator, in.Namespace, in.Base)
    creator = creator.SetInvoiceID(in.InvoiceID)
    return creator.SetLineID(in.LineID)
}`)
**Status invariants enforced in Base.Validate()** — StatusAuthorized requires Authorized != nil. StatusSettled requires both Authorized != nil and Settled != nil. New status values must add a corresponding validation case. (`case StatusSettled:
    if r.Settled == nil { errs = append(errs, ...) }
    if r.Authorized == nil { errs = append(errs, ...) }`)
**Domain error sentinels as ValidationIssue with HTTP status** — Package-level errors (ErrPaymentAlreadyAuthorized, ErrPaymentAlreadySettled, ErrCannotSettleNotAuthorizedPayment) are models.ValidationIssue values with HTTP 400 attached via commonhttp.WithHTTPStatusCodeAttribute. Use these sentinels directly. (`var ErrPaymentAlreadyAuthorized = models.NewValidationIssue(
    ErrCodePaymentAlreadyAuthorized,
    "payment already authorized",
    models.WithCriticalSeverity(),
    commonhttp.WithHTTPStatusCodeAttribute(http.StatusBadRequest),
)`)
**InvoicedMixin composes base Mixin + immutable line_id/invoice_id fields** — Use InvoicedMixin (= entutils.RecursiveMixin[invoicedMixin]) for invoiced-payment Ent entities. It composes Mixin{} and adds line_id + invoice_id as Immutable fields. (`func (MyInvoicedPayment) Mixin() []ent.Mixin {
    return []ent.Mixin{payment.InvoicedMixin{}}
}`)
**convert.TimePtrIn(ptr, time.UTC) in Update for timestamp normalization** — In the generic Update function, all timestamp pointers are passed through convert.TimePtrIn(ptr, time.UTC) before being set on the updater to ensure UTC storage. Never pass raw pointer without timezone conversion. (`SetNillableAuthorizedAt(convert.TimePtrIn(in.Authorized.GetTimeOrNull(), time.UTC))`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `models.go` | Base, Payment, Status enum, attribute key constants. Status invariants and amount-must-be-positive validation live here. | PaymentSettlementTypeExternal / PaymentSettlementTypeStandardInvoice are plain string constants — TODO says make a single typed constant per type; don't add a third as a bare string. |
| `mixin.go` | Ent mixin for base payment fields, Creator/Updater/Getter interfaces, generic Create/Update/mapPaymentFromDB. | convert.TimePtrIn(ptr, time.UTC) is required in Update for timestamp fields; raw pointers without timezone conversion cause UTC drift. |
| `invoiced.go` | Invoiced type with line_id + invoice_id, InvoicedCreator interface, InvoicedMixin for Ent schema. | line_id and invoice_id are Immutable in the Ent schema — they cannot be updated after creation. |
| `external.go` | External type and ExternalCreateInput. ExternalMixin = Mixin (no extra fields needed). CreateExternal delegates to the generic Create. | ErrorAttributes() on External includes the type string PaymentSettlementTypeExternal — keep in sync with models.go constants. |
| `errors.go` | Package-level error sentinel variables. All are ValidationIssue with critical severity and HTTP 400. | Each error has an ErrorCode constant that is the stable wire identifier — don't rename codes without a compatibility check. |

## Anti-Patterns

- Creating a StatusSettled payment without Authorized reference set
- Mutating line_id or invoice_id on an existing Invoiced entity — they are Immutable in the Ent schema
- Returning raw fmt.Errorf for domain payment errors instead of the package-level ValidationIssue sentinels
- Adding a new payment variant by copying Base fields instead of embedding Base via the mixin hierarchy
- Omitting convert.TimePtrIn(ptr, time.UTC) when writing timestamp pointers in the Update path

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
