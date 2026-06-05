# adapter

<!-- archie:ai-start -->

> Ent-backed persistence layer for product-catalog add-ons, implementing the addon.Repository interface. Owns all SQL queries, eager-loading, and bidirectional mapping between Ent rows and domain addon types.

## Patterns

**Config-validated constructor returning the domain interface** — New(Config) validates Client+Logger via Config.Validate() (collects errs, errors.Join) and returns addon.Repository, not *adapter. Compile-time asserts: var _ models.Validator = (*Config)(nil) and var _ addon.Repository = (*adapter)(nil). (`func New(config Config) (addon.Repository, error) { if err := config.Validate(); err != nil { return nil, err } ... }`)
**Transaction-aware repo methods via entutils.TransactingRepo** — Every Repository method wraps its body in a closure `fn(ctx, a *adapter)` passed to entutils.TransactingRepo[T, *adapter](ctx, a, fn). The adapter implements Tx/WithTx/Self so it rebinds to the tx client carried in ctx. (`return entutils.TransactingRepo[*addon.Addon, *adapter](ctx, a, fn)`)
**Validate params first inside fn** — Each closure begins with `if err := params.Validate(); err != nil { return ..., fmt.Errorf("invalid ... parameters: %w", err) }` before touching the DB. (`if err := params.Validate(); err != nil { return nil, fmt.Errorf("invalid create add-on parameters: %w", err) }`)
**Ent predicate filters via package aliases + filter.ApplyToQuery** — Queries use addondb/addonratecarddb/planaddondb aliases for typed predicates and pkg/filter ApplyToQuery for FilterString/FilterULID fields (ID, Key, Name, Currency). Status filters are hand-built EffectiveFrom/EffectiveTo predicate sets. (`query = filter.ApplyToQuery(query, params.Key, addondb.FieldKey)`)
**FromXRow mappers + asAddonRateCardRow** — DB→domain via FromAddonRow/FromAddonRateCardRow/FromPlanAddonRow/FromPlanRow/FromPlanPhaseRow/FromPlanRateCardRow in mapping.go; domain→DB via asAddonRateCardRow. RateCard.Type switches on FlatFeeRateCardType vs UsageBasedRateCardType. (`add, err := FromAddonRow(*addonRow)`)
**Soft-delete and refetch-after-write** — DeleteAddon sets DeletedAt (no hard delete) and list/get exclude DeletedAtIsNil unless IncludeDeleted. Create/Update refetch with WithRatecards(AddonEagerLoadRateCardsFn) to populate subresources before mapping. (`SetDeletedAt(deletedAt); query = query.Where(addondb.DeletedAtIsNil())`)
**Ratecards replaced wholesale, not patched** — UpdateAddon deletes all AddonRateCard rows for the addon then bulk-recreates via rateCardBulkCreate when params.RateCards != nil; if nil, ratecards are left untouched. (`a.db.AddonRateCard.Delete().Where(addonratecarddb.AddonID(add.ID)).Exec(ctx)`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `adapter.go` | Config/New constructor and adapter struct with Tx/WithTx/Self plumbing for transaction.Driver. | WithTx must rebuild via entdb.NewTxClientFromRawConfig; never share the base *entdb.Client across tx boundaries. |
| `addon.go` | Repository CRUD: ListAddons, CreateAddon, GetAddon, UpdateAddon, DeleteAddon plus eager-load funcs AddonEagerLoadRateCardsFn and addonEagerLoadActivePlans. | GetAddon-by-key uses raw sql.Selector MAX(version) subquery for IncludeLatest; status predicates rely on clock.Now() so freeze time in tests. AddonStatusInvalid means EffectiveTo < EffectiveFrom. |
| `mapping.go` | Row<->domain mappers including tax-code backfill (BackfillTaxConfig) and feature-key workaround when only FeatureID is set. | FromAddonRateCardRow needs Edges.FeaturesOrErr/TaxCodeOrErr eagerly loaded or it errors; asAddonRateCardRow requires a known RateCard type (Flat/UsageBased) else returns error. |
| `adapter_test.go` | Postgres integration test (TestPostgresAdapter) driving CRUD + status-filter scenarios through pctestutils.NewTestEnv. | Uses clock.SetTime/ResetTime for status assertions; requires Postgres (POSTGRES_HOST). |

## Anti-Patterns

- Calling a.db directly outside a TransactingRepo closure, bypassing tx rebinding in ctx.
- Hard-deleting addon rows instead of SetDeletedAt soft delete.
- Mapping a ratecard row without eager-loading its Features/TaxCode edges (FromAddonRateCardRow will error).
- Returning *adapter from New instead of addon.Repository.
- Patching individual ratecards on update instead of delete-all + bulk recreate.

## Decisions

- **Ratecards are eager-loaded via shared AddonEagerLoadRateCardsFn on every read path.** — Domain Addon is incomplete without ratecards; centralizing the eager-load predicate keeps deleted-ratecard filtering consistent.
- **Versioning by (namespace, key, version) with a MAX(version) subquery for latest.** — Add-ons are immutable versions; latest-by-key lookups must resolve the highest version atomically in SQL.

## Example: Transaction-aware repository method skeleton

```
func (a *adapter) GetAddon(ctx context.Context, params addon.GetAddonInput) (*addon.Addon, error) {
	fn := func(ctx context.Context, a *adapter) (*addon.Addon, error) {
		if err := params.Validate(); err != nil {
			return nil, fmt.Errorf("invalid get add-on parameters: %w", err)
		}
		addonRow, err := a.db.Addon.Query().
			Where(addondb.And(addondb.Namespace(params.Namespace), addondb.ID(params.ID))).
			WithRatecards(AddonEagerLoadRateCardsFn).First(ctx)
		if err != nil {
			if entdb.IsNotFound(err) {
				return nil, addon.NewNotFoundError(addon.NotFoundErrorParams{Namespace: params.Namespace, ID: params.ID})
			}
			return nil, fmt.Errorf("failed to get add-on: %w", err)
		}
		return FromAddonRow(*addonRow)
// ...
```

<!-- archie:ai-end -->
