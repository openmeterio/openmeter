# adapter

<!-- archie:ai-start -->

> Ent-backed persistence layer implementing plan.Repository for Plans, PlanPhases, and PlanRateCards. All DB access for the plan domain flows through here; the service layer never touches Ent directly.

## Patterns

**Transaction-aware repository methods** — Every public adapter method wraps its body in entutils.TransactingRepo[T, *adapter](ctx, a, fn) so it rebinds to any tx already carried in ctx. The adapter implements Tx/WithTx/Self to participate in transaction.Driver. (`return entutils.TransactingRepo[*plan.Plan, *adapter](ctx, a, fn)`)
**Config struct + Validate + New returning interface** — Constructor New(Config) (plan.Repository, error) validates Config (Client and Logger required) before building the unexported adapter struct. var _ plan.Repository = (*adapter)(nil) enforces the contract. (`func New(config Config) (plan.Repository, error) { if err := config.Validate(); err != nil { return nil, err } ... }`)
**FromXRow / asXRow mapping functions in mapping.go** — Ent row -> domain conversion lives in exported FromPlanRow/FromAddonRow/fromPlanPhaseRow/fromPlanRateCardRow; domain -> Ent in asPlanRateCardRow. RateCard type switch on r.Type dispatches FlatFee vs UsageBased. (`func FromPlanRow(p entdb.Plan) (*plan.Plan, error)`)
**Eager-load edges via OrErr to detect unloaded relations** — Mapping uses p.Edges.AddonsOrErr() / RatecardsOrErr() / TaxCodeOrErr() to distinguish 'not loaded' (set nil) from real data; queries attach WithPhases/WithRatecards using shared eager-load funcs (rateCardEagerLoadFeaturesFn, rateCardEagerLoadTaxCodesFn, planPhaseEagerLoadRateCardsFn). (`addons, err := p.Edges.AddonsOrErr(); if err != nil { pp.Addons = nil }`)
**Status filtering via EffectivePeriod predicates** — ListPlans translates productcatalog.PlanStatus values into Ent predicates over EffectiveFrom/EffectiveTo against clock.Now(); Active/Draft/Scheduled/Archived/Invalid each map to a distinct predicate combined with plandb.Or. (`plandb.And(plandb.EffectiveFromLTE(now), plandb.Or(plandb.EffectiveToGTE(now), plandb.EffectiveToIsNil()))`)
**Bulk RateCard creation under phase create** — createPhase builds child rate cards with rateCardBulkCreate then CreateBulk(...).Exec, re-queries the phase with eager loads, and maps it back. TaxConfig/TaxCodeID/TaxBehavior and Price are set conditionally (only when non-nil). (`bulk, err := rateCardBulkCreate(a.db.PlanRateCard, params.RateCards, planPhaseRow.ID, params.Namespace)`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `adapter.go` | Config/New constructor + adapter struct + Tx/WithTx/Self transaction plumbing | Both Client and Logger are required; New returns plan.Repository not *adapter. |
| `mapping.go` | Ent<->domain conversion for Plan/Addon/PlanAddon/RateCard/Phase | BackfillTaxConfig must be called to reconcile legacy TaxConfig with new TaxBehavior/TaxCode columns; FeatureKey is backfilled from the loaded feature edge when only FeatureID is set. |
| `phase.go` | createPhase + createPhaseInput validation + rateCardBulkCreate | Index is uint8 in DB (SetIndex(uint8(...))); validate Namespace/Key/PlanID/Name/Index>=0 before insert. |
| `plan.go` | ListPlans/CreatePlan/DeletePlan and other Plan CRUD with status-filter predicates | CreatePlan defaults Version to 1; OrderBy switch falls through to ByID; phases are created one-by-one via a.createPhase inside the same tx. |
| `adapter_test.go` | Postgres integration test via pctestutils.NewTestEnv exercising create/get/list/update/delete/status-filter | Uses env.Plan vs env.PlanRepository (service vs raw repo) deliberately; SettlementMode persistence is asserted on the repo path. |

## Anti-Patterns

- Calling a.db.* outside an entutils.TransactingRepo/TransactingRepoWithNoValue wrapper, breaking tx propagation from ctx
- Returning the concrete *adapter from New instead of plan.Repository
- Reading p.Edges.X directly without the *OrErr guard, conflating 'not loaded' with 'empty'
- Skipping BackfillTaxConfig when mapping rate cards, leaving legacy TaxConfig inconsistent with TaxBehavior/TaxCode columns
- Encoding business rules (status transitions, version increment policy) here instead of the service layer

## Decisions

- **Adapter owns only persistence + mapping; validation of business invariants lives in service** — Keeps Ent concerns isolated and lets the service compose multi-step transactional workflows (feature/tax resolution, events).
- **Eager-load helpers and OrErr-based loaded detection** — Plans are deep aggregates (phases->ratecards->tax/feature); explicit load signalling avoids N+1 and silent empty-vs-missing bugs.

## Example: Transaction-aware list with status predicates

```
func (a *adapter) ListPlans(ctx context.Context, params plan.ListPlansInput) (pagination.Result[plan.Plan], error) {
  fn := func(ctx context.Context, a *adapter) (pagination.Result[plan.Plan], error) {
    if err := params.Validate(); err != nil { return pagination.Result[plan.Plan]{}, err }
    query := a.db.Plan.Query().WithPhases(planPhaseIncludeDeleted(false), planPhaseEagerLoadRateCardsFn)
    // ... apply namespace/status/order predicates ...
    paged, err := query.Paginate(ctx, params.Page)
    if err != nil { return pagination.Result[plan.Plan]{}, err }
    // map paged.Items via FromPlanRow
    return response, nil
  }
  return entutils.TransactingRepo[pagination.Result[plan.Plan], *adapter](ctx, a, fn)
}
```

<!-- archie:ai-end -->
