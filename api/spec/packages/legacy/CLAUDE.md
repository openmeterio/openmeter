# legacy

<!-- archie:ai-start -->

> Root TypeSpec package for the v1/v2 OpenMeter REST API and the OpenMeter Cloud API. Compiles to api/openapi.yaml, api/openapi.cloud.yaml, and the Python SDK via make gen-api. Primary constraint: src/ is the composition root holding shared primitives (types.tsp, errors.tsp, filter.tsp, query.tsp) and two emit entry points — src/cloud/main.tsp for cloud OpenAPI and src/main.tsp for the Python SDK — that must stay in sync; new v1 domain features belong here, v3 changes go to api/spec/packages/aip.

## Patterns

**Dual entry points kept in sync** — src/cloud/main.tsp imports every sub-domain to produce api/openapi.cloud.yaml; src/main.tsp is the separate client-emit entry for the Python SDK. A new sub-domain must be registered in BOTH or it silently vanishes from one output. (`src/cloud/main.tsp: import "../meters.tsp"; import "../billing/main.tsp";`)
**@friendlyName on every named declaration** — Every model, enum, union, and interface must carry @friendlyName, which controls the OpenAPI schema name and the SDK class name. The linter enforces this at error severity — a missing @friendlyName blocks gen-api. (`@friendlyName("MeterCreate") model MeterCreate is TypeSpec.Rest.Resource.ResourceCreateModel<Meter>;`)
**@visibility lifecycle decorators on fields** — Use Lifecycle.Read / Lifecycle.Create / Lifecycle.Update to derive request and response shapes from a single model instead of defining separate request/response types. (`@visibility(Lifecycle.Read) id: ULID; @visibility(Lifecycle.Read, Lifecycle.Create) slug: Key;`)
**REST resource templates for CRUD bodies** — Request body models derive from TypeSpec.Rest.Resource.ResourceCreateModel<T> (POST → {Name}Create), ResourceReplaceModel<T> (PUT → {Name}Update), or ResourceCreateOrUpdateModel<T> (PATCH → {Name}Patch). No ad-hoc RequestBody/Input/Payload types. (`@friendlyName("MeterUpdate") model MeterUpdate is TypeSpec.Rest.Resource.ResourceReplaceModel<Meter>;`)
**Shared primitives centralised in types.tsp** — ULID, DateTime, Key, Resource, ResourceTimestamps, CadencedResource, and pagination models live in src/types.tsp under namespace OpenMeter. Sub-domain files reference them (OpenMeter.ULID); never re-declare. (`// Sub-domain file uses OpenMeter.ULID — not a local re-declaration`)
**@extension("x-omitempty", true) on filter operator fields** — Every operator property in filter models carries @extension("x-omitempty", true) so absent filters are excluded from serialised query strings rather than emitted as null. (`filter.tsp: @extension("x-omitempty", true) eq?: string;`)
**@operationId on every interface operation** — Explicit camelCase verb+noun @operationId prevents non-deterministic generated SDK function names across Go, JS, and Python. (`@operationId("createMeter") create(@body body: MeterCreate): Meter | OpenMeterError;`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `src/cloud/main.tsp` | Cloud-variant compilation entry; imports all sub-domains to produce api/openapi.cloud.yaml. | A new sub-domain file not imported here is silently excluded from api/openapi.cloud.yaml. |
| `src/main.tsp` | Client-emit entry for Python SDK generation via tspconfig.client.yaml. | Separate from cloud/main.tsp — new domains must be added to both to keep OpenAPI and Python SDK aligned. |
| `src/types.tsp` | Single source of truth for shared primitives: ULID, DateTime, Key, Resource, pagination models. | Re-declaring these in sub-domain files creates divergent $ref targets in generated OpenAPI. |
| `src/errors.tsp` | OpenMeterError and error union types carrying the @error decorator. | Extending Error without @error breaks OpenAPI error schema generation — error responses are not tagged correctly. |
| `src/filter.tsp` | Generic filter operator models; every field needs @extension("x-omitempty", true). | Missing x-omitempty serialises empty filter fields as null, breaking filter semantics. |
| `src/query.tsp` | Distinct pagination shapes: QueryPagination, QueryLimitOffset, QueryCursorPagination. | Mixing pagination shapes across endpoints produces inconsistent generated list signatures. |
| `lib/index.js` | Registers all v1 linter rules via defineLinter/$linter export. | A new rule file has zero effect without registration here; no compile error signals the omission. |
| `tspconfig.yaml / tspconfig.client.yaml` | Two emitter configs: OpenAPI (cloud) vs Python SDK output dirs; both extend @openmeter/api-spec-legacy/all. | Changing emitter-output-dir breaks make gen-api path expectations and the CI dirty-tree check; dropping the linter extends disables linting for that emit run. |

## Anti-Patterns

- Defining new domain models directly in root-level .tsp files instead of a sub-folder with its own main.tsp
- Re-declaring primitive types (ULID, DateTime, Key, Resource) in sub-domain files — they live in src/types.tsp
- Using `extends Error` without the @error decorator — breaks OpenAPI error schema generation
- Omitting @operationId on interface operations — yields non-deterministic generated SDK function names
- Adding a sub-domain file without registering it in both src/cloud/main.tsp and src/main.tsp imports

## Decisions

- **Two tspconfig files (tspconfig.yaml for OpenAPI, tspconfig.client.yaml for Python SDK)** — The OpenAPI and Python SDK emitters need different output directories and emitter-specific options that a single config cannot parameterise without complex per-run overrides.
- **Shared primitives centralised in types.tsp with Resource base models** — Keeps schema naming deterministic across sub-domains and prevents duplicate $ref targets that confuse SDK generators and produce mismatched type names.
- **friendlyNameRule is error-severity; all other rules are warnings** — @friendlyName controls SDK class names across Go, JS, and Python, so a missing or wrong name is an irreversible public API breakage that justifies a hard compile block.

## Example: Adding a new v1 resource with visibility-controlled CRUD

```
// src/widgets/main.tsp
import "@typespec/http";
import "@typespec/rest";
using TypeSpec.Http;
using TypeSpec.Rest;
using TypeSpec.Rest.Resource;

@friendlyName("Widget")
model Widget {
  @visibility(Lifecycle.Read) id: OpenMeter.ULID;
  @visibility(Lifecycle.Read, Lifecycle.Create, Lifecycle.Update) name: string;
}
@friendlyName("WidgetCreate") model WidgetCreate is ResourceCreateModel<Widget>;
// then register the import in both src/cloud/main.tsp and src/main.tsp
```

<!-- archie:ai-end -->
