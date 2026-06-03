# scripts

<!-- archie:ai-start -->

> Two post-generation Node.js scripts (generate.ts, add-as-const.ts) that clean up or transform generated JS SDK output after orval and openapi-typescript run; invoked as part of the 'make gen-api' pipeline, never imported by application code.

## Patterns

**Scripts read and write back to generated src/ files** — Each script opens an already-generated file under src/, transforms it in memory, and writes it back. They must be idempotent — running twice produces the same output. (`add-as-const.ts: readFileSync('../src/zod/index.ts') -> regex replace -> writeFileSync back to same path`)
**Type overrides applied via openapiTS transform hook, not post-hoc regex** — Custom type mappings (date-time -> Date, Event string fields -> optional string) are applied via the transform(schemaObject, metadata) callback passed to openapiTS, enabling context-aware overrides based on parameter location. (`if (schemaObject.format === 'date-time') { return allowString ? tsUnion([DATE, STRING]) : DATE }`)
**Schema source resolved relative to script location** — generate.ts resolves the schema as new URL('../../../openapi.cloud.yaml', import.meta.url); scripts must run from their scripts/ directory. Changing the output path requires updating the hardcoded writeFileSync target. (`const schema = new URL('../../../openapi.cloud.yaml', import.meta.url); fs.writeFileSync('./src/client/schemas.ts', contents)`)
**add-as-const.ts is a temporary orval workaround** — Appends 'as const' to exported object-literal defaults in zod/index.ts to fix orval issue #3244. Remove it and its invocation once the upstream fix lands. (`src.replace(/(^export const \w+Default =\s*\{[^{}]*\})/gm, '$1 as const')`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `generate.ts` | Generates src/client/schemas.ts from api/openapi.cloud.yaml using openapi-typescript with custom transform hooks for date-time and Event schema overrides. | The Event schema transform checks schemaObject.example against a hardcoded exclusion list ['customer-id', 'com.example.someevent'] — new Event string fields with different example values are silently made optional; update the list when the Event schema changes. |
| `add-as-const.ts` | Post-processes src/zod/index.ts to add 'as const' to single-level object-literal defaults. Temporary fix for orval#3244. | The regex only matches single-level object literals (no nested braces); complex nested defaults are not fixed. Remove once orval#3244 is resolved upstream. |

## Anti-Patterns

- Importing these scripts from application SDK code — they are build-time tools only
- Using regex transforms in generate.ts instead of the openapiTS transform hook for type overrides
- Hardcoding new type overrides in add-as-const.ts instead of the transform hook in generate.ts
- Running generate.ts from a directory other than scripts/ — the relative URL resolution for openapi.cloud.yaml breaks

## Decisions

- **Type overrides (date-time -> Date) applied via openapiTS transform hook rather than post-hoc regex** — The transform hook has typed access to schemaObject and metadata (including parameter location), enabling context-aware overrides like allowing string|Date in query parameters while enforcing Date only elsewhere.
- **add-as-const.ts exists as a clearly-labelled throwaway script** — The orval bug (missing 'as const' on object-literal defaults) is tracked upstream as issue #3244; a named script with an issue reference signals exactly when it can be deleted and minimises tech debt.

## Example: Adding a new date-time field that must also accept string in query params

```
// In generate.ts transform callback — already handled generically:
if (schemaObject.format === 'date-time') {
  const allowString =
    (metadata.schema && 'in' in metadata.schema && metadata.schema.in === 'query') ||
    metadata.path?.includes('/parameters/query')
  return allowString
    ? factory.createUnionTypeNode([DATE, NULL, STRING])
    : DATE
}
```

<!-- archie:ai-end -->
