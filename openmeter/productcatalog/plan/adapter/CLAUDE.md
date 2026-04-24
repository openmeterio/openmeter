# adapter

<!-- archie:ai-start -->

> Ent/PostgreSQL adapter implementing plan.Repository for plan lifecycle persistence (CRUD, phase management, rate card bulk creation). Primary constraint: all mutations must wrap Ent calls in entutils.TransactingRepo to honor ctx-carried transactions.

## Patterns

**TransactingRepo wrapping** — Every adapter method body must be a closure passed to entutils.TransactingRepo[T, *adapter](ctx, a, fn) so the call rebinds to any transaction already in ctx. (`return entutils.TransactingRepo[*plan.Plan, *adapter](ctx, a, fn)`)
**Config.Validate() on construction** — Constructor New(Config) calls config.Validate() first and returns error if Client or Logger are nil; Config implements models.Validator. (`var _ models.Validator = (*Config)(nil)`)
**WithTx / Self pattern** — adapter implements WithTx(ctx, *TxDriver) *adapter and Self() *adapter so entutils.TransactingRepo can rebind the db field to a tx-scoped client. (`func (a *adapter) WithTx(ctx context.Context, tx *entutils.TxDriver) *adapter { txClient := entdb.NewTxClientFromRawConfig(ctx, *tx.GetConfig()); return &adapter{db: txClient.Client(), logger: a.logger} }`)
**Eager load with typed helper vars** — Reusable query modifier vars (planPhaseEagerLoadRateCardsFn, rateCardEagerLoadFeaturesFn, rateCardEagerLoadTaxCodesFn, planEagerLoadActiveAddons) compose WithRatecards/WithFeatures/WithTaxCode so every query loads the full graph consistently. (`query = query.WithPhases(planPhaseIncludeDeleted(false), planPhaseEagerLoadRateCardsFn)`)
**Soft-delete via DeletedAt** — DeletePlan sets DeletedAt timestamp; phases are intentionally NOT soft-deleted so state before deletion remains inspectable. Queries filter with plandb.DeletedAtIsNil() unless IncludeDeleted is true. (`err = a.db.Plan.UpdateOneID(p.ID).Where(plandb.Namespace(p.Namespace)).SetDeletedAt(deletedAt).Exec(ctx)`)
**Status filtering via clock.Now()** — ListPlans maps PlanStatus enum values to EffectiveFrom/EffectiveTo predicates using clock.Now().UTC() so tests can override time via clock.SetTime. (`now := clock.Now().UTC(); predicates = append(predicates, plandb.And(plandb.EffectiveFromLTE(now), plandb.Or(plandb.EffectiveToGTE(now), plandb.EffectiveToIsNil())))`)
**Bulk rate card create then re-query** — After CreateBulk for rate cards, the phase row is re-fetched with eager loads (WithRatecards+WithFeatures+WithTaxCode) to return a fully populated struct instead of relying on in-memory assembly. (`planPhaseRow, err = a.db.PlanPhase.Query().Where(...).WithRatecards(rateCardEagerLoadFeaturesFn, rateCardEagerLoadTaxCodesFn).First(ctx)`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `adapter.go` | Config, constructor, Tx/WithTx/Self methods — the transaction rebinding plumbing lives here. | Never store a tx-scoped client on the struct permanently; WithTx returns a new *adapter. |
| `plan.go` | ListPlans, CreatePlan, GetPlan, UpdatePlan, DeletePlan — all core CRUD wrapped in TransactingRepo. | UpdatePlan phase replacement uses delete-all + re-create, not an incremental diff; callers must supply the full phase set. |
| `phase.go` | createPhase helper and rateCardBulkCreate — internal helpers used by CreatePlan and UpdatePlan. | rateCardBulkCreate accepts raw *entdb.PlanRateCardClient; ensure it is always called from within a TransactingRepo closure so it inherits the tx. |
| `mapping.go` | FromPlanRow, FromPlanPhaseRow, fromPlanRateCardRow, asPlanRateCardRow — bidirectional Ent↔domain conversion. Checks Edges.AddonsOrErr() to distinguish 'not loaded' (nil) from 'loaded empty slice'. | New plan fields added to the Ent schema must be mapped here in both directions; missing field silently drops data. |

## Anti-Patterns

- Calling a.db directly outside a TransactingRepo closure — bypasses ctx-carried transaction.
- Assembling plan.Plan from in-memory state after CreateBulk instead of re-querying with eager loads.
- Using entdb.IsNotFound without wrapping in plan.NewNotFoundError — callers expect domain errors.
- Soft-deleting phases on plan deletion — breaks ability to inspect pre-deletion plan state (see comment in DeletePlan).
- Directly editing openmeter/ent/db/ generated files — always regenerate with make generate.

## Decisions

- **All methods wrap Ent access in entutils.TransactingRepo even for reads.** — Ent transactions propagate via ctx; not rebinding causes a helper to fall off the caller's transaction, risking partial writes.
- **Phases are deleted and re-created wholesale on UpdatePlan when params.Phases != nil.** — Simplifies correctness over incremental diffing; callers that only change plan metadata pass nil Phases to skip phase reconciliation.
- **Status filtering built from clock predicates at query time rather than a stored status column.** — Plan status is a derived function of EffectiveFrom/EffectiveTo; no separate status field avoids dual-write consistency bugs.

## Example: Add a new adapter method that reads and writes inside the caller's transaction

```
func (a *adapter) MyMethod(ctx context.Context, params plan.MyInput) (*plan.Plan, error) {
	fn := func(ctx context.Context, a *adapter) (*plan.Plan, error) {
		if err := params.Validate(); err != nil {
			return nil, fmt.Errorf("invalid params: %w", err)
		}
		row, err := a.db.Plan.Query().Where(plandb.Namespace(params.Namespace), plandb.ID(params.ID)).
			WithPhases(planPhaseIncludeDeleted(false), planPhaseEagerLoadRateCardsFn).
			First(ctx)
		if err != nil {
			if entdb.IsNotFound(err) {
				return nil, plan.NewNotFoundError(plan.NotFoundErrorParams{Namespace: params.Namespace, ID: params.ID})
			}
			return nil, fmt.Errorf("failed to get Plan: %w", err)
		}
		return FromPlanRow(*row)
// ...
```

<!-- archie:ai-end -->
