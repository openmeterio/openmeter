# testutils

<!-- archie:ai-start -->

> Test-only helpers (package testutils) for the plan-to-subscription bridge. Provides PlanSubscriptionAdapter, a thin adapter over plan.Service that resolves a plansubscription.Plan from a PlanRefInput or a plan.CreatePlanInput, mirroring the production service's plan-resolution logic for use in subscription/billing tests.

## Patterns

**Interface + Config + constructor + compile assertion** — Declares PlanSubscriptionAdapter interface, PlanSubscriptionAdapterConfig{PlanService, Logger}, adapter struct embedding the config, var _ PlanSubscriptionAdapter = &adapter{}, and NewPlanSubscriptionAdapter(config). (`var _ PlanSubscriptionAdapter = &adapter{}; func NewPlanSubscriptionAdapter(config PlanSubscriptionAdapterConfig) PlanSubscriptionAdapter { return &adapter{config} }`)
**Reuse production PlanFromPlanInput** — FromInput delegates straight to service.PlanFromPlanInput(input) instead of reimplementing the cheat-validation, keeping test behavior aligned with production. (`func (a *adapter) FromInput(...) (subscription.Plan, error) { return service.PlanFromPlanInput(input) }`)
**PlanNotFound normalization mirrors service** — GetVersion uses defaultx.WithDefault(ref.Version, 0) and maps plan.IsNotFound / nil plan to subscription.NewPlanNotFoundError, and rejects deleted plans via clock.Now(). (`if plan.IsNotFound(err) { return nil, subscription.NewPlanNotFoundError(planKey, version) }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `adapter.go` | Sole file: PlanSubscriptionAdapter interface (GetVersion, FromInput) and its plan.Service-backed implementation. | Marked 'TODO: we can get rid of this'; GetVersion duplicates service.getPlanByVersion logic plus a deleted-plan check — keep them aligned if either changes. |

## Anti-Patterns

- Reimplementing plan-input-to-subscription.Plan conversion instead of calling service.PlanFromPlanInput.
- Importing this test adapter from production (app/common) code — it lives under testutils and is intended for tests only.
- Diverging GetVersion's not-found/deleted handling from the production service.getPlanByVersion.

## Decisions

- **The adapter exists as a slim re-implementation/wrapper of service plan resolution for tests.** — Tests need a PlanSubscriptionAdapter seam without pulling the full plansubscription service wiring; the TODO acknowledges it is a candidate for removal once APIs align.

<!-- archie:ai-end -->
