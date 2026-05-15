# testutils

<!-- archie:ai-start -->

> Test-only helper package providing a PlanSubscriptionAdapter implementation that wraps plan.Service into the interface needed by service_test files, enabling plan resolution (by ref or inline input) without importing app/common Wire wiring.

## Patterns

**PlanSubscriptionAdapter interface for test isolation** — Defines a minimal PlanSubscriptionAdapter interface (GetVersion, FromInput) with a compile-time assertion (var _ PlanSubscriptionAdapter = &adapter{}) so tests can resolve plans without wiring the full DI graph. (`var _ PlanSubscriptionAdapter = &adapter{}`)
**GetVersion maps plan.IsNotFound to subscription.NewPlanNotFoundError** — GetVersion delegates to plan.Service.GetPlan, maps IsNotFound errors to subscription.NewPlanNotFoundError, and checks DeletedAt before returning. Tests that bypass this mapping will see raw storage errors instead of typed domain errors. (`if plan.IsNotFound(err) { return nil, subscription.NewPlanNotFoundError(planKey, version) }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `adapter.go` | Entire testutils package: PlanSubscriptionAdapter interface, PlanSubscriptionAdapterConfig, adapter struct, NewPlanSubscriptionAdapter constructor, GetVersion and FromInput implementations. | GetVersion checks p.DeletedAt before returning — removing that guard silently allows deleted plans in tests. FromInput delegates to service.PlanFromPlanInput, which uses the 'cheat' key/version — tests must not rely on the returned plan having a real key or version. |

## Anti-Patterns

- Importing app/common or any Wire provider from this package — it must stay independent to avoid import cycles in test dependencies.
- Adding helpers that reach into the Ent DB directly instead of going through plan.Service — bypasses the same validation paths production code uses.

## Decisions

- **PlanSubscriptionAdapter lives in testutils rather than being inlined in each test** — Multiple service_test files need the same plan-resolution logic; the adapter DRYs that up without leaking production code paths.

<!-- archie:ai-end -->
