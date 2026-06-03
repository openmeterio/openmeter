# v2

<!-- archie:ai-start -->

> TypeSpec definitions for the v2 entitlements and grants API under /api/v2/. Composes V1 entitlement/grant models via OmitProperties spreads to expose customer-scoped CRUD, value/history/reset sub-routes, and admin list endpoints — all compiled into Go server stubs and SDKs.

## Patterns

**Discriminated union for polymorphic entitlement types** — Unions use @discriminated(#{ envelope: "none", discriminatorPropertyName: "type" }) with a literal type field per variant; both response (EntitlementV2) and input (EntitlementV2CreateInputs) follow this shape. (`@discriminated(#{ envelope: "none", discriminatorPropertyName: "type" }) union EntitlementV2 { metered: EntitlementMeteredV2, static: EntitlementStaticV2, boolean: EntitlementBooleanV2 }`)
**OmitProperties for model composition** — V2 models reuse V1 via OmitProperties<Base, "f1" | "f2"> spreads instead of copy-pasting fields; deprecated fields stay on create-input models with #deprecated. (`model EntitlementMeteredV2 { ...OmitProperties<EntitlementMeteredV2CreateInputs, "type" | "measureUsageFrom" | "grants">; ...EntitlementCustomerFields; }`)
**Interface-per-route-group with @route + @tag** — Each distinct URL prefix gets its own interface with @route/@tag/@friendlyName; sub-routes use @route on individual operations. Never mix URL prefixes in one interface. (`@route("/api/v2/customers/{customerIdOrKey}/entitlements") @tag("Entitlements") @friendlyName("CustomerEntitlementsV2") interface CustomerEntitlementsV2Endpoints { ... }`)
**Explicit operationId with V2 suffix** — Every operation carries @operationId("verbNounV2") in camelCase with a V2 suffix for stable SDK method names distinct from V1. (`@operationId("createCustomerEntitlementV2")`)
**V2 suffix on all exported symbols** — Every model, union, and interface friendlyName ends with V2 (EntitlementV2, GrantV2, GrantCreateInputV2) to avoid SDK symbol collisions with V1. (`model GrantV2 { ... }  // NOT Grant or EntitlementGrant`)
**Typed error unions on every operation** — Every operation returns | CommonErrors at minimum, plus | NotFoundError / | ConflictError where applicable. Never omit error unions. (`get(...): EntitlementV2 | CommonErrors | NotFoundError;`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `main.tsp` | Package entry point: imports all sibling .tsp files and the parent namespace; declares no models. | Every new .tsp in this folder must be imported here or it compiles in isolation and is invisible. |
| `entitlements.tsp` | Admin list/get at /api/v2/entitlements plus the canonical discriminated unions (EntitlementV2, EntitlementV2CreateInputs) and the three variant models. | A new entitlement sub-type requires updating both union definitions here. |
| `customer.tsp` | Customer-scoped CRUD plus value/history/reset/grants sub-routes split across two interfaces (two URL prefixes); defines EntitlementValueV2. | Sub-resource routes (/grants, /value, /history, /reset) belong in the second interface (CustomerEntitlementV2Endpoints), not the collection interface. |
| `grant.tsp` | Pure model file: GrantV2 response and GrantCreateInputV2 input. No HTTP routes. | Does NOT use `using TypeSpec.Http` — adding HTTP decorators requires adding the import and using declaration first. |
| `grants.tsp` | Admin list endpoint at /api/v2/grants; intentionally thin (no per-grant routes). | Uses both QueryPagination and QueryLimitOffset spreads — all new list endpoints here must do the same. |

## Anti-Patterns

- Redefining V1 models instead of composing via OmitProperties — creates divergence when V1 types change
- Omitting the V2 suffix from a model/union/interface friendlyName — causes SDK collisions with V1
- Adding a new entitlement type variant without updating both EntitlementV2 and EntitlementV2CreateInputs unions
- Forgetting to import a new .tsp file in main.tsp — it compiles in isolation but is invisible
- Using HTTP decorators in a file lacking import "@typespec/http" and using TypeSpec.Http (e.g. grant.tsp)

## Decisions

- **V2 models compose V1 types via OmitProperties rather than standalone redefinitions** — Keeps the V1->V2 delta minimal and auditable; V1 field changes propagate automatically unless explicitly omitted.
- **Deprecated fields (issueAfterReset, issueAfterResetPriority) kept on create-input with #deprecated instead of removed** — V1-style SDK consumers must not break on upgrade; the V2 replacement (issue: IssueAfterReset) is additive.
- **Two separate interfaces in customer.tsp for the two /customers/{id}/entitlements URL prefixes** — Interface-level @route applies to all operations; splitting keeps sub-resource paths from polluting the top-level collection interface.

## Example: Adding a new list filter param to an existing V2 list endpoint

```
// In entitlements.tsp — add the param inside the list() operation signature:
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
