# legacy

<!-- archie:ai-start -->

> TypeSpec source for the v1/v2 OpenMeter REST API and OpenMeter Cloud API; compiles to api/openapi.yaml, api/openapi.cloud.yaml, and the Python SDK. src/ is the composition root with shared primitives (types.tsp, errors.tsp, filter.tsp); cloud/main.tsp is the cloud-variant entry point. New v1 domain features go here; v3 changes go to api/spec/packages/aip.

## Patterns

**cloud/main.tsp as sole OpenAPI compilation entry point** — All sub-domain .tsp files must be imported into src/cloud/main.tsp. Files not listed are silently excluded from api/openapi.cloud.yaml. src/main.tsp is the separate entry for Python SDK generation. (`src/cloud/main.tsp: import "../meters.tsp"; import "../billing/main.tsp";`)
**@friendlyName on every named type** — Every model, enum, union, and interface must have @friendlyName controlling the OpenAPI schema name and SDK class name. The linter enforces this at error severity — missing @friendlyName blocks gen-api. (`@friendlyName("MeterCreate") model MeterCreate is TypeSpec.Rest.Resource.ResourceCreateModel<Meter>;`)
**@visibility lifecycle decorators on model fields** — Use Lifecycle.Read, Lifecycle.Create, Lifecycle.Update to control which fields appear in which request/response shapes from a single model definition instead of defining separate request/response models. (`@visibility(Lifecycle.Read) id: ULID; @visibility(Lifecycle.Read, Lifecycle.Create) slug: Key;`)
**REST resource templates for CRUD request bodies** — Derive request body models from TypeSpec.Rest.Resource.ResourceCreateModel<T>, ResourceReplaceModel<T>, ResourceCreateOrUpdateModel<T>. Do not define ad-hoc request body types. (`@friendlyName("MeterUpdate") model MeterUpdate is TypeSpec.Rest.Resource.ResourceReplaceModel<Meter>;`)
**Shared primitives in types.tsp — never re-declared** — ULID, DateTime, Key, Resource, ResourceTimestamps, CadencedResource, and all pagination models live in src/types.tsp under the OpenMeter namespace. Sub-domain files import from there; never re-declare these types. (`// Sub-domain file uses: OpenMeter.ULID — not a local re-declaration`)
**@extension("x-omitempty", true) on all filter operator fields** — Every operator property in filter models must carry @extension("x-omitempty", true) so absent filters are excluded from serialised query strings. (`filter.tsp: @extension("x-omitempty", true) eq?: string;`)
**@operationId on every interface operation** — Explicit camelCase verb+noun @operationId prevents non-deterministic SDK function names across Go, JS, and Python clients. (`@operationId("createMeter") create(@body body: MeterCreate): Meter | OpenMeterError;`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `src/cloud/main.tsp` | Cloud-variant compilation entry point; imports all sub-domain files to produce api/openapi.cloud.yaml. | New sub-domain file not imported here is silently excluded from api/openapi.cloud.yaml. |
| `src/main.tsp` | Client-emit entry point for Python SDK generation via tspconfig.client.yaml. | Separate from cloud/main.tsp; new domains must be added to both to keep OpenAPI and Python SDK in sync. |
| `src/types.tsp` | Shared primitive types: ULID, DateTime, Key, Resource, pagination models. Single source of truth for all shared types. | Never re-declare these in sub-domain files; any duplication creates divergent $ref targets in generated OpenAPI. |
| `src/errors.tsp` | OpenMeterError and error union types with @error decorator. | Extending Error without @error decorator breaks OpenAPI error schema generation — error responses will not be tagged correctly. |
| `src/filter.tsp` | Generic filter operator models; every field must have @extension("x-omitempty", true). | Missing x-omitempty causes empty filter fields to be serialised as null in query strings, breaking filter semantics. |
| `lib/index.js` | Registers all v1 linter rules via defineLinter/$linter export. | New rule file has zero effect without registration here; no compile error signals the omission. |
| `tspconfig.yaml` | OpenAPI emitter config; output dir must match Makefile expectations for api/openapi.cloud.yaml. | Changing emitter-output-dir breaks the `make gen-api` path expectations and the CI dirty-tree check. |
| `tspconfig.client.yaml` | Python SDK emitter config; points to api/client/python output directory. | Both tspconfig files extend @openmeter/api-spec-legacy/all linter; removing this breaks linting for whichever emit run omits it. |

## Anti-Patterns

- Defining new domain models directly in root-level .tsp files — create a sub-folder with its own main.tsp instead
- Re-declaring primitive types (ULID, DateTime, Key, Resource) in sub-domain files — they already exist in src/types.tsp
- Using `extends Error` without the @error decorator — breaks OpenAPI error schema generation
- Omitting @operationId on interface operations — causes non-deterministic generated SDK function names
- Adding new sub-domain files without registering them in both src/cloud/main.tsp and src/main.tsp imports

## Decisions

- **Two separate tspconfig files (tspconfig.yaml for OpenAPI, tspconfig.client.yaml for Python SDK)** — OpenAPI and Python SDK emitters need different output directories and emitter-specific options; a single config cannot parameterise both without complex per-run overrides.
- **Shared primitive types centralised in types.tsp with Resource base models** — Keeps schema naming deterministic across all sub-domains and prevents duplicate $ref targets in generated OpenAPI that confuse SDK generators and produce mismatched type names.
- **friendlyNameRule is error-severity (blocks gen-api); all other rules are warnings** — @friendlyName controls SDK class names across Go, JS, and Python; a missing or wrong name produces an irreversible public API breakage, justifying a hard block on compilation.

## Example: Adding a new v1 resource (Widget) with CRUD operations

```
// src/widgets/main.tsp
import "@typespec/http";
import "@typespec/rest";
using TypeSpec.Http;
using TypeSpec.Rest;
using TypeSpec.Rest.Resource;

@friendlyName("Widget")
model Widget {
  @visibility(Lifecycle.Read)
  id: OpenMeter.ULID;

  @visibility(Lifecycle.Read, Lifecycle.Create, Lifecycle.Update)
  name: string;
}
// ...
```

<!-- archie:ai-end -->
