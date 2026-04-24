# repo

<!-- archie:ai-start -->

> Ent/PostgreSQL persistence layer for subscriptions, phases, and items. Implements subscription.SubscriptionRepository, SubscriptionPhaseRepository, and SubscriptionItemRepository. All three repos share the same TransactingRepo + transaction.go Tx/WithTx/Self pattern.

## Patterns

**entutils.TransactingRepo wrapping on every method** — Every public repo method body is wrapped in entutils.TransactingRepo (or TransactingRepoWithNoValue) so the method rebinds to any transaction already in ctx. (`func (r *subscriptionItemRepo) GetByID(ctx context.Context, id models.NamespacedID) (subscription.SubscriptionItem, error) {
	return entutils.TransactingRepo(ctx, r, func(ctx context.Context, repo *subscriptionItemRepo) (subscription.SubscriptionItem, error) { ... })
}`)
**Tx / Self / WithTx triad in transaction.go** — Each repo implements Tx() to start a new tx, Self() to return itself, and WithTx() to re-create the repo bound to a tx client. Required by entutils.TransactingRepo contract. (`func (r *subscriptionRepo) WithTx(ctx context.Context, tx *entutils.TxDriver) *subscriptionRepo {
	txClient := db.NewTxClientFromRawConfig(ctx, *tx.GetConfig())
	return NewSubscriptionRepo(txClient.Client())
}`)
**Soft-delete via SetDeletedAt(clock.Now())** — Deletes set DeletedAt rather than removing rows. Queries filter with DeletedAtIsNil() OR DeletedAtGT(now) to exclude soft-deleted records. (`err := repo.db.SubscriptionItem.UpdateOneID(input.ID).SetDeletedAt(at).Exec(ctx)`)
**DB error mapping to domain errors** — db.IsNotFound(err) is checked after every query and mapped to domain-specific not-found errors (subscription.NewItemNotFoundError, NewPhaseNotFoundError, NewSubscriptionNotFoundError). (`if db.IsNotFound(err) { return subscription.SubscriptionItem{}, subscription.NewItemNotFoundError(id.ID) }`)
**MapDB* functions for Ent→domain conversion** — mapping.go contains MapDBSubscription, MapDBSubscripitonPhase, MapDBSubscriptionItem. All repo reads must go through these — never inline Ent→domain field mapping. (`return pagination.MapResultErr(paged, MapDBSubscription)`)
**Predicate helper functions in utils.go** — SubscriptionActiveAt, SubscriptionActiveInPeriod, SubscriptionNotDeletedAt return []predicate.Subscription slices; compose them with Where(preds...) in queries. (`query.Where(SubscriptionNotDeletedAt(now)...).Where(SubscriptionActiveAt(at)...)`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `subscriptionrepo.go` | CRUD + List for Subscription entity; List supports filtering by customer, status, active period, and pagination. | Status filtering (Active/Canceled/Inactive/Scheduled) builds OR predicate blocks — adding a new status requires a new predicate branch. |
| `subscriptionphaserepo.go` | CRUD for SubscriptionPhase; GetForSubscriptionsAt accepts a batch of inputs for bulk loading. | Phase queries do NOT eager-load items; item loading is done separately by subscriptionItemRepo. |
| `subscriptionitemrepo.go` | CRUD for SubscriptionItem; Create maps rate card fields (price, discounts, tax config) manually due to custom Ent value scanners. | EntitlementTemplate and TaxConfig fields use custom value scanners and must be set conditionally (nil-check before Set). |
| `mapping.go` | Converts Ent DB rows to domain types; MapDBSubscriptionItem requires eager-loaded Phase edge (PhaseOrErr). | Always call WithPhase() on item queries; missing edge causes MapDBSubscriptionItem to error. |
| `transaction.go` | Implements Tx/Self/WithTx for all three repo types using db.HijackTx + entutils.NewTxDriver. | All three repo types must implement this triad; omitting WithTx causes TransactingRepo to fall back to a new transaction instead of joining the caller's. |
| `utils.go` | Shared Ent predicate builders for subscription time-range queries. | SubscriptionActiveInPeriod uses StartBoundedPeriod (open end); To==nil means no upper bound. |

## Anti-Patterns

- Omitting entutils.TransactingRepo wrapper — the method will bypass any caller-supplied transaction.
- Querying items without WithPhase() eager-load — MapDBSubscriptionItem will fail at runtime.
- Inlining Ent→domain conversion instead of using MapDB* functions.
- Hard-deleting rows instead of soft-deleting via SetDeletedAt.

## Decisions

- **Soft delete via DeletedAt timestamp for all three entity types.** — Billing sync reads historical item cadences; permanent deletion would lose the window data needed for invoice line generation.

## Example: Add a new query method with transaction support

```
import (
	"github.com/openmeterio/openmeter/openmeter/ent/db"
	dbsubscription "github.com/openmeterio/openmeter/openmeter/ent/db/subscription"
	"github.com/openmeterio/openmeter/openmeter/subscription"
	"github.com/openmeterio/openmeter/pkg/framework/entutils"
)

func (r *subscriptionRepo) GetByCustomerID(ctx context.Context, ns, customerID string) ([]subscription.Subscription, error) {
	return entutils.TransactingRepo(ctx, r, func(ctx context.Context, repo *subscriptionRepo) ([]subscription.Subscription, error) {
		rows, err := repo.db.Subscription.Query().
			WithPlan().
			Where(dbsubscription.Namespace(ns), dbsubscription.CustomerID(customerID)).
			Where(SubscriptionNotDeletedAt(clock.Now())...).
			All(ctx)
		if err != nil {
// ...
```

<!-- archie:ai-end -->
