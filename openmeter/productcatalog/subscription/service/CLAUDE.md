# service

<!-- archie:ai-start -->

> Concrete implementation of plansubscription.PlanSubscriptionService; orchestrates plan resolution (by ref or inline input), status validation, zeroPhasesBeforeStartingPhase, and delegates to subscriptionworkflow.Service for create/change/migrate lifecycle operations.

## Patterns

**PlanInput dual-path: ref vs inline** — Every method that accepts a PlanInput checks AsInput() vs AsRef(); ref path calls getPlanByVersion then validates PlanStatusActive; inline path calls PlanFromPlanInput (which uses a key/version cheat for validation) (`if request.PlanInput.AsInput() != nil { p, err := PlanFromPlanInput(*request.PlanInput.AsInput()) } else if request.PlanInput.AsRef() != nil { p, err := s.getPlanByVersion(...) }`)
**Plan status validation before change** — Before calling WorkflowService, verify p.StatusAt(clock.Now()) == productcatalog.PlanStatusActive; Migrate additionally allows Archived; return models.NewGenericValidationError if not satisfied (`if pStatus != productcatalog.PlanStatusActive { return def, models.NewGenericValidationError(fmt.Errorf("plan %s@%d is not active", ...)) }`)
**zeroPhasesBeforeStartingPhase mutates plan phases in place** — When StartingPhase is set, phases before the named phase get Duration set to time.Duration(0) ISODuration rather than being removed; phases after are unchanged (`if request.StartingPhase != nil { if err := s.zeroPhasesBeforeStartingPhase(p, *request.StartingPhase); err != nil { return def, err } }`)
**Delegation to WorkflowService for actual persistence** — service.go delegates all subscription writes to subscriptionworkflow.Service.CreateFromPlan / ChangeToPlan; the plan subscription service only resolves and validates the plan (`curr, new, err := s.WorkflowService.ChangeToPlan(ctx, request.ID, request.WorkflowInput, plan)`)
**Migrate timing auto-selection** — When request.Timing is nil, Migrate tries TimingImmediate first; if ValidateForAction returns an error it falls back to TimingNextBillingCycle (`timing = subscription.Timing{Enum: lo.ToPtr(subscription.TimingImmediate)}; if err := timing.ValidateForAction(subscription.SubscriptionActionCancel, &currView); err != nil { timing = subscription.Timing{...NextBillingCycle} }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `service.go` | Config struct and service constructor; zeroPhasesBeforeStartingPhase helper; New() returns plansubscription.PlanSubscriptionService | Config embeds WorkflowService and SubscriptionService as separate fields; do not collapse them — they serve different roles |
| `plan.go` | getPlanByVersion (wraps plan.Service.GetPlan, maps IsNotFound to subscription.NewPlanNotFoundError); PlanFromPlanInput (inline plan validation cheat); PlanFromPlan (wraps plan.Plan as plansubscription.Plan with Ref) | PlanFromPlanInput temporarily sets Key='cheat' and Version=1 for validation to pass, then unsets them — downstream code must not rely on those values |
| `create.go` | Implements Create: resolves plan, validates status, optionally zeroes pre-starting phases, calls WorkflowService.CreateFromPlan | Returns sub.Subscription from the view, not the full view — callers needing the view must call GetView separately |
| `migrate.go` | Implements Migrate: enforces version > current, validates plan is active or archived, auto-selects timing when nil, calls ChangeToPlan | Migrate allows archived plans (unlike Change which requires active); version equality check is strict greater-than |

## Anti-Patterns

- Adding persistence calls directly in this service — all DB writes go through WorkflowService or SubscriptionService
- Removing the validation cheat in PlanFromPlanInput without providing an alternative validation path for inline plans that have no key/version yet
- Calling plan.Service.GetPlan directly from a method instead of going through getPlanByVersion — the helper normalises NotFound to subscription.NewPlanNotFoundError
- Setting StartingPhase without calling zeroPhasesBeforeStartingPhase — phases will not be collapsed correctly

## Decisions

- **Plan resolution and validation separated from WorkflowService** — WorkflowService operates on the generic subscription.Plan interface and is unaware of the catalog; PlanSubscriptionService bridges catalog concepts (key/version, status) to that interface
- **zeroPhasesBeforeStartingPhase sets Duration=0 rather than deleting phases** — Deleting phases would change array indices and break keyed lookups; zero-duration phases are effectively skipped without restructuring the phase list

<!-- archie:ai-end -->
