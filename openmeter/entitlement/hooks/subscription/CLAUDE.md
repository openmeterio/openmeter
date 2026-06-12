# subscription

<!-- archie:ai-start -->

> A single-file ServiceHook that guards entitlement.Service Update/Delete operations so that subscription-managed entitlements cannot be mutated or deleted outside of a subscription operation. Its sole responsibility is enforcing the invariant that subscription-owned entitlements are only changed via the subscription lifecycle.

## Patterns

**ServiceHook via embedded Noop base** — The hook struct embeds NoopEntitlementSubscriptionHook (alias of models.NoopServiceHook[entitlement.Entitlement]) and only overrides the lifecycle methods it cares about; unoverridden hook points default to no-op. (`type hook struct { NoopEntitlementSubscriptionHook }`)
**Compile-time interface assertion** — A var _ assertion guarantees *hook satisfies models.ServiceHook[entitlement.Entitlement] so wiring breaks at compile time if the contract drifts. (`var _ models.ServiceHook[entitlement.Entitlement] = (*hook)(nil)`)
**Subscription-operation bypass guard** — Every guarded method first calls subscription.IsSubscriptionOperation(ctx) and returns nil early when true, allowing the subscription engine itself to mutate the entitlement. (`if subscription.IsSubscriptionOperation(ctx) { return nil }`)
**Annotation-based ownership check** — Ownership is detected via subscription.AnnotationParser.HasSubscription(ent.Annotations); when true and not a subscription operation, return a forbidden error rather than mutating state. (`if subscription.AnnotationParser.HasSubscription(ent.Annotations) { return models.NewGenericForbiddenError(...) }`)
**Config-injected constructor returning the interface type** — NewEntitlementSubscriptionHook takes an EntitlementSubscriptionHookConfig (currently empty struct) and returns the EntitlementSubscriptionHook interface alias, not the concrete *hook. (`func NewEntitlementSubscriptionHook(_ EntitlementSubscriptionHookConfig) EntitlementSubscriptionHook`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `hook.go` | Defines the package entitlementsubscriptionhook: the hook struct, the EntitlementSubscriptionHook/NoopEntitlementSubscriptionHook type aliases, the constructor, and the PreDelete/PreUpdate guard implementations. | PreDelete and PreUpdate share identical logic; keep them in sync if you change the guard. The leading comment warns that entitlement.Service methods are not conventionally named, so verify the actual call site before assuming when these hooks fire. EntitlementSubscriptionHookConfig is intentionally empty - do not add fields unless a real dependency is needed. |

## Anti-Patterns

- Mutating or deleting a subscription-owned entitlement without first checking subscription.IsSubscriptionOperation(ctx) - it bypasses the ownership guard.
- Returning the concrete *hook from the constructor instead of the EntitlementSubscriptionHook interface alias.
- Replacing the embedded NoopServiceHook with a struct that implements all hook methods manually, defeating the no-op default for unhandled lifecycle points.
- Using a non-forbidden error type for the ownership violation instead of models.NewGenericForbiddenError.
- Diverging PreUpdate and PreDelete logic so that one path allows mutation the other blocks.

## Decisions

- **Enforce subscription ownership through a ServiceHook rather than inside entitlement.Service itself.** — Keeps the entitlement service unaware of subscription concerns; the cross-domain invariant is injected as a hook so the dependency points from subscription guard into entitlement, not the reverse.
- **Detect a subscription operation via context (IsSubscriptionOperation) plus annotation ownership rather than a DB flag.** — The subscription engine marks its own operations on the context, so the guard can distinguish legitimate subscription-driven mutations from external API calls without extra storage or lookups.

## Example: Blocking external mutation of a subscription-managed entitlement

```
func (h *hook) PreUpdate(ctx context.Context, ent *entitlement.Entitlement) error {
	if subscription.IsSubscriptionOperation(ctx) {
		return nil
	}
	if subscription.AnnotationParser.HasSubscription(ent.Annotations) {
		return models.NewGenericForbiddenError(fmt.Errorf("entitlement is managed by subscription"))
	}
	return nil
}
```

<!-- archie:ai-end -->
