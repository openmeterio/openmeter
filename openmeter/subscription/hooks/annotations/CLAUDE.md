# annotations

<!-- archie:ai-start -->

> Implements a SubscriptionCommandHook that maintains referential integrity of annotation-based subscription chain links (previous/superseding IDs) when a subscription is deleted. Ensures the doubly-linked list of chained subscriptions remains consistent without orphaned pointers.

## Patterns

**Embed NoOpSubscriptionCommandHook** — Every hook struct must embed subscription.NoOpSubscriptionCommandHook so only the methods it overrides need implementation; unimplemented hook methods compile as no-ops. (`type AnnotationCleanupHook struct { subscription.NoOpSubscriptionCommandHook; ... }`)
**Constructor validates all required deps** — NewAnnotationCleanupHook returns (*T, error) and checks every injected dep for nil, returning a descriptive error string before constructing the struct. (`if subscriptionQueryService == nil { return nil, fmt.Errorf("subscription query service is required") }`)
**Use AnnotationParser for annotation access** — All reads/writes of annotation keys (previous ID, superseding ID) must go through subscription.AnnotationParser methods — never read/write annotation map keys directly. (`subscription.AnnotationParser.GetSupersedingSubscriptionID(view.Subscription.Annotations)`)
**Clone annotations before mutating** — Always call maps.Clone on the annotation map before modifying it to avoid mutating the original view's annotations in-place. (`supersedingAnnotations = maps.Clone(supersedingAnnotations)`)
**Soft-skip on NotFound, hard-error on others** — When fetching a linked subscription, check subscription.IsSubscriptionNotFoundError; log and continue on not-found, but return the error for all other failures. (`if subscription.IsSubscriptionNotFoundError(err) { h.logger.Error(...); return nil }`)
**Nil-out empty annotation maps** — After deleting keys, if len(annotations) == 0 set the map to nil before persisting to avoid storing empty maps. (`if len(supersedingAnnotations) == 0 { supersedingAnnotations = nil }`)
**Persist via SubscriptionRepository.UpdateAnnotations** — All annotation writes use subscriptionRepo.UpdateAnnotations(ctx, namespacedID, annotations) — never mutate through the query service. (`h.subscriptionRepo.UpdateAnnotations(ctx, supersedingView.Subscription.NamespacedID, supersedingAnnotations)`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `hook.go` | Sole file; defines AnnotationCleanupHook and its BeforeDelete handler that repairs the previous/superseding chain when a subscription is removed. | BeforeDelete is called before deletion; if it returns an error the delete is aborted. Both updateSupersedingSubscriptionAnnotations and updatePreviousSubscriptionAnnotations must succeed for the operation to proceed. |

## Anti-Patterns

- Directly reading or writing annotation map keys by string literal instead of using subscription.AnnotationParser
- Mutating view.Subscription.Annotations in-place without cloning first
- Returning an error on subscription.IsSubscriptionNotFoundError — these are expected races and must be logged and skipped
- Implementing hook interface methods without embedding NoOpSubscriptionCommandHook, forcing all methods to be explicitly implemented
- Storing an empty (len==0) annotations map instead of nil when all keys are removed

## Decisions

- **Hook is placed in a dedicated sub-package (hooks/annotations) rather than inline in the subscription service** — Keeps annotation-chain repair logic isolated from core subscription lifecycle code, and avoids circular imports between the service and the hook.
- **BeforeDelete is the only overridden hook method** — Annotation chain links only need repair at deletion time; create/update paths set annotations through normal AnnotationParser flows elsewhere.
- **Uses subscription.SubscriptionRepository directly (not subscription.Service) for annotation writes** — UpdateAnnotations is a low-level repository operation; routing through the full service would re-trigger hooks and risk re-entrancy.

## Example: Wiring the hook during application startup

```
import (
	annotationhook "github.com/openmeterio/openmeter/openmeter/subscription/hooks/annotations"
)

hook, err := annotationhook.NewAnnotationCleanupHook(subscriptionQuerySvc, subscriptionRepo, logger)
if err != nil {
	return fmt.Errorf("create annotation cleanup hook: %w", err)
}
subscriptionSvc.RegisterHook(hook)
```

<!-- archie:ai-end -->
