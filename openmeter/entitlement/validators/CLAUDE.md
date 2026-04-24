# validators

<!-- archie:ai-start -->

> Organisational container for customer.RequestValidator implementations that enforce entitlement-domain pre-mutation constraints on customer operations. Each child package prevents illegal customer lifecycle transitions (e.g. deleting a customer with active entitlements) without creating circular imports.

## Patterns

**Embed customer.NoopRequestValidator** — Validator structs embed customer.NoopRequestValidator so only ValidateDeleteCustomer (or other relevant methods) need to be overridden. Compile-time assertion is mandatory. (`type validator struct { customer.NoopRequestValidator }`)
**Depend on EntitlementRepo, not entitlement.Service** — Use entitlement.EntitlementRepo for validation queries to avoid heavy dependency on the full service and to prevent import cycles. (`r.entitlementRepo.ListEntitlements(ctx, entitlement.ListEntitlementsParams{CustomerIDs: []string{id}, ActiveAt: lo.ToPtr(clock.Now())})`)
**Return models.GenericConflictError for blocked operations** — The HTTP layer maps only GenericConflictError to 409. Validation failures must use this error type. (`return models.NewGenericConflictError(fmt.Errorf("customer has active entitlements"))`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `openmeter/entitlement/validators/customer/validator.go` | ValidateDeleteCustomer: lists active entitlements for the customer at clock.Now(), returns GenericConflictError if any exist. | Must use clock.Now() not time.Now() for test determinism. Depends on EntitlementRepo, not entitlement.Service. |

## Anti-Patterns

- Importing entitlement.Service instead of entitlement.EntitlementRepo — causes heavy dependency and risks import cycles
- Using time.Now() instead of clock.Now() — breaks test determinism
- Returning non-GenericConflictError for 'has active entitlements' — HTTP layer maps only ConflictError to 409
- Adding state-mutating logic inside validator methods — validators are read-only guards, not service operations

## Decisions

- **Depend on EntitlementRepo rather than entitlement.Service** — RequestValidator is called from within the customer service; depending on entitlement.Service would risk import cycles since the service layer imports customer for hooks

<!-- archie:ai-end -->
