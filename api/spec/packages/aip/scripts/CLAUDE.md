# scripts

<!-- archie:ai-start -->

> Post-processing scripts for TypeSpec-generated OpenAPI YAML. Currently contains a single Node.js script that normalises allOf schemas by moving sibling properties into allOf branches, ensuring SDK generators receive well-formed output.

## Patterns

**allOf normalisation via x-flatten-allOf marker** — flattenAllOf() recurses the parsed YAML tree; when a node has allOf with at least one $ref member, moveSiblingPropertiesIntoAllOf() migrates non-allOf, non-extension sibling keys into a new allOf entry and stamps x-flatten-allOf: true to prevent double-processing. (`node flatten-allof.mjs api/v3/openapi.yaml`)
**Idempotent transformation** — The x-flatten-allOf marker guards re-runs: a node already processed (no movable keys left) receives the marker without further mutation. Running the script twice is safe. (`if (node[FLATTEN_MARKER] !== true) { node[FLATTEN_MARKER] = true; return true; } return false;`)
**YAML round-trip with controlled formatting** — Input is parsed with the yaml package and written back with indent:2, lineWidth:0 options to avoid unwanted line-wrapping in the generated spec. (`const YAML_OPTIONS = { indent: 2, lineWidth: 0 }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `flatten-allof.mjs` | CLI post-processor invoked by make gen-api after TypeSpec compilation to fix allOf schema shape for downstream SDK generators. | Only processes nodes where allOf contains at least one $ref — pure allOf without a $ref is left untouched. Extension keys (x-*) are never moved, preventing accidental stripping of TypeSpec-emitted vendor extensions. |

## Anti-Patterns

- Do not add business-logic or spec-authoring code here — this folder is strictly a post-processing utility layer.
- Do not remove the x-flatten-allOf marker check; doing so causes infinite loops on re-runs.
- Do not change YAML_OPTIONS.lineWidth from 0 — non-zero values produce spurious diffs in the generated openapi.yaml.

## Decisions

- **Script is a standalone ESM .mjs file with no build step.** — It runs directly via Node.js after TypeSpec compilation in make gen-api; no transpilation dependency keeps the toolchain simple.
- **Marker key x-flatten-allOf is an OpenAPI extension (x- prefix) so it survives downstream validator passes without raising unknown-field errors.** — OpenAPI tooling ignores x- keys by convention, making the idempotency marker invisible to Spectral, oapi-codegen, and the JavaScript SDK generator.

## Example: Adding a new post-processing script alongside flatten-allof.mjs

```
#!/usr/bin/env node
import fs from 'node:fs/promises'
import YAML from 'yaml'

const YAML_OPTIONS = { indent: 2, lineWidth: 0 }

async function main() {
  const [filePath] = process.argv.slice(2)
  const raw = await fs.readFile(filePath, 'utf8')
  const doc = YAML.parse(raw)
  // ... transform doc ...
  await fs.writeFile(filePath, YAML.stringify(doc, YAML_OPTIONS), 'utf8')
}
await main()
```

<!-- archie:ai-end -->
