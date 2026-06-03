# subscription

<!-- archie:ai-start -->

> Bridge package (package plansubscription) connecting the productcatalog plan model to the subscription domain: defines PlanSubscriptionService (Create, Migrate, Change), PlanInput (ref vs inline discriminator), and Plan/Phase/RateCard adapter types implementing the subscription.Plan/PlanPhase/PlanRateCard interfaces; children split into http (v1 driver), service (concrete bridge), and testutils. All subscription DB writes go through subscriptionworkflow.Service, never directly through subscription.Service.

## Patterns

**PlanInput dual-path discriminator** — PlanInput holds either a *PlanRefInput (existing plan by key/version) or *plan.CreatePlanInput (inline spec). Use FromRef/FromInput setters; AsRef()/AsInput() return nil for the unused path; Validate() errors if both are nil. (`var pi plansubscription.PlanInput; pi.FromRef(&PlanRefInput{Key: "pro", Version: lo.ToPtr(2)})`)
**Adapter types implement subscription interfaces** — plan.go defines Plan, Phase, RateCard wrappers implementing subscription.Plan/PlanPhase/PlanRateCard. Plan.GetPhases() accumulates StartAfter by summing phase durations in order — reordering Phases breaks timing. (`p := &plansubscription.Plan{Plan: fetchedPlan.AsProductCatalogPlan(), Ref: &models.NamespacedID{ID: fetchedPlan.ID}}`)
**All persistence delegated to WorkflowService** — The service resolves and validates the plan, then calls subscriptionworkflow.Service.CreateFromPlan / ChangeToPlan / Migrate. No direct subscription.Repository or Ent calls in this package. (`return s.workflowService.CreateFromPlan(ctx, workflowInput, &adaptedPlan)`)
**zeroPhasesBeforeStartingPhase collapses earlier phases** — When StartingPhase is set, phases before it have Duration zeroed (not deleted) before passing the plan to WorkflowService. (`// service/create.go: zeroPhasesBeforeStartingPhase(plan, *request.StartingPhase)`)
**Custom-plan discriminator sniff via marshal+unmarshal** — In the http/ child, custom-plan detection marshals then unmarshals into a plan-specific struct rather than using oapi-codegen As* helpers, which succeed on wrong types and silently produce zero values. (`// http/create.go: marshal body to JSON then unmarshal into plan-specific struct to detect type`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `service.go` | PlanSubscriptionService interface (Create, Migrate, Change) and request types. | ChangeSubscriptionRequest.StartingPhase only applies when PlanInput.AsRef() is non-nil; set both consistently. |
| `plan.go` | Plan/Phase/RateCard adapter types wrapping productcatalog types to implement subscription.Plan interfaces. | Plan.GetPhases() accumulates StartAfter by summing phase durations in order — any reordering of Phases breaks subscription timing. |

## Anti-Patterns

- Calling subscription.Service directly for create/change/migrate — always route through PlanSubscriptionService or WorkflowService.
- Calling plan.Service.GetPlan directly instead of the getPlanByVersion helper — the helper normalises NotFound to subscription.NewPlanNotFoundError.
- Setting StartingPhase without calling zeroPhasesBeforeStartingPhase — earlier phases won't collapse.
- Importing app/common from testutils — it must stay independent to avoid import cycles.
- Adding persistence calls directly in service/service.go — all DB writes go through WorkflowService.

## Decisions

- **PlanInput discriminator (ref vs inline) instead of separate Create methods.** — Callers may subscribe to an existing stored plan or a transient inline spec without separate entry points; the discriminator validates exactly one path is set.
- **Plan/Phase/RateCard adapter types implement subscription interfaces rather than using productcatalog types directly.** — Subscription defines its own Plan/PlanPhase/PlanRateCard interfaces; the adapter types bridge productcatalog.Plan to them without circular imports.

## Example: Resolving a plan by ref and delegating subscription creation to WorkflowService

```
import plansubscription "github.com/openmeterio/openmeter/openmeter/productcatalog/subscription"

func (s *service) Create(ctx context.Context, req plansubscription.CreateSubscriptionRequest) (*subscription.Subscription, error) {
    if ref := req.PlanInput.AsRef(); ref != nil {
        fetchedPlan, err := s.getPlanByVersion(ctx, ns, ref.Key, ref.Version)
        if err != nil { return nil, err }
        adaptedPlan := &plansubscription.Plan{Plan: fetchedPlan.AsProductCatalogPlan(), Ref: &models.NamespacedID{ID: fetchedPlan.ID}}
        if req.StartingPhase != nil {
            zeroPhasesBeforeStartingPhase(adaptedPlan, *req.StartingPhase)
        }
        return s.workflowService.CreateFromPlan(ctx, workflowInput, adaptedPlan)
    }
    // ... inline plan path ...
}
```

<!-- archie:ai-end -->
