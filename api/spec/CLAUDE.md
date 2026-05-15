# spec

<!-- archie:ai-start -->

> Single source of truth for all OpenMeter HTTP APIs. TypeSpec packages under packages/ compile to api/openapi.yaml (v1), api/openapi.cloud.yaml (v1 cloud), and api/v3/openapi.yaml (v3 AIP). All Go server stubs, Go/JS/Python SDKs are downstream — never edit generated artefacts directly.

## Patterns

**Two-package split: aip vs legacy** — packages/aip/ owns v3 AIP-style spec (tspconfig compiles to api/v3/openapi.yaml); packages/legacy/ owns v1/v2 spec (compiles to api/openapi.yaml and api/openapi.cloud.yaml). Never mix v3 content into legacy/ or v1 content into aip/. (`New billing v3 endpoint: add .tsp file under packages/aip/src/, import in openmeter.tsp root namespace.`)
**Route/tag binding at root namespace only** — Domain sub-folder .tsp files declare models and operations; @route and @tag decorators are bound only in root namespace files (openmeter.tsp, konnect.tsp, main.tsp). (`packages/aip/src/openmeter.tsp imports sub-domain files and applies @route at the top-level interface.`)
**make gen-api is the only regeneration path** — Running pnpm generate → pnpm --filter compile → openapi bundle → cp produces all output artefacts. Direct tsc or tsp compile calls skip the yq filter-schema loop and flatten-allof post-processing. (`Makefile generate target: pnpm --filter @openmeter/api-spec-aip run compile → openapi bundle → ../../../v3/openapi.yaml`)
**Vendored patches applied via pnpm patchedDependencies** — TypeSpec compiler/emitter bugs are fixed via patches/ files applied at npm install time. Each patch targets compiled dist/ not TypeScript src/. (`package.json pnpm.patchedDependencies maps @typespec/compiler to patches/@typespec__compiler.patch`)
**New legacy sub-domain file must register in both main.tsp and cloud/main.tsp** — packages/legacy/src/main.tsp and packages/legacy/src/cloud/main.tsp are the compilation entry points; files not imported there are silently ignored. (`Adding packages/legacy/src/myfeature.tsp: add `import './myfeature.tsp';` to both main.tsp files.`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `api/spec/packages/aip/src/openmeter.tsp` | Root namespace file for v3 OpenMeter spec; all @route/@tag bindings live here. | Adding @route inside sub-domain files breaks the routing contract. |
| `api/spec/packages/aip/scripts/flatten-allof.mjs` | Post-processing step that flattens allOf in the AIP output YAML; runs after tsp compile. | If upstream TypeSpec fixes allOf generation this script may become a no-op but must not be deleted without verification. |
| `api/spec/packages/legacy/src/main.tsp` | Entry point for v1/v2 OpenMeter compilation. | New sub-domain files must be imported here AND in cloud/main.tsp. |
| `api/spec/packages/legacy/src/types.tsp` | Shared primitive types (ULID, DateTime, Resource) for v1 spec. | Never re-declare these in sub-domain files. |
| `api/spec/Makefile` | Canonical generate/lint/format targets; called by root `make gen-api`. | The yq filter-schema loop replaces inline filter definitions with $ref after compile — filter schemas must match expected names. |
| `api/spec/package.json` | Workspace root; declares TypeSpec version pins and patchedDependencies. | Bumping a patched package version requires updating the corresponding patch file hash; re-test full make gen-api pipeline. |
| `api/spec/patches/` | Vendored patch files (patch-package format) applied to TypeSpec compiler/emitter packages at install time. | Patch compiled dist/ not TypeScript src/; one concern per patch; remove stale patches when upstream ships the fix. |

## Anti-Patterns

- Hand-editing api/openapi.yaml, api/openapi.cloud.yaml, or api/v3/openapi.yaml — always regenerate via make gen-api
- Declaring @route or @tag inside domain sub-folder operation files — routing belongs in root namespace files only
- Patching TypeScript source files (src/**/*.ts) in patches/ — node_modules ships dist/ only; source patches are silently ignored
- Adding v3 domain content into packages/legacy/ or v1 content into packages/aip/
- Adding a new legacy sub-domain file without registering it in both main.tsp and cloud/main.tsp

## Decisions

- **Two separate packages (aip/ and legacy/) with independent tspconfig files** — v3 AIP spec and v1 legacy spec have different emitter configurations, versioning strategies, and output targets; a single package would require complex conditional compilation.
- **Post-processing flatten-allof.mjs in aip/ rather than upstream TypeSpec changes** — Upstream TypeSpec allOf generation cannot be changed without forking; a post-process script keeps the TypeSpec source clean.
- **Vendored patches on dist/ of TypeSpec packages instead of forking** — Forking would block upstream upgrades; minimal dist/ patches are reviewable and can be dropped when upstream fixes land.

## Example: Add a new v3 resource endpoint in the AIP package

```
// packages/aip/src/resources/myresource.tsp
import "@typespec/http";
import "@typespec/rest";
using TypeSpec.Http;
using TypeSpec.Rest;

model MyResource {
  id: string;
  name: string;
}

// Register in packages/aip/src/openmeter.tsp:
// import "./resources/myresource.tsp";
// @route("/api/v1/myresources")
// interface MyResourceRoutes { ... }
```

<!-- archie:ai-end -->
