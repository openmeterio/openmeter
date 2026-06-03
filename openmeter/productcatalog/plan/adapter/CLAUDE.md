# adapter

<!-- archie:ai-start -->

> Ent/PostgreSQL adapter implementing plan.Repository for plan lifecycle persistence (CRUD, phase management, rate card bulk creation). Primary constraint: every method body must wrap Ent calls in entutils.TransactingRepo to honor ctx-carried transactions.

## Patterns

**TransactingRepo wrapping on every method** — Every exported method body is a closure passed to entutils.TransactingRepo[T, *adapter](ctx, a, fn) so it rebinds to any tx in ctx; this is required even for reads. (`return entutils.TransactingRepo[*plan.Plan, *adapter](ctx, a, func(ctx context.Context, a *adapter) (*plan.Plan, error) { ... })`)
**Tx / WithTx / Self triad** — adapter implements Tx (HijackTx + NewTxDriver), WithTx (NewTxClientFromRawConfig returning a fresh *adapter), and Self(); all three must exist or TransactingRepo panics. (`func (a *adapter) WithTx(ctx context.Context, tx *entutils.TxDriver) *adapter { txClient := entdb.NewTxClientFromRawConfig(ctx, *tx.GetConfig()); return &adapter{db: txClient.Client(), logger: a.logger} }`)
**Config.Validate() on construction** — New(Config) calls config.Validate() to ensure Client and Logger are non-nil; Config satisfies models.Validator. (`func New(config Config) (plan.Repository, error) { if err := config.Validate(); err != nil { return nil, err } ... }`)
**Typed eager-load var helpers** — Reusable query modifiers (planPhaseEagerLoadRateCardsFn, rateCardEagerLoadFeaturesFn, rateCardEagerLoadTaxCodesFn, planEagerLoadActiveAddons) compose WithRatecards/WithFeatures/WithTaxCode so every query loads the full graph. (`query = query.WithPhases(planPhaseIncludeDeleted(false), planPhaseEagerLoadRateCardsFn)`)
**Soft-delete plan only; phases hard-deleted** — DeletePlan sets DeletedAt on the plan row; phases are hard-deleted on DeletePlan and on UpdatePlan reconciliation so pre-deletion plan state stays inspectable. Queries filter plandb.DeletedAtIsNil() unless IncludeDeleted. (`a.db.Plan.UpdateOneID(p.ID).Where(plandb.Namespace(p.Namespace)).SetDeletedAt(deletedAt).Exec(ctx)`)
**Status filtering via clock.Now() predicates** — ListPlans maps PlanStatus enum values to EffectiveFrom/EffectiveTo Ent predicates using clock.Now().UTC() so tests can override time via clock.SetTime. (`now := clock.Now().UTC(); predicates = append(predicates, plandb.And(plandb.EffectiveFromLTE(now), plandb.Or(plandb.EffectiveToGTE(now), plandb.EffectiveToIsNil())))`)
**Bulk rate card create then re-query** — After CreateBulk for rate cards, re-fetch the phase row with eager loads to return a fully populated struct instead of assembling in memory. (`planPhaseRow, _ = a.db.PlanPhase.Query().Where(...).WithRatecards(rateCardEagerLoadFeaturesFn, rateCardEagerLoadTaxCodesFn).First(ctx)`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `adapter.go` | Config struct, New(Config) constructor, and the Tx/WithTx/Self transaction rebinding plumbing. | Never store a tx-scoped client permanently on the struct; WithTx must return a new *adapter with the tx-bound db. |
| `plan.go` | Core plan CRUD (ListPlans, CreatePlan, GetPlan, UpdatePlan, DeletePlan). UpdatePlan phase reconciliation is delete-all + re-create via createPhase, not an incremental diff. | Callers of UpdatePlan must supply the FULL phase set when Phases != nil; nil Phases skips phase reconciliation entirely. |
| `phase.go` | createPhase helper and rateCardBulkCreate, called by CreatePlan/UpdatePlan. Both accept a raw *entdb.PlanRateCardClient. | Must always be called inside a TransactingRepo closure. rateCardBulkCreate builds []*entdb.PlanRateCardCreate but does NOT execute — the caller must CreateBulk(...).Exec(ctx). |
| `mapping.go` | Bidirectional Ent↔domain conversion: FromPlanRow, fromPlanPhaseRow, fromPlanRateCardRow, asPlanRateCardRow; uses Edges.AddonsOrErr() to distinguish not-loaded from loaded-empty. | New Ent schema fields must be mapped both directions or data silently drops. TaxConfig must call productcatalog.BackfillTaxConfig after mapping the TaxCode edge. |

## Anti-Patterns

- Calling a.db directly outside a TransactingRepo closure — bypasses ctx-carried transaction.
- Assembling plan.Plan from in-memory state after CreateBulk instead of re-querying with eager loads.
- Returning plain entdb.IsNotFound(err) without wrapping in plan.NewNotFoundError.
- Soft-deleting phases on plan deletion — breaks inspection of pre-deletion plan state.
- Directly editing openmeter/ent/db/ generated files instead of make generate.

## Decisions

- **All methods wrap Ent access in entutils.TransactingRepo even for reads.** — Ent transactions propagate via ctx; not rebinding causes a helper to fall off the caller's transaction, risking partial writes in multi-step operations.
- **Phases are deleted and re-created wholesale on UpdatePlan when params.Phases != nil.** — Correctness is simpler than incremental diffing; metadata-only updates pass nil Phases to skip reconciliation.
- **Plan status is derived from EffectiveFrom/EffectiveTo at query time, not a stored status column.** — Avoids dual-write consistency bugs and Atlas migrations when status logic changes.

## Example: Adapter method that queries and writes inside the caller's transaction

```
func (a *adapter) MyMethod(ctx context.Context, params plan.MyInput) (*plan.Plan, error) {
	fn := func(ctx context.Context, a *adapter) (*plan.Plan, error) {
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
		return FromPlanRow(*row)
	}
	return entutils.TransactingRepo[*plan.Plan, *adapter](ctx, a, fn)
// ...
```

<!-- archie:ai-end -->
