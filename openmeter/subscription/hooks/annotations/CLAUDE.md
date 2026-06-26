# annotations

<!-- archie:ai-start -->

> Single-file (hook.go, package annotationhook) command hook that keeps the previous/superseding subscription cross-link annotations consistent when a subscription is deleted. It runs on BeforeDelete to repair the doubly-linked list of subscriptions so neither neighbor points at a soon-to-be-deleted subscription.

## Patterns

**Embed NoOpSubscriptionCommandHook** — The hook struct embeds subscription.NoOpSubscriptionCommandHook so it only overrides the lifecycle methods it cares about (BeforeDelete) and inherits no-op implementations for the rest. (`type AnnotationCleanupHook struct { subscription.NoOpSubscriptionCommandHook; subscriptionQueryService subscription.QueryService; subscriptionRepo subscription.SubscriptionRepository; logger *slog.Logger }`)
**Constructor validates non-nil deps and returns error** — NewAnnotationCleanupHook returns (*AnnotationCleanupHook, error), nil-checking subscriptionQueryService, subscriptionRepository, and logger before constructing. Logger is injected explicitly, never slog.Default(). (`if subscriptionQueryService == nil { return nil, fmt.Errorf("subscription query service is required") }`)
**Read links via subscription.AnnotationParser** — Never index the annotation map directly. Read superseding/previous IDs with AnnotationParser.GetSupersedingSubscriptionID / GetPreviousSubscriptionID and write with SetPreviousSubscriptionID / SetSupersedingSubscriptionID / ClearSupersedingSubscriptionID. (`supersedingID := subscription.AnnotationParser.GetSupersedingSubscriptionID(view.Subscription.Annotations)`)
**Clone annotations before mutating** — Annotations from a fetched view are cloned with maps.Clone before mutation (or initialized to models.Annotations{}), so the in-memory view is never mutated in place. (`if supersedingAnnotations != nil { supersedingAnnotations = maps.Clone(supersedingAnnotations) } else { supersedingAnnotations = models.Annotations{} }`)
**Tolerate missing neighbor subscriptions** — GetView failures are inspected with subscription.IsSubscriptionNotFoundError; a missing neighbor is logged and skipped (return nil) rather than failing the delete. Other errors are wrapped and propagated. (`if subscription.IsSubscriptionNotFoundError(err) { h.logger.Error("superseding subscription not found, continuing without cleanup", ...); return nil }`)
**Persist via SubscriptionRepository.UpdateAnnotations** — Annotation writes go through subscriptionRepo.UpdateAnnotations(ctx, NamespacedID, annotations); when the resulting map is empty it is collapsed to nil before persisting. (`if len(supersedingAnnotations) == 0 { supersedingAnnotations = nil }; _, err = h.subscriptionRepo.UpdateAnnotations(ctx, supersedingView.Subscription.NamespacedID, supersedingAnnotations)`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `hook.go` | Defines AnnotationCleanupHook + NewAnnotationCleanupHook and the BeforeDelete handler delegating to updateSupersedingSubscriptionAnnotations and updatePreviousSubscriptionAnnotations. | The two update methods are near-mirror images (superseding vs previous direction). When editing one, keep the other symmetric. Note the asymmetry: superseding-side clears the previous key via raw delete(annotations, subscription.AnnotationPreviousSubscriptionID), while previous-side clears via AnnotationParser.ClearSupersedingSubscriptionID — don't accidentally unify them incorrectly. BeforeDelete fails the whole delete if either update returns a non-not-found error. |

## Anti-Patterns

- Mutating view.Subscription.Annotations in place instead of cloning — corrupts the caller's in-memory view.
- Indexing or writing the annotation map with raw string keys instead of going through subscription.AnnotationParser (except the existing delete of AnnotationPreviousSubscriptionID).
- Failing the delete when a neighbor subscription is not found — must detect IsSubscriptionNotFoundError, log, and continue.
- Falling back to slog.Default() instead of requiring an injected *slog.Logger in the constructor.
- Persisting an empty (zero-length) annotation map instead of collapsing it to nil before UpdateAnnotations.

## Decisions

- **Cleanup runs in BeforeDelete and relinks the surviving neighbors to each other.** — Subscriptions form a doubly-linked previous/superseding chain via annotations; deleting a middle node would leave dangling references, so the hook stitches previous<->superseding together (or clears the dangling side) to keep the chain valid.
- **Hook embeds NoOpSubscriptionCommandHook and depends only on QueryService + SubscriptionRepository.** — Keeps the hook narrowly scoped to annotation repair and avoids pulling in the full mutation service, preventing import cycles and side effects beyond annotation updates.

## Example: Repairing the superseding neighbor's previous-link when a subscription is deleted

```
func (h *AnnotationCleanupHook) updateSupersedingSubscriptionAnnotations(ctx context.Context, view subscription.SubscriptionView) error {
	supersedingID := subscription.AnnotationParser.GetSupersedingSubscriptionID(view.Subscription.Annotations)
	previousID := subscription.AnnotationParser.GetPreviousSubscriptionID(view.Subscription.Annotations)
	if supersedingID == nil {
		return nil
	}
	supersedingView, err := h.subscriptionQueryService.GetView(ctx, models.NamespacedID{ID: lo.FromPtr(supersedingID), Namespace: view.Subscription.Namespace})
	if err != nil {
		if subscription.IsSubscriptionNotFoundError(err) {
			h.logger.Error("superseding subscription not found, continuing without cleanup", "error", err)
			return nil
		}
		return fmt.Errorf("failed to get superseding subscription: %w", err)
	}
	ann := maps.Clone(supersedingView.Subscription.Annotations)
// ...
```

<!-- archie:ai-end -->
