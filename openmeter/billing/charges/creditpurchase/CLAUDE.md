# creditpurchase

<!-- archie:ai-start -->

> Domain package for credit-purchase charge lifecycle: defines the Service interface (Create, GetByIDs, List, ListFundedCreditActivities, ExternalPaymentLifecycle, InvoicePaymentLifecycle), domain types (Charge, Intent, Settlement, Status), and the Handler interface that mediates all ledger side-effects. Settlement type (promotional/external/invoice) is a tagged-union with MarshalJSON/UnmarshalJSON.

## Patterns

**Settlement tagged union** — Settlement is a private-field struct with a SettlementType discriminator; construct via NewSettlement[T] and access via As*Settlement(). Never set t/invoice/external/promotional fields directly. (`s := creditpurchase.NewSettlement(creditpurchase.ExternalSettlement{GenericSettlement: ..., InitialStatus: CreatedInitialPaymentSettlementStatus})`)
**Intent.Normalized() before Validate()** — Always call intent.Normalized() (which calls meta.Intent.Normalized() and NormalizeOptionalTimestamp) before Validate() to truncate timestamps to streaming.MinimumWindowSizeDuration and round CreditAmount to currency precision. (`input.Intent = input.Intent.Normalized(); if err := input.Validate(); err != nil { ... }`)
**Handler abstraction for ledger** — All ledger writes must go through the creditpurchase.Handler interface methods (OnPromotionalCreditPurchase, OnCreditPurchaseInitiated, OnCreditPurchasePaymentAuthorized, OnCreditPurchasePaymentSettled). Never call ledger APIs directly from charge code. (`ref, err := h.handler.OnCreditPurchaseInitiated(ctx, charge)`)
**models.NewNillableGenericValidationError for Validate()** — All Validate() methods return models.NewNillableGenericValidationError(errors.Join(errs...)) so nil is returned when errs is empty. (`return models.NewNillableGenericValidationError(errors.Join(errs...))`)
**ValidationIssue sentinel errors** — Domain-level errors are defined as package-level var using models.NewValidationIssue with ErrCode constant and commonhttp.WithHTTPStatusCodeAttribute, not fmt.Errorf. (`var ErrCreditPurchaseChargeNotActive = models.NewValidationIssue(ErrCodeCreditPurchaseChargeNotActive, ..., commonhttp.WithHTTPStatusCodeAttribute(http.StatusBadRequest))`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `adapter.go` | Defines Adapter (ChargeAdapter + CreditGrantAdapter + ExternalPaymentAdapter + InvoicedPaymentAdapter + TxCreator) and all input types with Validate(). | Every input type must have a Validate() method returning models.NewNillableGenericValidationError. |
| `charge.go` | Charge, ChargeBase, Intent, Settlement, Realizations, and state types. Intent.Normalized() truncates timestamps. | CreditAmount must be positive; settlement currency must match credit currency; EffectiveAt is not yet supported and always errors. |
| `settlement.go` | Settlement tagged-union with JSON serde. InvoiceSettlement, ExternalSettlement, PromotionalSettlement. | Never instantiate Settlement{} directly; always use NewSettlement[T]. |
| `handler.go` | Handler interface — single abstraction for all ledger side-effects. Promotional path only uses OnPromotionalCreditPurchase; paid paths go through Initiated→Authorized→Settled. | Handler methods must not be called outside the service layer. |
| `service.go` | Service interface = CreditPurchaseService + ExternalPaymentLifecycle + InvoicePaymentLifecycle. | ChargeWithGatheringLine is returned from Create; caller must process GatheringLineToCreate. |
| `statemachine.go` | Status enum mapping meta.ChargeStatus values for credit-purchase. | ToMetaChargeStatus() must be used to convert back to meta.ChargeStatus. |
| `errors.go` | ErrCreditPurchaseChargeNotActive sentinel with HTTP 400. | Use this sentinel (errors.Is) rather than creating new errors for this condition. |
| `funded_credit_activity.go` | FundedCreditActivity, cursor-based pagination input/result for ListFundedCreditActivities. | After and Before cannot both be set; Limit must be >= 1. |

## Anti-Patterns

- Calling ledger APIs directly instead of routing through creditpurchase.Handler — bypasses credits.enabled=false guard.
- Constructing Settlement{} with direct field assignment instead of NewSettlement[T] — leaves discriminator field t empty.
- Skipping intent.Normalized() before Validate() — allows denormalized timestamps and un-rounded amounts to reach the adapter.
- Returning raw fmt.Errorf from Validate() instead of models.NewNillableGenericValidationError — breaks error-type detection upstream.
- Setting EffectiveAt on Intent — it is explicitly unsupported and will return a validation error.

## Decisions

- **Settlement is a tagged union (private fields + discriminator) rather than an interface** — Enables exhaustive JSON serde and GetCostBasis() on all variants without type assertions at call sites.
- **Handler interface decouples ledger writes from charge service** — Allows noop handler when credits.enabled=false without branching inside service code.
- **ChargeWithGatheringLine returned from Create** — Gathering line creation must be deferred to the caller (charges.Service) to fit the billing line lifecycle model without circular dependency.

## Example: Create a credit purchase with invoice settlement

```
import (
    "github.com/openmeterio/openmeter/openmeter/billing/charges/creditpurchase"
    "github.com/openmeterio/openmeter/pkg/currencyx"
)

intent := creditpurchase.Intent{
    Intent:       metaIntent,
    CreditAmount: decimal.NewFromFloat(100),
    Settlement:   creditpurchase.NewSettlement(creditpurchase.InvoiceSettlement{
        GenericSettlement: creditpurchase.GenericSettlement{
            Currency:  currencyx.Code("USD"),
            CostBasis: decimal.NewFromFloat(100),
        },
    }),
}
// ...
```

<!-- archie:ai-end -->
