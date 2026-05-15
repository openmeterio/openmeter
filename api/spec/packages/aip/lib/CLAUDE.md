# lib

<!-- archie:ai-start -->

> TypeSpec linter plugin for the v3 AIP spec — exports a $linter via index.js that registers all custom rules (casing, docs, friendlyName, no-nullable, composition, field-prefix) enforcing AIP naming conventions and structural constraints. Rules run during `make gen-api` and block or warn before any OpenAPI or SDK artifact is generated.

## Patterns

**createRule factory for every rule** — Every lint rule must be defined via createRule from '@typespec/compiler'. Raw functions calling context.report directly are not registered by the TypeSpec rule engine and have no effect. (`import { createRule } from '@typespec/compiler'
export const myRule = createRule({ name: 'my-rule', severity: 'error', messages: { default: 'msg' }, create: (context) => ({ model: (m) => { /* ... */ } }) })`)
**Two-severity model: error vs warning** — Rules blocking compilation (missing @friendlyName, enum value casing) use severity: 'error'. Stylistic preferences (model name casing, prefix grouping, composition) use severity: 'warning'. Never use error for stylistic-only rules. (`// error: blocks gen-api
export const friendlyNameRule = createRule({ severity: 'error', ... })
// warning: informs but does not block
export const casingRule = createRule({ severity: 'warning', ... })`)
**Register every rule in index.js** — A new rule file has zero effect until imported and added to the rules array in index.js, which is the sole entry point consumed by defineLinter. (`// index.js
import { myRule } from './rules/my-rule.js'
const rules = [ ..., myRule ]
export const $linter = defineLinter({ rules })`)
**PascalCase exceptions live exclusively in utils.js** — Accepted acronym exceptions for isPascalCaseNoAcronyms (OAuth2, URL, API, UI, ID) are defined in utils.js. Add new acronyms there only — not inline in individual rule files. (`// utils.js
const pascalCaseExceptions = ['OAuth2', 'URL', 'API', 'UI', 'ID']`)
**EXCLUDED_PREFIXES for boolean-flag field groups** — field-prefix.js exempts AIP-standard boolean-flag prefixes (is*, enable*, disable*, allow*, etc.) via EXCLUDED_PREFIXES. New boolean-flag prefixes must be added there to avoid spurious warnings. (`// field-prefix.js
const EXCLUDED_PREFIXES = ['is_', 'enable_', 'disable_', 'allow_', ...]`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `index.js` | Sole entry point — exports $linter via defineLinter, imports and registers every rule. The TypeSpec compiler loads only this file. | Adding a new rule file without importing it here — the rule is silently ignored. |
| `rules/utils.js` | Shared casing helpers: isPascalCaseNoAcronyms, isCamelCaseNoAcronyms, isSnakeCase, isKebabCase. Holds the pascalCaseExceptions allowlist. | Acronyms missing from pascalCaseExceptions will be flagged as casing errors — add them here, not per-rule. |
| `rules/casing.js` | casingRule (warning): models/enums PascalCase, non-path properties snake_case, path properties camelCase, enum member names PascalCase. casingErrorsRule (error): enum member string values must be snake_case. | @path-decorated params are checked for camelCase not snake_case — renaming them to snake_case to silence a warning produces a new error. |
| `rules/friendly-name.js` | Enforces @friendlyName on all models, enums, unions, and non-Endpoints/Operations interfaces (severity: error). Interfaces ending in 'Endpoints'/'Operations' must NOT have @friendlyName. | Every new TypeSpec model, enum, union, and non-route interface needs @friendlyName — missing it blocks gen-api. |
| `rules/field-prefix.js` | Warns when 2+ model properties share a common prefix_ and should be grouped into a sub-object. EXCLUDED_PREFIXES exempts AIP-standard boolean-flag prefixes. | New boolean-flag property groups without an entry in EXCLUDED_PREFIXES produce spurious warnings. |
| `rules/composition-over-inheritance.js` | Warns when a model extends a base without @discriminator. Prefers spread (...Base) or 'model is Base' for composition. | Using 'extends' for mixins without @discriminator produces warnings — prefer spread syntax. |
| `rules/docs.js` | Enforces @doc on models, enums, and properties; docFormatRule checks format conventions. | Missing @doc on new types blocks gen-api if configured as error severity. |

## Anti-Patterns

- Defining a rule as a raw function calling context.report directly instead of using createRule — TypeSpec rule registration requires createRule.
- Using severity: 'error' for stylistic preferences (casing, prefix grouping) — errors block gen-api; use 'warning'.
- Adding new accepted PascalCase acronyms anywhere other than pascalCaseExceptions in utils.js.
- Forgetting to import a new rule in index.js — the rule file is valid JS but has zero effect.
- Adding underscore-prefixed boolean-flag field groups without adding the prefix to EXCLUDED_PREFIXES in field-prefix.js.

## Decisions

- **Two-severity model: error blocks compilation, warning surfaces in lint output but does not fail gen-api.** — Hard naming contracts (friendlyName, enum value casing) must be enforced before code generation succeeds; stylistic preferences should guide authors without halting the build on every iteration.
- **PascalCase exceptions codified in utils.js rather than inline per rule.** — Centralising exceptions makes them reviewable and extensible without touching each rule that uses isPascalCaseNoAcronyms.
- **EXCLUDED_PREFIXES in field-prefix.js exempts AIP-standard boolean-flag prefixes.** — AIP guidelines explicitly allow these prefix patterns for boolean fields; grouping them into sub-objects would violate AIP conventions.

## Example: Adding a new AIP linter rule that enforces a required decorator on all operations

```
// rules/require-my-decorator.js
import { createRule, paramMessage } from '@typespec/compiler'

export const requireMyDecoratorRule = createRule({
  name: 'require-my-decorator',
  severity: 'error',
  description: 'Ensure @myDecorator is present on all operations.',
  messages: {
    default: paramMessage`Operation ${'name'} must have @myDecorator.`,
  },
  create: (context) => ({
    operation: (op) => {
      if (!op.decorators.some((d) => d.decorator.name === '$myDecorator')) {
        context.reportDiagnostic({ format: { name: op.name }, target: op, messageId: 'default' })
      }
// ...
```

<!-- archie:ai-end -->
