# entitlement

<!-- archie:ai-start -->

> Structural parent for entitlement integration and regression test packages — owns the test/entitlement/regression sub-package that guards grant burn-down, expiry, voiding, and reset edge cases against real Postgres using raw package constructors and MockStreamingConnector. The directory itself holds only a CLAUDE.md.

## Patterns

**No source files at parent level** — All test code lives in sub-packages (regression/); never add .go files directly to test/entitlement. (`// All code in: test/entitlement/regression/*.go`)
**setupDependencies for self-contained bootstrap** — Each regression test file wires its own dependency graph via setupDependencies using raw package constructors plus MockStreamingConnector — never app/common. (`deps := setupDependencies(t) // real Ent adapters + MockStreamingConnector`)
**Controlled clock with defer reset** — Time-based grant burn-down tests advance the global clock via clock.SetTime() and must defer clock.ResetTime() to avoid leaking state into parallel tests. (`clock.SetTime(now); defer clock.ResetTime()`)
**Scenario tests named after the bug they guard** — Regression functions in scenario_test.go are named after the specific production bug/edge case for immediate root-cause traceability. (`func TestGrantExpiryBeforeReset(t *testing.T) { ... }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `regression/framework_test.go` | Defines setupDependencies and createCustomerAndSubject helpers plus shared test wiring. | Pre-existing context.Background() uses are not a model — new code uses t.Context(). Never import app/common. |
| `regression/scenario_test.go` | Individual regression tests, one per production bug; each calls setupDependencies independently (fresh pgtestdb DB). | Never reuse subject keys or feature keys across test functions. |

## Anti-Patterns

- Adding .go source files directly in test/entitlement — all code belongs in regression/
- Importing app/common or any Wire provider set — causes import cycles
- Using context.Background() in new test code where t.Context() is available
- Reusing subject or feature keys across tests without isolated setupDependencies calls
- Skipping defer clock.ResetTime() — leaves global clock state dirty for parallel tests

## Decisions

- **Regression tests isolated here rather than co-located with openmeter/entitlement/** — Keeps the regression harness separate from production code so test-only imports (pgtestdb, MockStreamingConnector) don't pollute the production package graph.
- **MockStreamingConnector instead of real ClickHouse** — ClickHouse is unavailable in standard CI; the mock gives deterministic usage responses without external dependencies.

<!-- archie:ai-end -->
