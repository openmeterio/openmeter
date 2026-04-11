# AIP-126 — Enums

Reference: https://kong-aip.netlify.app/aip/126/

- All enum wire values must be `snake_case` (enforced as an error by the `casing-aip-errors` linter rule).
- Every enum must define an `Unknown` member as the zero/default value.
- Prefer enums over booleans for two-state fields — this allows a third state to be added later without a breaking change.
