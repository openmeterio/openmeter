# scripts

<!-- archie:ai-start -->

> Two post-generation Node.js scripts that clean up or transform the generated JS SDK output after orval and openapi-typescript run. They are invoked as part of the 'make gen-api' pipeline, not imported by application code.

## Patterns

**Scripts read from and write back to generated src/ files** — Each script opens an already-generated file under src/, transforms it in memory, and writes it back. They must be idempotent — running them twice produces the same output. (`add-as-const.ts: readFileSync('../src/zod/index.ts') → regex replace → writeFileSync back to same path`)
**generate.ts uses openapiTS transform hook for type overrides** — Custom type mappings (date-time → Date, Event string fields → optional string) are applied via the transform(schemaObject, metadata) callback passed to openapiTS, not as post-hoc regex replacements. (`if (schemaObject.format === 'date-time') { return allowString ? tsUnion([DATE, STRING]) : DATE }`)
**Schema source is api/openapi.cloud.yaml (relative URL)** — generate.ts resolves the schema path as new URL('../../../openapi.cloud.yaml', import.meta.url), so the scripts must be run from their location inside scripts/. Changing the output path (./src/client/schemas.ts) requires updating the hardcoded writeFileSync target. (`const schema = new URL('../../../openapi.cloud.yaml', import.meta.url)`)
**add-as-const.ts is a temporary workaround for an orval upstream bug** — The script exists solely to append 'as const' to exported object-literal defaults in zod/index.ts until orval#3244 is fixed. Remove the script and its invocation once the upstream fix lands. (`src.replace(/(^export const \w+Default =\s*\{[^{}]*\})/gm, '$1 as const')`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `generate.ts` | Generates src/client/schemas.ts from api/openapi.cloud.yaml using openapi-typescript with custom transform hooks. Primary entry point for TypeScript type generation. | The Event schema transform (schemaObject.example check against a hardcoded list) will silently miss new Event fields — update the example exclusion list when the Event schema changes. |
| `add-as-const.ts` | Post-processes src/zod/index.ts to add 'as const' to object-literal defaults. Temporary fix for orval issue #3244. | The regex only matches single-level object literals (no nested braces). Complex defaults will not be fixed. Remove once orval releases the upstream fix. |

## Anti-Patterns

- Importing these scripts from application SDK code — they are build-time tools only
- Using regex transforms in generate.ts instead of the openapiTS transform hook for type changes
- Hardcoding new type overrides in add-as-const.ts instead of the transform hook in generate.ts
- Running generate.ts from a directory other than scripts/ — the relative URL resolution for openapi.cloud.yaml will break
- Removing the 'as const' fix without first verifying orval upstream issue #3244 is resolved

## Decisions

- **Type overrides (date-time → Date) are applied via openapiTS transform hook, not post-hoc regex** — The transform hook has typed access to schemaObject and metadata (including parameter location), enabling context-aware overrides like allowing string|Date in query parameters while enforcing Date elsewhere.
- **add-as-const.ts exists as a temporary script rather than a permanent transformation** — The orval bug (missing 'as const' on object-literal defaults) is tracked upstream; keeping the fix as a clearly-labelled throwaway script with an issue reference minimises tech debt and signals when it can be deleted.

## Example: Adding a new date-time field that must also accept string in query params

```
// In generate.ts transform callback — already handled generically:
if (schemaObject.format === 'date-time') {
  const allowString =
    (metadata.schema && 'in' in metadata.schema && metadata.schema.in === 'query') ||
    metadata.path?.includes('/parameters/query')
  return allowString
    ? factory.createUnionTypeNode([DATE, NULL, STRING]) // or [DATE, STRING] if non-nullable
    : DATE
}
```

<!-- archie:ai-end -->
