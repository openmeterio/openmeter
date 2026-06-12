# service

<!-- archie:ai-start -->

> Service implementation (package service) of plansubscription.PlanSubscriptionService — the bridge that turns a plan (inline CreatePlanInput or a published plan reference) into a subscription via the subscription workflow layer. Owns Create, Change, Migrate plus plan-resolution/validation; it does not persist anything itself, it orchestrates WorkflowService and PlanService.

## Patterns

**Config struct + New returns the interface** — service embeds Config{WorkflowService, SubscriptionService, PlanService, Logger, CustomerService}; New(c Config) returns plansubscription.PlanSubscriptionService, never the concrete *service. (`func New(c Config) plansubscription.PlanSubscriptionService { return &service{Config: c} }`)
**PlanInput dispatch: AsInput vs AsRef** — Create/Change first call request.PlanInput.Validate() (wrap failures in NewGenericValidationError), then branch: AsInput() -> PlanFromPlanInput, AsRef() -> getPlanByVersion + status/deleted checks, else hard error 'should have validated already'. (`if request.PlanInput.AsInput() != nil { plan, _ = PlanFromPlanInput(*request.PlanInput.AsInput()) } else if request.PlanInput.AsRef() != nil { ... }`)
**Plan lifecycle validation before subscribing** — Referenced plans are validated against clock.Now(): reject if DeletedAt is past, and require PlanStatusActive (Migrate also allows PlanStatusArchived). Errors are models.NewGenericValidationError. (`if p.StatusAt(now) != productcatalog.PlanStatusActive { return def, models.NewGenericValidationError(...) }`)
**StartingPhase zeroes earlier phases** — When request.StartingPhase is set, zeroPhasesBeforeStartingPhase sets each earlier phase Duration to ISODurationFromDuration(0) (does not delete them) and errors if the starting phase key is never found. (`phase.Duration = lo.ToPtr(datetime.ISODurationFromDuration(time.Duration(0)))`)
**Delegation to WorkflowService** — Create calls WorkflowService.CreateFromPlan; Change/Migrate call WorkflowService.ChangeToPlan returning (current, next). This service never writes the subscription directly. (`curr, new, err := s.WorkflowService.ChangeToPlan(ctx, request.ID, request.WorkflowInput, plan)`)
**Plan-not-found normalization** — getPlanByVersion uses defaultx.WithDefault(ref.Version, 0) (plan service treats 0 as latest) and maps plan.IsNotFound(err)/nil plan to subscription.NewPlanNotFoundError(planKey, version). (`if plan.IsNotFound(err) { return nil, subscription.NewPlanNotFoundError(planKey, version) }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `service.go` | Config, service struct, New, and zeroPhasesBeforeStartingPhase helper. | Config.WorkflowService and SubscriptionService are noted as potentially mergeable (TODO); all methods assume both are present. |
| `create.go` | Create: builds subscription.Plan then WorkflowService.CreateFromPlan, returns subView.Subscription. | Status check differs from Change (no archived allowance); keep validation order deleted-then-status. |
| `change.go` | Change: resolves plan, validates active-only, returns SubscriptionChangeResponse{Current, Next}. | Unlike Create, Change requires PlanStatusActive strictly and returns both current and next views. |
| `migrate.go` | Migrate: fetches current sub, requires PlanRef, resolves target version, allows Active OR Archived, computes timing. | Rejects target version <= current; timing falls back to NextBillingCycle when immediate cancel timing fails ValidateForAction. |
| `plan.go` | getPlanByVersion, PlanFromPlanInput (cheats key='cheat'/version=1 to pass validation then unsets), PlanFromPlan. | PlanFromPlanInput temporarily sets and clears Key/Version to satisfy plan.Validate(); defaults SettlementMode to CreditThenInvoice. Marked redundant TODO — keep in sync with adapter. |

## Anti-Patterns

- Persisting subscriptions directly instead of delegating to WorkflowService.CreateFromPlan / ChangeToPlan.
- Skipping request.PlanInput.Validate() or the deleted/status lifecycle checks before subscribing.
- Returning bare fmt.Errorf for user-facing plan validation failures instead of models.NewGenericValidationError.
- Allowing PlanStatusArchived in Create/Change (only Migrate permits archived plans).
- Deleting earlier phases for StartingPhase instead of zeroing their Duration.

## Decisions

- **Inline custom plans reuse plan.CreatePlanInput by faking Key/Version for validation then clearing them.** — plan.Validate() requires key and reference, but custom subscription plans have none; PlanFromPlanInput notes this as deliberate cheating pending a partial productcatalog type.
- **Migrate accepts archived plans while Create/Change require active.** — Migration moves an existing subscription forward to a newer version that may already be archived, whereas new/changed subscriptions must point at an active plan.

## Example: Resolve a plan ref, validate lifecycle, and change subscription via the workflow service

```
func (s *service) Change(ctx context.Context, request plansubscription.ChangeSubscriptionRequest) (plansubscription.SubscriptionChangeResponse, error) {
	var def plansubscription.SubscriptionChangeResponse
	if err := request.PlanInput.Validate(); err != nil { return def, models.NewGenericValidationError(err) }
	var plan subscription.Plan
	if request.PlanInput.AsRef() != nil {
		p, err := s.getPlanByVersion(ctx, request.ID.Namespace, *request.PlanInput.AsRef())
		if err != nil { return def, err }
		now := clock.Now()
		if p.StatusAt(now) != productcatalog.PlanStatusActive { return def, models.NewGenericValidationError(fmt.Errorf("plan %s@%d is not active", p.Key, p.Version)) }
		plan = PlanFromPlan(*p)
	}
	curr, next, err := s.WorkflowService.ChangeToPlan(ctx, request.ID, request.WorkflowInput, plan)
	if err != nil { return def, err }
	return plansubscription.SubscriptionChangeResponse{Current: curr, Next: next}, nil
}
```

<!-- archie:ai-end -->
