# validators

<!-- archie:ai-start -->

> Organisational package grouping cross-domain billing guard validators (customer/, subscription/) that register pre/post-mutation hooks into customer.Service and subscription.Service, enforcing billing pre-conditions without creating import cycles between the billing, customer, and subscription packages.

## Patterns

**Registration at wiring time, not via cross-domain import** — Each validator is constructed in app/common and registered via customerService.RegisterRequestValidator() or subscriptionService.RegisterHook(); coupling lives in the DI layer only. (`// app/common: v := NewValidator(...); customerService.RegisterRequestValidator(v)`)
**Embed noop base structs** — Validators embed customer.NoopRequestValidator or subscription.NoOpSubscriptionCommandHook so they override only the lifecycle methods they need. (`type Validator struct { customer.NoopRequestValidator; billingService billing.Service }`)
**Compile-time interface assertion** — Each file asserts compliance with var _ Interface = (*Validator)(nil). (`var _ customer.RequestValidator = (*Validator)(nil)`)
**Nil-guard constructors returning error** — NewValidator checks each injected service for nil, catching misconfigured wiring at startup. (`if billingService == nil { return nil, fmt.Errorf("billing service is required") }`)
**models.NewGenericConflictError for billing setup failures** — subscription/validator.go wraps failures so the HTTP layer renders 409. (`return models.NewGenericConflictError(fmt.Errorf("invalid billing setup: %w", err))`)
**Sync-before-validate in customer deletion** — customer/customer.go calls SynchronizeSubscription before ListInvoices so pending charges are reflected before the gate check. (`v.syncService.SynchronizeSubscription(ctx, view, time.Now()) // must precede ListInvoices`)
**errors.Join for multi-entity validation** — customer/customer.go collects per-invoice errors and returns errors.Join so the caller sees all blocking invoices at once. (`return errors.Join(errs...)`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `openmeter/billing/validators/customer/customer.go` | Implements customer.RequestValidator; ValidateDeleteCustomer syncs subscriptions then blocks deletion when non-final standard invoices or active gathering invoices exist. | Skipping SynchronizeSubscription before ListInvoices causes false-negative gate checks; subscriptionService is intentionally nullable — check app/common wiring before adding a nil guard. |
| `openmeter/billing/validators/subscription/validator.go` | Implements subscription.SubscriptionCommandHook via AfterCreate/AfterUpdate; validates the customer's billing app supports tax, invoicing, and payment collection before a priced subscription activates. | hasBillableItems short-circuits free subscriptions — re-check for new ratecard types; capability slice is hardcoded to three app.CapabilityType values. |

## Anti-Patterns

- Adding ValidateCreateCustomer/ValidateUpdateCustomer without a billing-domain reason — this package only guards delete
- Calling billing.Adapter or Ent directly from either validator — all reads must go through billing.Service
- Returning a plain error instead of models.NewGenericConflictError for billing setup failures in the subscription validator
- Adding app-type-specific branching (e.g. 'if Stripe do X') — capability validation is intentionally app-agnostic
- Using context.Background() instead of propagating the incoming ctx

## Decisions

- **Validators live under billing/validators/ rather than inside customer/ or subscription/** — Billing has dependency edges to both customer and subscription; placing validators here avoids import cycles by keeping billing-domain logic in billing's own tree.
- **Subscription sync driven inside the customer deletion validator, not as a subscription-service pre-hook** — Ensures invoice state is current at deletion time while keeping the customer package free of billing concerns.
- **Post-create/post-update (AfterCreate/AfterUpdate) hooks for subscription validation rather than pre-create** — Billing setup validation checks state that only exists after persistence (override profile, app assignment); a pre-create check would race incomplete state.

## Example: Adding a new billing validator (embed noop, nil-guard constructor, compile-time assertion)

```
package myentity

import (
	"context"
	"fmt"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/myentity"
	"github.com/openmeterio/openmeter/pkg/models"
)

var _ myentity.RequestValidator = (*Validator)(nil)

type Validator struct {
	myentity.NoopRequestValidator
// ...
```

<!-- archie:ai-end -->
