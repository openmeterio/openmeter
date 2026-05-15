# validators

<!-- archie:ai-start -->

> Organisational container for customer.RequestValidator implementations that enforce entitlement-domain pre-mutation constraints on customer operations. Prevents illegal customer lifecycle transitions (e.g. deleting a customer with active entitlements) without creating circular imports.

## Patterns

**Embed customer.NoopRequestValidator** — Validator structs embed customer.NoopRequestValidator so only the relevant Validate* methods need to be overridden. Compile-time assertion var _ customer.RequestValidator = (*Validator)(nil) is mandatory. (`type Validator struct {
	customer.NoopRequestValidator
	entitlementRepo entitlement.EntitlementRepo
}
var _ customer.RequestValidator = (*Validator)(nil)`)
**Depend on EntitlementRepo, not entitlement.Service** — Use entitlement.EntitlementRepo for validation queries to avoid heavy dependency on the full service and to prevent import cycles. Constructor nil-guards the repo and returns an error if nil. (`func NewValidator(entitlementRepo entitlement.EntitlementRepo) (*Validator, error) {
	if entitlementRepo == nil {
		return nil, fmt.Errorf("entitlement repository is required")
	}
	return &Validator{entitlementRepo: entitlementRepo}, nil
}`)
**Return models.GenericConflictError for blocked operations** — Pre-mutation blocking failures must use models.NewGenericConflictError — the HTTP error encoder maps ConflictError to 409. Using GenericValidationError or plain errors produces incorrect status codes. (`return models.NewGenericConflictError(fmt.Errorf("customer %s still has active entitlements", input.ID))`)
**Use clock.Now() for testable time** — All time references inside validators must use pkg/clock.Now() rather than time.Now() to allow test clock injection and ensure deterministic test behaviour. (`now := clock.Now()
entitlements, err := v.entitlementRepo.ListEntitlements(ctx, entitlement.ListEntitlementsParams{ActiveAt: lo.ToPtr(now), ...})`)
**Validate input before querying** — Call input.Validate() at the start of each ValidateXxx method to catch malformed inputs before issuing any DB queries. (`if err := input.Validate(); err != nil { return err }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `openmeter/entitlement/validators/customer/validator.go` | ValidateDeleteCustomer: lists active entitlements for the customer at clock.Now() (including soft-deleted ones active at that instant), returns GenericConflictError if any exist. Depends on entitlement.EntitlementRepo, not entitlement.Service. | Must use clock.Now() not time.Now(). Depends on EntitlementRepo, not entitlement.Service. IncludeDeleted+IncludeDeletedAfter pair is required to catch entitlements soft-deleted after the check instant. Return GenericConflictError (not ValidationError) so HTTP maps to 409. |

## Anti-Patterns

- Importing entitlement.Service instead of entitlement.EntitlementRepo — creates import cycles and heavier dependency graph
- Using time.Now() instead of clock.Now() — breaks test determinism
- Returning non-GenericConflictError for 'has active entitlements' — HTTP layer maps only ConflictError to 409
- Adding state-mutating logic inside validator methods — validators are read-only pre-mutation guards only
- Omitting the compile-time interface assertion var _ customer.RequestValidator = (*Validator)(nil)

## Decisions

- **Depend on entitlement.EntitlementRepo rather than entitlement.Service** — RequestValidator is called from within the customer service; depending on entitlement.Service would risk import cycles since the service layer imports customer for hooks. The repo interface is a lighter, cycle-safe dependency.
- **One sub-package per external domain concern (customer/)** — Mirrors the hooks/ layout — each validator lives in its own compilation unit so it can be individually wired/unwired in app/common without touching other validators.

## Example: Correct ValidateDeleteCustomer implementation checking active entitlements

```
package customer

import (
	"context"
	"fmt"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/openmeter/entitlement"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/models"
)

var _ customer.RequestValidator = (*Validator)(nil)
// ...
```

<!-- archie:ai-end -->
