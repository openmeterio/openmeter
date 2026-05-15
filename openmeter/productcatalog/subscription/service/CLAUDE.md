# service

<!-- archie:ai-start -->

> Concrete implementation of plansubscription.PlanSubscriptionService; bridges the product catalog (plan key/version, status) to the generic subscription.Plan interface by resolving plans from the catalog, validating status, applying zeroPhasesBeforeStartingPhase, and delegating all persistence to subscriptionworkflow.Service.

## Patterns

**PlanInput dual-path: ref vs inline** — Every method accepting PlanInput first checks AsInput() for an inline custom plan (calling PlanFromPlanInput), then AsRef() for a catalog reference (calling getPlanByVersion with status validation). Both paths must end with the same WorkflowService delegation. (`if request.PlanInput.AsInput() != nil { p, err := PlanFromPlanInput(*request.PlanInput.AsInput()) } else if request.PlanInput.AsRef() != nil { p, err := s.getPlanByVersion(ctx, ns, *request.PlanInput.AsRef()) }`)
**Plan status validation before workflow delegation** — After fetching a plan by ref, always verify p.StatusAt(clock.Now()) == productcatalog.PlanStatusActive (Migrate also allows Archived). Return models.NewGenericValidationError if not satisfied. (`if pStatus != productcatalog.PlanStatusActive { return def, models.NewGenericValidationError(fmt.Errorf("plan %s@%d is not active at %s", p.Key, p.Version, now)) }`)
**zeroPhasesBeforeStartingPhase on StartingPhase** — When request.StartingPhase is set, always call s.zeroPhasesBeforeStartingPhase(p, *request.StartingPhase) before calling PlanFromPlan. This sets Duration to zero-duration ISODuration for phases before the named phase without removing them. (`if request.StartingPhase != nil { if err := s.zeroPhasesBeforeStartingPhase(p, *request.StartingPhase); err != nil { return def, err } }`)
**All persistence delegated to WorkflowService** — No DB writes occur in this package; only WorkflowService.CreateFromPlan and WorkflowService.ChangeToPlan are called for actual persistence. (`curr, new, err := s.WorkflowService.ChangeToPlan(ctx, request.ID, request.WorkflowInput, plan)`)
**Migrate timing auto-selection when nil** — When Migrate receives request.Timing == nil, it tries TimingImmediate first; if timing.ValidateForAction returns an error it falls back to TimingNextBillingCycle. (`timing = subscription.Timing{Enum: lo.ToPtr(subscription.TimingImmediate)}; if err := timing.ValidateForAction(subscription.SubscriptionActionCancel, &currView); err != nil { timing = subscription.Timing{Enum: lo.ToPtr(subscription.TimingNextBillingCycle)} }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `service.go` | Config struct (WorkflowService + SubscriptionService + PlanService + CustomerService + Logger), New() constructor returning plansubscription.PlanSubscriptionService, and zeroPhasesBeforeStartingPhase helper. | WorkflowService and SubscriptionService are intentionally separate fields; do not collapse them — they serve different roles. |
| `plan.go` | getPlanByVersion (wraps plan.Service.GetPlan, maps IsNotFound to subscription.NewPlanNotFoundError), PlanFromPlanInput (inline plan validation with temporary key='cheat'/version=1), PlanFromPlan. | PlanFromPlanInput sets Key='cheat' and Version=1 temporarily for validation; downstream code must not rely on those values being real. |
| `create.go` | Implements Create: resolves plan, validates status, applies zeroPhasesBeforeStartingPhase, calls WorkflowService.CreateFromPlan, returns sub.Subscription (not the full view). | Returns Subscription not SubscriptionView — callers needing the view must call GetView separately. |
| `migrate.go` | Implements Migrate: enforces version strictly greater than current (not equal), allows Active or Archived plan status, auto-selects timing when nil, delegates to ChangeToPlan. | Migrate allows Archived plans (unlike Change which requires Active only); the version strict-greater check is critical correctness. |

## Anti-Patterns

- Adding persistence (Ent/DB) calls directly in this service — all DB writes must go through WorkflowService or SubscriptionService.
- Removing or bypassing the validation cheat in PlanFromPlanInput without providing an alternative validation path for inline plans that have no real key/version yet.
- Calling plan.Service.GetPlan directly from a method instead of using getPlanByVersion — the helper normalises NotFound to subscription.NewPlanNotFoundError.
- Setting StartingPhase without calling zeroPhasesBeforeStartingPhase — phases before the starting phase will not collapse correctly.

## Decisions

- **Plan resolution and validation are separated from WorkflowService** — WorkflowService operates on the generic subscription.Plan interface and is unaware of the product catalog; PlanSubscriptionService bridges catalog concepts (key/version, status) to that interface.
- **zeroPhasesBeforeStartingPhase sets Duration=0 instead of deleting phases** — Deleting phases would change array indices and break keyed lookups; zero-duration phases are effectively skipped at the subscription layer without restructuring the phase list.

## Example: Adding a new plan-aware operation (e.g. ScheduleSubscription)

```
// service.go: add to Config if new dep needed
// new file schedule.go:
func (s *service) Schedule(ctx context.Context, request plansubscription.ScheduleSubscriptionRequest) (subscription.Subscription, error) {
    var def subscription.Subscription
    if err := request.PlanInput.Validate(); err != nil {
        return def, models.NewGenericValidationError(err)
    }
    var plan subscription.Plan
    if request.PlanInput.AsInput() != nil {
        p, err := PlanFromPlanInput(*request.PlanInput.AsInput())
        if err != nil { return def, err }
        plan = p
    } else if request.PlanInput.AsRef() != nil {
        p, err := s.getPlanByVersion(ctx, request.WorkflowInput.Namespace, *request.PlanInput.AsRef())
        if err != nil { return def, err }
// ...
```

<!-- archie:ai-end -->
