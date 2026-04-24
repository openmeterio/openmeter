# service

<!-- archie:ai-start -->

> Implements the creditgrant.Service interface — a facade that orchestrates credit grant creation, retrieval, listing, and external settlement transitions by delegating to charges.Service and creditpurchase.Service. It owns no direct DB access; all persistence flows through charges.Service.

## Patterns

**Config struct with Validate() + New()** — All dependencies are declared in a Config struct. New() calls Config.Validate() before constructing the service, returning an error if any required dependency is nil. (`func New(config Config) (creditgrant.Service, error) { if err := config.Validate(); err != nil { return nil, fmt.Errorf("invalid config: %w", err) } ... }`)
**Input.Validate() called at top of every method** — Every public service method calls input.Validate() as its first statement and wraps the error with fmt.Errorf before returning. (`func (s *service) Create(ctx context.Context, input creditgrant.CreateInput) (...) { if err := input.Validate(); err != nil { return ..., fmt.Errorf("invalid input: %w", err) } }`)
**Delegate to charges.Service, never touch Ent directly** — The service layer has no entdb import and performs no Ent queries. All data mutations go through chargesService.Create / GetByID / HandleCreditPurchaseExternalPaymentStateTransition. (`result, err := s.chargesService.Create(ctx, charges.CreateInput{Namespace: input.Namespace, Intents: charges.ChargeIntents{charges.NewChargeIntent(intent)}})`)
**Ownership check before returning a charge** — Get() verifies cpCharge.Intent.CustomerID == input.CustomerID and returns models.NewGenericNotFoundError if there is a mismatch, preventing cross-tenant data leaks. (`if cpCharge.Intent.CustomerID != input.CustomerID { return ..., models.NewGenericNotFoundError(...) }`)
**ValidationIssue for business-rule rejections** — Business-rule violations (e.g. wrong settlement type) use models.NewValidationIssue with commonhttp.WithHTTPStatusCodeAttribute to carry an explicit HTTP status code to the encoder. (`return ..., models.NewValidationIssue("credit_grant_external_settlement_not_supported", "credit grant is not externally funded", models.WithCriticalSeverity(), commonhttp.WithHTTPStatusCodeAttribute(http.StatusBadRequest), ...)`)
**toIntent / toSettlement conversion helpers are private** — Mapping from CreateInput to creditpurchase.Intent is isolated in package-private toIntent() and toSettlement() functions, keeping the service methods clean. (`intent := toIntent(input); result, err := s.chargesService.Create(ctx, charges.CreateInput{..., Intents: charges.ChargeIntents{charges.NewChargeIntent(intent)}})`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `service.go` | Single file implementing creditgrant.Service. Wraps charges.Service and creditpurchase.Service; performs customer existence check, ownership assertion, and settlement-type validation before delegating. | toIntent() hardcodes clock.Now() for ServicePeriod/BillingPeriod/FullServicePeriod with a TODO — real period derivation is not yet implemented. Do not rely on these values being meaningful. |

## Anti-Patterns

- Importing openmeter/ent/db or calling Ent builders directly — all DB access must go through charges.Service or creditpurchase.Service
- Skipping input.Validate() at the top of a new method — every public method must validate before acting
- Returning a charge without verifying CustomerID ownership — always assert cpCharge.Intent.CustomerID == input.CustomerID in Get-style methods
- Using context.Background() instead of the caller-supplied ctx
- Adding business logic inside toIntent()/toSettlement() beyond pure field mapping — keep them as dumb converters

## Decisions

- **Service delegates entirely to charges.Service rather than holding its own adapter** — Credit grants are a specialised view of creditpurchase.Charge; all lifecycle state (advance, realizations, ledger writes) is already managed by the charges pipeline, so duplicating persistence here would create divergent write paths.
- **Ownership check done in the service layer (not in charges.Service)** — charges.Service is namespace-scoped but not customer-scoped; the creditgrant service is the API boundary that adds per-customer access control without polluting the generic charges layer.

## Example: Creating a credit grant — full path from CreateInput to creditpurchase.Charge

```
import (
    "github.com/openmeterio/openmeter/openmeter/billing/charges"
    "github.com/openmeterio/openmeter/openmeter/billing/charges/meta"
    "github.com/openmeterio/openmeter/openmeter/billing/creditgrant"
)

func (s *service) Create(ctx context.Context, input creditgrant.CreateInput) (creditpurchase.Charge, error) {
    if err := input.Validate(); err != nil {
        return creditpurchase.Charge{}, fmt.Errorf("invalid input: %w", err)
    }
    _, err := s.customerService.GetCustomer(ctx, customer.GetCustomerInput{CustomerID: &customer.CustomerID{Namespace: input.Namespace, ID: input.CustomerID}})
    if err != nil {
        return creditpurchase.Charge{}, fmt.Errorf("get customer: %w", err)
    }
    intent := toIntent(input)
// ...
```

<!-- archie:ai-end -->
