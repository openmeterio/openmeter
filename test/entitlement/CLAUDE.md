# entitlement

<!-- archie:ai-start -->

> Structural parent for entitlement integration and regression test packages — owns the test/entitlement/regression sub-package that guards grant burn-down, expiry, voiding, and reset edge cases against real Postgres using raw package constructors and MockStreamingConnector.

## Patterns

**No source files at parent level** — All test code lives in sub-packages (regression/). The test/entitlement directory itself contains only a CLAUDE.md. Never add .go source files directly here. (`// All code in: test/entitlement/regression/*.go`)
**setupDependencies for self-contained bootstrap** — Each regression test file wires its own dependency graph via a setupDependencies function using raw package constructors — never app/common or Wire provider sets. (`deps := setupDependencies(t) // returns real Ent adapters + MockStreamingConnector`)
**Controlled clock with defer reset** — Tests that exercise time-based grant burn-down advance the global clock via clock.SetTime() and must always defer clock.ResetTime() to prevent state leaking into parallel tests. (`clock.SetTime(now)
defer clock.ResetTime()`)
**Scenario tests named after the bug they guard** — Regression test functions in scenario_test.go are named after the specific production bug or edge case they cover, making root-cause traceability immediate. (`func TestGrantExpiryBeforeReset(t *testing.T) { ... }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `test/entitlement/regression/framework_test.go` | Defines setupDependencies, createCustomerAndSubject helpers, and the shared test wiring used by all scenario tests. | Pre-existing context.Background() uses in this file are not models to follow — new code should use t.Context(). Never import app/common here. |
| `test/entitlement/regression/scenario_test.go` | Individual regression test functions, one per production bug or edge case. | Each test must call setupDependencies independently — pgtestdb provides a fresh DB per test. Never reuse subject keys or feature keys across test functions. |

## Anti-Patterns

- Adding .go source files directly in test/entitlement — all code belongs in the regression sub-package
- Importing app/common or any Wire provider set from any test under this folder — causes import cycles
- Using context.Background() in new test code where t.Context() is available
- Reusing subject or feature keys across test functions without isolated setupDependencies calls
- Skipping defer clock.ResetTime() — leaves global clock state dirty for parallel tests

## Decisions

- **Regression tests isolated in test/entitlement/regression rather than co-located with openmeter/entitlement/** — Keeps the regression harness separate from production domain code, allowing test-only imports (pgtestdb, MockStreamingConnector) without polluting the production package graph.
- **MockStreamingConnector instead of real ClickHouse** — ClickHouse is unavailable in the standard CI test environment; MockStreamingConnector provides deterministic usage query responses without external dependencies.

<!-- archie:ai-end -->
