# hooks

<!-- archie:ai-start -->

> Organisational container for entitlement ServiceHook implementations that enforce cross-domain constraints without creating circular imports. Each child package implements models.ServiceHook[entitlement.Entitlement] for a specific external domain concern.

## Patterns

**Embed NoopServiceHook, override only needed methods** — Hook structs embed models.NoopServiceHook[entitlement.Entitlement] so only PreDelete/PreUpdate methods that need logic are overridden. Compile-time assertion var _ models.ServiceHook[entitlement.Entitlement] = (*hook)(nil) is mandatory. (`type hook struct { models.NoopServiceHook[entitlement.Entitlement] }`)
**Context-based operation gate** — subscription.IsSubscriptionOperation(ctx) must be checked first in any PreDelete/PreUpdate override to allow the subscription service itself to bypass the guard during its own lifecycle operations. (`if subscription.IsSubscriptionOperation(ctx) { return nil }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `openmeter/entitlement/hooks/subscription/hook.go` | Implements PreDelete/PreUpdate guard: blocks deletion/mutation of subscription-annotated entitlements unless called from within a subscription operation. Returns models.GenericForbiddenError (maps to 403). | Must check subscription.IsSubscriptionOperation(ctx) before checking annotation presence. Returning any error type other than GenericForbiddenError breaks HTTP status mapping. |

## Anti-Patterns

- Adding hooks that import entitlement.Service (rather than the entitlement domain types) — risks import cycles
- Omitting the compile-time var _ models.ServiceHook[entitlement.Entitlement] assertion
- Adding PostCreate/PostUpdate with DB side-effects without verifying no import cycle with the entitlement adapter package

## Decisions

- **One sub-package per external domain concern (subscription/)** — Keeps each cross-domain constraint in its own compilation unit, preventing any single hook from accumulating unrelated concerns and making it easy to wire/unwire individual hooks in app/common

<!-- archie:ai-end -->
