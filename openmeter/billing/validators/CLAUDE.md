# validators

<!-- archie:ai-start -->

> Organisational package grouping cross-domain billing guard validators. Its two children register pre/post-mutation hooks into customer.Service and subscription.Service respectively, enforcing billing pre-conditions without introducing circular imports between the billing, customer, and subscription packages.

## Patterns

**Registration at wiring time, not compile time** — Each validator is constructed in app/common and registered via customerService.RegisterRequestValidator() or subscriptionService.RegisterHook(). Neither child imports the other domain's package at the call site — the coupling is in the DI layer only. (`app/common wires NewValidator(...) then calls customerService.RegisterRequestValidator(v) or subscriptionService.RegisterHook(v)`)
**Embed noop base structs** — Validators embed customer.NoopRequestValidator or subscription.NoOpSubscriptionCommandHook so they only override the specific lifecycle methods they need. Unused hook methods are left as noops. (`type Validator struct { customer.NoopRequestValidator; ... } or type Validator struct { subscription.NoOpSubscriptionCommandHook; ... }`)
**Compile-time interface assertion** — customer/customer.go uses `var _ customer.RequestValidator = (*Validator)(nil)` to assert the interface at compile time. Add this to every new validator. (`var _ customer.RequestValidator = (*Validator)(nil)`)
**Nil-guard constructors returning error** — Both NewValidator functions check each injected service for nil and return an error. This catches misconfigured wiring at startup. (`if billingService == nil { return nil, fmt.Errorf("billing service is required") }`)
**models.NewGenericConflictError for billing setup failures** — subscription/validator.go wraps validation errors in models.NewGenericConflictError so the HTTP layer renders a 409. Use this — not a plain error — for billing pre-condition failures. (`return models.NewGenericConflictError(fmt.Errorf("invalid billing setup: %w", err))`)
**Sync-before-validate in customer deletion** — customer/customer.go calls syncService.SynchronizeSubscription before listing invoices so pending subscription charges are reflected in the invoice state before the gate check runs. (`v.syncService.SynchronizeSubscription(ctx, view, time.Now()) // must precede ListInvoices`)
**errors.Join for multi-entity validation errors** — customer/customer.go collects per-invoice errors into a slice and returns errors.Join(errs...) so the caller sees all blocking invoices at once, not just the first. (`errs = append(errs, fmt.Errorf("invoice %s is not in final state...", id)); return errors.Join(errs...)`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `openmeter/billing/validators/customer/customer.go` | Implements customer.RequestValidator. ValidateDeleteCustomer syncs pending subscriptions then blocks deletion when non-final standard invoices or active gathering invoices exist. | Skipping the SynchronizeSubscription call before ListInvoices will cause false-negative gate checks; the subscriptionService field is intentionally nullable (no nil guard) — do not add nil guard without checking app/common wiring. |
| `openmeter/billing/validators/subscription/validator.go` | Implements subscription.SubscriptionCommandHook via AfterCreate/AfterUpdate. Validates that the customer's installed billing app supports tax calculation, invoicing, and payment collection before a subscription with priced ratecards becomes active. | hasBillableItems short-circuits validation for free subscriptions — ensure this exit is still correct for any new ratecard types; capability slice is hardcoded to three app.CapabilityType values, not sourced from the app. |

## Anti-Patterns

- Adding ValidateCreateCustomer or ValidateUpdateCustomer in the customer validator without a billing-domain reason — general lifecycle belongs in the customer package.
- Calling billing.Adapter or Ent directly from either validator — all reads must go through billing.Service.
- Returning a plain error instead of models.NewGenericConflictError for billing setup failures in the subscription validator — the HTTP layer maps GenericConflictError to 409.
- Adding app-type-specific branching (e.g., 'if Stripe do X') in the subscription validator — capability validation is intentionally app-agnostic.
- Using context.Background() instead of propagating the incoming ctx parameter.

## Decisions

- **Validators live under billing/validators/ rather than inside customer/ or subscription/** — Billing has dependency edges to both customer and subscription; placing validators here avoids import cycles by keeping the billing-domain logic in billing's own package tree.
- **Subscription sync is driven inside the customer deletion validator rather than as a pre-hook in the subscription service** — The sync ensures invoice state is up-to-date at the moment of deletion validation. Doing it inside the billing validator keeps the customer package free of billing concerns.
- **Post-create/post-update (AfterCreate/AfterUpdate) hooks for subscription validation rather than pre-create** — Billing setup validation checks state that only exists after the subscription is persisted (e.g. customer override profile, app assignment); a pre-create check would race against incomplete state.

## Example: Add a new billing validator for a domain entity (pattern: embed noop, nil-guard constructor, compile-time assertion, wrap errors in GenericConflictError)

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
