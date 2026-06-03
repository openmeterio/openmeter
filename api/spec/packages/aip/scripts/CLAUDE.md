# scripts

<!-- archie:ai-start -->

> Post-processing Node.js ESM scripts that fix TypeSpec-generated OpenAPI YAML before SDK generators and runtime validators consume it. Each is a standalone CLI tool invoked by make gen-api after tsp compile.

## Patterns

**YAML round-trip with fixed formatting** — Each script reads with YAML.parse, mutates in place, then writes with YAML.stringify(doc, { indent: 2, lineWidth: 0 }). lineWidth:0 is mandatory to avoid spurious line-wrap diffs. (`const parsed = await readYamlFile(filePath); if (flattenAllOf(parsed)) await writeYamlFile(filePath, parsed)`)
**Idempotency marker x-flatten-allOf** — flatten-allof.mjs stamps x-flatten-allOf:true on processed nodes to prevent double-processing on re-runs; the x- prefix is ignored by OpenAPI tooling. Check the marker before mutating. (`if (node[FLATTEN_MARKER] !== true) { node[FLATTEN_MARKER] = true; return true; } return false`)
**Recursive walk returning changed boolean** — Transform functions recurse the YAML tree and return a boolean; main() skips the write when changed === false to avoid no-op file touches. (`for (const value of Object.values(node)) changed = flattenAllOf(value) || changed`)
**additionalProperties:{not:{}} -> false rewrite** — seal-object-schemas.mjs rewrites the TypeSpec seal-object-schemas form `{ not: {} }` to `false` because kin-openapi's deepObject query decoder cannot handle the not:{} form. (`if (isNotEmptyObject(node.additionalProperties)) { node.additionalProperties = false; changed = true }`)
**apply-doc-fixes uses the TypeSpec compiler API** — apply-doc-fixes.mjs calls compile() and applyCodeFixes() directly (not the CLI), deriving the entry point from package.json#exports['.'].typespec and ruleset from package.json#name so renaming the package stays correct. (`const program = await compile(NodeHost, entry, { noEmit: true, linterRuleSet: { extends: [RULESET] } }); await applyCodeFixes(NodeHost, fixes)`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `flatten-allof.mjs` | Moves sibling properties on allOf-with-$ref schemas into a new allOf branch for well-formed SDK output. Invoked as `node flatten-allof.mjs api/v3/openapi.yaml`. | Only processes nodes whose allOf contains a $ref; x-* keys are never moved; removing the marker check causes infinite loops on re-run. |
| `seal-object-schemas.mjs` | Rewrites additionalProperties:{not:{}} to false across one or more YAML files (supports globs, dedups via a Set). | isNotEmptyObject is precise: exactly one key `not` whose value is an empty plain object. |
| `apply-doc-fixes.mjs` | Applies the TypeSpec format-doc-comment codefix in place via the compiler API. | Exits 0 (not 1) when there are no fixes — callers must not treat exit 0 as failure. Only applies fixes whose id === 'format-doc-comment', skipping the auto-attached suppress codefix. |

## Anti-Patterns

- Adding business logic or spec-authoring code here — this is strictly a post-processing layer
- Changing YAML_OPTIONS.lineWidth from 0 — produces spurious line-wrap diffs
- Removing the x-flatten-allOf idempotency marker check — causes infinite loops on re-run
- Introducing a build/transpilation step — scripts run directly as ESM .mjs
- Mutating TypeSpec source files in api/spec/packages/ — these scripts only post-process generated YAML

## Decisions

- **Scripts are standalone ESM .mjs with no build step** — Keeps the toolchain simple — make gen-api invokes them as plain `node script.mjs`, no bundler or transpilation.
- **The idempotency marker uses the x- vendor extension prefix** — OpenAPI tooling ignores x- keys, so the marker survives downstream validation without unknown-field errors.
- **additionalProperties:{not:{}} is rewritten to false, not removed** — Preserves the closed-schema semantics TypeSpec intended; `false` is semantically equivalent and accepted by kin-openapi's deepObject decoder.

<!-- archie:ai-end -->
