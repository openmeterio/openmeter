# patches

<!-- archie:ai-start -->

> Holds a single unified patch file applied to the installed openapi-typescript npm package before JS SDK code generation runs. Its sole job is carrying local bug fixes and feature backports not yet in the upstream release so the SDK generates correct TypeScript output.

## Patterns

**Single consolidated diff file** — All upstream fixes live in exactly one file (openapi-typescript.patch) applied as a whole. Never split into multiple patch files per fix. (`openapi-typescript.patch — one unified diff touching bin/cli.js, dist/index.d.{c,m,}ts, dist/transform/schema-object.{c,}js`)
**Patch targets compiled dist/ artifacts** — The patch modifies the pre-built dist/ files of openapi-typescript (both .cjs and .mjs variants) and .d.ts type declarations, not TypeSpec or TypeScript source files. (`dist/index.d.cts, dist/index.d.mts, dist/index.d.ts all receive the same additionalProperties type addition`)
**BOOLEAN_FLAGS list kept in sync across patch** — The extracted BOOLEAN_FLAGS array in bin/cli.js must contain every flag from the original boolean: [] list. Adding a new CLI flag means updating BOOLEAN_FLAGS in the patch. (`BOOLEAN_FLAGS = ['additionalProperties', 'alphabetize', ... 'rootTypesNoSchemaPrefix']`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `openapi-typescript.patch` | Unified diff applied to the installed openapi-typescript package. Fixes: BOOLEAN_FLAGS extraction into a named const, additionalProperties schema type addition to SchemaObject, enum + additionalProperties interaction, generateSchema config spreading. | Regenerate and re-validate the patch whenever openapi-typescript is upgraded — the base commit hash in the diff header will no longer match and the patch will fail to apply silently. |

## Anti-Patterns

- Creating separate patch files for each fix — the tooling expects a single patch file
- Patching TypeSpec source files or the generated openapi.yaml instead of the openapi-typescript dist artifacts
- Editing dist/ files directly in the repo without encoding the change as a patch
- Forgetting to update the patch after upgrading openapi-typescript (stale base hash causes apply failure)

## Decisions

- **Patch dist/ artifacts rather than waiting for upstream releases** — The JS SDK generation pipeline requires fixes (additionalProperties schema type, BOOLEAN_FLAGS refactor, enum+additionalProperties interaction) not yet in the published npm package; patching the installed artifact unblocks generation without forking the whole package.

<!-- archie:ai-end -->
