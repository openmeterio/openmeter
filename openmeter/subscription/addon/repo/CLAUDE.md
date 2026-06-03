# repo

<!-- archie:ai-start -->

> Ent-backed PostgreSQL repository for subscription addons and their quantity timeline, implementing entutils.TransactingRepo so all mutations participate in caller-supplied transactions. Quantities are append-only.

## Patterns

**TransactingRepo wrapping all mutations** — Every method body wraps in entutils.TransactingRepo(ctx, r, func(ctx, repo) ...) so the repo rebinds to the ctx transaction or starts its own via Self(). Never call repo.db directly in a method body. (`return entutils.TransactingRepo(ctx, r, func(ctx context.Context, repo *subscriptionAddonRepo) (*models.NamespacedID, error) { entity, err := repo.db.SubscriptionAddon.Create()...Save(ctx); return &models.NamespacedID{ID: entity.ID, Namespace: entity.Namespace}, err })`)
**Tx/Self/WithTx transaction trinity** — Each repo struct implements Tx() (HijackTx + NewTxDriver), Self() (identity), and WithTx() (rebind via NewTxClientFromRawConfig). Both subscriptionAddonRepo and subscriptionAddonQuantityRepo have their own block in transaction.go. (`func (r *subscriptionAddonRepo) WithTx(ctx context.Context, tx *entutils.TxDriver) *subscriptionAddonRepo { txClient := db.NewTxClientFromRawConfig(ctx, *tx.GetConfig()); return NewSubscriptionAddonRepo(txClient.Client()) }`)
**Eager-load via querySubscriptionAddon helper** — All Get/List queries go through querySubscriptionAddon() which loads WithAddon (Ratecards->Features, Ratecards->TaxCode) and WithQuantities ordered by ActiveFrom asc, CreatedAt asc. Never use a bare db.SubscriptionAddon.Query(). (`query.WithAddon(func(aq *db.AddonQuery) { aq.WithRatecards(func(arq *db.AddonRateCardQuery) { arq.WithFeatures(); arq.WithTaxCode() }) }).WithQuantities(...)`)
**Paginate-or-return-all for List** — List returns all results when filter.Page.IsZero(); otherwise uses Ent's Paginate and entutils.MapPagedWithErr. (`if filter.Page.IsZero() { entities, _ := query.All(ctx) } else { paged, _ := query.Paginate(ctx, filter.Page); return entutils.MapPagedWithErr(paged, MapSubscriptionAddon) }`)
**Quantities are append-only** — SubscriptionAddonQuantityRepository has only Create — no Update or Delete. The quantity timeline is an audit-safe ledger; changes are new records with a later ActiveFrom. (`// Only Create is defined — no Update/Delete on subscriptionAddonQuantityRepo`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `subscriptionaddon.go` | Create, Get, List for SubscriptionAddon; Get wraps db.IsNotFound in models.NewGenericNotFoundError. | All queries must use querySubscriptionAddon() — bare queries miss eager-loaded RateCard/Quantity edges. |
| `subscriptionaddonquantity.go` | Append-only Create for SubscriptionAddonQuantity; namespace comes from parent subscriptionAddonID.Namespace. | No Update or Delete methods exist by design — do not add them. |
| `mapping.go` | MapSubscriptionAddon(s) convert db rows to domain types, delegating RateCard mapping to addonrepo.FromAddonRateCardRow. | MapSubscriptionAddon errors if Edges.Addon is nil for rate card mapping — the edge must always be loaded via querySubscriptionAddon. |
| `transaction.go` | Tx/Self/WithTx for both subscriptionAddonRepo and subscriptionAddonQuantityRepo. | Both repos use db.HijackTx independently — a third repo type needs its own Tx/Self/WithTx block. |

## Anti-Patterns

- Calling repo.db methods directly inside Create/Get/List without TransactingRepo wrapping
- Fetching SubscriptionAddon without the querySubscriptionAddon() helper — misses eager-loaded edges
- Updating or deleting SubscriptionAddonQuantity rows — quantities are immutable append-only
- Using a raw db.Client in WithTx instead of db.NewTxClientFromRawConfig

## Decisions

- **Quantities are append-only, never updated** — The quantity timeline is an audit-safe ledger; changes are new records with a later ActiveFrom rather than modifying existing rows.

## Example: Create subscription addon inside a transaction

```
import (
	subscriptionaddonrepo "github.com/openmeterio/openmeter/openmeter/subscription/addon/repo"
)

repo := subscriptionaddonrepo.NewSubscriptionAddonRepo(dbClient)
id, err := repo.Create(ctx, ns, subscriptionaddon.CreateSubscriptionAddonRepositoryInput{
	AddonID:        addonID,
	SubscriptionID: subID,
})
// ctx carries the tx — repo.Create rebinds automatically via TransactingRepo
```

<!-- archie:ai-end -->
