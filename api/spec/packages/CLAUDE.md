# packages

<!-- archie:ai-start -->

> pnpm workspace container holding the two TypeSpec source-of-truth packages that author the OpenMeter API contract. It splits the surface by API generation: legacy/ emits the v1/v2 + Cloud OpenAPI (and a Python client), while aip/ emits the Google-AIP-style v3 OpenAPI. Everything downstream (api/openapi.yaml, api/v3/openapi.yaml, Go/JS SDKs) is generated from these .tsp sources and must never be hand-edited.

## Patterns

**Two packages, two API generations** — legacy/ owns v1/v2 + Cloud; aip/ owns the AIP-style v3 surface. Pick the package by which API version the change targets; do not cross-author v3 types in legacy or vice versa. (`v3 list-filter type -> packages/aip/src; new v1 portal endpoint -> packages/legacy/src`)
**Workspace member discovery via pnpm-workspace.yaml** — ../pnpm-workspace.yaml globs 'packages/*', so each child is a standalone pnpm package with its own package.json/tspconfig.yaml. A new sibling only participates once it lives directly under packages/. (`packages:\n  - packages/*`)
**Build orchestrated from the parent spec Makefile, not here** — ../Makefile generate target runs pnpm generate across both members, then post-processes (yq $ref rewrite of AIP filters), bundles the aip output, and copies emitted YAML to api/openapi.yaml, api/openapi.cloud.yaml, api/v3/openapi.yaml. Output paths are fixed by that target. (`cp packages/legacy/output/openapi.OpenMeter.yaml ../openapi.yaml`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `aip/` | TypeSpec package for the AIP-style v3 API; stricter custom linter (kebab op-ids, no-nullable, field-prefix); two-stage tsp-compile-then-Node-post-process build with shared common/definitions fragments. | Skipping flatten-allof / seal-object-schemas post-processors, or editing emitted output/definitions YAML instead of src/. |
| `legacy/` | TypeSpec package for v1/v2 + Cloud; dual emit (OpenAPI3 + Python client) via tspconfig.yaml + tspconfig.client.yaml, with a lighter custom linter than aip. | Pointing the OpenAPI emit at ./src instead of ./src/cloud, and CRUD bodies not using the Rest.Resource ResourceCreate/ReplaceModel templates from README. |

## Anti-Patterns

- Editing generated artifacts (api/openapi.yaml, api/v3/openapi.yaml, api/api.gen.go, Go/JS/Python SDKs) instead of the .tsp sources here.
- Adding v3 surface to legacy/ or v1 surface to aip/ instead of the package matching the API generation.
- Adding a package outside packages/* so pnpm-workspace.yaml never picks it up.
- Running tsp compile directly and bypassing the parent Makefile generate target (misses the yq filter $ref rewrite, the aip openapi bundle, and the copy-to-api steps).

## Decisions

- **Two separate TypeSpec packages rather than one shared spec.** — v1/legacy and v3/AIP have divergent conventions and linters (AIP enforces kebab op-ids, no-nullable, field prefixes); isolating them lets each evolve and lint independently while sharing one pnpm workspace.
- **Cross-package generation logic lives in the parent spec Makefile, not inside packages/.** — Steps spanning both members (yq filter $ref rewrite, openapi bundle of the aip output, copying emitted YAML into api/ and api/v3/) must run in a fixed order, so they sit one level up rather than in either package.

<!-- archie:ai-end -->
