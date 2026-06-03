# service

<!-- archie:ai-start -->

> Concrete plansubscription.PlanSubscriptionService — bridges the product catalog (plan key/version, status) to the generic subscription.Plan interface by resolving plans, validating status, applying zeroPhasesBeforeStartingPhase, and delegating all persistence to subscriptionworkflow.Service.

## Patterns

**PlanInput dual-path: ref vs inline** — Methods accepting PlanInput check AsInput() for an inline custom plan (PlanFromPlanInput), then AsRef() for a catalog reference (getPlanByVersion with status validation); both end in WorkflowService delegation. (`if request.PlanInput.AsInput() != nil { p, err := PlanFromPlanInput(*request.PlanInput.AsInput()) } else if request.PlanInput.AsRef() != nil { p, err := s.getPlanByVersion(ctx, ns, *request.PlanInput.AsRef()) }`)
**Plan status validation before workflow delegation** — After fetching by ref, verify p.StatusAt(clock.Now()) == PlanStatusActive (Migrate also allows Archived); else return models.NewGenericValidationError. (`if pStatus != productcatalog.PlanStatusActive { return def, models.NewGenericValidationError(fmt.Errorf("plan %s@%d is not active at %s", p.Key, p.Version, now)) }`)
**zeroPhasesBeforeStartingPhase on StartingPhase** — When request.StartingPhase is set, call s.zeroPhasesBeforeStartingPhase(p, *request.StartingPhase) before PlanFromPlan; it sets zero-duration ISODuration on earlier phases without removing them. (`if request.StartingPhase != nil { if err := s.zeroPhasesBeforeStartingPhase(p, *request.StartingPhase); err != nil { return def, err } }`)
**All persistence delegated to WorkflowService** — No DB writes in this package; only WorkflowService.CreateFromPlan and ChangeToPlan persist. (`curr, new, err := s.WorkflowService.ChangeToPlan(ctx, request.ID, request.WorkflowInput, plan)`)
**Migrate timing auto-selection when nil** — Migrate tries TimingImmediate first; if ValidateForAction fails it falls back to TimingNextBillingCycle. (`timing = subscription.Timing{Enum: lo.ToPtr(subscription.TimingImmediate)}; if err := timing.ValidateForAction(...); err != nil { timing = subscription.Timing{Enum: lo.ToPtr(subscription.TimingNextBillingCycle)} }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `service.go` | Config (WorkflowService + SubscriptionService + PlanService + CustomerService + Logger), New() returning PlanSubscriptionService, zeroPhasesBeforeStartingPhase helper. | WorkflowService and SubscriptionService are separate fields serving different roles — do not collapse them. |
| `plan.go` | getPlanByVersion (maps IsNotFound to subscription.NewPlanNotFoundError), PlanFromPlanInput (inline validation with temporary key='cheat'/version=1), PlanFromPlan. | PlanFromPlanInput sets Key='cheat', Version=1 temporarily for validation; downstream must not rely on those being real. |
| `create.go` | Create: resolves plan, validates status, applies zeroPhasesBeforeStartingPhase, calls WorkflowService.CreateFromPlan, returns subscription.Subscription. | Returns Subscription not SubscriptionView — callers needing the view call GetView separately. |
| `migrate.go` | Migrate: enforces version strictly greater than current, allows Active or Archived status, auto-selects timing when nil, delegates to ChangeToPlan. | Migrate allows Archived plans (unlike Change which requires Active); the strict-greater version check is critical correctness. |

## Anti-Patterns

- Adding Ent/DB calls directly — all DB writes go through WorkflowService or SubscriptionService.
- Removing the validation 'cheat' in PlanFromPlanInput without an alternative validation path for inline plans.
- Calling plan.Service.GetPlan directly instead of getPlanByVersion — the helper normalises NotFound to subscription.NewPlanNotFoundError.
- Setting StartingPhase without calling zeroPhasesBeforeStartingPhase — earlier phases won't collapse correctly.

## Decisions

- **Plan resolution/validation separated from WorkflowService.** — WorkflowService operates on the generic subscription.Plan interface and is catalog-unaware; PlanSubscriptionService bridges catalog key/version/status to it.
- **zeroPhasesBeforeStartingPhase sets Duration=0 instead of deleting phases.** — Deleting phases would shift array indices and break keyed lookups; zero-duration phases are effectively skipped without restructuring.

## Example: Adding a new plan-aware operation (ScheduleSubscription)

```
func (s *service) Schedule(ctx context.Context, request plansubscription.ScheduleSubscriptionRequest) (subscription.Subscription, error) {
    var def subscription.Subscription
    if err := request.PlanInput.Validate(); err != nil { return def, models.NewGenericValidationError(err) }
    var plan subscription.Plan
    if request.PlanInput.AsInput() != nil {
        p, err := PlanFromPlanInput(*request.PlanInput.AsInput())
        if err != nil { return def, err }
        plan = p
    } else if request.PlanInput.AsRef() != nil {
        p, err := s.getPlanByVersion(ctx, request.WorkflowInput.Namespace, *request.PlanInput.AsRef())
        if err != nil { return def, err }
        plan = PlanFromPlan(*p)
    }
    // delegate to s.WorkflowService.CreateFromPlan
    return def, nil
// ...
```

<!-- archie:ai-end -->
