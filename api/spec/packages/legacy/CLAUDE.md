# legacy

<!-- archie:ai-start -->

> TypeSpec source for the v1/v2 OpenMeter REST API and OpenMeter Cloud API; compiles to api/openapi.yaml and api/openapi.cloud.yaml plus the Python SDK. src/ is the composition root with shared primitives; cloud/ is the cloud-variant entry point. New v1 domain features go here; v3 changes go to api/spec/packages/aip instead.

## Patterns

**main.tsp as sole compilation entry point** — All sub-domain .tsp files must be imported into src/cloud/main.tsp (or src/main.tsp for client). Unlisted files are silently excluded from the compiled spec. (`src/cloud/main.tsp: import "../meters.tsp"; import "../billing/main.tsp";`)
**@friendlyName on every model, enum, union, interface** — Controls the OpenAPI schema name; required on all types to produce deterministic SDK class names. linter enforces this at error severity. (`@friendlyName("MeterCreate") model MeterCreate is TypeSpec.Rest.Resource.ResourceCreateModel<Meter>;`)
**@visibility lifecycle decorators on model fields** — Use Lifecycle.Read, Lifecycle.Create, Lifecycle.Update to control which fields appear in which request/response shapes from a single model definition. (`@visibility(Lifecycle.Read) id: ULID; @visibility(Lifecycle.Read, Lifecycle.Create) slug: Key;`)
**REST resource templates for CRUD request bodies** — Use TypeSpec.Rest.Resource.ResourceCreateModel<T>, ResourceReplaceModel<T>, ResourceCreateOrUpdateModel<T> to derive request bodies; avoids ad-hoc types and ensures visibility filtering. (`@friendlyName("MeterUpdate") model MeterUpdate is TypeSpec.Rest.Resource.ResourceReplaceModel<Meter>;`)
**Shared primitives in types.tsp, never re-declared in sub-domains** — ULID, DateTime, Key, Resource, ResourceTimestamps, CadencedResource and pagination models live in src/types.tsp under the OpenMeter namespace; importing them avoids duplicate schema definitions. (`// In a sub-domain file: use OpenMeter.ULID, not a local re-declaration`)
**Filter models with @extension("x-omitempty", true) on all operator fields** — Filter types in filter.tsp mark every operator property with @extension("x-omitempty", true) so absent filters are excluded from serialised queries. (`@extension("x-omitempty", true) eq?: string;`)
**@operationId on every interface operation** — Explicit camelCase verb+noun operationId prevents non-deterministic SDK function names. (`@operationId("createMeter") create(@body body: MeterCreate): Meter | OpenMeterError;`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `src/cloud/main.tsp` | Cloud-variant compilation entry point; imports all sub-domain files | New sub-domain file not imported here is silently excluded from api/openapi.cloud.yaml |
| `src/main.tsp` | Client-emit entry point for Python SDK generation | Separate from cloud/main.tsp; both must be kept in sync for new domains |
| `src/types.tsp` | Shared primitive types: ULID, DateTime, Key, Resource, pagination models | Never re-declare these in sub-domain files; import from this file instead |
| `src/errors.tsp` | OpenMeterError and error union types; @error decorator required | Extending Error without @error breaks OpenAPI error schema generation |
| `src/filter.tsp` | Generic filter operator models; all fields must have @extension("x-omitempty", true) | Missing x-omitempty causes empty filter fields to be serialised as null in query strings |
| `lib/index.js` | Registers all v1 linter rules via defineLinter | New rule file has zero effect without registration here |
| `tspconfig.yaml / tspconfig.client.yaml` | Two configs: OpenAPI emit and Python SDK emit respectively | Both extend @openmeter/api-spec-legacy/all linter; changing emitter-output-dir breaks `make gen-api` path expectations |

## Anti-Patterns

- Defining new domain models directly in root-level .tsp files instead of a sub-folder with its own main.tsp
- Re-declaring primitive types (ULID, DateTime, Key, Resource) in sub-domain files — they already exist in types.tsp
- Using `extends Error` without the @error decorator — breaks OpenAPI error schema generation
- Omitting @operationId on interface operations — causes non-deterministic generated SDK function names
- Adding new sub-domain files without registering them in cloud/main.tsp and src/main.tsp imports

## Decisions

- **Two separate tspconfig files (tspconfig.yaml for OpenAPI, tspconfig.client.yaml for Python SDK)** — OpenAPI and Python SDK emitters need different output directories and emitter options; a single config cannot parameterise both without overriding emitter-output-dir per run
- **Shared primitive types centralised in types.tsp with Resource base models** — Keeps schema naming deterministic across all sub-domains and prevents duplicate $ref targets in generated OpenAPI that confuse SDK generators
- **friendlyNameRule is error-severity (blocks gen-api); all other rules are warnings** — @friendlyName controls SDK class names; a missing or wrong name produces irreversible public API breakage, justifying a hard block on compilation

## Example: Adding a new v1 resource (e.g. Widget) with CRUD operations

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
