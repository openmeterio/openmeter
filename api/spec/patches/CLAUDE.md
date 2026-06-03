# patches

<!-- archie:ai-start -->

> Vendored patch-package diffs applied to TypeSpec compiler/emitter npm packages during `npm install` to fix upstream bugs or add features the OpenMeter API-generation pipeline needs. Primary constraint: each patch is tightly coupled to the exact package version pinned in api/spec/package.json.

## Patterns

**patch-package unified diff format** — Each file is a unified diff named `@scope__package.patch` exactly matching the npm package name under node_modules, targeting compiled dist/ JS and .d.ts(.map) files — never TypeScript src/ (node_modules ships only dist/). (`@typespec__openapi3.patch modifies dist/src/attach-extensions.js to skip the x-inline extension key during OpenAPI emission`)
**Single-concern per patch** — One bug fix or feature addition per patch file to keep diffs reviewable and reduce conflicts on version upgrades. (`@typespec__http.patch only adds the `style` field to QueryOptions`)
**Version-coupled patches** — A patch references specific dist/ layout and commit hashes; any package version bump in package.json requires re-verifying the patch still applies or is still needed. (`@typespec__compiler.patch targets dist/manifest.js commit hash 478dfed5`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `@typespec__compiler.patch` | Adds $functions binder support, propagates ModifierFlags.Internal, replaces globby with Node built-in glob in formatter-fs.js, plus config-schema linter changes. | On compiler upgrade re-verify binder.js still lacks $functions/internal-modifier handling; remove if fixed upstream. |
| `@typespec__http.patch` | Adds `style` field to QueryOptions enabling form|spaceDelimited|pipeDelimited|deepObject serialization on @query params. | May be merged upstream — check release notes before upgrading. |
| `@typespec__openapi.patch` | Adds `parent` to TagMetadata for hierarchical tag grouping and exports getOperationId/setOperationId. | The parent field is OpenMeter-specific; spec tag grouping depends on it — do not drop on upgrade. |
| `@typespec__openapi3.patch` | Skips x-inline extension key in attachExtensions so it is not emitted into generated YAML; adds directive support in generateOperationParameter. | Removing it lets x-inline leak into YAML, breaking SDK generators. |
| `@typespec__http-client-python.patch` | Python emitter: silences a pylint false positive and adds constants.js (blob URLs), unbranded flavor detection, and Pyodide generation code. | Goes well beyond a trivial style fix — review the emitter.js Pyodide changes carefully on upgrade. |
| `CLAUDE.md` | Manually maintained docs on patch purposes and the patch-package mechanism, with archie:ai markers. | Update whenever a patch is added/modified/removed; note which upstream fixes allow patch removal. |

## Anti-Patterns

- Patching TypeScript source files (src/**/*.ts) — node_modules only ships dist/, so source patches are silently ignored
- Bundling multiple unrelated fixes into one patch file
- Using patches to change API semantics or add TypeSpec language features — those belong in api/spec/packages/
- Bumping a patched package version without re-running the full make gen-api pipeline
- Leaving stale patches in place after upstream ships the fix

## Decisions

- **Patch compiled dist/ files rather than forking the packages** — patch-package applies diffs non-invasively, reverts cleanly on upgrade, and avoids maintaining a separate registry or changing import paths in TypeSpec source.
- **Co-locate patches in api/spec/patches/ next to the TypeSpec source** — Patches are exclusively part of the TypeSpec compilation toolchain, not the Go backend; adjacency makes that scope obvious.

<!-- archie:ai-end -->
