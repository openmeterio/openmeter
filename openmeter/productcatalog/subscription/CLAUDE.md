# subscription

<!-- archie:ai-start -->

> Bridges the productcatalog plan model to the subscription domain: defines PlanSubscriptionService (Create, Migrate, Change), PlanInput (ref vs inline discriminator), and adapter types (Plan, Phase, RateCard) that implement the subscription.Plan/PlanPhase/PlanRateCard interfaces. Primary constraint: all subscription DB writes go through subscriptionworkflow.Service, not directly through subscription.Service.

## Patterns

**PlanInput dual-path: ref vs inline plan.CreatePlanInput** — PlanInput holds either a *PlanRefInput (existing plan lookup) or *plan.CreatePlanInput (inline spec). Use FromRef/FromInput setters; AsRef()/AsInput() return nil for the unused path. (`var pi plansubscription.PlanInput; pi.FromRef(&PlanRefInput{Key: 'pro', Version: lo.ToPtr(2)})`)
**Plan/Phase/RateCard adapter types implement subscription.Plan/PlanPhase/PlanRateCard interfaces** — plan.go defines Plan, Phase, RateCard wrappers that implement ToCreateSubscriptionPlanInput, ToCreateSubscriptionPhasePlanInput, ToCreateSubscriptionItemPlanInput. (`p := &plansubscription.Plan{Plan: fetchedPlan.AsProductCatalogPlan(), Ref: &models.NamespacedID{ID: fetchedPlan.ID}}`)
**PlanSubscriptionService delegates persistence to WorkflowService** — Service layer resolves and validates the plan, then calls subscriptionworkflow.Service.CreateFromPlan / ChangeToPlan / Migrate. No direct subscription.Repository calls. (`return s.workflowService.CreateFromPlan(ctx, workflowInput, &adaptedPlan)`)
**zeroPhasesBeforeStartingPhase collapses earlier phases in place** — When StartingPhase is set, all phases before the starting phase have their Duration zeroed (not deleted) so the subscription begins at the correct phase. (`// service/create.go: zeroPhasesBeforeStartingPhase(plan, *request.StartingPhase)`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `service.go` | PlanSubscriptionService interface with Create, Migrate, Change; request types MigrateSubscriptionRequest, ChangeSubscriptionRequest, CreateSubscriptionRequest. | ChangeSubscriptionRequest.PlanInput and StartingPhase must both be set consistently; StartingPhase is only used when PlanInput.AsRef() is non-nil. |
| `plan.go` | Plan/Phase/RateCard adapter types that wrap productcatalog types and implement subscription.Plan interfaces; GetPhases() computes cumulative StartAfter durations. | Plan.GetPhases() computes StartAfter by accumulating phase durations in order — any reordering of Phases breaks timing. |

## Anti-Patterns

- Calling subscription.Service directly for create/change/migrate — always route through PlanSubscriptionService or WorkflowService.
- Calling plan.Service.GetPlan directly instead of the internal getPlanByVersion helper — the helper normalises NotFound to subscription.NewPlanNotFoundError.
- Setting StartingPhase without calling zeroPhasesBeforeStartingPhase — phases before the start will not be collapsed.
- Importing app/common from testutils — it must stay independent to avoid import cycles.
- Adding persistence calls directly in service/service.go — all DB writes go through WorkflowService.

## Decisions

- **PlanInput discriminator (ref vs inline) instead of separate Create methods** — Callers need to subscribe to either an existing stored plan or a transient inline spec without the service needing separate entry points; the discriminator makes the choice explicit.
- **Plan/Phase/RateCard adapter types implement subscription interfaces rather than using productcatalog types directly** — Subscription domain defines its own Plan/PlanPhase/PlanRateCard interfaces; the adapter types bridge productcatalog.Plan to those interfaces without circular imports.

<!-- archie:ai-end -->
