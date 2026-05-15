# annotations

<!-- archie:ai-start -->

> Implements a SubscriptionCommandHook that repairs annotation-based chain links (previous/superseding subscription IDs) when a subscription is deleted, maintaining doubly-linked list integrity across the subscription chain without orphaned pointers.

## Patterns

**Embed NoOpSubscriptionCommandHook** — Every hook struct must embed subscription.NoOpSubscriptionCommandHook so only overridden methods need implementation; unimplemented methods compile as no-ops. (`type AnnotationCleanupHook struct { subscription.NoOpSubscriptionCommandHook; subscriptionQueryService subscription.QueryService; ... }`)
**Constructor validates all deps** — NewAnnotationCleanupHook returns (*T, error) and nil-checks every injected dependency before constructing the struct. (`if subscriptionQueryService == nil { return nil, fmt.Errorf("subscription query service is required") }`)
**Use AnnotationParser for all annotation access** — All reads and writes of annotation keys must go through subscription.AnnotationParser methods — never read/write annotation map keys by string literal directly. (`supersedingID := subscription.AnnotationParser.GetSupersedingSubscriptionID(view.Subscription.Annotations)`)
**Clone before mutating annotations** — Always call maps.Clone on the annotation map before any modification to avoid mutating the original view's annotations in-place. (`supersedingAnnotations = maps.Clone(supersedingAnnotations)`)
**Soft-skip NotFound, hard-error on all others** — When fetching a linked subscription, check subscription.IsSubscriptionNotFoundError; log-and-continue on not-found, return error for all other failures. (`if subscription.IsSubscriptionNotFoundError(err) { h.logger.Error("..."); return nil }`)
**Nil-out empty annotation maps after deletion** — After deleting keys, if len(annotations) == 0 set the map to nil before persisting to avoid storing empty maps. (`if len(supersedingAnnotations) == 0 { supersedingAnnotations = nil }`)
**Persist via SubscriptionRepository.UpdateAnnotations** — All annotation writes use subscriptionRepo.UpdateAnnotations(ctx, namespacedID, annotations) — never route through the full Service to avoid re-triggering hooks. (`h.subscriptionRepo.UpdateAnnotations(ctx, supersedingView.Subscription.NamespacedID, supersedingAnnotations)`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `hook.go` | Sole file; defines AnnotationCleanupHook with BeforeDelete that repairs both the superseding and previous subscription annotation pointers when a subscription is deleted. | BeforeDelete aborts the delete if it returns an error. Both updateSupersedingSubscriptionAnnotations and updatePreviousSubscriptionAnnotations must succeed. The two helpers are symmetric: each reads the opposite link ID and re-points it, or clears it when there is nothing to point to. |

## Anti-Patterns

- Reading or writing annotation map keys by string literal instead of using subscription.AnnotationParser
- Mutating view.Subscription.Annotations in-place without maps.Clone first
- Returning an error on subscription.IsSubscriptionNotFoundError — these are expected races and must be logged and skipped
- Implementing hook interface methods without embedding NoOpSubscriptionCommandHook, forcing all unrelated methods to be explicitly stubbed
- Storing an empty (len==0) annotations map instead of nil when all keys are removed

## Decisions

- **Hook lives in a dedicated sub-package (hooks/annotations) rather than inline in the subscription service** — Keeps annotation-chain repair logic isolated from core subscription lifecycle code and avoids circular imports between the service and the hook.
- **Only BeforeDelete is overridden** — Annotation chain links only need repair at deletion time; create/update paths set annotations through normal AnnotationParser flows elsewhere.
- **Uses subscription.SubscriptionRepository directly for annotation writes, not subscription.Service** — UpdateAnnotations is a low-level repository operation; routing through the full service would re-trigger hooks and risk re-entrancy.

## Example: Wiring the hook during application startup in app/common

```
import annotationhook "github.com/openmeterio/openmeter/openmeter/subscription/hooks/annotations"

hook, err := annotationhook.NewAnnotationCleanupHook(subscriptionQuerySvc, subscriptionRepo, logger)
if err != nil {
    return fmt.Errorf("create annotation cleanup hook: %w", err)
}
subscriptionSvc.RegisterHook(hook)
```

<!-- archie:ai-end -->
