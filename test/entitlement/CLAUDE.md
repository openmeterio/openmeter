# entitlement

<!-- archie:ai-start -->

> Structural parent for entitlement integration and regression test packages — owns the test/entitlement/regression sub-package that guards grant burn-down, expiry, voiding, and reset edge cases against real Postgres.

## Anti-Patterns

- Adding source files directly in test/entitlement — all code belongs in the regression sub-package
- Importing app/common or Wire provider sets from any test under this folder

## Decisions

- **Regression tests are isolated in a sub-package rather than co-located with production code** — Keeps the regression harness separate from openmeter/entitlement/ domain code, allowing test-only imports (pgtestdb, MockStreamingConnector) without polluting the production package graph.

<!-- archie:ai-end -->
