# AIP-181 — API stability markers

Reference: https://kong-aip.netlify.app/aip/181/

AIP-181 defines stability levels as custom OpenAPI extensions applied either at the operation level (alongside `operationId`) or on individual schema definitions in `components.schemas`.

## Stability levels

| Constant                   | Extension key | Meaning                                                                                |
| -------------------------- | ------------- | -------------------------------------------------------------------------------------- |
| _(no annotation)_          | —             | **Stable** — default state, production-ready                                           |
| `Shared.UnstableExtension` | `x-unstable`  | Functionality under development; breaking changes are possible                         |
| `Shared.InternalExtension` | `x-internal`  | Reserved for Kong-internal use only; excluded from public documentation                |
| `Shared.PrivateExtension`  | `x-private`   | Not exposed by Gateways (e.g., metadata endpoints) — different from "hidden from docs" |

`x-unstable` and `x-internal` may be combined on the same operation to mark a feature that is both internal and still in development.

## Lifecycle progression

The typical maturity path is:

1. **Development** — both `x-unstable`, `x-internal` and `x-private` applied.
2. **Stabilizing** — `x-internal` and `x-private`; internal consumers may rely on the shape, but the API is not yet public
3. **Public** — no annotations present
4. **Deprecated** — use the standard OpenAPI `deprecated: true` field (optionally with a `Sunset` header)

## Schema-level annotations

For shared schemas, additional extensions refine visibility:

- `x-property-annotations` — marks individual properties on a shared schema
- `x-enum-dev` / `x-enum-internal` — controls enum value visibility across published specification versions
