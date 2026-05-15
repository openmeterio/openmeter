# adapter

<!-- archie:ai-start -->

> Ent/PostgreSQL adapter implementing addon.Repository: full CRUD and list for add-ons and their rate cards, with all operations wrapped in entutils.TransactingRepo to honor ctx-bound transactions. This is the persistence boundary for the addon domain.

## Patterns

**TransactingRepo wrapping on every method** — Every exported adapter method wraps its fn body in entutils.TransactingRepo[T, *adapter](ctx, a, fn). Never access a.db directly in a method body without this wrapper. (`return entutils.TransactingRepo[*addon.Addon, *adapter](ctx, a, fn)`)
**Tx / WithTx / Self triad** — adapter implements TxCreator via Tx() (HijackTx + NewTxDriver), TxUser via WithTx() (NewTxClientFromRawConfig -> Client()), and Self(). All three are required by entutils.TransactingRepo. (`func (a *adapter) WithTx(ctx context.Context, tx *entutils.TxDriver) *adapter { txClient := entdb.NewTxClientFromRawConfig(ctx, *tx.GetConfig()); return &adapter{db: txClient.Client(), logger: a.logger} }`)
**Eager load rate cards with WithRatecards** — Every query returning an Addon calls query.WithRatecards(AddonEagerLoadRateCardsFn). PlanAddons are loaded only when Expand.PlanAddons is true via WithPlans(addonEagerLoadActivePlans). (`query = query.WithRatecards(AddonEagerLoadRateCardsFn)`)
**Soft delete via DeletedAt** — DeleteAddon sets DeletedAt to time.Now().UTC() via UpdateOneID; ListAddons excludes deleted rows with addondb.DeletedAtIsNil() unless IncludeDeleted is set. (`err = a.db.Addon.UpdateOneID(add.ID).Where(addondb.Namespace(add.Namespace)).SetDeletedAt(deletedAt).Exec(ctx)`)
**Status filtering via EffectiveFrom/EffectiveTo time predicates** — AddonStatus is derived at query time: active = EffectiveFromLTE(now) AND (EffectiveToGTE(now) OR nil), draft = both nil, archived = EffectiveToLT(now). Never compute status in Go after fetching all rows. (`predicates = append(predicates, addondb.And(addondb.EffectiveFromLTE(now), addondb.Or(addondb.EffectiveToGTE(now), addondb.EffectiveToIsNil())))`)
**Rate card updates replace all cards (delete + bulk-create)** — UpdateAddon deletes all existing AddonRateCard rows then calls rateCardBulkCreate. No diff logic — full replacement ensures consistency and key uniqueness. (`a.db.AddonRateCard.Delete().Where(addonratecarddb.AddonID(add.ID)).Exec(ctx)`)
**All type conversions live in mapping.go** — CRUD methods call FromAddonRow, FromAddonRateCardRow, FromPlanAddonRow, asAddonRateCardRow from mapping.go. Never embed conversion logic inside query methods. (`add, err := FromAddonRow(*addonRow)`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `adapter.go` | Config struct, New constructor with Validate(), adapter struct, and Tx/WithTx/Self transaction plumbing. | Never remove Tx/WithTx/Self — they are required by entutils.TransactingRepo. Always call config.Validate() before constructing. |
| `addon.go` | All CRUD + list methods (ListAddons, CreateAddon, GetAddon, UpdateAddon, DeleteAddon) plus rateCardBulkCreate helper and addonEagerLoadActivePlans closure. | Every method must wrap its fn body with TransactingRepo. GetAddon uses complex SQL subquery for latest-version-by-key lookup; prefer using the existing addondb predicates rather than raw SQL selectors. |
| `mapping.go` | Bidirectional mapping: FromAddonRow/FromAddonRateCardRow/FromPlanAddonRow (Ent->domain) and asAddonRateCardRow (domain->Ent). Also imports plan adapter mapping functions (FromPlanRow, FromPlanPhaseRow, FromPlanRateCardRow). | TaxConfig.BackfillTaxConfig must be called after mapping TaxCode edge. BillingCadence uses ISOString parse/stringify (r.BillingCadence.ParsePtrOrNil()), never raw string assignment. |
| `adapter_test.go` | Integration tests against real Postgres via pctestutils.NewTestEnv. Covers create/get/list/update/delete plus status-filter logic. | Tests use context.WithCancel — prefer t.Context() for new tests. TestEnv is closed in t.Cleanup. |

## Anti-Patterns

- Calling a.db.Foo() directly inside an adapter method without wrapping in entutils.TransactingRepo — this bypasses ctx-bound transactions.
- Embedding type-conversion logic inside CRUD methods instead of using mapping.go helpers.
- Querying add-ons without WithRatecards eager load — callers always expect rate cards populated.
- Hard-deleting rows instead of setting DeletedAt for soft delete.
- Building raw SQL predicates for status filtering instead of using the EffectiveFrom/EffectiveTo Ent predicate helpers.

## Decisions

- **TransactingRepo wraps every adapter method rather than passing *entdb.Tx explicitly.** — Ent transactions propagate implicitly via ctx; explicit tx parameters leak plumbing to every call site and prevent safe nesting via savepoints.
- **Rate card updates replace all existing cards (delete + bulk-create) rather than diffing.** — Simplifies consistency guarantees — no partial state after an update, ordering and key uniqueness enforced atomically without complex diff logic.

## Example: Add a new adapter method for an addon sub-resource

```
func (a *adapter) GetAddonFoo(ctx context.Context, params addon.GetFooInput) (*addon.Foo, error) {
	fn := func(ctx context.Context, a *adapter) (*addon.Foo, error) {
		if err := params.Validate(); err != nil {
			return nil, fmt.Errorf("invalid params: %w", err)
		}
		row, err := a.db.AddonFoo.Query().Where(...).WithRatecards(AddonEagerLoadRateCardsFn).First(ctx)
		if err != nil {
			if entdb.IsNotFound(err) { return nil, addon.NewNotFoundError(...) }
			return nil, fmt.Errorf("failed to get foo: %w", err)
		}
		return FromAddonFooRow(*row)
	}
	return entutils.TransactingRepo[*addon.Foo, *adapter](ctx, a, fn)
}
```

<!-- archie:ai-end -->
