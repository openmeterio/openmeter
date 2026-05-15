# hooks

<!-- archie:ai-start -->

> Organisational container for entitlement ServiceHook implementations that enforce cross-domain constraints (e.g. subscription ownership) without creating circular imports. Each child package implements models.ServiceHook[entitlement.Entitlement] for a specific external domain concern.

## Patterns

**Embed NoopServiceHook, override only needed methods** — Hook structs embed models.NoopServiceHook[entitlement.Entitlement] (aliased as NoopEntitlementSubscriptionHook) so only PreDelete/PreUpdate methods that need logic are overridden. Compile-time assertion var _ models.ServiceHook[entitlement.Entitlement] = (*hook)(nil) is mandatory. (`type hook struct { NoopEntitlementSubscriptionHook }
var _ models.ServiceHook[entitlement.Entitlement] = (*hook)(nil)`)
**IsSubscriptionOperation gate must be checked first** — Any PreDelete/PreUpdate override must call subscription.IsSubscriptionOperation(ctx) before the annotation check to allow the subscription service to bypass the guard during its own lifecycle operations. (`if subscription.IsSubscriptionOperation(ctx) { return nil }
if subscription.AnnotationParser.HasSubscription(ent.Annotations) { return models.NewGenericForbiddenError(...) }`)
**Return GenericForbiddenError for annotation guards** — Blocked operations must return models.NewGenericForbiddenError — the HTTP error encoder maps this to 403. Using any other error type breaks status code mapping. (`return models.NewGenericForbiddenError(fmt.Errorf("entitlement is managed by subscription"))`)
**Constructor accepts typed config struct** — Constructors follow the pattern NewXxxHook(cfg XxxHookConfig) XxxHook — even if the config struct is empty — to allow future extension without breaking callers. (`func NewEntitlementSubscriptionHook(_ EntitlementSubscriptionHookConfig) EntitlementSubscriptionHook { ... }`)
**Type aliases for the hook interface and noop** — Each child package defines type aliases EntitlementXxxHook = models.ServiceHook[entitlement.Entitlement] and NoopEntitlementXxxHook = models.NoopServiceHook[entitlement.Entitlement] so callers reference a domain-named type. (`type EntitlementSubscriptionHook = models.ServiceHook[entitlement.Entitlement]
type NoopEntitlementSubscriptionHook = models.NoopServiceHook[entitlement.Entitlement]`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `openmeter/entitlement/hooks/subscription/hook.go` | Implements PreDelete/PreUpdate guard: blocks deletion/mutation of subscription-annotated entitlements unless called from within a subscription operation. Returns models.GenericForbiddenError (maps to 403). | Must check subscription.IsSubscriptionOperation(ctx) before subscription.AnnotationParser.HasSubscription. Returning any error type other than GenericForbiddenError breaks HTTP status mapping. No I/O or DB calls inside hook methods. |

## Anti-Patterns

- Implementing a new Pre* method without the IsSubscriptionOperation(ctx) check — breaks subscription self-management
- Returning a non-GenericForbiddenError for annotation guard failures — HTTP layer maps only ForbiddenError to 403
- Adding DB calls or business logic inside hook methods — hooks must be pure guards with no I/O
- Omitting the compile-time var _ models.ServiceHook[entitlement.Entitlement] = (*hook)(nil) assertion
- Importing entitlement.Service (rather than entitlement domain types) — risks import cycles

## Decisions

- **One sub-package per external domain concern (subscription/)** — Keeps each cross-domain constraint in its own compilation unit, preventing any single hook from accumulating unrelated concerns and making it trivial to wire/unwire individual hooks in app/common.
- **Embed models.NoopServiceHook rather than implementing all ServiceHook methods manually** — Reduces boilerplate and ensures the hook compiles correctly even when the ServiceHook interface gains new methods, as long as the noop provides default no-op implementations.

## Example: Adding a new cross-domain hook that guards entitlement mutation

```
package entitlementbillinghook

import (
	"context"
	"fmt"

	"github.com/openmeterio/openmeter/openmeter/entitlement"
	"github.com/openmeterio/openmeter/openmeter/subscription"
	"github.com/openmeterio/openmeter/pkg/models"
)

type (
	EntitlementBillingHook     = models.ServiceHook[entitlement.Entitlement]
	NoopEntitlementBillingHook = models.NoopServiceHook[entitlement.Entitlement]
)
// ...
```

<!-- archie:ai-end -->
