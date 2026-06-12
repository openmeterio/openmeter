# repo

<!-- archie:ai-start -->

> Ent-backed persistence for subscriptions, phases, and items. Implements the three subscription repository interfaces and owns DB<->domain mapping. Constraint: every method is transaction-aware via entutils.TransactingRepo.

## Patterns

**Repo implements subscription repository interface** — Each repo holds *db.Client and asserts the matching interface (SubscriptionRepository / SubscriptionPhaseRepository / SubscriptionItemRepository). (`var _ subscription.SubscriptionRepository = (*subscriptionRepo)(nil)`)
**Wrap every method in entutils.TransactingRepo** — All read/write methods run inside entutils.TransactingRepo(ctx, r, func(ctx, repo) ...) so they rebind to the ctx transaction; use TransactingRepoWithNoValue for void ops. (`return entutils.TransactingRepo(ctx, r, func(ctx context.Context, repo *subscriptionRepo) (subscription.Subscription, error) { ... })`)
**Map DB rows in mapping.go, never inline** — MapDBSubscription, MapDBSubscripitonPhase, MapDBSubscriptionItem are the only DB->domain converters; queries call them. UTC-normalize times via convert.SafeToUTC. (`return MapDBSubscription(res)`)
**Soft-delete with DeletedAt + not-deleted predicates** — Delete sets DeletedAt = clock.Now(); reads filter via SubscriptionNotDeletedAt(at) / DeletedAtGT helpers in utils.go. (`Where(SubscriptionNotDeletedAt(clock.Now())...)`)
**Map ent NotFound to domain not-found errors** — Check db.IsNotFound(err) and return subscription.NewSubscriptionNotFoundError / NewItemNotFoundError / NewPhaseNotFoundError instead of leaking ent errors. (`if db.IsNotFound(err) { return subscription.SubscriptionItem{}, subscription.NewItemNotFoundError(id.ID) }`)
**Tx/Self/WithTx triplet per repo** — transaction.go implements Tx (HijackTx), Self, and WithTx (NewTxClientFromRawConfig) for each repo so entutils can manage transactions. (`func (r *subscriptionRepo) WithTx(ctx context.Context, tx *entutils.TxDriver) *subscriptionRepo { ... }`)
**Filter/order/paginate via pkg helpers** — List uses filter.ApplyToQuery / filter.SelectPredicate, entutils.GetOrdering, sortx, and pagination.MapResultErr; status filters compose dbsubscription predicates. (`return pagination.MapResultErr(paged, MapDBSubscription)`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `mapping.go` | DB->domain mappers; reconstructs RateCard (FlatFee vs UsageBased) from item columns | RateCard type is chosen by Price type/cadence; usage-based requires a non-nil billing cadence. FeatureID is left nil (resolved at service level). TaxConfig is backfilled from normalized TaxBehavior/TaxCode columns via BackfillTaxConfig. Item mapping requires the Phase edge to be eagerly loaded (PhaseOrErr). |
| `subscriptionitemrepo.go` | Item CRUD; Create translates RateCard meta into Set* columns | Queries must .WithPhase().WithTaxCode() or MapDBSubscriptionItem fails. EntitlementTemplate/TaxConfig/Price use non-Nillable setters guarded by nil checks (the value scanners panic on nil). |
| `subscriptionphaserepo.go` | Phase CRUD and at-time fetch filters | getPhaseForSubscriptionAtFilter scopes by SubscriptionID+Namespace and DeletedAt; GetForSubscriptionsAt errors on empty input filter. |
| `subscriptionrepo.go` | Subscription CRUD, SetEndOfCadence, UpdateAnnotations, List with status/filter logic | Create eagerly attaches the Plan edge after save so MapDBSubscription can read PlanRef. List computes Active/Canceled/Inactive/Scheduled status as predicate groups over now. |
| `utils.go` | Reusable subscription predicate builders (active at/after/in-period, not-deleted) | SubscriptionActiveAt uses ActiveFromLTE + (ActiveToIsNil OR ActiveToGT); reuse these instead of hand-rolling time predicates. |
| `transaction.go` | Tx/Self/WithTx for all three repos | Each new repo must add its triplet here or TransactingRepo cannot rebind to the active transaction. |

## Anti-Patterns

- Querying items/phases without .WithPhase()/.WithTaxCode(), causing MapDB* to error on missing edges.
- Returning raw ent errors instead of mapping db.IsNotFound to subscription.New*NotFoundError.
- Hard-deleting rows instead of soft-deleting via SetDeletedAt(clock.Now()).
- Writing a repo method that bypasses entutils.TransactingRepo and uses r.db directly outside the active transaction.
- Duplicating mapping logic in query methods instead of calling the MapDB* functions.

## Decisions

- **RateCard kind is inferred from persisted columns rather than a stored discriminator** — Flat-fee vs usage-based is derivable from Price type and presence of a billing cadence, avoiding a separate column and keeping the schema lean.
- **Phase/Item Create accept *EntityInput types, not domain specs** — Specs are mapped to flat entity inputs by the service layer; the repo only persists fully-resolved entity inputs, keeping spec semantics out of the DB layer.

## Example: Transaction-aware repo read with not-found mapping

```
func (r *subscriptionRepo) GetByID(ctx context.Context, subscriptionID models.NamespacedID) (subscription.Subscription, error) {
	return entutils.TransactingRepo(ctx, r, func(ctx context.Context, repo *subscriptionRepo) (subscription.Subscription, error) {
		res, err := repo.db.Subscription.Query().WithPlan().
			Where(dbsubscription.ID(subscriptionID.ID), dbsubscription.Namespace(subscriptionID.Namespace)).
			Where(SubscriptionNotDeletedAt(clock.Now())...).First(ctx)
		if db.IsNotFound(err) {
			return subscription.Subscription{}, subscription.NewSubscriptionNotFoundError(subscriptionID.ID)
		} else if err != nil {
			return subscription.Subscription{}, err
		}
		return MapDBSubscription(res)
	})
}
```

<!-- archie:ai-end -->
