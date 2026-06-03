# annotations

<!-- archie:ai-start -->

> Implements a SubscriptionCommandHook (AnnotationCleanupHook) that repairs annotation-based chain links (previous/superseding subscription IDs) when a subscription is deleted, maintaining doubly-linked-list integrity across the subscription chain without orphaned pointers.

## Patterns

**Embed NoOpSubscriptionCommandHook** — Every hook struct embeds subscription.NoOpSubscriptionCommandHook so only overridden methods need implementation; the rest compile as no-ops. (`type AnnotationCleanupHook struct { subscription.NoOpSubscriptionCommandHook; subscriptionQueryService subscription.QueryService; ... }`)
**Constructor validates all deps** — NewAnnotationCleanupHook returns (*T, error) and nil-checks every injected dependency before constructing. (`if subscriptionQueryService == nil { return nil, fmt.Errorf("subscription query service is required") }`)
**AnnotationParser for all annotation access** — All reads/writes of annotation keys go through subscription.AnnotationParser methods — never read/write annotation map keys by string literal. (`supersedingID := subscription.AnnotationParser.GetSupersedingSubscriptionID(view.Subscription.Annotations)`)
**Clone before mutating annotations** — Always maps.Clone the annotation map before any modification to avoid mutating the original view in place. (`supersedingAnnotations = maps.Clone(supersedingAnnotations)`)
**Soft-skip NotFound, hard-error otherwise** — When fetching a linked subscription, check subscription.IsSubscriptionNotFoundError; log-and-continue on not-found, return error for all other failures. (`if subscription.IsSubscriptionNotFoundError(err) { h.logger.Error(...); return nil }`)
**Nil-out empty annotation maps** — After deleting keys, if len(annotations)==0 set the map to nil before persisting to avoid storing empty maps. (`if len(supersedingAnnotations) == 0 { supersedingAnnotations = nil }`)
**Persist via SubscriptionRepository.UpdateAnnotations** — Annotation writes use subscriptionRepo.UpdateAnnotations directly — never route through the full Service, to avoid re-triggering hooks (re-entrancy). (`h.subscriptionRepo.UpdateAnnotations(ctx, supersedingView.Subscription.NamespacedID, supersedingAnnotations)`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `hook.go` | Sole file: AnnotationCleanupHook with BeforeDelete repairing both superseding and previous subscription pointers on delete. | BeforeDelete aborts the delete if it errors. Both updateSupersedingSubscriptionAnnotations and updatePreviousSubscriptionAnnotations must succeed; the two helpers are symmetric — each re-points the opposite link or clears it. |

## Anti-Patterns

- Reading/writing annotation keys by string literal instead of subscription.AnnotationParser.
- Mutating view.Subscription.Annotations in place without maps.Clone first.
- Returning an error on IsSubscriptionNotFoundError — these are expected races; log and skip.
- Implementing hook methods without embedding NoOpSubscriptionCommandHook.
- Storing an empty (len==0) annotations map instead of nil.

## Decisions

- **Hook lives in a dedicated hooks/annotations sub-package, not inline in the service.** — Isolates annotation-chain repair from core lifecycle code and avoids circular imports between service and hook.
- **Only BeforeDelete is overridden.** — Chain links only need repair at deletion; create/update set annotations through normal AnnotationParser flows elsewhere.
- **Uses SubscriptionRepository directly for annotation writes, not subscription.Service.** — UpdateAnnotations is a low-level repo op; routing through the full service would re-trigger hooks and risk re-entrancy.

## Example: Wire the hook during startup in app/common

```
import annotationhook "github.com/openmeterio/openmeter/openmeter/subscription/hooks/annotations"

hook, err := annotationhook.NewAnnotationCleanupHook(subscriptionQuerySvc, subscriptionRepo, logger)
if err != nil { return fmt.Errorf("create annotation cleanup hook: %w", err) }
subscriptionSvc.RegisterHook(hook)
```

<!-- archie:ai-end -->
