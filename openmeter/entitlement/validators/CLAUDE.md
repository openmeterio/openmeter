# validators

<!-- archie:ai-start -->

> Organisational container for customer.RequestValidator implementations that enforce entitlement-domain pre-mutation constraints on customer operations (e.g. blocking deletion of a customer with active entitlements) without circular imports.

## Patterns

**Embed customer.NoopRequestValidator** — Validator structs embed customer.NoopRequestValidator so only the relevant Validate* methods are overridden; a compile-time var _ customer.RequestValidator = (*Validator)(nil) is mandatory. (`type Validator struct { customer.NoopRequestValidator; entitlementRepo entitlement.EntitlementRepo }
var _ customer.RequestValidator = (*Validator)(nil)`)
**Depend on EntitlementRepo, not entitlement.Service** — Validators use entitlement.EntitlementRepo (lighter, cycle-safe) and nil-guard it in the constructor, returning an error if nil. (`func NewValidator(entitlementRepo entitlement.EntitlementRepo) (*Validator, error) { if entitlementRepo == nil { return nil, fmt.Errorf("entitlement repository is required") }; ... }`)
**Return GenericConflictError for blocked operations** — Pre-mutation blocking failures use models.NewGenericConflictError so the HTTP encoder maps to 409. (`return models.NewGenericConflictError(fmt.Errorf("customer %s still has active entitlements", input.ID))`)
**Use clock.Now() for testable time** — All time references use pkg/clock.Now() rather than time.Now() to allow test clock injection. (`now := clock.Now(); entitlements, err := v.entitlementRepo.ListEntitlements(ctx, entitlement.ListEntitlementsParams{ActiveAt: lo.ToPtr(now), ...})`)
**Validate input before querying** — Each ValidateXxx calls input.Validate() first to reject malformed input before any DB query. (`if err := input.Validate(); err != nil { return err }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `customer/validator.go` | ValidateDeleteCustomer lists active entitlements at clock.Now() (including soft-deleted ones active at that instant) and returns GenericConflictError if any exist; depends on entitlement.EntitlementRepo. | Must use clock.Now() not time.Now(). The IncludeDeleted+IncludeDeletedAfter pair is required to catch entitlements soft-deleted after the check. Return GenericConflictError so HTTP maps to 409. |

## Anti-Patterns

- Importing entitlement.Service instead of entitlement.EntitlementRepo — creates import cycles and a heavier dependency graph
- Using time.Now() instead of clock.Now() — breaks test determinism
- Returning a non-GenericConflictError for 'has active entitlements' — the HTTP layer maps only ConflictError to 409
- Adding state-mutating or post-mutation logic — validators are read-only pre-mutation guards only
- Omitting the compile-time var _ customer.RequestValidator = (*Validator)(nil) assertion

## Decisions

- **Depend on entitlement.EntitlementRepo rather than entitlement.Service** — RequestValidator is called from within the customer service; depending on entitlement.Service would risk import cycles since the service layer imports customer for hooks.
- **One sub-package per external-domain concern (customer/)** — Mirrors the hooks/ layout so each validator can be independently wired/unwired in app/common.

<!-- archie:ai-end -->
