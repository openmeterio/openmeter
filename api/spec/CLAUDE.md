# spec

<!-- archie:ai-start -->

> TypeSpec source of truth for the OpenMeter API contract, authored as a pnpm workspace. It compiles two TypeSpec packages into the OpenAPI specs and SDKs the rest of the repo depends on; everything under api/ (openapi.yaml, api/v3/openapi.yaml, api.gen.go, all client SDKs) is generated downstream from here.

## Patterns

**Two packages, two API generations** — packages/legacy/ emits the v1/v2 OpenMeter + Cloud specs and the Python client; packages/aip/ emits the AIP-style v3 spec. Add new surface to the package matching the API generation. (`package.json `generate` runs `--filter @openmeter/api-spec-legacy run generate && --filter @openmeter/api-spec-aip run generate``)
**Build orchestration lives in this Makefile, not the children** — The api/spec Makefile runs pnpm generate, then AIP $ref rewriting (yq), `openapi bundle` for v3, and copies outputs into ../ and ../v3/. Running `tsp compile` directly skips these steps. (`Makefile generate rewrites SortQuery/StringFieldFilter schemas to $ref common/definitions/aip_filters.yaml, then cp legacy outputs to ../openapi.yaml and ../openapi.cloud.yaml`)
**pnpm workspace with patched, version-pinned TypeSpec** — pnpm-workspace.yaml globs packages/*; TypeSpec compiler/http/openapi/openapi3 and http-client-python are pinned and locally patched under patches/. .npmrc enforces save-exact=true. (`package.json pnpm.patchedDependencies maps @typespec/http, @typespec/compiler, @typespec/openapi(3), @typespec/http-client-python to files in patches/`)
**Format and lint before generate** — `format` (prettier + aip format) and `lint` (prettier --check + per-package lint incl. custom AIP/legacy rules) run via pnpm scripts; the Makefile generate target depends on format. (`Makefile `generate: format`; package.json `lint` runs prettier --check plus both package lint scripts`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `Makefile` | Top-level orchestration: install, format, generate (yq $ref rewrite + openapi bundle + copy-to-api), lint. | The copy/bundle/yq steps live ONLY here; bypassing them leaves api/openapi.yaml and api/v3/openapi.yaml stale or with inline filter schemas instead of $ref |
| `package.json` | Workspace root: generate/format/lint scripts, pinned TypeSpec deps, patchedDependencies/overrides. | private, version 0.1.0; exports map points at packages/*/output, which are gitignored build artifacts |
| `pnpm-workspace.yaml` | Declares workspace members as packages/* with trustPolicy/minimumReleaseAge guards. | A new package outside packages/* is never discovered by pnpm |
| `patches/` | Local patches for the pinned TypeSpec compiler/http/openapi/openapi3/http-client-python packages. | Bumping a patched dep requires regenerating its patch hash in package.json |
| `.gitignore / .prettierignore` | Excludes generated output (**/output/, packages/**/output/), node_modules, and pnpm-lock from formatting/VCS. | Generated OpenAPI under packages/*/output is intentionally untracked; the committed artifacts are the copies in api/ |

## Anti-Patterns

- Editing generated artifacts (api/openapi.yaml, api/v3/openapi.yaml, api.gen.go, Go/JS/Python SDKs) instead of the .tsp sources here
- Running tsp compile directly and skipping the Makefile (misses the yq $ref rewrite, AIP openapi bundle, and copy-to-api steps)
- Adding v3 surface to packages/legacy or v1/v2 surface to packages/aip
- Putting cross-package generation logic inside packages/* instead of the parent Makefile
- Unpinning or floating TypeSpec dependency versions, invalidating the local patches

## Decisions

- **Two separate TypeSpec packages (legacy v1/v2/cloud and AIP v3) rather than one shared spec.** — The v3 surface follows Kong AIP conventions and a different pipeline (bundle + filter $ref rewrite) than the legacy surface, so they stay isolated.
- **TypeSpec is the single source of truth; all OpenAPI and SDKs are generated.** — Keeps multi-language clients and server stubs consistent and lets the API contract be authored once.

## Example: Regenerate all specs and SDKs from the TypeSpec sources

```
make gen-api          # repo root, or:
make -C api/spec generate   # pnpm generate, yq $ref rewrite, openapi bundle, then copy to api/ and api/v3/
```

<!-- archie:ai-end -->
