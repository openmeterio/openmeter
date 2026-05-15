# subscription

<!-- archie:ai-start -->

> Bridges the productcatalog plan model to the subscription domain: defines PlanSubscriptionService (Create, Migrate, Change), PlanInput (ref vs inline discriminator), and Plan/Phase/RateCard adapter types that implement the subscription.Plan/PlanPhase/PlanRateCard interfaces. All subscription DB writes go through subscriptionworkflow.Service, never directly through subscription.Service.

## Patterns

**PlanInput dual-path: ref vs inline plan.CreatePlanInput** — PlanInput holds either a *PlanRefInput (existing plan lookup by key/version) or *plan.CreatePlanInput (inline spec). Use FromRef/FromInput setters; AsRef()/AsInput() return nil for the unused path. Validate() returns error if both are nil. (`var pi plansubscription.PlanInput; pi.FromRef(&PlanRefInput{Key: "pro", Version: lo.ToPtr(2)})`)
**Plan/Phase/RateCard adapter types implement subscription interfaces** — plan.go defines Plan, Phase, RateCard wrappers that implement subscription.Plan, PlanPhase, PlanRateCard. Plan.GetPhases() computes cumulative StartAfter durations by accumulating phase durations in order — reordering Phases breaks timing. (`p := &plansubscription.Plan{Plan: fetchedPlan.AsProductCatalogPlan(), Ref: &models.NamespacedID{ID: fetchedPlan.ID}}`)
**PlanSubscriptionService delegates all persistence to WorkflowService** — Service layer resolves and validates the plan, then calls subscriptionworkflow.Service.CreateFromPlan / ChangeToPlan / Migrate. No direct subscription.Repository or Ent calls allowed in this service. (`return s.workflowService.CreateFromPlan(ctx, workflowInput, &adaptedPlan)`)
**zeroPhasesBeforeStartingPhase collapses earlier phases in place** — When StartingPhase is set, all phases before the starting phase have their Duration zeroed (not deleted). Must be called on the plan before passing to WorkflowService when StartingPhase is specified. (`// service/create.go: zeroPhasesBeforeStartingPhase(plan, *request.StartingPhase)`)
**Custom-plan discriminator sniff via marshal+unmarshal in HTTP handler** — In the http/ sub-package, custom-plan detection uses marshal+unmarshal to discriminate plan type rather than oapi-codegen As* helpers, which succeed on wrong types and silently produce zero values. (`// http/create.go: marshal body to JSON then unmarshal into plan-specific struct to detect type`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `service.go` | PlanSubscriptionService interface with Create, Migrate, Change; request types MigrateSubscriptionRequest, ChangeSubscriptionRequest, CreateSubscriptionRequest. | ChangeSubscriptionRequest.PlanInput and StartingPhase must be set consistently; StartingPhase only applies when PlanInput.AsRef() is non-nil. |
| `plan.go` | Plan/Phase/RateCard adapter types wrapping productcatalog types to implement subscription.Plan interfaces; Plan.GetPhases() accumulates StartAfter durations. | Plan.GetPhases() accumulates StartAfter by summing phase durations in order — any reordering of Phases breaks subscription timing. |

## Anti-Patterns

- Calling subscription.Service directly for create/change/migrate — always route through PlanSubscriptionService or WorkflowService.
- Calling plan.Service.GetPlan directly instead of the internal getPlanByVersion helper — the helper normalises NotFound to subscription.NewPlanNotFoundError.
- Setting StartingPhase without calling zeroPhasesBeforeStartingPhase — phases before the start will not be collapsed.
- Importing app/common from testutils — it must stay independent to avoid import cycles.
- Adding persistence calls directly in service/service.go — all DB writes go through WorkflowService.

## Decisions

- **PlanInput discriminator (ref vs inline) instead of separate Create methods** — Callers need to subscribe to either an existing stored plan or a transient inline spec without the service needing separate entry points; the discriminator makes the choice explicit and validates that exactly one path is set.
- **Plan/Phase/RateCard adapter types implement subscription interfaces rather than using productcatalog types directly** — Subscription domain defines its own Plan/PlanPhase/PlanRateCard interfaces; the adapter types bridge productcatalog.Plan to those interfaces without circular imports between productcatalog and subscription packages.

## Example: Resolving a plan by ref and delegating subscription creation to WorkflowService

```
import (
    plansubscription "github.com/openmeterio/openmeter/openmeter/productcatalog/subscription"
)

func (s *service) Create(ctx context.Context, req plansubscription.CreateSubscriptionRequest) (*subscription.Subscription, error) {
    pi := req.PlanInput
    if ref := pi.AsRef(); ref != nil {
        fetchedPlan, err := s.getPlanByVersion(ctx, ns, ref.Key, ref.Version)
        if err != nil { return nil, err }
        adaptedPlan := &plansubscription.Plan{
            Plan: fetchedPlan.AsProductCatalogPlan(),
            Ref:  &models.NamespacedID{ID: fetchedPlan.ID},
        }
        if req.StartingPhase != nil {
            zeroPhasesBeforeStartingPhase(adaptedPlan, *req.StartingPhase)
// ...
```

<!-- archie:ai-end -->
