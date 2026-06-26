# subscription

<!-- archie:ai-start -->

> The plan-to-subscription bridge (package plansubscription): converts a plan — either an inline plan.CreatePlanInput or a published plan reference — into subscription domain inputs, and defines PlanSubscriptionService (Create/Change/Migrate). It owns plan→subscription translation; persistence is delegated to the subscription workflow layer.

## Patterns

**PlanInput dispatch: ref vs inline** — PlanInput holds either *PlanRefInput or *plan.CreatePlanInput; consumers branch on AsRef()/AsInput(). Validate requires exactly one to be set. (`func (p *PlanInput) Validate() error { if p.ref == nil && p.plan == nil { return fmt.Errorf("plan or plan reference must be provided") }; return nil }`)
**Domain adapters implement subscription.* interfaces** — Plan/Phase/RateCard implement subscription.Plan / subscription.PlanPhase / subscription.PlanRateCard via compile assertions and expose ToCreateSubscription*Input converters. (`var _ subscription.Plan = &Plan{} ; func (p *Plan) ToCreateSubscriptionPlanInput() subscription.CreateSubscriptionPlanInput { ... }`)
**Phases compute cumulative StartAfter** — GetPhases walks plan phases accumulating each phase Duration into startAfter, assigning Index for SortHint, rather than storing absolute times. (`ps = append(ps, &Phase{Phase: ph, StartAfter: startAfter, Index: idx}); startAfter, _ = startAfter.Add(lo.FromPtr(ph.Duration))`)
**Ref only set when the plan exists** — ToCreateSubscriptionPlanInput attaches a subscription.PlanRef only when Plan.Ref != nil, so inline custom plans subscribe without a stored reference. (`if p.Ref != nil { ref = &subscription.PlanRef{Id: p.Ref.ID, Key: p.Key, Version: p.Version} }`)
**Service orchestrates, never persists** — PlanSubscriptionService delegates to subscriptionworkflow (CreateFromPlan/ChangeToPlan) and plan.Service; Create/Change require active plans while Migrate accepts archived. (`type PlanSubscriptionService interface { Create(...); Migrate(...); Change(...) }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `plan.go` | PlanInput/PlanRefInput + Plan/Phase/RateCard adapters to subscription.* | RateCard.ToCreateSubscriptionItemPlanInput clones the rate card (RateCard.Clone()); Ref must stay nil for inline plans |
| `service.go` | PlanSubscriptionService interface + request structs | Create/Change carry a WorkflowInput plus PlanInput and optional StartingPhase; Migrate takes TargetVersion |

## Anti-Patterns

- Persisting subscriptions directly here instead of delegating to the subscription workflow layer
- Branching on whether AsRef()/AsInput() is nil without going through PlanInput.Validate first
- Setting subscription.PlanRef for an inline custom plan (Ref must be nil unless the plan is stored)
- Computing phase StartAfter as absolute timestamps instead of cumulative durations
- Allowing archived plans in Create/Change (only Migrate accepts archived plans)

## Decisions

- **One endpoint accepts both an inline custom plan and a plan reference via PlanInput** — Subscriptions can be created from a saved catalog plan or an ad-hoc inline plan; PlanInput's ref/plan dispatch unifies both code paths into the same workflow call.
- **Phases use cumulative StartAfter durations, not absolute times** — Subscription phases are anchored relative to subscription start; storing relative ISO durations keeps the plan independent of any specific subscription start date.

## Example: Converting plan phases into relative subscription phases

```
func (p *Plan) GetPhases() []subscription.PlanPhase {
	ps := make([]subscription.PlanPhase, 0, len(p.Phases))
	startAfter := datetime.ISODuration{}
	for idx, ph := range p.Phases {
		ps = append(ps, &Phase{Phase: ph, StartAfter: startAfter, Index: idx})
		startAfter, _ = startAfter.Add(lo.FromPtr(ph.Duration))
	}
	return ps
}
```

<!-- archie:ai-end -->
