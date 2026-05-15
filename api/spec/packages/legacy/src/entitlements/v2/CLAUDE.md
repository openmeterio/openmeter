# v2

<!-- archie:ai-start -->

> TypeSpec definitions for the v2 entitlements and grants API surface under /api/v2/. Composes V1 entitlement/grant models via OmitProperties spreads to expose customer-scoped CRUD, value/history/reset sub-routes, and admin list endpoints — all compiled into api/openapi.yaml Go server stubs and SDKs.

## Patterns

**Discriminated union for polymorphic entitlement types** — Union types use @discriminated(#{ envelope: "none", discriminatorPropertyName: "type" }) with a literal `type` field on each variant. Both response (EntitlementV2) and input (EntitlementV2CreateInputs) unions follow this shape. (`@discriminated(#{ envelope: "none", discriminatorPropertyName: "type" })
union EntitlementV2 { metered: EntitlementMeteredV2, static: EntitlementStaticV2, boolean: EntitlementBooleanV2 }`)
**OmitProperties for model composition** — V2 models reuse V1 models via OmitProperties<Base, "field1" | "field2"> spreads instead of copy-pasting fields. Deprecated fields are kept on create-input models with #deprecated annotations. (`model EntitlementMeteredV2 {
  ...OmitProperties<EntitlementMeteredV2CreateInputs, "type" | "measureUsageFrom" | "grants">;
  ...EntitlementCustomerFields;
}`)
**Interface-per-route-group with @route + @tag** — Each distinct URL prefix gets its own interface decorated with @route, @tag, and @friendlyName. Sub-routes use @route on individual operations. Never mix URL prefixes inside one interface. (`@route("/api/v2/customers/{customerIdOrKey}/entitlements")
@tag("Entitlements")
@friendlyName("CustomerEntitlementsV2")
interface CustomerEntitlementsV2Endpoints { ... }`)
**Explicit operationId with V2 suffix** — Every operation carries @operationId("verbNounV2") in camelCase with a V2 suffix to produce stable SDK method names distinct from V1 counterparts. (`@operationId("createCustomerEntitlementV2")`)
**PaginatedResponse with QueryPagination/QueryOrdering spreads** — All list operations return PaginatedResponse<T> and spread ...QueryPagination, ...QueryLimitOffset, and ...QueryOrdering<OrderByEnum> for consistent pagination params. (`list(...QueryPagination, ...QueryLimitOffset, ...QueryOrdering<EntitlementOrderBy>): PaginatedResponse<EntitlementV2> | CommonErrors;`)
**V2 suffix on all exported symbols** — Every model, union, and interface name exported from this package ends with V2 (e.g. EntitlementV2, GrantV2, GrantCreateInputV2) to avoid SDK symbol collisions with V1. (`model GrantV2 { ... }  // NOT Grant or EntitlementGrant`)
**CommonErrors + typed error union on every operation** — Every operation returns | CommonErrors at minimum, plus | NotFoundError, | ConflictError where applicable. Never omit error unions. (`get(...): EntitlementV2 | CommonErrors | NotFoundError;`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `main.tsp` | Package entry point: imports all sibling .tsp files and the parent namespace. Declares no models — only wires the package together. | All new .tsp files added to this folder must be imported here; forgetting breaks compilation silently. |
| `entitlements.tsp` | Defines admin list/get endpoints at /api/v2/entitlements and the canonical discriminated union models (EntitlementV2, EntitlementV2CreateInputs) plus all three variant models (Metered, Boolean, Static). | Adding a new entitlement sub-type requires updating both EntitlementV2 and EntitlementV2CreateInputs union definitions here. |
| `customer.tsp` | Customer-scoped CRUD plus value/history/reset/grants sub-routes split across two interfaces for two URL prefixes. Defines EntitlementValueV2 response model. | Two separate interfaces for the two URL prefixes; sub-resource routes (/grants, /value, /history, /reset) belong in CustomerEntitlementV2Endpoints (the second interface), not the first. |
| `grant.tsp` | Pure model file defining GrantV2 response model and GrantCreateInputV2 create-input model. No HTTP routes. | Does NOT use `using TypeSpec.Http` — adding HTTP decorators here requires adding the import and using declaration first. |
| `grants.tsp` | Admin list endpoint at /api/v2/grants. Intentionally thin — no per-grant routes. | Uses both QueryPagination and QueryLimitOffset spread params; all new list endpoints in this package must do the same. |

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
