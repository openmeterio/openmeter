# hooks

<!-- archie:ai-start -->

> Organisational folder grouping SubscriptionCommandHook implementations that maintain subscription invariants across lifecycle events. Currently contains hooks/annotations which repairs doubly-linked chain annotations on subscription deletion. Hooks here must never import subscription.Service (write path) to avoid re-entrant hook calls.

## Patterns

**Embed NoOpSubscriptionCommandHook** — All hook implementations embed subscription.NoOpSubscriptionCommandHook so only the overridden methods (e.g. BeforeDelete) need to be implemented. This prevents future interface changes from breaking existing hooks. (`type AnnotationCleanupHook struct { subscription.NoOpSubscriptionCommandHook; ... }`)
**Use SubscriptionRepository for writes, not Service** — Hook implementations call subscription.SubscriptionRepository.UpdateAnnotations directly rather than subscription.Service to avoid re-entrant hook invocations. (`h.subRepo.UpdateAnnotations(ctx, id, annotations)`)
**Soft-skip on NotFound, hard-error on others** — When a linked subscription is not found during annotation cleanup, log and skip — it may have been deleted in a race. Any other error is returned as a hard failure. (`if subscription.IsSubscriptionNotFoundError(err) { logger.Warn(...); return nil }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `hooks/annotations/hook.go` | Implements BeforeDelete to clean up previous/superseding subscription ID annotations on chain-linked subscriptions. | Must clone annotations before mutating; must nil-out empty maps after key removal. |

## Anti-Patterns

- Calling subscription.Service write methods from inside a hook — creates re-entrant hook invocations.
- Mutating annotation maps in-place without cloning first.
- Not embedding NoOpSubscriptionCommandHook — forces explicit implementation of all hook methods.
- Returning an error on subscription.IsSubscriptionNotFoundError — these are expected concurrent-delete races.

## Decisions

- **Hooks placed in a dedicated sub-package rather than inline in subscription/service** — Prevents circular imports between the service and cross-cutting lifecycle concerns like annotation chain repair.

<!-- archie:ai-end -->
