# testutils

<!-- archie:ai-start -->

> Test-only helper providing a PlanSubscriptionAdapter implementation that wraps plan.Service into the interface needed by service_test files, enabling plan resolution (by ref or inline input) without importing app/common Wire wiring.

## Patterns

**PlanSubscriptionAdapter interface for test isolation** — Minimal PlanSubscriptionAdapter (GetVersion, FromInput) with a compile-time assertion so tests resolve plans without the full DI graph. (`var _ PlanSubscriptionAdapter = &adapter{}`)
**GetVersion maps plan.IsNotFound to subscription.NewPlanNotFoundError** — GetVersion delegates to plan.Service.GetPlan, maps IsNotFound to subscription.NewPlanNotFoundError, and checks DeletedAt before returning. (`if plan.IsNotFound(err) { return nil, subscription.NewPlanNotFoundError(planKey, version) }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `adapter.go` | Entire package: PlanSubscriptionAdapter interface, Config, adapter struct, NewPlanSubscriptionAdapter, GetVersion and FromInput. | GetVersion checks p.DeletedAt before returning — removing it silently allows deleted plans in tests. FromInput delegates to service.PlanFromPlanInput which uses the 'cheat' key/version — tests must not rely on a real key/version. |

## Anti-Patterns

- Importing app/common or any Wire provider — this package must stay independent to avoid test import cycles.
- Reaching into the Ent DB directly instead of going through plan.Service — bypasses the validation paths production uses.

## Decisions

- **PlanSubscriptionAdapter lives in testutils rather than inlined per test.** — Multiple service_test files need the same plan-resolution logic; the adapter DRYs it up without leaking production code paths.

<!-- archie:ai-end -->
