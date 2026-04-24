# testutils

<!-- archie:ai-start -->

> Test-only adapter wrapping plan.Service into the PlanSubscriptionAdapter interface used by service_test files; provides GetVersion and FromInput helpers so tests can resolve plans without importing app/common wiring.

## Patterns

**PlanSubscriptionAdapter interface for test isolation** — Defines a minimal PlanSubscriptionAdapter interface (GetVersion, FromInput) implemented by adapter struct; tests use NewPlanSubscriptionAdapter(config) to get a usable instance without wiring the full service graph (`var _ PlanSubscriptionAdapter = &adapter{}`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `adapter.go` | Sole file; PlanSubscriptionAdapter interface, adapter struct, NewPlanSubscriptionAdapter constructor, GetVersion (delegates to plan.Service.GetPlan with NotFound mapping), FromInput (delegates to service.PlanFromPlanInput) | GetVersion checks DeletedAt before returning — do not remove that guard or tests will silently allow deleted plans |

## Anti-Patterns

- Importing app/common or any Wire provider from this package — it must stay independent to avoid import cycles in tests
- Adding test helpers that reach into the DB directly instead of going through plan.Service

## Decisions

- **PlanSubscriptionAdapter lives in testutils rather than inlining in each test** — Multiple service_test files need the same plan-resolution logic; the adapter DRYs that up without leaking to production code

<!-- archie:ai-end -->
