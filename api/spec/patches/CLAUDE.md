# patches

<!-- archie:ai-start -->

> Holds vendored patch files (patch-package format) applied to TypeSpec compiler and emitter npm packages to fix upstream bugs or add missing features required by OpenMeter's API generation pipeline. Patches are applied during `npm install` and must stay synchronized with the exact package versions pinned in api/spec/package.json.

## Patterns

**patch-package unified diff format** — Each patch file is a unified diff named `@scope__package.patch` exactly matching the npm package name under node_modules. The diff targets compiled dist/ JS and type declaration map files, not TypeScript source files, because node_modules ships only the built distribution. (``@typespec__openapi3.patch` modifies `dist/src/attach-extensions.js` to skip the `x-inline` extension key during OpenAPI YAML emission.`)
**Single-concern per patch** — Each patch file addresses exactly one bug fix or feature addition to keep diffs reviewable and reduce merge conflicts when upgrading package versions. (``@typespec__http-client-python.patch` only removes premature initialization of `fileContent` to silence a pylint false positive.`)
**Version-coupled patches** — Patches are tightly coupled to the specific package version in package.json. Any package version upgrade requires re-verifying whether the patch is still necessary or conflicts with the new dist/ layout. (``@typespec__compiler.patch` targets `dist/manifest.js` commit hash `478dfed5`; upgrading the compiler invalidates the patch.`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `@typespec__compiler.patch` | Fixes TypeSpec compiler binder: adds `$functions` export support, propagates `ModifierFlags.Internal` to all symbol declaration sites, replaces globby with Node.js built-in `glob` in formatter-fs.js, and renames `_` to `_decs` in ts-test files. | If @typespec/compiler is upgraded, re-verify that binder.js still lacks $functions handling and internal modifier propagation; if fixed upstream, remove this patch. The formatter-fs.js change removing `globby` dependency is critical for Node.js compatibility. |
| `@typespec__http.patch` | Adds `style` field to `QueryOptions` interface enabling `form|spaceDelimited|pipeDelimited|deepObject` serialization styles on `@query` parameters. | The `style` option may be merged upstream in a future @typespec/http release; check release notes before upgrading and remove the patch if fixed upstream. |
| `@typespec__openapi.patch` | Adds `parent` field to `TagMetadata` interface for hierarchical tag grouping and exports `getOperationId`/`setOperationId` via `useStateMap`. | The `parent` field on TagMetadata is an OpenMeter-specific extension. Do not remove it when upgrading @typespec/openapi — API spec grouping depends on it. |
| `@typespec__openapi3.patch` | Skips the `x-inline` extension key in `attachExtensions` so it is not emitted into generated OpenAPI YAML, and adds directive support in `generateOperationParameter`. | Removing this patch will cause `x-inline` to appear in generated YAML, breaking SDK generators that don't understand it. Verify any schema inlining suppression before removing. |
| `@typespec__http-client-python.patch` | Removes premature initialization of `fileContent` variable in Python emitter to avoid pylint `possibly-used-before-assignment` false positive. Also adds `constants.js` exporting blob storage URLs, flavor detection for `unbranded` SDK generation, and Pyodide generation logic. | This patch also includes substantial Pyodide-based generation code. Review carefully when upgrading — the emitter.js changes go beyond the trivial style fix described in the original CLAUDE.md. |
| `CLAUDE.md` | Manually maintained documentation describing patch purposes, the patch-package mechanism, and anti-patterns. Contains `archie:ai-start`/`archie:ai-end` markers for Archie's intent layer. | Update this file whenever a patch is added, modified, or removed, especially noting which upstream fixes should allow patch removal on next package upgrade. |

## Anti-Patterns

- Patching TypeScript source files (src/**/*.ts) — node_modules only ships dist/; source patches are silently ignored
- Adding multiple unrelated bug fixes in a single patch file — one concern per patch keeps diffs reviewable and upgrades manageable
- Using patches to change API semantics or add new TypeSpec language features — API changes belong in api/spec/packages/, not in toolchain patches
- Forgetting to update package.json version constraints or re-test the full `make gen-api` pipeline after bumping a patched package version
- Leaving patches in place after the upstream package ships the fix — stale patches cause confusing conflicts on version upgrades

## Decisions

- **Patch compiled dist/ files rather than forking the packages** — Forking TypeSpec packages requires maintaining a separate registry and updating import paths throughout api/spec/. patch-package applies diffs non-invasively and is cleanly reverted on version upgrade without changing any import statements in the TypeSpec source.
- **Co-locate patches in api/spec/patches/ next to the TypeSpec source packages** — Patches are exclusively relevant to the TypeSpec compilation pipeline; placing them adjacent to api/spec/packages/ makes it clear they are part of the API generation toolchain and not part of the Go backend or any other subsystem.

<!-- archie:ai-end -->
