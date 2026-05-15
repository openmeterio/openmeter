# service

<!-- archie:ai-start -->

> Implements creditgrant.Service as a facade that orchestrates credit grant creation, retrieval, listing, and external settlement transitions by delegating entirely to charges.Service and creditpurchase.Service. Owns no direct DB access — all persistence flows through the charges pipeline.

## Patterns

**Config struct with Validate() + New()** — All dependencies declared in a Config struct. New() calls Config.Validate() before constructing the service; returns an error if any required dependency is nil. (`func New(config Config) (creditgrant.Service, error) { if err := config.Validate(); err != nil { return nil, fmt.Errorf("invalid config: %w", err) } ... }`)
**input.Validate() as the first statement in every public method** — Every public method calls input.Validate() before any logic and wraps the error with fmt.Errorf before returning. (`func (s *service) Create(ctx context.Context, input creditgrant.CreateInput) (...) { if err := input.Validate(); err != nil { return ..., fmt.Errorf("invalid input: %w", err) } }`)
**Delegate to charges.Service — no Ent imports** — The service has no openmeter/ent/db import. All mutations go through chargesService.Create, GetByID, or HandleCreditPurchaseExternalPaymentStateTransition. (`result, err := s.chargesService.Create(ctx, charges.CreateInput{Namespace: input.Namespace, Intents: charges.ChargeIntents{charges.NewChargeIntent(intent)}})`)
**Ownership check before returning a charge** — Get-style methods verify cpCharge.Intent.CustomerID == input.CustomerID and return models.NewGenericNotFoundError on mismatch to prevent cross-tenant leaks. (`if cpCharge.Intent.CustomerID != input.CustomerID { return creditpurchase.Charge{}, fmt.Errorf("get charge: %w", models.NewGenericNotFoundError(...)) }`)
**ValidationIssue with HTTP status for business-rule rejections** — Business-rule violations (e.g. wrong settlement type) use models.NewValidationIssue with commonhttp.WithHTTPStatusCodeAttribute to carry an explicit HTTP status to the encoder. (`return ..., models.NewValidationIssue("credit_grant_external_settlement_not_supported", "credit grant is not externally funded", models.WithCriticalSeverity(), commonhttp.WithHTTPStatusCodeAttribute(http.StatusBadRequest))`)
**Private toIntent/toSettlement converters for CreateInput mapping** — Mapping from CreateInput to creditpurchase.Intent is isolated in package-private toIntent() and toSettlement() functions; keep them as pure field mappers with no business logic. (`intent := toIntent(input); result, err := s.chargesService.Create(ctx, charges.CreateInput{..., Intents: charges.ChargeIntents{charges.NewChargeIntent(intent)}})`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `service.go` | Single file implementing creditgrant.Service. Wraps charges.Service and creditpurchase.Service; performs customer existence check, ownership assertion, and settlement-type validation before delegating. | toIntent() hardcodes clock.Now() for ServicePeriod/BillingPeriod/FullServicePeriod with a TODO — real period derivation is not yet implemented. Do not rely on these values being meaningful. After Create(), the service does a second chargesService.GetByID call to fetch the realizations; this is a deliberate two-step fetch, not a bug. |

## Anti-Patterns

- Importing openmeter/ent/db or calling Ent builders directly — all DB access must go through charges.Service or creditpurchase.Service
- Skipping input.Validate() at the top of a new method — every public method must validate before acting
- Returning a charge without verifying CustomerID ownership — always assert cpCharge.Intent.CustomerID == input.CustomerID in Get-style methods
- Using context.Background() instead of the caller-supplied ctx
- Adding business logic inside toIntent()/toSettlement() beyond pure field mapping — keep them as dumb converters

## Decisions

- **Service delegates entirely to charges.Service rather than holding its own adapter** — Credit grants are a specialised view of creditpurchase.Charge; all lifecycle state (advance, realizations, ledger writes) is already managed by the charges pipeline, so duplicating persistence here would create divergent write paths.
- **Ownership check done in the service layer, not in charges.Service** — charges.Service is namespace-scoped but not customer-scoped; the creditgrant service is the API boundary that adds per-customer access control without polluting the generic charges layer.
- **charges.NewChargeIntent used instead of struct-literal ChargeIntent{}** — The Charge tagged-union has a private discriminator field set only by NewCharge[T]/NewChargeIntent; struct-literal construction leaves the discriminator zero-valued and all type-switch accessors return errors.

## Example: Creating a credit grant — full path from CreateInput to creditpurchase.Charge

```
import (
    "github.com/openmeterio/openmeter/openmeter/billing/charges"
    "github.com/openmeterio/openmeter/openmeter/billing/charges/creditpurchase"
    "github.com/openmeterio/openmeter/openmeter/billing/creditgrant"
    "github.com/openmeterio/openmeter/openmeter/customer"
)

func (s *service) Create(ctx context.Context, input creditgrant.CreateInput) (creditpurchase.Charge, error) {
    if err := input.Validate(); err != nil {
        return creditpurchase.Charge{}, fmt.Errorf("invalid input: %w", err)
    }
    _, err := s.customerService.GetCustomer(ctx, customer.GetCustomerInput{
        CustomerID: &customer.CustomerID{Namespace: input.Namespace, ID: input.CustomerID},
    })
    if err != nil {
// ...
```

<!-- archie:ai-end -->
