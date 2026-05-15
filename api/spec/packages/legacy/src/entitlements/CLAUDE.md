# entitlements

<!-- archie:ai-start -->

> TypeSpec schema-only package defining the v1 entitlements API surface — subjects-scoped entitlement CRUD, grant management, customer access checks, and shared value/history/reset sub-routes. All V1 symbols are deprecated in favour of v2 equivalents in the v2/ sub-folder; new entitlement work belongs there.

## Patterns

**Discriminated union with envelope:none for polymorphic entitlement types** — The Entitlement union and EntitlementCreateInputs union use @discriminated(#{ envelope: "none", discriminatorPropertyName: "type" }). Adding a new entitlement type requires a new arm in BOTH unions plus a new model with a matching `type` literal field. (`@discriminated(#{ envelope: "none", discriminatorPropertyName: "type" })
@friendlyName("EntitlementCreateInputs")
union EntitlementCreateInputs {
  metered: EntitlementMeteredCreateInputs,
  static: EntitlementStaticCreateInputs,
  boolean: EntitlementBooleanCreateInputs,
}`)
**OmitProperties<T, "field1" | "field2"> for model composition** — Response models spread OmitProperties over a base create-input type and add read-only fields, avoiding full redefinition and keeping create/read models in sync automatically when the base type changes. (`model EntitlementMetered {
  type: EntitlementType.metered;
  ...OmitProperties<EntitlementMeteredCreateInputs, "type" | "measureUsageFrom" | "usagePeriod">;
  ...EntitlementMeteredCalculatedFields;
}`)
**#deprecated + #suppress pair on every V1 symbol with a V2 successor** — Every V1 model, union, and operation that has a V2 successor must carry both #deprecated and #suppress. Missing either causes compiler warnings that break `make gen-api`. (`#deprecated "Use EntitlementMeteredV2 instead"
#suppress "deprecated" "V1 Entitlements APIs will be removed on December 1st, 2025"
@friendlyName("EntitlementMetered")
model EntitlementMetered { ... }`)
**Explicit @operationId on every operation** — All operations carry @operationId to ensure stable generated Go function names. Omitting it causes name instability across TypeSpec compiler upgrades. (`@get
@operationId("getEntitlementValue")
@route("/{entitlementIdOrFeatureKey}/value")
getEntitlementValue(...): EntitlementValue | CommonErrors | NotFoundError;`)
**main.tsp as the sole entry point importing all sibling files** — A new .tsp file in this folder must be imported in main.tsp (which also cross-imports productcatalog/features.tsp) or it produces no output in api/openapi.yaml. (`// main.tsp
import "./entitlements.tsp";
import "../productcatalog/features.tsp";
import "./grant.tsp";
import "./subjects.tsp";
import "./customer.tsp";`)
**Separate interface groups per distinct URL prefix** — Each interface maps to exactly one URL prefix so adding operations does not pollute another prefix's operation set. (`@route("/api/v1/customers/{customerIdOrKey}")
@tag("Entitlements")
@friendlyName("Customer")
interface CustomerEndpoints { ... }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `main.tsp` | Package entry point: imports all sibling .tsp files including the cross-package productcatalog/features.tsp. No model or interface declarations live here. | Forgetting to import a new .tsp file — it compiles in isolation but is invisible to the full package output. |
| `entitlements.tsp` | Defines the Entitlement discriminated union, EntitlementCreateInputs union, EntitlementSharedFields, all three per-type create-input models (metered/boolean/static), and the admin list/get endpoints at /api/v1/entitlements. | Adding a new entitlement type variant requires updating BOTH the Entitlement read union AND the EntitlementCreateInputs create union atomically, or the discriminator will be incomplete. |
| `subjects.tsp` | Defines all V1 subject-scoped entitlement CRUD and grant endpoints at /api/v1/subjects/{subjectIdOrKey}/entitlements. All operations are deprecated and point to V2 equivalents. | Do not add new operations here — they belong in the v2 sub-folder. Every operation here carries #deprecated + #suppress. |
| `grant.tsp` | Defines Grant, GrantCreateInput, ExpirationPeriod, ExpirationDuration, and the global /api/v1/grants admin list/void endpoints. Already imports @typespec/http and uses TypeSpec.Http. | Missing `import "@typespec/http"` and `using TypeSpec.Http` in new files referencing these models will cause compilation failure. |
| `customer.tsp` | Defines CustomerAccess and the customer-scoped access-check endpoints (getCustomerAccess, getCustomerEntitlementValue) at /api/v1/customers/{customerIdOrKey}. | CustomerAccess.entitlements is typed as Record<EntitlementValue> — do not split into a separate typed map without updating all SDK consumers. |

## Anti-Patterns

- Adding new V1 operations or models in this folder instead of the v2 sub-folder — all V1 entitlement APIs are deprecated
- Adding a new entitlement type variant to only one of the two discriminated unions (Entitlement or EntitlementCreateInputs) — both must be updated atomically
- Omitting #deprecated + #suppress pair on new V1 symbols — causes compiler warnings that break `make gen-api`
- Defining models without @friendlyName — autogenerated names can clash with or shadow V2 names in the same namespace
- Adding HTTP decorators to a new .tsp file without `import "@typespec/http"` and `using TypeSpec.Http` — causes compilation failure

## Decisions

- **V1 symbols are kept but fully deprecated in favour of V2 equivalents in the v2 sub-folder** — Breaking API changes require a deprecation window; keeping V1 symbols with #deprecated annotations lets the compiler warn SDK consumers while the generator still emits the endpoints for backward-compatible clients.
- **OmitProperties composition instead of standalone read-model redefinitions** — Avoids maintaining two parallel type hierarchies for create vs. read; when a create-input field changes, the read model reflects it automatically unless explicitly omitted.
- **EntitlementValue shared across all entitlement types with optional type-specific fields** — A single EntitlementValue model with optional metered-specific fields (balance, usage, overage) and shared hasAccess allows the /value endpoint to return a uniform shape regardless of entitlement type, reducing client branching logic.

## Example: Add a new V2-only entitlement endpoint (this work belongs in v2/, not here)

```
// In api/spec/packages/legacy/src/entitlements/v2/customer.tsp:

@route("/api/v2/customers/{customerIdOrKey}/entitlements")
@tag("Entitlements")
@friendlyName("CustomerEntitlementsV2")
interface CustomerEntitlementsV2Endpoints {
  @post
  @operationId("createEntitlementV2")
  @summary("Create customer entitlement")
  post(
    @path customerIdOrKey: ULIDOrExternalKey,
    @body entitlement: EntitlementV2CreateInputs,
  ): {
    @statusCode _: 201;
    @body body: EntitlementV2;
// ...
```

<!-- archie:ai-end -->
