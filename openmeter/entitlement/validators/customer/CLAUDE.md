# customer

<!-- archie:ai-start -->

> Implements customer.RequestValidator to block deletion of customers that still have active entitlements. Single-file package that bridges the entitlement and customer domains without creating a circular import.

## Patterns

**RequestValidator interface compliance** — The Validator struct must satisfy customer.RequestValidator via a compile-time assertion `var _ customer.RequestValidator = (*Validator)(nil)`. Embed customer.NoopRequestValidator to get default no-op implementations for any methods not explicitly overridden. (`var _ customer.RequestValidator = (*Validator)(nil)
type Validator struct {
    customer.NoopRequestValidator
    entitlementRepo entitlement.EntitlementRepo
}`)
**Constructor nil-guard** — NewValidator returns (*Validator, error) and explicitly rejects a nil entitlementRepo dependency with a descriptive error. All constructors in this pattern must guard every required dependency. (`if entitlementRepo == nil {
    return nil, fmt.Errorf("entitlement repository is required")
}`)
**Conflict error for blocked deletions** — When validation fails because a resource still has dependents, wrap the error with models.NewGenericConflictError so the HTTP layer maps it to HTTP 409. (`return models.NewGenericConflictError(fmt.Errorf("customer %s still has active entitlements", input.ID))`)
**Use EntitlementRepo, not entitlement.Service** — This package depends on entitlement.EntitlementRepo (the low-level repository interface) rather than entitlement.Service to avoid a heavy dependency on the full service and prevent import cycles. (`entitlementRepo entitlement.EntitlementRepo`)
**Clock-aware active check** — Use clock.Now() (not time.Now()) when computing the current time for ActiveAt filtering so the package is testable with a fake clock. (`now := clock.Now()
ActiveAt: lo.ToPtr(now),
IncludeDeletedAfter: now,`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `validator.go` | Only file in the package. Defines Validator, its constructor, and the ValidateDeleteCustomer guard. All future validation methods for customer mutations (create, update) would be added here. | Do not call entitlement.Service here — use entitlement.EntitlementRepo to avoid import cycles. Always embed NoopRequestValidator so unimplemented methods have safe defaults. |

## Anti-Patterns

- Importing entitlement.Service instead of entitlement.EntitlementRepo — causes a heavy dependency and risks import cycles
- Returning a non-conflict error type for 'has active entitlements' — the HTTP layer maps only models.GenericConflictError to 409
- Using time.Now() instead of clock.Now() — makes tests non-deterministic
- Adding business logic beyond pre-mutation validation — this package is only for RequestValidator guards, not for state transitions
- Omitting the compile-time interface assertion — silent drift if customer.RequestValidator evolves

## Decisions

- **Depend on entitlement.EntitlementRepo rather than entitlement.Service** — The service layer would create a circular dependency (customer → entitlement → customer). The repo interface is a lower-level dependency that satisfies the validation need without pulling in the full service graph.
- **Embed customer.NoopRequestValidator** — Provides safe no-op defaults for all RequestValidator methods not yet implemented, so adding a new method to the interface does not break this validator until it needs a real implementation.

## Example: Adding a new ValidateUpdateCustomer guard that blocks updates when active entitlements exist

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
    if err := input.Validate(); err != nil {
        return err
    }
// ...
```

<!-- archie:ai-end -->
