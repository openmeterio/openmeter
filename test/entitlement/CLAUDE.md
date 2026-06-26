# entitlement

<!-- archie:ai-start -->

> Structural parent for entitlement integration/regression tests. It owns the cross-layer test surface where metered entitlements, credit grants, balance snapshots, resets, and voids are exercised end-to-end against a real Postgres DB; its sole child (regression) holds the actual code.

## Patterns

**Children own all source** — test/entitlement has no direct source files; navigate into regression for the entitlement+credit stack tests built via setupDependencies and driven by pkg/clock. (`test/entitlement/regression/{framework_test.go,scenario_test.go}`)

## Anti-Patterns

- Adding entitlement test source directly at this level instead of inside a typed child folder (e.g. regression).

## Decisions

- **Group entitlement tests under a structural parent split by test kind (regression).** — Separates historically-reproduced balance regressions from other entitlement test concerns while sharing the import surface (credit, entitlement, streaming, clock).

<!-- archie:ai-end -->
