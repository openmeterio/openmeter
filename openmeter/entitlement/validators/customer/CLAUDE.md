# customer

<!-- archie:ai-start -->

> Implements customer.RequestValidator to block deletion of customers that still have active entitlements. Single-file bridge between the entitlement and customer domains that avoids circular imports by depending on entitlement.EntitlementRepo instead of entitlement.Service.

## Patterns

**Compile-time interface assertion** — Always declare var _ customer.RequestValidator = (*Validator)(nil) at package level so interface drift is caught at compile time. (`var _ customer.RequestValidator = (*Validator)(nil)`)
**Embed NoopRequestValidator for default methods** — Embed customer.NoopRequestValidator in the Validator struct to get safe no-op defaults for any RequestValidator methods not explicitly overridden, preventing breakage when the interface adds methods. (`type Validator struct { customer.NoopRequestValidator; entitlementRepo entitlement.EntitlementRepo }`)
**Constructor nil-guard returning error** — NewValidator returns (*Validator, error) and rejects nil dependencies with a descriptive error before constructing the struct. (`if entitlementRepo == nil { return nil, fmt.Errorf("entitlement repository is required") }`)
**GenericConflictError for blocked deletions** — Return models.NewGenericConflictError when a customer cannot be deleted due to active dependents; the HTTP GenericErrorEncoder maps this to 409. (`return models.NewGenericConflictError(fmt.Errorf("customer %s still has active entitlements", input.ID))`)
**clock.Now() for testable time** — Use clock.Now() (not time.Now()) when computing the current time for ActiveAt filtering so tests can inject a fake clock. (`now := clock.Now(); ActiveAt: lo.ToPtr(now), IncludeDeletedAfter: now`)
**Depend on EntitlementRepo, not entitlement.Service** — Use entitlement.EntitlementRepo (low-level repo interface) to avoid a heavy dependency on the full service graph and prevent import cycles with the customer domain. (`entitlementRepo entitlement.EntitlementRepo`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `validator.go` | Only file in the package. Defines Validator, its constructor, and the ValidateDeleteCustomer pre-mutation guard. Future RequestValidator methods (ValidateCreateCustomer, ValidateUpdateCustomer) would be added here. | Do not call entitlement.Service — only entitlement.EntitlementRepo. Always embed NoopRequestValidator. Return models.GenericConflictError (not GenericValidationError) for 'has active dependents'. |

## Anti-Patterns

- Importing entitlement.Service instead of entitlement.EntitlementRepo — creates import cycles and a heavier dependency graph.
- Returning a non-GenericConflictError for 'has active entitlements' — the encoder maps only GenericConflictError to 409.
- Using time.Now() instead of clock.Now() — makes tests non-deterministic.
- Adding state-transition or post-mutation logic here — this package is exclusively for pre-mutation blocking guards.
- Omitting the compile-time interface assertion — silent breakage if customer.RequestValidator gains methods.

## Decisions

- **Depend on entitlement.EntitlementRepo rather than entitlement.Service.** — entitlement.Service transitively imports customer, creating a circular dependency; the repo interface satisfies the validation need without pulling in the full service graph.
- **Embed customer.NoopRequestValidator instead of implementing all methods.** — Provides safe no-op defaults so adding a method to the interface does not silently break this validator until it needs a real implementation.

## Example: Add ValidateUpdateCustomer that blocks updates when active entitlements exist

```
import (
  "context"
  "fmt"
  "github.com/samber/lo"
  "github.com/openmeterio/openmeter/openmeter/customer"
  "github.com/openmeterio/openmeter/openmeter/entitlement"
  "github.com/openmeterio/openmeter/pkg/clock"
  "github.com/openmeterio/openmeter/pkg/models"
)

func (v *Validator) ValidateUpdateCustomer(ctx context.Context, input customer.UpdateCustomerInput) error {
  if err := input.Validate(); err != nil { return err }
  now := clock.Now()
  ents, err := v.entitlementRepo.ListActiveEntitlementsOfCustomer(ctx, input.Namespace, input.ID, now)
  if err != nil { return err }
// ...
```

<!-- archie:ai-end -->
