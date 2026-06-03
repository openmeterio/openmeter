# hooks

<!-- archie:ai-start -->

> Organisational container for entitlement ServiceHook implementations that enforce cross-domain constraints (currently subscription ownership) without creating circular imports. Each child package implements models.ServiceHook[entitlement.Entitlement] for one external-domain concern.

## Patterns

**Embed NoopServiceHook, override only needed methods** — Hook structs embed models.NoopServiceHook[entitlement.Entitlement] so only PreDelete/PreUpdate are overridden; a compile-time var _ models.ServiceHook[entitlement.Entitlement] = (*hook)(nil) is mandatory. (`type hook struct { NoopEntitlementSubscriptionHook }
var _ models.ServiceHook[entitlement.Entitlement] = (*hook)(nil)`)
**IsSubscriptionOperation gate first** — Any Pre* override calls subscription.IsSubscriptionOperation(ctx) before the annotation check so the subscription service can bypass the guard during its own lifecycle. (`if subscription.IsSubscriptionOperation(ctx) { return nil }; if subscription.AnnotationParser.HasSubscription(ent.Annotations) { return models.NewGenericForbiddenError(...) }`)
**Return GenericForbiddenError for annotation guards** — Blocked operations return models.NewGenericForbiddenError so the HTTP encoder maps to 403; any other error type breaks status mapping. (`return models.NewGenericForbiddenError(fmt.Errorf("entitlement is managed by subscription"))`)
**Constructor accepts a typed config struct** — Constructors follow NewXxxHook(cfg XxxHookConfig) XxxHook even when the config is empty, allowing future extension without breaking callers. (`func NewEntitlementSubscriptionHook(_ EntitlementSubscriptionHookConfig) EntitlementSubscriptionHook { ... }`)
**Type aliases for hook interface and noop** — Each child defines EntitlementXxxHook = models.ServiceHook[entitlement.Entitlement] and NoopEntitlementXxxHook = models.NoopServiceHook[entitlement.Entitlement] so callers use a domain-named type. (`type EntitlementSubscriptionHook = models.ServiceHook[entitlement.Entitlement]`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `subscription/hook.go` | PreDelete/PreUpdate guard blocking mutation/deletion of subscription-annotated entitlements unless called from within a subscription operation; returns GenericForbiddenError (403). | Must check subscription.IsSubscriptionOperation(ctx) before HasSubscription. No I/O or DB calls inside hook methods. |

## Anti-Patterns

- Implementing a new Pre* method without the IsSubscriptionOperation(ctx) check — breaks subscription self-management
- Returning a non-GenericForbiddenError for annotation-guard failures — the HTTP layer maps only ForbiddenError to 403
- Adding DB calls or business logic inside hook methods — hooks must be pure guards with no I/O
- Omitting the compile-time var _ models.ServiceHook[entitlement.Entitlement] = (*hook)(nil) assertion
- Importing entitlement.Service (rather than entitlement domain types) — risks import cycles

## Decisions

- **One sub-package per external-domain concern (subscription/)** — Keeps each cross-domain constraint in its own compilation unit, making it trivial to wire/unwire individual hooks in app/common.
- **Embed models.NoopServiceHook rather than implementing all methods** — Reduces boilerplate and keeps hooks compiling when the ServiceHook interface gains methods.

<!-- archie:ai-end -->
