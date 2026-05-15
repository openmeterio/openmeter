# subscription

<!-- archie:ai-start -->

> Single-file package implementing a models.ServiceHook[entitlement.Entitlement] that guards subscription-managed entitlements from direct mutation or deletion. Prevents bypassing subscription lifecycle constraints by blocking PreDelete and PreUpdate on entitlements annotated with subscription metadata.

## Patterns

**Embed NoopServiceHook, override only needed methods** — The hook struct embeds NoopEntitlementSubscriptionHook (alias for models.NoopServiceHook[entitlement.Entitlement]) so unimplemented lifecycle methods are no-ops by default. Only PreDelete and PreUpdate are overridden. (`type hook struct { NoopEntitlementSubscriptionHook }`)
**Two-step gate: IsSubscriptionOperation then HasSubscription** — Every guarded hook method must first check subscription.IsSubscriptionOperation(ctx) and return nil if true (allows subscription package self-management), then check subscription.AnnotationParser.HasSubscription(ent.Annotations) and return GenericForbiddenError if annotated. (`if subscription.IsSubscriptionOperation(ctx) { return nil }
if subscription.AnnotationParser.HasSubscription(ent.Annotations) { return models.NewGenericForbiddenError(fmt.Errorf("entitlement is managed by subscription")) }`)
**Compile-time interface assertion** — A package-level blank-identifier var asserts *hook satisfies models.ServiceHook[entitlement.Entitlement] at compile time. (`var _ models.ServiceHook[entitlement.Entitlement] = (*hook)(nil)`)
**Constructor accepts typed config struct** — NewEntitlementSubscriptionHook accepts EntitlementSubscriptionHookConfig (currently empty) so future config fields can be added without a signature change. (`func NewEntitlementSubscriptionHook(_ EntitlementSubscriptionHookConfig) EntitlementSubscriptionHook { return &hook{} }`)
**Return GenericForbiddenError (not other error types) for annotation guard** — The annotation guard must return models.NewGenericForbiddenError so the HTTP error encoder maps it to 403 Forbidden via GenericErrorEncoder. (`return models.NewGenericForbiddenError(fmt.Errorf("entitlement is managed by subscription"))`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `hook.go` | Entire package implementation: defines hook struct, type aliases for the hook and noop base, constructor, and PreDelete/PreUpdate guards. | Both PreDelete and PreUpdate must apply the identical two-step gate (IsSubscriptionOperation then HasSubscription). Adding a new hook method that skips either step silently allows direct mutation of subscription-managed entitlements. |

## Anti-Patterns

- Implementing a new hook method (e.g. PreCreate) without the IsSubscriptionOperation check — breaks subscription self-management
- Returning any error type other than GenericForbiddenError for the annotation guard — callers expect 403 HTTP status mapping
- Adding DB calls or business logic inside hook methods — hooks must be pure guards with no I/O
- Removing the compile-time interface assertion var _ models.ServiceHook[entitlement.Entitlement] = (*hook)(nil)
- Implementing PostCreate/PostUpdate/PostDelete with side effects that import entitlement or subscription packages — risks import cycles

## Decisions

- **Package lives in hooks/subscription sub-package rather than inline in the entitlement service** — Avoids a circular import between openmeter/entitlement and openmeter/subscription; the hook depends on both but neither depends on it.
- **Embed models.NoopServiceHook rather than implementing all ServiceHook methods manually** — ServiceHook has multiple lifecycle methods; embedding the noop ensures new interface methods are automatically no-ops, preventing accidental breakage when the interface evolves.

## Example: Adding a new guarded hook method following the established two-step gate pattern

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
