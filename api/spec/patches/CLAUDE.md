# patches

<!-- archie:ai-start -->

> Holds vendored patch files applied to TypeSpec compiler and emitter npm packages to fix upstream bugs or add missing features needed by OpenMeter's API generation pipeline. These patches are applied via `patch-package` during `npm install` and must stay in sync with the pinned package versions in package.json.

## Patterns

**patch-package format** — Each file is a unified diff named `@scope__package.patch` matching the exact package name under node_modules. The diff is against the package's compiled dist/ files, not source TypeScript. (`@typespec__compiler.patch patches dist/src/core/binder.js and dist/manifest.js`)
**Compiled artifact targeting** — Patches modify compiled JS (dist/**/\*.js) and type declaration maps (dist/**/\*.d.ts.map), never raw TypeScript sources, because node_modules only contains the built distribution. (`@typespec__openapi3.patch modifies dist/src/attach-extensions.js to skip x-inline extension key`)
**Minimal scope** — Each patch targets only the smallest change required — a single bug fix or feature addition — keeping the diff reviewable and reducing merge conflicts on version upgrades. (`@typespec__http-client-python.patch changes `let fileContent = ''`to`let fileContent;` to avoid a pylint false positive`)

## Key Files

| File                                  | Role                                                                                                                                                                                                                    | Watch For                                                                                                                                                            |
| ------------------------------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ------------- | -------------------------------------------------------------------------------------------- | ----------------------------------------------------------------------------------------------- |
| `@typespec__compiler.patch`           | Fixes TypeSpec compiler binder: adds `$functions` export support, propagates `ModifierFlags.Internal` to all symbol declaration sites, and renames `_` to `_decs` in ts-test files to silence unused-variable warnings. | If @typespec/compiler is upgraded, re-verify that binder.js still lacks $functions handling and internal modifier propagation; if fixed upstream, remove this patch. |
| `@typespec__http.patch`               | Adds `style` field to `QueryOptions` interface in @typespec/http, enabling `form                                                                                                                                        | spaceDelimited                                                                                                                                                       | pipeDelimited | deepObject`serialization styles on @query parameters. Also renames`\_`to`\_decs` in ts-test. | style option may be merged upstream in a future @typespec/http release; check before upgrading. |
| `@typespec__openapi.patch`            | Adds `parent` field to `TagMetadata` interface for hierarchical tag grouping, exports `getOperationId`/`setOperationId` via `useStateMap` instead of a raw state map, and renames `_` to `_decs` in ts-test.            | The `parent` field on TagMetadata is an OpenMeter-specific extension; do not remove it when upgrading @typespec/openapi.                                             |
| `@typespec__openapi3.patch`           | Skips the `x-inline` extension key in `attachExtensions` so it is not emitted into the generated OpenAPI YAML, and adds directive support in `generateOperationParameter` for OpenAPI 3 conversion.                     | If x-inline is used in TypeSpec files to suppress schema inlining, removing this patch will cause x-inline to appear in generated YAML, breaking SDK generators.     |
| `@typespec__http-client-python.patch` | Removes premature initialization of `fileContent` variable in Python emitter to avoid pylint `possibly-used-before-assignment` false positive.                                                                          | Trivial style patch; safe to drop if upstream fixes the initialization.                                                                                              |

## Anti-Patterns

- Patching TypeScript source files (src/\*_/_.ts) — node_modules only ships dist/; source patches are ignored
- Adding patches that exceed a single concern per file — one bug fix per patch keeps diffs reviewable
- Forgetting to update atlas.sum or package-lock.json equivalent when bumping patched package versions
- Adding new TypeSpec language features via patches instead of upstream contributions or TypeSpec library extensions in api/spec/packages/
- Using these patches to change API semantics rather than fixing toolchain bugs — API changes belong in api/spec/packages/

## Decisions

- **Patch compiled dist/ files rather than forking the packages** — Forking TypeSpec packages would require maintaining a separate registry and updating import paths throughout api/spec/. patch-package applies diffs non-invasively and is reverted cleanly on version upgrade.
- **Keep patches in api/spec/patches/ co-located with the TypeSpec spec** — Patches are only relevant to the TypeSpec compilation pipeline; placing them next to api/spec/packages/ makes it clear they are part of the API generation toolchain, not the Go backend.

<!-- archie:ai-end -->
