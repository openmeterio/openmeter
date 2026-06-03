# creditpurchase

<!-- archie:ai-start -->

> The credit-purchase charge type within the billing/charges tagged-union family: defines the Charge/Intent/Settlement domain (this folder) and splits implementation across adapter/ (Ent persistence), service/ (transaction.Run orchestration + settlement dispatch), and lineengine/ (rating delegation). Primary constraint: all ledger side-effects flow through the creditpurchase.Handler interface so a noop handler can satisfy credits.enabled=false.

## Patterns

**Settlement tagged-union via NewSettlement[T]** — Settlement is a private-field struct with discriminator t; construct only via NewSettlement[T] and read via AsInvoiceSettlement/AsExternalSettlement/AsPromotionalSettlement. Never set t/invoice/external/promotional directly. (`s := creditpurchase.NewSettlement(creditpurchase.ExternalSettlement{GenericSettlement: creditpurchase.GenericSettlement{Currency: "USD", CostBasis: amount}, InitialStatus: creditpurchase.CreatedInitialPaymentSettlementStatus})`)
**Intent.Normalized() before Validate()** — Always call input.Intent = input.Intent.Normalized() (truncates timestamps, rounds CreditAmount to currency precision) before input.Validate() in Create. The service mirrors this before reaching the adapter. (`input.Intent = input.Intent.Normalized()
if err := input.Validate(); err != nil { return err }`)
**Handler mediation for all ledger interactions** — Every ledger write goes through Handler methods (OnPromotionalCreditPurchase, OnCreditPurchaseInitiated, OnCreditPurchasePaymentAuthorized, OnCreditPurchasePaymentSettled). Promotional uses only OnPromotionalCreditPurchase; paid paths run Initiated -> Authorized -> Settled. (`ref, err := h.handler.OnCreditPurchaseInitiated(ctx, charge)`)
**models.NewNillableGenericValidationError in every Validate()** — All input/domain Validate() methods collect into []error and return models.NewNillableGenericValidationError(errors.Join(errs...)) so nil is returned when empty. Never return raw fmt.Errorf from Validate(). (`return models.NewNillableGenericValidationError(errors.Join(errs...))`)
**ValidationIssue sentinels with HTTP status** — Domain conditions are package-level var ValidationIssue with an ErrCode constant and commonhttp.WithHTTPStatusCodeAttribute, matched upstream via errors.Is. Never invent a fresh error for a covered condition. (`var ErrCreditPurchaseChargeNotActive = models.NewValidationIssue(ErrCodeCreditPurchaseChargeNotActive, "...", models.WithCriticalSeverity(), commonhttp.WithHTTPStatusCodeAttribute(http.StatusBadRequest))`)
**Charge value-semantics for the generic state machine** — Charge implements GetStatus/WithStatus/GetBase/WithBase as value receivers returning copies (meta.ChargeAccessor). Pointer-mutating these breaks the generic Machine external-storage pattern. (`func (c Charge) WithStatus(status Status) Charge { c.Status = status; return c }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `charge.go` | Charge, ChargeBase, Intent, Realizations, State. Intent.Normalized() truncates timestamps and rounds CreditAmount. | CreditAmount must be positive; settlement currency must match credit currency; EffectiveAt is unsupported and always errors in Validate(). |
| `settlement.go` | Settlement tagged-union with JSON serde; three variants (Invoice/External/Promotional). | Never instantiate Settlement{} directly; GetCostBasis() returns zero for promotional. |
| `handler.go` | Handler interface — sole abstraction for ledger side-effects, documenting the Initiated->Authorized->Settled happy path. | Do not call Handler methods outside the service layer. |
| `service.go` | Service interface (CreditPurchaseService + ExternalPaymentLifecycle + InvoicePaymentLifecycle). Create returns ChargeWithGatheringLine. | GatheringLineToCreate may be non-nil; the parent charges.Service creates the gathering line after Create returns. |
| `adapter.go` | Adapter (Charge+CreditGrant+ExternalPayment+InvoicedPayment + TxCreator) and input types, each with Validate(). | Every new input type needs a Validate(); namespace must be non-empty. |
| `statemachine.go` | Status enum mirroring meta.ChargeStatus; ToMetaChargeStatus() converts back. | Use ToMetaChargeStatus() not a direct cast — direct cast skips Validate(). |
| `errors.go` | ErrCreditPurchaseChargeNotActive sentinel (HTTP 400). | Match with errors.Is; do not create new errors for this condition. |

## Anti-Patterns

- Calling ledger APIs directly instead of routing through creditpurchase.Handler — bypasses the credits.enabled=false noop guard.
- Constructing Settlement{} via field assignment instead of NewSettlement[T] — leaves discriminator t empty.
- Skipping intent.Normalized() before Validate() — denormalized timestamps and un-rounded amounts reach the adapter.
- Returning raw fmt.Errorf from Validate() instead of models.NewNillableGenericValidationError — breaks error-type detection upstream.
- Calling adapter methods outside transaction.Run in service code — breaks atomicity of multi-step charge flows.

## Decisions

- **Settlement is a tagged union (private fields + discriminator) rather than an interface.** — Enables exhaustive JSON serde and GetCostBasis() across variants without type assertions, and prevents partially-constructed Settlement values.
- **Handler interface decouples ledger writes from charge service.** — Allows a noop handler when credits.enabled=false without branching inside service code; keeps the ledger boundary clean.
- **ChargeWithGatheringLine returned from Create rather than creating the line inline.** — Gathering-line creation is deferred to the parent charges.Service to avoid a circular dependency between creditpurchase and billing.

## Example: Create a credit purchase charge with invoice settlement

```
import (
  "github.com/openmeterio/openmeter/openmeter/billing/charges/creditpurchase"
  "github.com/openmeterio/openmeter/pkg/currencyx"
)

intent := creditpurchase.Intent{
  Intent:       metaIntent,
  CreditAmount: decimal.NewFromFloat(100),
  Settlement: creditpurchase.NewSettlement(creditpurchase.InvoiceSettlement{
    GenericSettlement: creditpurchase.GenericSettlement{Currency: currencyx.Code("USD"), CostBasis: decimal.NewFromFloat(100)},
  }),
}.Normalized()
```

<!-- archie:ai-end -->
