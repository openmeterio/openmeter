# scripts

<!-- archie:ai-start -->

> Post-processing Node.js ESM scripts that fix TypeSpec-generated OpenAPI YAML before SDK generators and runtime validators consume it. Each script is a standalone CLI tool invoked by `make gen-api` after `tsp compile`.

## Patterns

**YAML round-trip with fixed formatting** — Every script reads the input file with `YAML.parse`, transforms the parsed object in-place, then writes back with `YAML.stringify(doc, { indent: 2, lineWidth: 0 })`. The `lineWidth: 0` is mandatory — any non-zero value produces spurious line-wrap diffs in the committed openapi.yaml. (`const parsed = await readYamlFile(filePath); const changed = flattenAllOf(parsed); if (changed) { await writeYamlFile(filePath, parsed); }`)
**Idempotency marker (x-flatten-allOf)** — flatten-allof.mjs stamps `x-flatten-allOf: true` on every processed node to prevent double-processing on re-runs. The marker uses the `x-` prefix so OpenAPI tooling ignores it. Check `node[FLATTEN_MARKER] !== true` before mutating. (`if (node[FLATTEN_MARKER] !== true) { node[FLATTEN_MARKER] = true; return true; } return false;`)
**Recursive tree walk returning changed boolean** — All transform functions (flattenAllOf, rewriteAdditionalProperties) recurse the YAML tree and return `boolean` indicating whether any mutation occurred. The top-level `main()` skips the write step when `changed === false`, avoiding no-op file touches. (`function flattenAllOf(node) { let changed = false; for (const value of Object.values(node)) { changed = flattenAllOf(value) || changed; } return changed; }`)
**additionalProperties: {not:{}} → false rewrite** — seal-object-schemas.mjs rewrites `additionalProperties: { not: {} }` (emitted by TypeSpec with seal-object-schemas:true) to `additionalProperties: false`. This is required because kin-openapi's deepObject decoder cannot handle the `not:{}` form for query params. (`if (isNotEmptyObject(node.additionalProperties)) { node.additionalProperties = false; changed = true; }`)
**apply-doc-fixes uses TypeSpec compiler API directly** — apply-doc-fixes.mjs calls `compile()` and `applyCodeFixes()` from `@typespec/compiler` — not the CLI. It reads the entry point from `package.json#exports['.'].typespec` and the ruleset from `package.json#name`, so renaming the package keeps it correct. (`const program = await compile(NodeHost, entry, { noEmit: true, linterRuleSet: { extends: [RULESET] } }); await applyCodeFixes(NodeHost, fixes);`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `flatten-allof.mjs` | Moves sibling properties on allOf-with-$ref schemas into a new allOf branch so SDK generators receive well-formed output. Invoked as `node flatten-allof.mjs api/v3/openapi.yaml`. | Only processes nodes where allOf contains at least one $ref — pure allOf without $ref is untouched. Extension keys (x-*) are never moved. Removing the marker check causes infinite loops. |
| `seal-object-schemas.mjs` | Rewrites `additionalProperties: { not: {} }` to `additionalProperties: false` across one or more YAML files (supports glob patterns). Fixes kin-openapi deepObject decoder incompatibility. | Accepts multiple file arguments and globs; deduplicates paths via a Set. The isNotEmptyObject check is precise: exactly one key `not` whose value is an empty plain object. |
| `apply-doc-fixes.mjs` | Applies TypeSpec linter `format-doc-comment` codefixes in-place using the compiler API. Entry point and ruleset are derived from package.json to survive package renames. | Exits 0 (not 1) when there are no fixes to apply — callers must not treat exit 0 as failure. Only applies fixes whose `id === 'format-doc-comment'`; the compiler also attaches a `suppress` codefix which is intentionally skipped. |

## Anti-Patterns

- Do not add business logic or spec-authoring code here — this folder is strictly a post-processing utility layer called after `tsp compile`.
- Do not change YAML_OPTIONS.lineWidth from 0 — non-zero values produce spurious line-wrap diffs in the committed openapi.yaml.
- Do not remove the x-flatten-allOf idempotency marker check in flatten-allof.mjs — doing so causes infinite loops when the script is run twice.
- Do not introduce a build/transpilation step — scripts run directly via Node.js as standalone ESM (.mjs) files with no bundler.
- Do not write scripts that mutate the TypeSpec source files in api/spec/packages/ — these scripts only post-process generated YAML output files.

## Decisions

- **Scripts are standalone ESM .mjs files with no build step, run directly by Node.js after `tsp compile`.** — Keeps the toolchain simple — no transpilation dependency, no separate build target. `make gen-api` can invoke them as plain `node script.mjs` commands.
- **The x-flatten-allOf idempotency marker uses the `x-` vendor extension prefix.** — OpenAPI tooling (Spectral, oapi-codegen, SDK generators) ignores x- keys by convention, so the marker survives all downstream validation passes without raising unknown-field errors.
- **additionalProperties: {not:{}} is rewritten to false rather than removed, to preserve the closed-schema semantics TypeSpec intended.** — kin-openapi's deepObject query param decoder cannot branch on `{not:{}}` form; `false` is semantically equivalent and universally accepted by validators and SDK generators.

## Example: Adding a new post-processing script that transforms the generated OpenAPI YAML

```
#!/usr/bin/env node
import fs from 'node:fs/promises'
import YAML from 'yaml'

const YAML_OPTIONS = { indent: 2, lineWidth: 0 }

function transform(node) {
  if (Array.isArray(node)) {
    let changed = false
    for (const item of node) changed = transform(item) || changed
    return changed
  }
  if (node == null || typeof node !== 'object') return false
  let changed = false
  // ... your mutation logic here ...
// ...
```

<!-- archie:ai-end -->
