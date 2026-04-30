# v2

<!-- archie:ai-start -->

> TypeSpec definitions for the v2 entitlements and grants API surface under `/api/v2/`. Extends and reshapes v1 entitlement/grant models (via OmitProperties + new fields) to expose customer-scoped CRUD, value/history/reset sub-routes, and admin list endpoints — all compiled into api/openapi.yaml Go server stubs and SDKs.

## Patterns

**Discriminated union for polymorphic entitlement types** — Union types with `@discriminated(#{ envelope: "none", discriminatorPropertyName: "type" })` and a `type` literal field on each variant. Both response unions (EntitlementV2) and input unions (EntitlementV2CreateInputs) follow this shape. (`@discriminated(#{ envelope: "none", discriminatorPropertyName: "type" })
union EntitlementV2 { metered: EntitlementMeteredV2, static: EntitlementStaticV2, boolean: EntitlementBooleanV2 }`)
**OmitProperties for model composition** — New V2 models reuse existing V1 models via `OmitProperties<Base, "field1" | "field2">` spreads rather than copy-pasting fields. Deprecated fields are kept on the create-input model with `#deprecated` annotations. (`model EntitlementMeteredV2 { ...OmitProperties<EntitlementMeteredV2CreateInputs, "type" | "grants">; ...EntitlementCustomerFields; }`)
**Interface-per-route-group with @route + @tag** — Each logical URL prefix gets its own `interface` decorated with `@route`, `@tag`, and `@friendlyName`. Sub-routes use `@route` on individual operations. Never mix URL prefixes inside one interface. (`@route("/api/v2/customers/{customerIdOrKey}/entitlements")
@tag("Entitlements")
interface CustomerEntitlementsV2Endpoints { ... }`)
**Explicit operationId on every operation** — Every operation carries `@operationId("verbNounV2")` with a camelCase V2 suffix so generated SDK method names are stable and distinct from V1 counterparts. (`@operationId("createCustomerEntitlementV2")`)
**PaginatedResponse wrapper with QueryPagination/QueryOrdering spreads** — List operations return `PaginatedResponse<T>` and spread `...QueryPagination`, `...QueryLimitOffset`, and `...QueryOrdering<OrderByEnum>` for consistent pagination params across all list endpoints. (`list(...QueryPagination, ...QueryLimitOffset, ...QueryOrdering<EntitlementOrderBy>): PaginatedResponse<EntitlementV2> | CommonErrors`)
**Version suffix naming (V2) on all public symbols** — Every model, union, and interface name exported from this package ends with `V2` (e.g. EntitlementV2, GrantV2, GrantCreateInputV2). main.tsp documents this as mandatory for versioned packages to avoid SDK symbol collisions. (`model GrantV2 { ... }  // NOT Grant or EntitlementGrant`)
**CommonErrors + typed error union on every operation** — Every operation returns `| CommonErrors` at minimum, plus `| NotFoundError`, `| ConflictError` where applicable. Never return a bare error or omit error unions. (`get(...): EntitlementV2 | CommonErrors | NotFoundError`)

## Key Files

| File               | Role                                                                                                                                                                                 | Watch For                                                                                                                                                                           |
| ------------------ | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `main.tsp`         | Package entry point: imports all sibling .tsp files and the parent namespace. Declares no models — only wires the package together.                                                  | All new .tsp files in this folder must be imported here; forgetting breaks compilation silently.                                                                                    |
| `entitlements.tsp` | Defines admin list/get endpoints at /api/v2/entitlements and the core discriminated union models (EntitlementV2, EntitlementV2CreateInputs) plus all three variant models.           | EntitlementV2 and EntitlementV2CreateInputs are the canonical union types — adding a new entitlement sub-type requires updating both unions here.                                   |
| `customer.tsp`     | Customer-scoped CRUD plus value/history/reset/grants sub-routes. Also defines EntitlementValueV2 response model and EntitlementMeteredV2CreateInputs with deprecated field bridging. | Two separate interfaces for the two URL prefixes; the second interface (CustomerEntitlementV2Endpoints) owns all sub-resource routes — don't add sub-routes to the first interface. |
| `grant.tsp`        | Defines GrantV2 response model and GrantCreateInputV2 create-input model; imported by customer.tsp and grants.tsp. No HTTP routes here — pure model file.                            | Does NOT use `using TypeSpec.Http` — adding HTTP decorators here requires adding the import and using declaration first.                                                            |
| `grants.tsp`       | Admin list endpoint at /api/v2/grants. Intentionally thin — no per-grant routes.                                                                                                     | Uses both `QueryPagination` and `QueryLimitOffset` spread params; all new list endpoints in this package should do the same.                                                        |

## Anti-Patterns

- Redefining V1 models instead of composing via OmitProperties — creates divergence when V1 types change.
- Omitting the V2 suffix from model, union, or interface friendly names — causes SDK name collisions with V1 symbols.
- Adding a new entitlement type variant without updating both EntitlementV2 and EntitlementV2CreateInputs discriminated unions in entitlements.tsp.
- Forgetting to import a new .tsp file in main.tsp — the file compiles in isolation but is invisible to the package.
- Using HTTP decorators in a file that lacks `import "@typespec/http"` and `using TypeSpec.Http` (e.g. grant.tsp) — causes compilation failure.

## Decisions

- **V2 models compose V1 types via OmitProperties rather than standalone redefinitions** — Keeps V1→V2 delta minimal and auditable; V1 field changes propagate automatically unless explicitly omitted.
- **Deprecated fields (issueAfterReset, issueAfterResetPriority) kept on create-input with #deprecated instead of removed** — SDK consumers using V1-style fields must not break on upgrade; the V2 replacement (issue: IssueAfterReset) is additive.
- **Two separate interfaces in customer.tsp for the two /customers/{id}/entitlements URL prefixes** — TypeSpec interface-level @route applies to all operations; splitting ensures sub-resource paths (/grants, /value, /history, /reset) don't pollute the top-level collection interface.

## Example: Adding a new list filter param to an existing V2 list endpoint

```
// In entitlements.tsp — add param inside the list() operation signature:
list(
  @query(#{ explode: true }) feature?: string[],
  // new filter:
  @query(#{ explode: true }) subjectKeys?: string[],
  ...QueryPagination,
  ...QueryLimitOffset,
  ...QueryOrdering<EntitlementOrderBy>,
): PaginatedResponse<EntitlementV2> | CommonErrors;
```

<!-- archie:ai-end -->
