# adapter

<!-- archie:ai-start -->

> Ent/PostgreSQL adapter implementing addon.Repository — full CRUD and list for add-ons and their rate cards, every operation wrapped in entutils.TransactingRepo to honor ctx-bound transactions. The persistence boundary for the addon domain.

## Patterns

**TransactingRepo on every method** — Every exported method wraps its fn body in entutils.TransactingRepo[T, *adapter](ctx, a, fn). Never access a.db directly outside the wrapper. (`return entutils.TransactingRepo[*addon.Addon, *adapter](ctx, a, fn)`)
**Tx/WithTx/Self triad** — adapter implements TxCreator (Tx via HijackTx+NewTxDriver), TxUser (WithTx via NewTxClientFromRawConfig().Client()), and Self() — all required by TransactingRepo. (`func (a *adapter) WithTx(ctx context.Context, tx *entutils.TxDriver) *adapter { return &adapter{db: entdb.NewTxClientFromRawConfig(ctx, *tx.GetConfig()).Client(), logger: a.logger} }`)
**Eager-load rate cards** — Every query returning an Addon calls WithRatecards(AddonEagerLoadRateCardsFn); PlanAddons loaded only when Expand.PlanAddons via WithPlans(addonEagerLoadActivePlans). (`query = query.WithRatecards(AddonEagerLoadRateCardsFn)`)
**Soft delete via DeletedAt** — DeleteAddon sets DeletedAt=time.Now().UTC() via UpdateOneID; ListAddons excludes deleted rows with addondb.DeletedAtIsNil() unless IncludeDeleted is set. (`a.db.Addon.UpdateOneID(add.ID).Where(addondb.Namespace(add.Namespace)).SetDeletedAt(deletedAt).Exec(ctx)`)
**Status filtering via Effective time predicates** — AddonStatus is derived at query time from EffectiveFrom/EffectiveTo predicates (active/draft/archived). Never compute status in Go after fetching all rows. (`addondb.And(addondb.EffectiveFromLTE(now), addondb.Or(addondb.EffectiveToGTE(now), addondb.EffectiveToIsNil()))`)
**Rate card updates replace all (delete + bulk-create)** — UpdateAddon deletes all existing AddonRateCard rows then rateCardBulkCreate — full replacement, no diff logic. (`a.db.AddonRateCard.Delete().Where(addonratecarddb.AddonID(add.ID)).Exec(ctx)`)
**All conversions in mapping.go** — CRUD methods call FromAddonRow / FromAddonRateCardRow / FromPlanAddonRow / asAddonRateCardRow. Never embed conversion in query methods. (`add, err := FromAddonRow(*addonRow)`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `adapter.go` | Config struct, New constructor with Validate(), adapter struct, and Tx/WithTx/Self plumbing. | Never remove Tx/WithTx/Self — required by TransactingRepo. Always call config.Validate() before constructing. |
| `addon.go` | All CRUD+list methods plus rateCardBulkCreate and addonEagerLoadActivePlans. | Wrap every method body with TransactingRepo. GetAddon uses an SQL subquery for latest-version-by-key — prefer existing addondb predicates over raw selectors. |
| `mapping.go` | Bidirectional Ent<->domain mapping; reuses plan adapter mappers (FromPlanRow, FromPlanPhaseRow, FromPlanRateCardRow). | Call TaxConfig.BackfillTaxConfig after mapping the TaxCode edge. BillingCadence uses ParsePtrOrNil()/ISOString, never raw string assignment. |
| `adapter_test.go` | Integration tests against real Postgres via pctestutils.NewTestEnv covering CRUD plus status-filter logic. | Prefer t.Context() over context.WithCancel for new tests; TestEnv closed in t.Cleanup. |

## Anti-Patterns

- Calling a.db.Foo() directly inside a method without TransactingRepo — bypasses ctx-bound transactions.
- Embedding type-conversion logic in CRUD methods instead of mapping.go.
- Querying add-ons without WithRatecards eager load.
- Hard-deleting rows instead of setting DeletedAt.
- Building raw SQL predicates for status filtering instead of EffectiveFrom/EffectiveTo Ent predicates.

## Decisions

- **TransactingRepo wraps every method rather than passing *entdb.Tx explicitly.** — Ent transactions propagate via ctx; explicit tx params leak plumbing and prevent safe savepoint nesting.
- **Rate card updates fully replace (delete + bulk-create) rather than diffing.** — No partial state after update; ordering and key uniqueness enforced atomically without diff complexity.

## Example: Add an adapter method for an addon sub-resource

```
func (a *adapter) GetAddonFoo(ctx context.Context, params addon.GetFooInput) (*addon.Foo, error) {
	fn := func(ctx context.Context, a *adapter) (*addon.Foo, error) {
		if err := params.Validate(); err != nil { return nil, fmt.Errorf("invalid params: %w", err) }
		row, err := a.db.AddonFoo.Query().Where(...).WithRatecards(AddonEagerLoadRateCardsFn).First(ctx)
		if err != nil { if entdb.IsNotFound(err) { return nil, addon.NewNotFoundError(...) }; return nil, err }
		return FromAddonFooRow(*row)
	}
	return entutils.TransactingRepo[*addon.Foo, *adapter](ctx, a, fn)
}
```

<!-- archie:ai-end -->
