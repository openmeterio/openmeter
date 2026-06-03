# adapter

<!-- archie:ai-start -->

> Ent/PostgreSQL adapter implementing planaddon.Repository — the persistence layer for plan-to-addon assignment records. Owns all SQL, soft-delete logic, and eager-loading of related plan and addon entities via the Tx/WithTx/Self triad required by entutils.TransactingRepo.

## Patterns

**TransactingRepo wrapping every method** — Every exported method wraps its body in entutils.TransactingRepo[T, *adapter](ctx, a, fn). The fn closure receives a tx-rebound *adapter; never call a.db outside the closure. (`return entutils.TransactingRepo[*planaddon.PlanAddon, *adapter](ctx, a, fn)`)
**Tx/WithTx/Self triad** — adapter implements Tx (a.db.HijackTx + entutils.NewTxDriver), WithTx (entdb.NewTxClientFromRawConfig), and Self (returns a). All three are required for TransactingRepo to rebind. (`func (a *adapter) WithTx(ctx, tx *entutils.TxDriver) *adapter { txClient := entdb.NewTxClientFromRawConfig(ctx, *tx.GetConfig()); return &adapter{db: txClient.Client(), logger: a.logger} }`)
**Post-create/update refetch with eager loads** — After Save()/Update(), re-query with WithPlan(PlanEagerLoadPhasesWithRateCardsWithFeaturesFn).WithAddon(AddonEagerLoadRateCardsWithFeaturesFn) because Ent Save() does not populate edges. (`row, err = a.db.PlanAddon.Query().Where(...).WithPlan(PlanEagerLoadPhasesWithRateCardsWithFeaturesFn).WithAddon(AddonEagerLoadRateCardsWithFeaturesFn).First(ctx)`)
**Soft-delete via SetDeletedAt + clock.Now()** — Delete sets DeletedAt to clock.Now().UTC() via UpdateOneID, never hard-deletes. List/Get add planaddondb.DeletedAtIsNil() unless IncludeDeleted is set. (`a.db.PlanAddon.UpdateOneID(planAddon.ID).Where(planaddondb.Namespace(...)).SetDeletedAt(clock.Now().UTC()).Exec(ctx)`)
**Typed NotFoundError wrapping** — Catch entdb.IsNotFound(err) and re-wrap as planaddon.NewNotFoundError(NotFoundErrorParams{...}) so callers get a domain-typed error. (`if entdb.IsNotFound(err) { return nil, planaddon.NewNotFoundError(planaddon.NotFoundErrorParams{Namespace: params.Namespace, ID: params.ID}) }`)
**Shared package-level eager-load vars** — PlanEagerLoadPhasesWithRateCardsWithFeaturesFn and AddonEagerLoadRateCardsWithFeaturesFn are reused across queries to keep edge loading consistent. (`var PlanEagerLoadPhasesWithRateCardsWithFeaturesFn = func(pq *entdb.PlanQuery) { pq.WithPhases(func(ppq *entdb.PlanPhaseQuery) { ppq.WithRatecards(func(prq *entdb.PlanRateCardQuery) { prq.WithFeatures() }) }) }`)
**Validated Config constructor** — New(Config) validates required fields (Client, Logger) via Config.Validate() and errors rather than panics. var _ models.Validator = (*Config)(nil) enforces the interface. (`var _ models.Validator = (*Config)(nil)`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `adapter.go` | Config, New constructor, adapter struct with Tx/WithTx/Self. No query logic. | Must implement the full TxCreator+TxUser triad exactly or TransactingRepo fails to rebind. var _ planaddon.Repository = (*adapter)(nil) must pass. |
| `planaddon.go` | All CRUD method implementations; each closure passed to entutils.TransactingRepo. | Never call a.db outside a TransactingRepo closure. DeletePlanAddon must keep the get-then-soft-delete pattern. |
| `mapping.go` | FromPlanAddonRow converts entdb.PlanAddon to planaddon.PlanAddon, delegating edge mapping to planadapter.FromPlanRow and addonadapter.FromAddonRow. | Edges are optional — guard with if a.Edges.Plan != nil / if a.Edges.Addon != nil before dereferencing. |
| `adapter_test.go` | Integration tests against real Postgres via pctestutils.NewTestEnv, covering Create/Get/List/Update/Delete. | env.DBSchemaMigrate(t) must run before any query. Tests use context.Background() (pre-t.Context() style) — keep consistent. |

## Anti-Patterns

- Calling a.db directly without wrapping in entutils.TransactingRepo — bypasses ctx-bound transaction.
- Hard-deleting rows via Delete() instead of SetDeletedAt soft-delete.
- Returning raw entdb.IsNotFound errors instead of planaddon.NewNotFoundError.
- Adding business-logic validation (plan status, conflict detection) in the adapter — belongs in the service.
- Skipping eager-load vars when re-fetching after create/update — callers expect populated Plan/Addon edges.

## Decisions

- **TransactingRepo for every method.** — Ent transactions propagate via ctx; the raw *entdb.Client falls off a caller transaction and causes partial writes under concurrency.
- **Post-create refetch instead of Save() return value.** — Ent's Save() does not populate relation edges; a query with WithPlan/WithAddon is the only way to return a hydrated object.
- **Soft-delete over hard-delete.** — Assignments must stay queryable (with IncludeDeleted) for audit and reconciliation after removal.

## Example: Add a new mutation method following the adapter pattern

```
func (a *adapter) SomeMutation(ctx context.Context, params planaddon.SomeMutationInput) (*planaddon.PlanAddon, error) {
  fn := func(ctx context.Context, a *adapter) (*planaddon.PlanAddon, error) {
    if err := params.Validate(); err != nil { return nil, fmt.Errorf("invalid params: %w", err) }
    row, err := a.db.PlanAddon.UpdateOneID(params.ID).Where(planaddondb.Namespace(params.Namespace)).SetSomeField(params.SomeField).Save(ctx)
    if err != nil {
      if entdb.IsNotFound(err) { return nil, planaddon.NewNotFoundError(planaddon.NotFoundErrorParams{Namespace: params.Namespace, ID: params.ID}) }
      return nil, err
    }
    return FromPlanAddonRow(*row)
  }
  return entutils.TransactingRepo[*planaddon.PlanAddon, *adapter](ctx, a, fn)
}
```

<!-- archie:ai-end -->
