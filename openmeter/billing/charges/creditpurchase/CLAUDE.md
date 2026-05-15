# creditpurchase

<!-- archie:ai-start -->

> Domain package for credit-purchase charge lifecycle: defines Service, Adapter, and Handler interfaces plus all domain types (Charge, Intent, Settlement tagged-union, Status, Realizations). Settlement type is a private-field tagged union (promotional/external/invoice) with full JSON serde. The Handler interface decouples all ledger side-effects from charge service code.

## Patterns

**Settlement tagged union via NewSettlement[T]** — Settlement is a private-field struct with a SettlementType discriminator t. Always construct via NewSettlement[T] and access via AsInvoiceSettlement/AsExternalSettlement/AsPromotionalSettlement. Never set t, invoice, external, or promotional fields directly. (`s := creditpurchase.NewSettlement(creditpurchase.ExternalSettlement{GenericSettlement: creditpurchase.GenericSettlement{Currency: "USD", CostBasis: decimal.NewFromFloat(100)}, InitialStatus: CreatedInitialPaymentSettlementStatus})`)
**Intent.Normalized() before Validate()** — Always call intent.Normalized() (which calls meta.Intent.Normalized() and NormalizeOptionalTimestamp, and rounds CreditAmount to currency precision) before Validate(). Skipping normalization allows denormalized timestamps and un-rounded amounts to reach the adapter. (`input.Intent = input.Intent.Normalized()
if err := input.Validate(); err != nil { return err }`)
**Handler abstraction for all ledger interactions** — All ledger writes must go through creditpurchase.Handler interface methods: OnPromotionalCreditPurchase, OnCreditPurchaseInitiated, OnCreditPurchasePaymentAuthorized, OnCreditPurchasePaymentSettled. Never call ledger APIs directly from charge code. (`ref, err := h.handler.OnCreditPurchaseInitiated(ctx, charge)`)
**models.NewNillableGenericValidationError in all Validate()** — All Validate() methods collect errors into []error and return models.NewNillableGenericValidationError(errors.Join(errs...)) so nil is returned when errs is empty. Never return raw fmt.Errorf from Validate(). (`return models.NewNillableGenericValidationError(errors.Join(errs...))`)
**ValidationIssue sentinel errors with HTTP status** — Domain-level errors are defined as package-level var using models.NewValidationIssue with an ErrCode constant and commonhttp.WithHTTPStatusCodeAttribute, not fmt.Errorf. Use errors.Is to match them upstream. (`var ErrCreditPurchaseChargeNotActive = models.NewValidationIssue(ErrCodeCreditPurchaseChargeNotActive, "credit purchase charge is not active", models.WithFieldString("namespace"), models.WithCriticalSeverity(), commonhttp.WithHTTPStatusCodeAttribute(http.StatusBadRequest))`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `adapter.go` | Defines Adapter (ChargeAdapter + CreditGrantAdapter + ExternalPaymentAdapter + InvoicedPaymentAdapter + TxCreator) and all input types, each with Validate() returning models.NewNillableGenericValidationError. | Every new input type must have a Validate() method; namespace must always be checked non-empty. |
| `charge.go` | Charge, ChargeBase, Intent, Settlement type, Realizations, and Status types. Intent.Normalized() truncates timestamps and rounds CreditAmount. | CreditAmount must be positive; settlement currency must match credit currency; EffectiveAt is unsupported and always errors in Validate(). |
| `settlement.go` | Settlement tagged-union with JSON MarshalJSON/UnmarshalJSON. Three variants: InvoiceSettlement, ExternalSettlement, PromotionalSettlement. | Never instantiate Settlement{} directly; always use NewSettlement[T]. GetCostBasis() returns zero for promotional. |
| `handler.go` | Handler interface — single abstraction for all ledger side-effects. Promotional uses OnPromotionalCreditPurchase only; paid paths go Initiated -> Authorized -> Settled. | Handler methods must not be called outside the service layer. |
| `service.go` | Service interface = CreditPurchaseService + ExternalPaymentLifecycle + InvoicePaymentLifecycle. Create returns ChargeWithGatheringLine — the caller must handle GatheringLineToCreate. | ChargeWithGatheringLine.GatheringLineToCreate may be non-nil; the parent charges.Service must create the gathering line after Create returns. |
| `errors.go` | ErrCreditPurchaseChargeNotActive sentinel ValidationIssue with HTTP 400. | Use errors.Is(err, ErrCreditPurchaseChargeNotActive) to detect this condition; do not create new errors for it. |
| `funded_credit_activity.go` | FundedCreditActivity and cursor-based ListFundedCreditActivitiesInput/Result for the credit activity feed. | After and Before cannot both be set; Limit must be >= 1. |
| `statemachine.go` | Status enum mapping meta.ChargeStatus values for credit-purchase; ToMetaChargeStatus() converts back. | Always use ToMetaChargeStatus() to convert Status back to meta.ChargeStatus; direct cast bypasses Validate(). |

## Anti-Patterns

- Calling ledger APIs directly instead of routing through creditpurchase.Handler — bypasses credits.enabled=false guard.
- Constructing Settlement{} with direct field assignment instead of NewSettlement[T] — leaves discriminator field t empty.
- Skipping intent.Normalized() before Validate() — allows denormalized timestamps and un-rounded amounts to reach the adapter.
- Returning raw fmt.Errorf from Validate() instead of models.NewNillableGenericValidationError — breaks error-type detection upstream.
- Setting EffectiveAt on Intent — it is explicitly unsupported and will return a validation error.

## Decisions

- **Settlement is a tagged union (private fields + discriminator) rather than an interface** — Enables exhaustive JSON serde and GetCostBasis() on all variants without type assertions at call sites, and prevents partially-constructed Settlement values.
- **Handler interface decouples ledger writes from charge service** — Allows a noop handler when credits.enabled=false without branching inside service code; keeps the ledger abstraction boundary clean.
- **ChargeWithGatheringLine returned from Create** — Gathering line creation must be deferred to the parent charges.Service to fit the billing line lifecycle model without a circular dependency between creditpurchase and billing.

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
        GenericSettlement: creditpurchase.GenericSettlement{
            Currency:  currencyx.Code("USD"),
            CostBasis: decimal.NewFromFloat(100),
        },
    }),
}
// ...
```

<!-- archie:ai-end -->
