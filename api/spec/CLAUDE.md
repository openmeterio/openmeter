# spec

<!-- archie:ai-start -->

> Single source of truth for every OpenMeter HTTP API, authored in TypeSpec and split into two independently compiled packages: packages/aip (v3 AIP-style -> api/v3/openapi.yaml + Konnect variant) and packages/legacy (v1/v2 + Cloud -> api/openapi.yaml, api/openapi.cloud.yaml). All OpenAPI YAMLs and the Go/JS/Python SDKs are downstream artefacts regenerated exclusively by make gen-api; vendored patches/ fix upstream TypeSpec compiler/emitter bugs at install time.

## Patterns

**Two-package version split: aip vs legacy** — packages/aip owns the v3 AIP spec (its tspconfig compiles to api/v3/openapi.yaml); packages/legacy owns v1/v2 and Cloud (compiles to api/openapi.yaml and api/openapi.cloud.yaml). Never mix v3 content into legacy/ or v1 content into aip/ — they compile to separate targets. (`New v3 endpoint: add .tsp under packages/aip/src/ and import it in the root openmeter.tsp.`)
**Route/tag binding only at root namespace files** — Domain sub-folder .tsp files declare models and operations; @route and @tag decorators are bound only in the root namespace files (aip/src/openmeter.tsp, konnect.tsp; legacy/src/main.tsp, cloud/main.tsp). (`packages/aip/src/openmeter.tsp imports sub-domain files and applies @route at the top-level interface.`)
**make gen-api is the only regeneration path** — The api/spec Makefile runs pnpm generate, then a yq filter-schema $ref loop, openapi bundle, flatten-allof post-processing, and cp into api/. Direct tsc/tsp calls skip these steps and desync the artefacts. (`Makefile generate: pnpm --filter @openmeter/api-spec-aip run compile -> openapi bundle -> ../../../v3/openapi.yaml`)
**Legacy sub-domain files must register in both main.tsp and cloud/main.tsp** — packages/legacy/src/main.tsp and packages/legacy/src/cloud/main.tsp are the two compilation entry points; a new legacy .tsp not imported in both is silently ignored. (`Adding packages/legacy/src/myfeature.tsp: add `import './myfeature.tsp';` to both main.tsp files.`)
**Vendored patches applied via pnpm patchedDependencies** — TypeSpec compiler/emitter bugs are fixed via single-concern unified diffs in patches/, mapped in package.json pnpm.patchedDependencies and applied at install; each patch targets compiled dist/ not TypeScript src/ and is version-coupled to the pinned package. (`package.json maps @typespec/compiler -> patches/@typespec__compiler.patch`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `api/spec/packages/aip/src/openmeter.tsp` | Root namespace file for the v3 spec; all @route/@tag bindings live here. | Declaring @route inside sub-domain files breaks the routing contract. |
| `api/spec/packages/aip/scripts/flatten-allof.mjs` | Post-processing step that flattens allOf in the AIP output YAML after tsp compile. | May become a no-op if upstream fixes allOf generation, but must not be deleted without verifying. |
| `api/spec/packages/legacy/src/main.tsp` | Entry point for v1/v2 compilation. | New sub-domain files must be imported here AND in cloud/main.tsp or they are silently dropped. |
| `api/spec/packages/legacy/src/types.tsp` | Shared primitive types (ULID, DateTime, Key, Resource) for the v1 spec. | Never re-declare these primitives in sub-domain files. |
| `api/spec/Makefile` | Canonical generate/lint/format targets called by root make gen-api. | The yq filter-schema loop replaces inline filter definitions with $ref to common/definitions/aip_filters.yaml — filter schema names must match expected (SortQuery, *FieldFilter, etc.). |
| `api/spec/package.json` | Workspace root; pins TypeSpec version (1.11.0) and declares patchedDependencies and named exports. | Bumping a patched package requires updating its patch and re-running the full make gen-api pipeline. |
| `api/spec/patches/` | Vendored patch-package diffs applied to TypeSpec compiler/emitter at install time. | Patch compiled dist/ not TypeScript src/ (node_modules ships dist only); one concern per patch; remove stale patches once upstream ships the fix. |

## Anti-Patterns

- Hand-editing api/openapi.yaml, api/openapi.cloud.yaml, or api/v3/openapi.yaml — always regenerate via make gen-api.
- Declaring @route or @tag inside a domain sub-folder operations.tsp instead of the root namespace files.
- Adding v3 content into packages/legacy/ or v1/v2 content into packages/aip/ — they compile to separate targets and mix-ins break both.
- Adding a new legacy sub-domain file without registering it in both main.tsp and cloud/main.tsp.
- Patching TypeScript src/**/*.ts in patches/ — node_modules ships dist/ only, so source patches are silently ignored.

## Decisions

- **Two separate packages (aip/ and legacy/) with independent tspconfig files and entry points.** — v3 AIP and v1 legacy have different emitter configs, versioning strategies, and output targets; a single package would require complex conditional compilation.
- **Post-processing via flatten-allof.mjs in aip/ rather than altering TypeSpec source.** — Upstream allOf generation cannot be changed without forking; a post-process script keeps the TypeSpec source clean.
- **Vendored patches on dist/ of TypeSpec packages instead of forking them.** — Forking would block upstream upgrades; minimal reviewable dist/ patches can be dropped when upstream fixes land.

## Example: Add a new v3 resource endpoint in the AIP package

```
// packages/aip/src/resources/myresource.tsp
import "@typespec/http";
import "@typespec/rest";
using TypeSpec.Http;
using TypeSpec.Rest;

model MyResource { id: string; name: string; }

// Register in packages/aip/src/openmeter.tsp:
// import "./resources/myresource.tsp";
// @route("/api/v1/myresources") interface MyResourceRoutes { ... }
```

<!-- archie:ai-end -->
