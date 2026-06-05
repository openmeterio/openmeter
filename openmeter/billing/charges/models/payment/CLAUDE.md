# payment

<!-- archie:ai-start -->

> Domain models, Ent mixins, and ValidationIssue errors for payment settlements on credit-purchase charges. Models a two-phase settlement (authorized -> settled) with two concrete variants: External (off-platform) and Invoiced (settled via an OpenMeter standard invoice line).

## Patterns

**Base + variant composition** — A shared Base (status, amount, service period, authorized/settled timed ledger refs) is embedded into Payment, which External and Invoiced embed; each variant adds only its extra fields (Invoiced adds LineID/InvoiceID). (`type Invoiced struct { Payment; LineID string; InvoiceID string }`)
**Status-driven invariant validation** — Base.Validate switches on Status: StatusAuthorized requires Authorized data; StatusSettled requires both Settled and Authorized data. Amount must be positive. (`case StatusSettled: if r.Settled == nil { ... }; if r.Authorized == nil { ... }`)
**ValidationIssue errors with HTTP attributes** — Lifecycle errors are package-level models.ValidationIssue values with an ErrorCode constant, critical severity, and a commonhttp HTTP status attribute (400). (`var ErrPaymentAlreadySettled = models.NewValidationIssue(ErrCodePaymentAlreadySettled, "payment already settled", models.WithCriticalSeverity(), commonhttp.WithHTTPStatusCodeAttribute(http.StatusBadRequest))`)
**Generic Create/Update/Map over variant creators** — Base Create/Update/Getter helpers operate on generic Creator[T]/Updater[T]/Getter; variant helpers (CreateExternal, CreateInvoiced) wrap them and add variant-only setters via extended interfaces (InvoicedCreator embeds Creator). (`func CreateInvoiced[T InvoicedCreator[T]](creator InvoicedCreator[T], in InvoicedCreate) T`)
**Timed ledger ref round-trip** — Authorized/Settled are stored as separate *_transaction_group_id + *_at columns and reconstructed into a *ledgertransaction.TimedGroupReference only when both the id and the time are non-nil. (`func mapTimedLedgerTransactionGroupReferenceFromDB(reference *string, at *time.Time) *ledgertransaction.TimedGroupReference`)
**ErrorAttributes for observability** — Each variant implements ErrorAttributes() returning settlement status/type/id attribute keys, distinguishing PaymentSettlementTypeExternal vs PaymentSettlementTypeStandardInvoice. (`func (r Invoiced) ErrorAttributes() models.Attributes`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `models.go` | Status enum, Base struct + status-driven Validate, Payment struct (NamespacedID+ManagedModel+Base), and settlement-type/attribute-key constants. | Authorized/Settled invariants depend on Status; amount must be positive. A TODO notes settlement-type constants should be unified. |
| `mixin.go` | Ent mixin (service period, status enum, numeric amount, authorized/settled group-id + at columns) plus generic Create/Update/Getter and the timed-ref mapping helpers. | Edges to ledger are TODO (only group-id strings stored). Update/Create coerce all times to UTC via convert.TimePtrIn. |
| `external.go` | External variant (off-platform settlement): ExternalCreateInput/Validate, CreateExternal/MapExternalFromDB/UpdateExternal. | ExternalMixin is just an alias for Mixin — External adds no columns beyond Base. |
| `invoiced.go` | Invoiced variant: invoicedMixin adds immutable line_id/invoice_id columns; InvoicedCreate/Validate, CreateInvoiced, MapInvoicedFromDB. | LineID and InvoiceID are Immutable and required; InvoicedCreator extends Creator with SetLineID/SetInvoiceID. |
| `errors.go` | Package-level payment lifecycle ValidationIssues (already-authorized, already-settled, cannot-settle-unauthorized). | All carry HTTP 400 and critical severity; reuse these rather than constructing ad-hoc errors for the same conditions. |

## Anti-Patterns

- Marking a payment StatusSettled without both Authorized and Settled ledger data populated.
- Returning plain fmt.Errorf for payment lifecycle conflicts instead of the predefined ValidationIssues in errors.go.
- Mutating line_id/invoice_id on an Invoiced payment — both are immutable in the schema.
- Reconstructing a TimedGroupReference when either the group id or the timestamp is nil.
- Allowing a non-positive settlement amount.

## Decisions

- **External and Invoiced share a Base via embedding rather than duplicating settlement fields** — Authorization/settlement lifecycle and amount/period semantics are identical across settlement types; only the linkage (invoice line vs none) differs.
- **Authorized/settled ledger linkage stored as id+timestamp column pairs, edges deferred** — Keeps the schema simple now (TODO to add real ledger edges) while still recording the transaction group and time for both phases.

## Example: Defining a reusable payment-lifecycle ValidationIssue with HTTP status

```
const ErrCodeCannotSettleNotAuthorizedPayment models.ErrorCode = "cannot_settle_not_authorized_payment"

var ErrCannotSettleNotAuthorizedPayment = models.NewValidationIssue(
	ErrCodeCannotSettleNotAuthorizedPayment,
	"cannot settle an unauthorized payment",
	models.WithCriticalSeverity(),
	commonhttp.WithHTTPStatusCodeAttribute(http.StatusBadRequest),
)
```

<!-- archie:ai-end -->
