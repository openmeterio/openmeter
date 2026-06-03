# repo

<!-- archie:ai-start -->

> Ent/PostgreSQL persistence layer for subscriptions, phases, and items. Implements subscription.SubscriptionRepository, SubscriptionPhaseRepository, and SubscriptionItemRepository sharing the TransactingRepo + Tx/WithTx/Self triad.

## Patterns

**TransactingRepo wrapping on every method** — Every public method body wraps in entutils.TransactingRepo (or WithNoValue) to rebind to a ctx-carried tx. (`return entutils.TransactingRepo(ctx, r, func(ctx context.Context, repo *subscriptionItemRepo) (...) { ... })`)
**Tx / Self / WithTx triad** — transaction.go implements the triad using db.HijackTx + entutils.NewTxDriver; WithTx rebuilds the repo from the tx client. (`txClient := db.NewTxClientFromRawConfig(ctx, *tx.GetConfig()); return NewSubscriptionRepo(txClient.Client())`)
**Soft-delete via SetDeletedAt** — Deletes set DeletedAt; queries filter with DeletedAtIsNil()/DeletedAtGT(now). (`repo.db.SubscriptionItem.UpdateOneID(input.ID).SetDeletedAt(at).Exec(ctx)`)
**DB error mapping to domain errors** — db.IsNotFound(err) after every query, mapped to domain not-found errors. (`if db.IsNotFound(err) { return ..., subscription.NewItemNotFoundError(id.ID) }`)
**MapDB* for Ent→domain conversion** — All reads go through mapping.go MapDBSubscription/MapDBSubscripitonPhase/MapDBSubscriptionItem — never inline field mapping. (`return pagination.MapResultErr(paged, MapDBSubscription)`)
**Predicate helpers in utils.go** — SubscriptionActiveAt/ActiveInPeriod/NotDeletedAt return []predicate.Subscription; compose with Where(preds...). (`query.Where(SubscriptionNotDeletedAt(now)...).Where(SubscriptionActiveAt(at)...)`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `subscriptionrepo.go` | CRUD + List for Subscription; List filters by customer, status, active period, pagination. | Status filtering builds OR predicate blocks — a new status needs a new predicate branch. |
| `subscriptionphaserepo.go` | CRUD for SubscriptionPhase; GetForSubscriptionsAt bulk-loads. | Phase queries do NOT eager-load items; items load separately via subscriptionItemRepo. |
| `subscriptionitemrepo.go` | CRUD for SubscriptionItem; Create maps rate-card fields manually. | EntitlementTemplate and TaxConfig use custom value scanners — nil-check before Set. |
| `mapping.go` | Ent→domain conversion; MapDBSubscriptionItem requires the eager-loaded Phase edge. | Always call WithPhase() on item queries; missing edge makes MapDBSubscriptionItem error at runtime. |
| `transaction.go` | Tx/Self/WithTx for all three repo types. | Omitting WithTx makes TransactingRepo open a new tx instead of joining the caller's. |
| `utils.go` | Shared Ent predicate builders for time-range queries. | SubscriptionActiveInPeriod uses StartBoundedPeriod (open end); To==nil means no upper bound. |

## Anti-Patterns

- Omitting the entutils.TransactingRepo wrapper — bypasses any caller-supplied transaction.
- Querying items without WithPhase() eager-load — MapDBSubscriptionItem fails at runtime.
- Inlining Ent→domain conversion instead of using MapDB* functions.
- Hard-deleting rows instead of soft-deleting via SetDeletedAt.

## Decisions

- **Soft delete via DeletedAt for all three entity types.** — Billing sync reads historical item cadences; permanent deletion would lose the window data needed for invoice line generation.

## Example: Add a new query method with transaction support and eager-loaded edges

```
func (r *subscriptionRepo) GetByCustomerID(ctx context.Context, ns, customerID string) ([]subscription.Subscription, error) {
  return entutils.TransactingRepo(ctx, r, func(ctx context.Context, repo *subscriptionRepo) ([]subscription.Subscription, error) {
    rows, err := repo.db.Subscription.Query().WithPlan().
      Where(dbsubscription.Namespace(ns), dbsubscription.CustomerID(customerID)).
      Where(SubscriptionNotDeletedAt(clock.Now())...).All(ctx)
    // ...
  })
}
```

<!-- archie:ai-end -->
