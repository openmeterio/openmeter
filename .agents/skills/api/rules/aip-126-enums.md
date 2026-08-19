# AIP-126 — Enums

Reference: https://kong-aip.netlify.app/aip/126/

- All enum wire values must be `snake_case`, or a dot-separated path of `snake_case` segments when the value names a nested field (`realization.detailed_lines`). Both are enforced as an error by the `casing-aip-errors` linter rule. Prefer a flat value; reach for a path only where the dot carries meaning, such as expands and `order_by` values that address a nested resource.
- Prefer defining an `unknown` member as the zero/default value. This is a recommendation, not a requirement — most domains (charges, subscriptions, currencies) omit it, so do not flag its absence as a violation.
- Prefer enums over booleans for two-state fields — this allows a third state to be added later without a breaking change.
