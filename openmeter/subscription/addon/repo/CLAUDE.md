# repo

<!-- archie:ai-start -->

> Ent-backed PostgreSQL repository for subscription addons (subscriptionaddon.SubscriptionAddonRepository) and their quantity timeline (SubscriptionAddonQuantityRepository). Implements the entutils.TransactingRepo pattern so all mutations participate in caller-supplied transactions.

## Patterns

**TransactingRepo wrapping all mutations** — Every method body is wrapped in entutils.TransactingRepo(ctx, r, func(ctx, repo) ...) so the repo rebinds to the transaction already in ctx, or starts its own. Never call repo.db directly inside a method body. (`func (r *subscriptionAddonRepo) Create(ctx, ns, input) (*models.NamespacedID, error) { return entutils.TransactingRepo(ctx, r, func(ctx, repo) (*models.NamespacedID, error) { ... }) }`)
**Tx / Self / WithTx transaction trinity** — Each repo struct implements Tx() (hijack), Self() (identity), and WithTx() (rebind to tx client) so entutils.TransactingRepo can correctly compose transactions. Both subscriptionAddonRepo and subscriptionAddonQuantityRepo have their own transaction.go block. (`func (r *subscriptionAddonRepo) WithTx(ctx, tx *entutils.TxDriver) *subscriptionAddonRepo { txClient := db.NewTxClientFromRawConfig(ctx, *tx.GetConfig()); return NewSubscriptionAddonRepo(txClient.Client()) }`)
**Eager-load via querySubscriptionAddon helper** — All Get/List queries go through querySubscriptionAddon() which WithAddon (including Ratecards->Features, Ratecards->TaxCode) and WithQuantities (ordered by ActiveFrom asc, CreatedAt asc). Adding a new edge requires updating this helper. (`query.WithAddon(func(aq *db.AddonQuery) { aq.WithRatecards(func(arq *db.AddonRateCardQuery) { arq.WithFeatures(); arq.WithTaxCode() }) }).WithQuantities(...)`)
**Quantities ordered timeline** — SubscriptionAddonQuantities are always fetched ordered ASC by ActiveFrom then CreatedAt; mapping constructs a timeutil.Timeline from the sorted slice to power GetInstanceAt/GetInstances logic. (`saqq.Order(db.Asc(dbsubscriptionaddonquantity.FieldActiveFrom), db.Asc(dbsubscriptionaddonquantity.FieldCreatedAt))`)
**Paginate or return all** — List returns all results without pagination when filter.Page.IsZero(); otherwise uses Ent's Paginate helper and entutils.MapPagedWithErr. (`if filter.Page.IsZero() { entities, _ := query.All(ctx); ... } else { paged, _ := query.Paginate(ctx, filter.Page); return entutils.MapPagedWithErr(paged, MapSubscriptionAddon) }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `subscriptionaddon.go` | Create, Get, List for SubscriptionAddon. Get wraps NotFound in models.NewGenericNotFoundError. | All queries use querySubscriptionAddon() to eager-load edges — never add a bare db.SubscriptionAddon.Query() without the helper. |
| `subscriptionaddonquantity.go` | Create only — quantities are append-only. No update or delete. | Namespace comes from the parent subscriptionAddonID.Namespace, not from a separate input field. |
| `mapping.go` | MapSubscriptionAddon and MapSubscriptionAddons convert db rows to domain types, delegating RateCard mapping to addonrepo.FromAddonRateCardRow. | MapSubscriptionAddon returns an error if Edges.Addon is nil for rate card mapping — ensure the edge is always loaded. |
| `transaction.go` | Tx/Self/WithTx implementations for both repo types. | Both repos use db.HijackTx independently — if you add a third repo type here, it needs its own block. |

## Anti-Patterns

- Calling repo.db methods directly inside Create/Get/List without TransactingRepo wrapping
- Fetching SubscriptionAddon without querySubscriptionAddon() helper, which would miss eager-loaded RateCard/Quantity edges
- Updating or deleting SubscriptionAddonQuantity rows — quantities are immutable append-only records
- Using a raw db.Client in WithTx instead of db.NewTxClientFromRawConfig

## Decisions

- **Quantities are append-only, never updated** — The quantity timeline is an audit-safe ledger; changes are represented by new quantity records with a later ActiveFrom rather than modifying existing rows.

## Example: Create subscription addon inside a transaction

```
import (
	subscriptionaddonrepo "github.com/openmeterio/openmeter/openmeter/subscription/addon/repo"
	"github.com/openmeterio/openmeter/openmeter/ent/db"
)

repo := subscriptionaddonrepo.NewSubscriptionAddonRepo(dbClient)
id, err := repo.Create(ctx, ns, subscriptionaddon.CreateSubscriptionAddonRepositoryInput{
	AddonID:        addonID,
	SubscriptionID: subID,
})
// ctx carries the tx — repo.Create rebinds automatically via TransactingRepo
```

<!-- archie:ai-end -->
