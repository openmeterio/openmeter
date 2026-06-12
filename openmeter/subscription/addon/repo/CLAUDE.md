# repo

<!-- archie:ai-start -->

> Ent-backed persistence (package subscriptionaddonrepo) for subscription addons and their quantity timeline. Implements subscriptionaddon.SubscriptionAddonRepository and SubscriptionAddonQuantityRepository, mapping db rows to domain types; constraint: every method is transaction-aware via entutils.TransactingRepo.

## Patterns

**TransactingRepo wrapping** — Every repo method body is wrapped in entutils.TransactingRepo(ctx, r, func(ctx, repo) {...}) so it rebinds to any tx already in ctx; Tx/Self/WithTx are implemented in transaction.go. (`return entutils.TransactingRepo(ctx, r, func(ctx context.Context, repo *subscriptionAddonRepo) (...) { ... })`)
**Interface assertion + constructor** — Each repo asserts the domain interface (var _ subscriptionaddon.SubscriptionAddonRepository = (*subscriptionAddonRepo)(nil)) and exposes a NewX(db *db.Client) constructor returning the unexported struct. (`var _ subscriptionaddon.SubscriptionAddonRepository = (*subscriptionAddonRepo)(nil)`)
**Eager-load via querySubscriptionAddon** — Reads always go through querySubscriptionAddon which WithAddon(WithRatecards(WithFeatures, WithTaxCode)) and WithQuantities ordered by ActiveFrom then CreatedAt; mapping assumes these edges are loaded. (`query.WithAddon(func(aq){ aq.WithRatecards(...) }).WithQuantities(...)`)
**Db-to-domain mapping in mapping.go** — MapSubscriptionAddon builds the domain SubscriptionAddon from edges; quantities become a timeutil.NewTimeline of Timed values; rate cards reuse addonrepo.FromAddonRow / FromAddonRateCardRow from productcatalog. (`base.Quantities = timeutil.NewTimeline(lo.Map(quantities, func(q, _){ return q.AsTimed() }))`)
**Quantity changes are append-only** — subscriptionAddonQuantityRepo.Create inserts a new SubscriptionAddonQuantity row (ActiveFrom, Quantity); history is never mutated in place — the timeline is rebuilt on read. (`repo.db.SubscriptionAddonQuantity.Create().SetActiveFrom(...).SetQuantity(...).Save(ctx)`)
**Optional pagination** — List returns all items as a single page when filter.Page.IsZero(), otherwise query.Paginate(ctx, filter.Page) + entutils.MapPagedWithErr; ordering selected by filter.OrderBy (ID/UpdatedAt/CreatedAt default). (`if filter.Page.IsZero() { ... pagination.NewPage(1, len(items)) } else { query.Paginate(ctx, filter.Page) }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `subscriptionaddon.go` | subscriptionAddonRepo: Create/Get/List plus the shared querySubscriptionAddon eager-load helper | Get returns models.NewGenericNotFoundError on db.IsNotFound; List branches on Page.IsZero; mapping relies on edges loaded by querySubscriptionAddon |
| `subscriptionaddonquantity.go` | subscriptionAddonQuantityRepo: append-only Create of a quantity segment | No update/delete — changing quantity means inserting a new row keyed by ActiveFrom |
| `mapping.go` | MapSubscriptionAddon(s), MapSubscriptionAddonRateCard(s), MapSubscriptionAddonQuantity(ies) | Reads entity.Edges.Addon.Edges.Ratecards (panics if Addon edge nil); reuses productcatalog addonrepo.FromAddonRow/FromAddonRateCardRow |
| `transaction.go` | Tx/Self/WithTx for both repos via db.HijackTx + entutils.NewTxDriver | Required boilerplate for TransactingRepo to work; WithTx rebuilds the client from raw tx config |

## Anti-Patterns

- Bypassing querySubscriptionAddon on reads (mapping assumes Addon/Ratecards/Quantities edges are loaded)
- Updating or deleting quantity rows instead of inserting a new ActiveFrom segment
- Writing a repo method without entutils.TransactingRepo wrapping
- Returning raw db.IsNotFound instead of models.NewGenericNotFoundError
- Hand-mapping rate cards instead of delegating to addonrepo.FromAddonRow/FromAddonRateCardRow

## Decisions

- **Quantity is an append-only timeline of (ActiveFrom, Quantity) rows** — Addon quantity changes over time; the domain reconstructs a timeutil.Timeline so historical quantities remain queryable
- **Reuse productcatalog addon adapter for rate-card mapping** — SubscriptionAddon rate cards are addon rate cards; sharing FromAddonRow avoids divergent mapping logic

<!-- archie:ai-end -->
