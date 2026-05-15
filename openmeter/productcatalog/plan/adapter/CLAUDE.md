# adapter

<!-- archie:ai-start -->

> Ent/PostgreSQL adapter implementing plan.Repository for plan lifecycle persistence (CRUD, phase management, rate card bulk creation). Primary constraint: every method body must wrap Ent calls in entutils.TransactingRepo to honor ctx-carried transactions.

## Patterns

**TransactingRepo wrapping on every method** — Every exported adapter method body must be a closure passed to entutils.TransactingRepo[T, *adapter](ctx, a, fn) so the call rebinds to any transaction already in ctx. Failure to do so silently reads/writes outside the caller's transaction. (`return entutils.TransactingRepo[*plan.Plan, *adapter](ctx, a, func(ctx context.Context, a *adapter) (*plan.Plan, error) { ... })`)
**Tx / WithTx / Self triad** — adapter implements Tx (HijackTx + NewTxDriver), WithTx (NewTxClientFromRawConfig returning a new *adapter), and Self() returning itself. All three must be present or TransactingRepo panics. (`func (a *adapter) WithTx(ctx context.Context, tx *entutils.TxDriver) *adapter { txClient := entdb.NewTxClientFromRawConfig(ctx, *tx.GetConfig()); return &adapter{db: txClient.Client(), logger: a.logger} }`)
**Config.Validate() on construction** — New(Config) calls config.Validate() which checks that Client and Logger are non-nil; returns error on failure. Config satisfies var _ models.Validator = (*Config)(nil). (`func New(config Config) (plan.Repository, error) { if err := config.Validate(); err != nil { return nil, err } ... }`)
**Typed eager-load var helpers** — Reusable query modifier vars (planPhaseEagerLoadRateCardsFn, rateCardEagerLoadFeaturesFn, rateCardEagerLoadTaxCodesFn, planEagerLoadActiveAddons) compose WithRatecards/WithFeatures/WithTaxCode so every query loads the full graph consistently. (`query = query.WithPhases(planPhaseIncludeDeleted(false), planPhaseEagerLoadRateCardsFn)`)
**Soft-delete via DeletedAt, phases NOT soft-deleted** — DeletePlan sets DeletedAt on the plan row only. Phases are hard-deleted on DeletePlan and on UpdatePlan phase reconciliation so pre-deletion state remains inspectable via the plan row. Queries filter with plandb.DeletedAtIsNil() unless IncludeDeleted is true. (`err = a.db.Plan.UpdateOneID(p.ID).Where(plandb.Namespace(p.Namespace)).SetDeletedAt(deletedAt).Exec(ctx)`)
**Status filtering via clock.Now() predicates** — ListPlans maps PlanStatus enum values (Draft, Active, Archived, Scheduled, Invalid) to EffectiveFrom/EffectiveTo Ent predicates using clock.Now().UTC() so tests can override time via clock.SetTime. (`now := clock.Now().UTC(); predicates = append(predicates, plandb.And(plandb.EffectiveFromLTE(now), plandb.Or(plandb.EffectiveToGTE(now), plandb.EffectiveToIsNil())))`)
**Bulk rate card create then re-query** — After CreateBulk for rate cards, the phase row is re-fetched with WithRatecards+WithFeatures+WithTaxCode eager loads to return a fully populated struct instead of relying on in-memory assembly. (`planPhaseRow, err = a.db.PlanPhase.Query().Where(...).WithRatecards(rateCardEagerLoadFeaturesFn, rateCardEagerLoadTaxCodesFn).First(ctx)`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `adapter.go` | Config struct, constructor New(Config), and the Tx/WithTx/Self transaction rebinding plumbing. | Never store a tx-scoped client on the struct permanently; WithTx must return a new *adapter with the tx-bound db. |
| `plan.go` | All core plan CRUD (ListPlans, CreatePlan, GetPlan, UpdatePlan, DeletePlan). UpdatePlan phase reconciliation uses delete-all + re-create via createPhase, not an incremental diff. | Callers of UpdatePlan must supply the FULL phase set when Phases != nil; nil Phases skips phase reconciliation entirely. |
| `phase.go` | createPhase helper and rateCardBulkCreate — internal helpers called by CreatePlan and UpdatePlan. Both accept a raw *entdb.PlanRateCardClient so they must always be called from within a TransactingRepo closure. | rateCardBulkCreate builds []*entdb.PlanRateCardCreate but does NOT execute; the caller must call CreateBulk(...).Exec(ctx). |
| `mapping.go` | Bidirectional Ent↔domain conversion: FromPlanRow, fromPlanPhaseRow, fromPlanRateCardRow, asPlanRateCardRow. Uses p.Edges.AddonsOrErr() to distinguish 'not loaded' (nil) from 'loaded empty slice'. | New plan fields added to the Ent schema must be mapped in both directions; a missing field silently drops data. TaxConfig must call productcatalog.BackfillTaxConfig after mapping TaxCode edge. |

## Anti-Patterns

- Calling a.db directly outside a TransactingRepo closure — bypasses ctx-carried transaction.
- Assembling plan.Plan from in-memory state after CreateBulk instead of re-querying with eager loads.
- Returning plain entdb.IsNotFound(err) without wrapping in plan.NewNotFoundError — callers expect domain errors.
- Soft-deleting phases on plan deletion — breaks ability to inspect pre-deletion plan state.
- Directly editing openmeter/ent/db/ generated files — always regenerate with make generate.

## Decisions

- **All methods wrap Ent access in entutils.TransactingRepo even for reads.** — Ent transactions propagate via ctx; not rebinding causes a helper to fall off the caller's transaction, risking partial writes during multi-step operations.
- **Phases are deleted and re-created wholesale on UpdatePlan when params.Phases != nil.** — Simplifies correctness over incremental diffing; callers that only change plan metadata pass nil Phases to skip phase reconciliation.
- **Plan status is derived from EffectiveFrom/EffectiveTo at query time using clock predicates, not stored in a separate status column.** — Avoids dual-write consistency bugs; no separate status field means Atlas migrations are not needed when status logic changes.

## Example: Add a new adapter method that queries and writes inside the caller's transaction

```
func (a *adapter) MyMethod(ctx context.Context, params plan.MyInput) (*plan.Plan, error) {
	fn := func(ctx context.Context, a *adapter) (*plan.Plan, error) {
		if err := params.Validate(); err != nil {
			return nil, fmt.Errorf("invalid params: %w", err)
		}
		row, err := a.db.Plan.Query().
			Where(plandb.Namespace(params.Namespace), plandb.ID(params.ID)).
			WithPhases(planPhaseIncludeDeleted(false), planPhaseEagerLoadRateCardsFn).
			First(ctx)
		if err != nil {
			if entdb.IsNotFound(err) {
				return nil, plan.NewNotFoundError(plan.NotFoundErrorParams{Namespace: params.Namespace, ID: params.ID})
			}
			return nil, fmt.Errorf("failed to get Plan: %w", err)
		}
// ...
```

<!-- archie:ai-end -->
