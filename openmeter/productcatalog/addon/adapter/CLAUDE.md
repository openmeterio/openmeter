# adapter

<!-- archie:ai-start -->

> Ent/PostgreSQL adapter implementing addon.Repository: CRUD + list for add-ons and their rate cards. All operations wrap adapter methods with entutils.TransactingRepo so the ctx-bound transaction is always honored.

## Patterns

**TransactingRepo wrapping** — Every exported adapter method wraps its body in entutils.TransactingRepo[T, *adapter](ctx, a, fn) to rebind to any caller-supplied Ent transaction carried in ctx. (`return entutils.TransactingRepo[*addon.Addon, *adapter](ctx, a, fn)`)
**Tx / WithTx / Self trio** — adapter implements transaction.Driver via Tx() (HijackTx), WithTx() (returns new *adapter bound to txClient), and Self() — required by entutils.TransactingRepo. (`func (a *adapter) WithTx(ctx context.Context, tx *entutils.TxDriver) *adapter { txClient := entdb.NewTxClientFromRawConfig(ctx, *tx.GetConfig()); return &adapter{db: txClient.Client(), ...} }`)
**Config + Validate constructor** — New(Config) validates all required fields (Client, Logger) before constructing. Config implements models.Validator. (`var _ models.Validator = (*Config)(nil)`)
**Eager loading via WithRatecards** — All queries that return an Addon call query.WithRatecards(AddonEagerLoadRateCardsFn) so rate cards are always populated; PlanAddons are loaded only when Expand.PlanAddons is true. (`query = query.WithRatecards(AddonEagerLoadRateCardsFn)`)
**mapping.go: FromAddonRow converts Ent row to domain type** — All type conversions live in mapping.go (FromAddonRow, FromAddonRateCardRow, FromPlanAddonRow, asAddonRateCardRow). Do not embed conversion logic inside CRUD methods. (`add, err := FromAddonRow(*addonRow)`)
**Soft delete via DeletedAt** — DeleteAddon sets DeletedAt to time.Now().UTC() via UpdateOneID; ListAddons excludes deleted rows with addondb.DeletedAtIsNil() unless IncludeDeleted is set. (`err = a.db.Addon.UpdateOneID(add.ID).SetDeletedAt(deletedAt).Exec(ctx)`)
**Status filtering via time predicates** — AddonStatus is derived from EffectiveFrom/EffectiveTo columns at query time: active = EffectiveFrom<=now AND (EffectiveTo>=now OR nil), draft = both nil, archived = EffectiveTo<now. (`predicates = append(predicates, addondb.And(addondb.EffectiveFromLTE(now), addondb.Or(addondb.EffectiveToGTE(now), addondb.EffectiveToIsNil())))`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `adapter.go` | Config struct, New constructor, adapter struct, Tx/WithTx/Self transaction plumbing. | Never remove Tx/WithTx/Self — they are required by entutils.TransactingRepo. Always validate Config before constructing. |
| `addon.go` | All CRUD + list methods for add-ons and their rate cards. Contains rateCardBulkCreate helper and addonEagerLoadActivePlans closure. | Every method must wrap its fn body with TransactingRepo. UpdateAddon replaces all rate cards by delete+bulk-create; partial updates on individual cards are not supported. |
| `mapping.go` | Bidirectional mapping between Ent rows and domain types: FromAddonRow, FromAddonRateCardRow, FromPlanAddonRow, asAddonRateCardRow, plus helpers that reuse plan/phase/ratecard converters from the plan adapter. | TaxConfig.BackfillTaxConfig must be called after mapping TaxCode edge to ensure legacy field compatibility. BillingCadence uses ISOString parse/stringify, never raw string. |
| `adapter_test.go` | Integration tests against a real Postgres DB using pctestutils.NewTestEnv. Tests cover create/get/list/update/delete and status-filter logic. | Tests use context.WithCancel — prefer t.Context() for new tests. TestEnv is closed in t.Cleanup. |

## Anti-Patterns

- Accepting *entdb.Client directly in helper functions without wrapping the body with entutils.TransactingRepo — this bypasses the ctx-bound transaction.
- Embedding type-conversion logic inside CRUD methods instead of mapping.go.
- Querying add-ons without WithRatecards eager load — callers expect rate cards always populated.
- Hard-deleting rows instead of setting DeletedAt for soft delete.
- Building raw SQL predicates for status filtering instead of using the established EffectiveFrom/EffectiveTo Ent predicates.

## Decisions

- **TransactingRepo wraps every adapter method rather than passing *entdb.Tx explicitly.** — Ent transactions are carried implicitly in ctx; explicit tx parameters leak plumbing to every call site and make helpers harder to compose.
- **Rate card updates replace all existing cards (delete + bulk-create) rather than diffing.** — Simplifies consistency guarantees — no partial state can exist after an update, and ordering/key uniqueness is enforced atomically.

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
