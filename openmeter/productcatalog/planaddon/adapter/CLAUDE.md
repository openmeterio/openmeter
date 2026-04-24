# adapter

<!-- archie:ai-start -->

> Ent/PostgreSQL adapter implementing planaddon.Repository — the persistence layer for plan-to-addon assignment records. Owns all SQL queries, soft-delete logic, and eager-loading of related plan and addon entities.

## Patterns

**TransactingRepo wrapping every method** — Every exported adapter method (ListPlanAddons, CreatePlanAddon, GetPlanAddon, UpdatePlanAddon, DeletePlanAddon) wraps its body in entutils.TransactingRepo[T, *adapter](ctx, a, fn) so the ctx-bound transaction is always honoured. (`return entutils.TransactingRepo[*planaddon.PlanAddon, *adapter](ctx, a, fn)`)
**Tx / WithTx / Self trinity** — adapter implements the transaction.Driver protocol via Tx (hijacks a new tx), WithTx (rebinds to caller's tx using entdb.NewTxClientFromRawConfig), and Self (returns a*adapter). Required by entutils.TransactingRepo. (`func (a *adapter) WithTx(ctx context.Context, tx *entutils.TxDriver) *adapter { txClient := entdb.NewTxClientFromRawConfig(ctx, *tx.GetConfig()); return &adapter{db: txClient.Client(), logger: a.logger} }`)
**Validated Config constructor** — New(Config) validates required fields (Client, Logger) via Config.Validate() implementing models.Validator before constructing the adapter, returning an error rather than panicking. (`var _ models.Validator = (*Config)(nil)`)
**Post-create refetch with eager loads** — After Save(), the adapter re-queries the new row with WithPlan(PlanEagerLoadPhasesWithRateCardsWithFeaturesFn).WithAddon(AddonEagerLoadRateCardsWithFeaturesFn) to populate nested edges before returning. (`planAddonRow, err = a.db.PlanAddon.Query().Where(...).WithPlan(PlanEagerLoadPhasesWithRateCardsWithFeaturesFn).WithAddon(AddonEagerLoadRateCardsWithFeaturesFn).First(ctx)`)
**Soft-delete via SetDeletedAt + clock.Now()** — Delete sets DeletedAt to clock.Now().UTC() via UpdateOneID rather than hard-deleting. Queries filter with planaddondb.DeletedAtIsNil() unless IncludeDeleted is set. (`err = a.db.PlanAddon.UpdateOneID(planAddon.ID).Where(planaddondb.Namespace(...)).SetDeletedAt(deletedAt).Exec(ctx)`)
**Domain-specific NotFoundError wrapping** — entdb.IsNotFound(err) is caught and re-wrapped as planaddon.NewNotFoundError(NotFoundErrorParams{...}) so callers receive a typed domain error, not a raw Ent error. (`if entdb.IsNotFound(err) { return nil, planaddon.NewNotFoundError(planaddon.NotFoundErrorParams{Namespace: params.Namespace, ID: params.ID}) }`)
**Shared eager-load function vars** — PlanEagerLoadPhasesWithRateCardsWithFeaturesFn and AddonEagerLoadRateCardsWithFeaturesFn are package-level vars reused across all queries to keep edge loading consistent. (`var PlanEagerLoadPhasesWithRateCardsWithFeaturesFn = func(pq *entdb.PlanQuery) { pq.WithPhases(func(ppq *entdb.PlanPhaseQuery) { ppq.WithRatecards(func(prq *entdb.PlanRateCardQuery) { prq.WithFeatures() }) }) }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `adapter.go` | Defines Config, constructor New, and the adapter struct with Tx/WithTx/Self. No query logic lives here. | Must implement transaction.TxCreator interface exactly (Tx, WithTx, Self) or TransactingRepo will fail to rebind. |
| `planaddon.go` | All CRUD method implementations. Each method closure is handed to entutils.TransactingRepo. | Avoid calling a.db directly outside the TransactingRepo closure — the closure's `a` is the tx-rebound copy. |
| `mapping.go` | FromPlanAddonRow converts entdb.PlanAddon to planaddon.PlanAddon, delegating plan/addon edge mapping to planadapter.FromPlanRow and addonadapter.FromAddonRow. | Edges are optional (nil when not eager-loaded); always guard with `if a.Edges.Plan != nil`. |
| `adapter_test.go` | Integration tests against a real Postgres schema using pctestutils.NewTestEnv. Tests cover Create/Get/List/Update/Delete via env.PlanAddonRepository. | env.DBSchemaMigrate(t) must be called before any query; uses context.Background() (pre-t.Context() era — keep consistent with existing style in this file). |

## Anti-Patterns

- Calling a.db directly inside a method without wrapping in entutils.TransactingRepo — bypasses ctx-bound transactions.
- Hard-deleting rows via Delete() instead of soft-deleting via SetDeletedAt.
- Returning raw entdb.IsNotFound errors — always wrap in planaddon.NewNotFoundError.
- Adding business-logic validation (plan status checks, conflict detection) inside the adapter — those belong in the service layer.
- Skipping eager-load functions when re-fetching after create/update — callers expect fully populated Plan and Addon edges.

## Decisions

- **TransactingRepo pattern for every method** — Ent transactions propagate implicitly in ctx; using the raw *entdb.Client would fall off a caller-supplied transaction and cause partial writes under concurrency.
- **Post-create refetch instead of using Save() return value** — Ent's Save() does not populate edges; a separate query with WithPlan/WithAddon is the only way to return a fully hydrated domain object.
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
