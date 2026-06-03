# lib

<!-- archie:ai-start -->

> TypeSpec linter plugin for the v3 AIP spec — index.js exports a $linter (defineLinter) registering all custom rules (casing, docs, friendlyName, no-nullable, composition, field-prefix, operation-summary, operation-id) enforcing AIP naming conventions and structural constraints. Rules run during `make gen-api` and block or warn before any OpenAPI or SDK artifact is generated. The rules/ child holds the rule implementations.

## Patterns

**createRule factory for every rule** — Every lint rule is defined via createRule from '@typespec/compiler'. Raw functions calling context.report directly are not registered by the rule engine. (`export const myRule = createRule({ name: 'my-rule', severity: 'error', messages: { default: 'msg' }, create: (context) => ({ model: (m) => { /* ... */ } }) })`)
**Two-severity model: error vs warning** — Rules blocking compilation (missing @friendlyName, enum value casing) use severity 'error'; stylistic preferences (model name casing, prefix grouping, composition) use 'warning'. Never use error for stylistic-only rules. (`export const friendlyNameRule = createRule({ severity: 'error', ... }); export const casingRule = createRule({ severity: 'warning', ... })`)
**Register every rule in index.js** — A new rule file has zero effect until imported and added to the rules array in index.js — the sole entry point consumed by defineLinter. (`import { myRule } from './rules/my-rule.js'; const rules = [ ..., myRule ]; export const $linter = defineLinter({ rules })`)
**PascalCase exceptions only in utils.js** — Accepted acronym exceptions for isPascalCaseNoAcronyms (OAuth2, URL, API, UI, ID) live in utils.js — add new acronyms there, never inline in a rule. (`const pascalCaseExceptions = ['OAuth2', 'URL', 'API', 'UI', 'ID']`)
**EXCLUDED_PREFIXES for boolean-flag field groups** — field-prefix.js exempts AIP-standard boolean-flag prefixes (is_, enable_, disable_, allow_) via EXCLUDED_PREFIXES; new boolean-flag prefixes must be added there to avoid spurious warnings. (`const EXCLUDED_PREFIXES = ['is_', 'enable_', 'disable_', 'allow_', ...]`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `index.js` | Sole entry point — exports $linter via defineLinter, imports and registers every rule (casing, casingErrors, doc, docFormat, friendlyName, noNullable, operationSummary, operationIdKebabCase, compositionOverInheritance, repeatedPrefixGrouping). | Adding a new rule file without importing it here — the rule is silently ignored. |
| `rules/utils.js` | Shared casing helpers (isPascalCaseNoAcronyms, isCamelCaseNoAcronyms, isSnakeCase, isKebabCase) and the pascalCaseExceptions allowlist. | Acronyms missing from pascalCaseExceptions are flagged as casing errors — add them here, not per-rule. |
| `rules/casing.js` | casingRule (warning) for model/enum/property casing; casingErrorsRule (error) requiring enum member string values to be snake_case. | @path-decorated params are checked for camelCase not snake_case — renaming them to silence a warning produces a new error. |
| `rules/friendly-name.js` | Enforces @friendlyName on all models/enums/unions/non-Endpoints-Operations interfaces (error); *Endpoints/*Operations interfaces must NOT have @friendlyName. | Every new model, enum, union, and non-route interface needs @friendlyName — missing it blocks gen-api. |
| `rules/field-prefix.js` | Warns when 2+ properties share a common prefix_ and should be grouped; EXCLUDED_PREFIXES exempts AIP boolean-flag prefixes. | New boolean-flag property groups without an EXCLUDED_PREFIXES entry produce spurious warnings. |
| `rules/composition-over-inheritance.js` | Warns when a model extends a base without @discriminator; prefers spread (...Base) or 'model is Base'. | Using 'extends' for mixins without @discriminator produces warnings — prefer spread syntax. |
| `rules/docs.js` | Enforces @doc on models/enums/properties; docFormatRule checks format conventions. | Missing @doc on new types blocks gen-api if configured as error severity. |

## Anti-Patterns

- Defining a rule as a raw function calling context.report directly instead of using createRule — TypeSpec rule registration requires createRule
- Using severity 'error' for stylistic preferences (casing, prefix grouping) — errors block gen-api; use 'warning'
- Adding new accepted PascalCase acronyms anywhere other than pascalCaseExceptions in utils.js
- Forgetting to import a new rule in index.js — the rule file is valid JS but has zero effect
- Adding underscore-prefixed boolean-flag field groups without adding the prefix to EXCLUDED_PREFIXES in field-prefix.js

## Decisions

- **Two-severity model: error blocks compilation, warning surfaces in lint output but does not fail gen-api** — Hard naming contracts (friendlyName, enum value casing) must be enforced before code generation; stylistic preferences should guide authors without halting every iteration.
- **PascalCase exceptions codified in utils.js rather than inline per rule** — Centralising exceptions makes them reviewable and extensible without touching each rule using isPascalCaseNoAcronyms.
- **EXCLUDED_PREFIXES in field-prefix.js exempts AIP-standard boolean-flag prefixes** — AIP guidelines explicitly allow these prefix patterns for boolean fields; grouping them into sub-objects would violate AIP conventions.

<!-- archie:ai-end -->
