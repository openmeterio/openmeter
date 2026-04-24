# entitlements

<!-- archie:ai-start -->

> TypeSpec schema-only folder defining the v1 entitlements API surface: subjects-scoped entitlement CRUD, grant management, customer access checks, and their read/create models. All v1 symbols are deprecated in favour of v2 equivalents; new work should go in the v2 sub-folder.

## Patterns

**Discriminated union with envelope:none for polymorphic entitlement types** — The Entitlement union and EntitlementCreateInputs union use @discriminated(#{ envelope: "none", discriminatorPropertyName: "type" }). A new entitlement variant requires a new arm in both unions plus a new model. (`@discriminated(#{ envelope: "none", discriminatorPropertyName: "type" })
@friendlyName("EntitlementCreateInputs")
union EntitlementCreateInputs {
  metered: EntitlementMeteredCreateInputs,
  static: EntitlementStaticCreateInputs,
  boolean: EntitlementBooleanCreateInputs,
}`)
**OmitProperties<T, "field1" | "field2"> for model composition** — Response models are built by spreading OmitProperties over a base create-input type and adding read-only fields, avoiding full redefinition and keeping create/read models in sync. (`model EntitlementMetered {
  type: EntitlementType.metered;
  ...OmitProperties<EntitlementMeteredCreateInputs,
    "type" | "measureUsageFrom" | "metadata" | "usagePeriod" | "featureKey" | "featureId" | "currentUsagePeriod"
  >;
  ...EntitlementMeteredCalculatedFields;
}`)
**#deprecated + #suppress pair for V1 symbols kept for backward compatibility** — Every V1 model, union, and operation that has a V2 successor must carry both `#deprecated "Use V2 instead"` and `#suppress "deprecated" "reason"`. Missing either causes compiler warnings that fail `make gen-api`. (`#deprecated "Use EntitlementMeteredV2 instead"
#suppress "deprecated" "V1 Entitlements APIs will be removed on December 1st, 2025"
@friendlyName("EntitlementMetered")
model EntitlementMetered { ... }`)
**Explicit @operationId on every operation** — All operations carry @operationId to ensure stable generated Go function names. Omitting it causes name instability across TypeSpec compiler upgrades. (`@get
@operationId("getEntitlementValue")
@route("/{entitlementIdOrFeatureKey}/value")
getEntitlementValue(...): EntitlementValue | CommonErrors | NotFoundError;`)
**main.tsp as the sole entry point importing all sibling files** — A new .tsp file in this folder must be imported in main.tsp or it produces no output in api/openapi.yaml. (`// main.tsp
import "./entitlements.tsp";
import "../productcatalog/features.tsp";
import "./grant.tsp";
import "./subjects.tsp";
import "./customer.tsp";`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `main.tsp` | Package entry point: imports all sibling .tsp files including the cross-package features.tsp from productcatalog. No model or interface declarations live here. | Forgetting to import a new .tsp file — it compiles in isolation but is invisible to the package output. |
| `entitlements.tsp` | Defines the Entitlement discriminated union, EntitlementCreateInputs union, EntitlementSharedFields, all three per-type create-input models (metered/boolean/static), and the admin list/get endpoints at /api/v1/entitlements. | Adding a new entitlement type variant requires updating BOTH the Entitlement read union AND the EntitlementCreateInputs create union, or the discriminator will be incomplete. |
| `subjects.tsp` | Defines all V1 subject-scoped entitlement CRUD and grant endpoints at /api/v1/subjects/{subjectIdOrKey}/entitlements. All operations are deprecated and point to V2 equivalents. | Do not add new operations here — they belong in the v2 sub-folder. Any new @route here requires `import "@typespec/http"` and `using TypeSpec.Http` since this file already has them. |
| `grant.tsp` | Defines Grant, GrantCreateInput, ExpirationPeriod, ExpirationDuration, and the global /api/v1/grants admin list/void endpoints. | grant.tsp already imports `@typespec/http` and uses TypeSpec.Http — adding a new HTTP decorator is safe; but missing this import in a new file referencing these models will fail compilation. |
| `customer.tsp` | Defines CustomerAccess and the customer-scoped access-check endpoints (getCustomerAccess, getCustomerEntitlementValue) at /api/v1/customers/{customerIdOrKey}. | CustomerAccess.entitlements is typed as Record<EntitlementValue> — the key is a feature key string and the value is the shared EntitlementValue model; do not split this into a separate typed map without updating all SDK consumers. |

## Anti-Patterns

- Adding new V1 operations or models in this folder instead of the v2 sub-folder — all V1 entitlement APIs are deprecated
- Adding a new entitlement type variant to only one of the two discriminated unions (Entitlement or EntitlementCreateInputs) — both must be updated atomically
- Omitting #deprecated + #suppress pair on new V1 symbols — causes compiler warnings that break `make gen-api`
- Defining models without @friendlyName — autogenerated names can clash with or shadow V2 names in the same namespace
- Adding HTTP decorators (@route, @get, etc.) to a new .tsp file without `import "@typespec/http"` and `using TypeSpec.Http` — causes compilation failure

## Decisions

- **V1 symbols are kept but fully deprecated in favour of V2 equivalents in the v2 sub-folder** — Breaking API changes require a deprecation window; keeping V1 symbols with #deprecated annotations lets the compiler warn SDK consumers while the generator still emits the endpoints for backward-compatible clients.
- **OmitProperties composition instead of standalone read-model redefinitions** — Avoids maintaining two parallel type hierarchies for create vs. read; when a create-input field changes, the read model reflects it automatically unless explicitly omitted.
- **Separate interface groups per route prefix (SubjectEntitlementsEndpoints, CustomerEndpoints, CustomerEntitlementEndpoints, EntitlementsEndpoints)** — Each interface maps to a distinct URL prefix, making it easy to see the full surface of a given route group in one place and to add operations without accidentally polluting another prefix's operation set.

## Example: Add a new entitlement type to the V1 discriminated union (while keeping V1 pattern for backward compat)

```
// In entitlements.tsp:

// 1. Add create-input model:
#deprecated "Use EntitlementMyTypeV2CreateInputs instead"
#suppress "deprecated" "V1 Entitlements APIs will be removed on December 1st, 2025"
@friendlyName("EntitlementMyTypeCreateInputs")
model EntitlementMyTypeCreateInputs {
  ...EntitlementCreateSharedFields;
  type: EntitlementType.myType;
  myField: string;
}

// 2. Add read model:
#deprecated "Use EntitlementMyTypeV2 instead"
#suppress "deprecated" "V1 Entitlements APIs will be removed on December 1st, 2025"
// ...
```

<!-- archie:ai-end -->
