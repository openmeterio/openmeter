# subscription

<!-- archie:ai-start -->

> Single-file package implementing a models.ServiceHook[entitlement.Entitlement] that guards subscription-managed entitlements from direct mutation or deletion. It enforces that entitlements annotated with subscription metadata can only be modified through subscription operations, preventing bypassing of subscription lifecycle constraints.

## Patterns

**Embed NoopServiceHook, override only needed methods** — The hook struct embeds NoopEntitlementSubscriptionHook (alias for models.NoopServiceHook[entitlement.Entitlement]) so unimplemented hook methods are no-ops by default. Only PreDelete and PreUpdate are overridden. (`type hook struct { NoopEntitlementSubscriptionHook }`)
**subscription.IsSubscriptionOperation(ctx) gate** — Every guarded hook method must first check subscription.IsSubscriptionOperation(ctx) and return nil early if true. This allows the subscription package itself to mutate subscription-managed entitlements. (`if subscription.IsSubscriptionOperation(ctx) { return nil }`)
**subscription.AnnotationParser.HasSubscription annotation check** — After the context gate, check ent.Annotations with subscription.AnnotationParser.HasSubscription to detect subscription ownership. Return models.NewGenericForbiddenError if the annotation is present. (`if subscription.AnnotationParser.HasSubscription(ent.Annotations) { return models.NewGenericForbiddenError(fmt.Errorf("entitlement is managed by subscription")) }`)
**Interface compliance assertion at package level** — A blank-identifier var declaration asserts at compile time that *hook satisfies models.ServiceHook[entitlement.Entitlement]. (`var _ models.ServiceHook[entitlement.Entitlement] = (*hook)(nil)`)
**Constructor accepts typed config struct** — NewEntitlementSubscriptionHook accepts an EntitlementSubscriptionHookConfig (currently empty) so future config fields can be added without a signature change. (`func NewEntitlementSubscriptionHook(_ EntitlementSubscriptionHookConfig) EntitlementSubscriptionHook { ... }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `hook.go` | Entire package implementation: defines hook struct, type aliases for the hook and noop base, constructor, and PreDelete/PreUpdate guards. | Both PreDelete and PreUpdate must apply the same two-step gate (IsSubscriptionOperation then HasSubscription). Adding a new hook method that skips either step will silently allow direct mutation of subscription-managed entitlements. |

## Anti-Patterns

- Calling PreDelete or PreUpdate logic without first checking subscription.IsSubscriptionOperation(ctx) — breaks subscription self-management
- Removing the compile-time interface assertion var _ models.ServiceHook[entitlement.Entitlement] = (*hook)(nil)
- Returning a non-GenericForbiddenError for the subscription-annotation guard — callers expect 403 status mapping
- Adding business logic outside the annotation guard pattern (e.g. DB calls) in hook methods
- Implementing PostCreate/PostUpdate/PostDelete with side effects without verifying it won't create import cycles with subscription or entitlement packages

## Decisions

- **Hook lives in a separate sub-package (hooks/subscription) rather than inline in the entitlement service** — Avoids a circular import between openmeter/entitlement and openmeter/subscription; the hook depends on both but neither depends on it.
- **Use models.NoopServiceHook embed rather than implementing all ServiceHook methods manually** — ServiceHook has multiple lifecycle methods; embedding the noop ensures new methods added to the interface are automatically no-ops, preventing accidental breakage.

## Example: Adding a new hook method (e.g. PreUpdate guard) following the existing pattern

```
import (
	"context"
	"fmt"

	"github.com/openmeterio/openmeter/openmeter/entitlement"
	"github.com/openmeterio/openmeter/openmeter/subscription"
	"github.com/openmeterio/openmeter/pkg/models"
)

func (h *hook) PreUpdate(ctx context.Context, ent *entitlement.Entitlement) error {
	if subscription.IsSubscriptionOperation(ctx) {
		return nil
	}
	if subscription.AnnotationParser.HasSubscription(ent.Annotations) {
		return models.NewGenericForbiddenError(fmt.Errorf("entitlement is managed by subscription"))
// ...
```

<!-- archie:ai-end -->
