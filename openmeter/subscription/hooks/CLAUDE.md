# hooks

<!-- archie:ai-start -->

> Organisational folder grouping SubscriptionCommandHook implementations that maintain subscription invariants across lifecycle events. Currently holds hooks/annotations, which repairs the doubly-linked chain annotations (previous/superseding subscription IDs) on deletion. Hooks here must never import subscription.Service (the write path) to avoid re-entrant hook invocations.

## Patterns

**Embed NoOpSubscriptionCommandHook** — Every hook embeds subscription.NoOpSubscriptionCommandHook so only the overridden methods (e.g. BeforeDelete) need implementing; interface additions don't break existing hooks. (`type AnnotationCleanupHook struct { subscription.NoOpSubscriptionCommandHook; ... }`)
**Write via SubscriptionRepository, not Service** — Hooks call subscription.SubscriptionRepository.UpdateAnnotations directly rather than subscription.Service to avoid re-entrant hook calls. (`h.subRepo.UpdateAnnotations(ctx, id, annotations)`)
**Soft-skip NotFound, hard-error otherwise** — A not-found linked subscription during cleanup is an expected concurrent-delete race — log and return nil; any other error is a hard failure. (`if subscription.IsSubscriptionNotFoundError(err) { logger.Warn(...); return nil }`)
**Clone-then-nil annotation maps** — Clone the annotations map before mutating, and nil out the map once it becomes empty rather than storing a len==0 map. (`annotations := maps.Clone(view.Subscription.Annotations); if len(annotations) == 0 { annotations = nil }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `annotations/hook.go` | Implements BeforeDelete to clean up previous/superseding subscription ID annotations on chain-linked subscriptions. | Clone annotations before mutating; nil-out empty maps after key removal to avoid storing empty annotation maps. |

## Anti-Patterns

- Calling subscription.Service write methods from inside a hook — creates re-entrant hook invocations.
- Mutating annotation maps in-place without cloning first.
- Not embedding NoOpSubscriptionCommandHook — forces explicit implementation of all hook methods.
- Returning an error on subscription.IsSubscriptionNotFoundError — expected concurrent-delete races must be logged and skipped.
- Storing an empty (len==0) annotations map instead of nil.

## Decisions

- **Hooks placed in a dedicated sub-package rather than inline in subscription/service.** — Prevents circular imports between the service and cross-cutting lifecycle concerns like annotation chain repair.

<!-- archie:ai-end -->
