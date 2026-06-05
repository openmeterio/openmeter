# adapter

<!-- archie:ai-start -->

> Ent-backed persistence layer for plan↔addon assignments (the PlanAddon join entity). Implements the planaddon.Repository interface and is the only place that touches the entdb.PlanAddon table.

## Patterns

**Transaction-aware repo methods** — Every public method wraps its body in entutils.TransactingRepo[...] so it rebinds to a tx carried in ctx; the inner fn takes (ctx, *adapter). (`return entutils.TransactingRepo[*planaddon.PlanAddon, *adapter](ctx, a, fn)`)
**Transaction driver trio** — adapter implements Tx (HijackTx), WithTx (NewTxClientFromRawConfig), and Self() to satisfy the entutils transactable contract. (`func (a *adapter) WithTx(ctx, tx) *adapter { txClient := entdb.NewTxClientFromRawConfig(ctx, *tx.GetConfig()); ... }`)
**Validate params first** — Each method calls params.Validate() before any DB access and wraps the error with namespace/plan.id/addon.id context. (`if err := params.Validate(); err != nil { return nil, fmt.Errorf("invalid ...: %w", err) }`)
**Eager-load via shared loader vars** — All reads attach PlanEagerLoadPhasesWithRateCardsWithFeaturesFn and AddonEagerLoadRateCardsWithFeaturesFn so edges (phases→ratecards→features) are populated before mapping. (`query.WithPlan(PlanEagerLoadPhasesWithRateCardsWithFeaturesFn).WithAddon(AddonEagerLoadRateCardsWithFeaturesFn)`)
**Create-then-refetch** — CreatePlanAddon does Save() then re-queries by ID with eager loads to populate sub-resources before mapping to domain. (`planAddonRow, err = a.db.PlanAddon.Query().Where(...).WithPlan(...).WithAddon(...).First(ctx)`)
**Soft delete** — DeletePlanAddon sets DeletedAt = clock.Now().UTC() via UpdateOneID; reads filter planaddondb.DeletedAtIsNil() unless IncludeDeleted. (`a.db.PlanAddon.UpdateOneID(planAddon.ID).SetDeletedAt(deletedAt).Exec(ctx)`)
**Typed not-found errors** — On entdb.IsNotFound, return planaddon.NewNotFoundError(planaddon.NotFoundErrorParams{...}) rather than the raw ent error. (`if entdb.IsNotFound(err) { return nil, planaddon.NewNotFoundError(...) }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `adapter.go` | Config{Client,Logger} + Validate, New() returning planaddon.Repository, and the Tx/WithTx/Self transaction plumbing. | New returns the interface, not *adapter; both Client and Logger are required (validated). |
| `planaddon.go` | All CRUD: ListPlanAddons, CreatePlanAddon, GetPlanAddon, UpdatePlanAddon, DeletePlanAddon plus the two eager-load loader vars. | List builds OR-of-OR predicate trees (plan/addon sub-filters via HasPlanWith/HasAddonWith); PlanKeyVersions loop has a dead planKeyVersionFilters slice that is appended empty — mirror existing style, don't rely on it. Update uses SetOrClearMaxQuantity (nil clears). |
| `mapping.go` | FromPlanAddunRow maps entdb.PlanAddon → planaddon.PlanAddon, delegating to planadapter.FromPlanRow and addonadapter.FromAddonRow for edges. | If Edges.Plan/Edges.Addon is nil it falls back to a stub with only NamespacedID; non-nil-but-cast-nil is treated as an error. |
| `adapter_test.go` | Postgres integration test (TestPostgresAdapter) using pctestutils.NewTestEnv; drives repository directly via env.PlanAddonRepository. | Requires Postgres (POSTGRES_HOST=127.0.0.1) and env.DBSchemaMigrate(t); exercises full plan+addon+feature setup before assignment. |

## Anti-Patterns

- Calling a.db directly in a method without wrapping in entutils.TransactingRepo (breaks tx propagation from ctx).
- Reading PlanAddon rows without the shared eager-load loaders, leaving Plan/Addon edges unpopulated for mapping.
- Returning raw entdb errors instead of planaddon.NewNotFoundError on IsNotFound.
- Skipping params.Validate() before DB access.
- Hard-deleting rows instead of setting DeletedAt.

## Decisions

- **Repository returns fully eager-loaded domain PlanAddon (with nested plan phases/ratecards/features).** — Callers (service, http) need the full plan and addon shape to validate compatibility and render API responses without N+1 refetches.
- **Create refetches after Save rather than mapping the insert result.** — The insert result lacks edges; a single eager-loaded refetch guarantees a complete domain object.

<!-- archie:ai-end -->
