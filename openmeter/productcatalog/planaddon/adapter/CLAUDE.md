# adapter

<!-- archie:ai-start -->

> Ent/PostgreSQL adapter implementing planaddon.Repository — the persistence layer for plan-to-addon assignment records. Owns all SQL queries, soft-delete logic, and eager-loading of related plan and addon entities via the Tx/WithTx/Self triad required by entutils.TransactingRepo.

## Patterns

**TransactingRepo wrapping every method** — Every exported adapter method wraps its body in entutils.TransactingRepo[T, *adapter](ctx, a, fn). The fn closure receives a tx-rebound *adapter; never call a.db directly outside this closure. (`return entutils.TransactingRepo[*planaddon.PlanAddon, *adapter](ctx, a, fn)`)
**Tx/WithTx/Self triad** — adapter implements Tx (via a.db.HijackTx + entutils.NewTxDriver), WithTx (via entdb.NewTxClientFromRawConfig), and Self (returns a). All three are required for TransactingRepo to rebind to caller-supplied transactions. (`func (a *adapter) WithTx(ctx context.Context, tx *entutils.TxDriver) *adapter { txClient := entdb.NewTxClientFromRawConfig(ctx, *tx.GetConfig()); return &adapter{db: txClient.Client(), logger: a.logger} }`)
**Post-create/update refetch with eager loads** — After Save() or Update(), re-query with WithPlan(PlanEagerLoadPhasesWithRateCardsWithFeaturesFn).WithAddon(AddonEagerLoadRateCardsWithFeaturesFn) because Ent Save() does not populate edges. (`planAddonRow, err = a.db.PlanAddon.Query().Where(...).WithPlan(PlanEagerLoadPhasesWithRateCardsWithFeaturesFn).WithAddon(AddonEagerLoadRateCardsWithFeaturesFn).First(ctx)`)
**Soft-delete via SetDeletedAt + clock.Now()** — Delete sets DeletedAt to clock.Now().UTC() via UpdateOneID, never hard-deletes. List/Get queries add planaddondb.DeletedAtIsNil() unless IncludeDeleted is set. (`a.db.PlanAddon.UpdateOneID(planAddon.ID).Where(planaddondb.Namespace(...)).SetDeletedAt(clock.Now().UTC()).Exec(ctx)`)
**Typed NotFoundError wrapping** — entdb.IsNotFound(err) must be caught and re-wrapped as planaddon.NewNotFoundError(NotFoundErrorParams{...}) so callers receive a domain-typed error instead of a raw Ent error. (`if entdb.IsNotFound(err) { return nil, planaddon.NewNotFoundError(planaddon.NotFoundErrorParams{Namespace: params.Namespace, ID: params.ID}) }`)
**Shared package-level eager-load vars** — PlanEagerLoadPhasesWithRateCardsWithFeaturesFn and AddonEagerLoadRateCardsWithFeaturesFn are package-level vars reused across all queries to keep edge loading consistent across methods. (`var PlanEagerLoadPhasesWithRateCardsWithFeaturesFn = func(pq *entdb.PlanQuery) { pq.WithPhases(func(ppq *entdb.PlanPhaseQuery) { ppq.WithRatecards(func(prq *entdb.PlanRateCardQuery) { prq.WithFeatures() }) }) }`)
**Validated Config constructor** — New(Config) validates required fields (Client, Logger) via Config.Validate() — returns an error rather than panicking if any field is nil. var _ models.Validator = (*Config)(nil) enforces the interface at compile time. (`var _ models.Validator = (*Config)(nil)`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `adapter.go` | Defines Config, constructor New, and the adapter struct with Tx/WithTx/Self. No query logic lives here. | Must implement the full TxCreator+TxUser triad exactly or TransactingRepo will fail to rebind. var _ planaddon.Repository = (*adapter)(nil) compile-check must pass. |
| `planaddon.go` | All CRUD method implementations. Each method closure is passed to entutils.TransactingRepo. | Avoid calling a.db directly outside a TransactingRepo closure — always use the closure's rebound *adapter. For DeletePlanAddon, get-then-soft-delete pattern must stay consistent. |
| `mapping.go` | FromPlanAddonRow converts entdb.PlanAddon to planaddon.PlanAddon, delegating plan/addon edge mapping to planadapter.FromPlanRow and addonadapter.FromAddonRow. | Edges are optional — always guard with `if a.Edges.Plan != nil` and `if a.Edges.Addon != nil` before dereferencing. |
| `adapter_test.go` | Integration tests against a real Postgres schema using pctestutils.NewTestEnv. Tests cover Create/Get/List/Update/Delete via env.PlanAddonRepository. | env.DBSchemaMigrate(t) must be called before any query. Tests use context.Background() (pre-t.Context() era) — keep consistent with existing style. |

## Anti-Patterns

- Calling a.db directly in a method body without wrapping in entutils.TransactingRepo — bypasses ctx-bound transaction.
- Hard-deleting rows via Delete() instead of soft-deleting via SetDeletedAt.
- Returning raw entdb.IsNotFound errors — always wrap in planaddon.NewNotFoundError.
- Adding business-logic validation (plan status checks, conflict detection) inside the adapter — those belong in the service layer.
- Skipping eager-load function vars when re-fetching after create/update — callers expect fully populated Plan and Addon edges.

## Decisions

- **TransactingRepo pattern for every method** — Ent transactions propagate implicitly in ctx; using the raw *entdb.Client falls off a caller-supplied transaction and causes partial writes under concurrency.
- **Post-create refetch instead of using Save() return value** — Ent's Save() does not populate relation edges; a separate query with WithPlan/WithAddon is the only way to return a fully hydrated domain object.
- **Soft-delete over hard-delete** — Plan-addon assignments must remain queryable (with IncludeDeleted) for audit and reconciliation after removal.

## Example: Add a new mutation method following the established adapter pattern

```
func (a *adapter) SomeMutation(ctx context.Context, params planaddon.SomeMutationInput) (*planaddon.PlanAddon, error) {
	fn := func(ctx context.Context, a *adapter) (*planaddon.PlanAddon, error) {
		if err := params.Validate(); err != nil {
			return nil, fmt.Errorf("invalid params: %w", err)
		}
		row, err := a.db.PlanAddon.UpdateOneID(params.ID).
			Where(planaddondb.Namespace(params.Namespace)).
			SetSomeField(params.SomeField).
			Save(ctx)
		if err != nil {
			if entdb.IsNotFound(err) {
				return nil, planaddon.NewNotFoundError(planaddon.NotFoundErrorParams{Namespace: params.Namespace, ID: params.ID})
			}
			return nil, fmt.Errorf("failed to mutate: %w", err)
		}
// ...
```

<!-- archie:ai-end -->
