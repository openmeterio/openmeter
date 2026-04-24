# lib

<!-- archie:ai-start -->

> TypeSpec linter entry point for the v1 (legacy) API spec. Assembles all lint rules into a single `$linter` export via `defineLinter`, which the TypeSpec compiler picks up during `make gen-api` to enforce naming, documentation, and structural constraints before OpenAPI is generated.

## Patterns

**Rule registration via defineLinter** — All rules are collected in the `rules` array inside `index.js` and passed to `defineLinter({ rules })`. A new rule file must export a named const and be imported and added to this array — otherwise it never runs. (`import { casingRule } from './rules/casing.js'; const rules = [casingRule, ...]; export const $linter = defineLinter({ rules });`)
**createRule factory with paramMessage** — Every rule uses `createRule` from `@typespec/compiler` with a `messages` map of `paramMessage` templates. Plain string messages are not used — `paramMessage` provides type-safe interpolation and correct editor diagnostics. (`export const casingRule = createRule({ name: 'casing', severity: 'warning', messages: { name: paramMessage`The names of ${'type'} types must use ${'casing'}` }, create: (context) => ({ model: (m) => { ... } }) })`)
**context.reportDiagnostic with AST node as target** — Diagnostics are always reported with `target` set to the offending AST node (model, property, decorator node, etc.) so editor tooling highlights the correct location. (`context.reportDiagnostic({ format: { type: 'Model', casing: 'PascalCase' }, target: model, messageId: 'name' })`)
**Casing helpers centralised in utils.js** — `isPascalCaseNoAcronyms` and `isCamelCaseNoAcronyms` live exclusively in `utils.js` with a hardcoded `pascalCaseExceptions` list (`['OAuth2', 'URL', 'API', 'UI', 'ID']`). Rule files import from utils.js — never inline their own regex. (`import { isCamelCaseNoAcronyms, isPascalCaseNoAcronyms } from './utils.js'`)
**friendlyName rule is error-severity (blocking)** — `friendlyNameRule` uses `severity: 'error'` which blocks compilation. All other rules use `severity: 'warning'`. Do not change friendlyName to warning — it is intentionally blocking to enforce SDK-visible names. (`export const friendlyNameRule = createRule({ name: 'friendlyName', severity: 'error', ... })`)
**Model property doc exemptions** — `docDecoratorRule` skips doc checks for `_` and `contentType` properties, and skips all property checks for models whose names end with `Response`. Matching these exemptions prevents false positives when adding new response models. (`if (!['_', 'contentType'].includes(name) && !getDoc(context.program, property)) { ... }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `index.js` | Linter entry point; exports `$linter` recognized by `@typespec/compiler`. Must include every rule in the `rules` array. | Forgetting to add a new rule import and array entry here means the rule is silently never applied. |
| `rules/utils.js` | Shared casing helpers (`isPascalCaseNoAcronyms`, `isCamelCaseNoAcronyms`, `isSnakeCase`, `isKebabCase`) and the canonical `pascalCaseExceptions` list. | Adding acronym exceptions anywhere other than the `pascalCaseExceptions` array in this file will cause inconsistent behaviour across rules. |
| `rules/friendly-name.js` | Error-severity rule requiring `@friendlyName` on all interfaces, models, enums, and unions. Blocks `make gen-api` on violation. | Changing severity to `warning` silently removes the block; this is the only error-level rule. |
| `rules/casing.js` | Warns when model names are not PascalCase or model properties are not camelCase. Imports casing helpers from utils.js. | The `_` property name is exempt from camelCase check (see `property.name !== '_'` guard); do not remove this guard. |
| `rules/docs.js` | Warns on missing `@doc` decorator for models, enums, unions, and their non-exempt properties. Uses `getDoc` from `@typespec/compiler`. | Models ending with `Response` skip property-level doc checks — adding a non-Response suffix to a response model will start triggering warnings on all its properties. |
| `rules/composition-over-inheritance.js` | Warns when a model extends a base model without a `@discriminator` decorator, encouraging spread or `model is` composition instead. | Uses `SyntaxKind` from `@typespec/compiler/ast` (not the main package) — ensure this sub-path import stays correct after compiler upgrades. |
| `rules/operation-summary.js` | Warns when an operation lacks a `@summary` decorator. Checks `node.decorators` for `$summary` (compiler internal name). | The decorator internal name is `$summary`, not `summary`. Searching for `summary` without the `$` prefix will never match. |

## Anti-Patterns

- Adding a rule file without exporting a named const and adding it to the `rules` array in `index.js` — the rule will never execute.
- Duplicating casing regex or acronym exceptions inline in a rule file instead of importing from `utils.js` — the exception list diverges.
- Using a plain string in `messages` instead of `paramMessage` — loses type-safe interpolation and breaks diagnostic formatting.
- Reporting a diagnostic with `target` set to something other than the offending AST node — editor highlights the wrong location.
- Changing `friendlyNameRule` severity from `error` to `warning` — removes the compilation block that enforces SDK-visible names.

## Decisions

- **`friendlyNameRule` is the only error-severity rule; all others are warnings.** — `@friendlyName` directly affects generated SDK type names across Go, JS, and Python — a missing decorator causes silent SDK naming drift, which is a hard contract break. Casing and doc issues are fixable post-hoc without breaking SDK consumers.
- **Casing helpers and the acronym exception list are centralised in `utils.js`.** — Multiple rules (casing, future rules) check the same naming conventions. A single source of truth for `pascalCaseExceptions` prevents divergence when a new acronym (e.g. `SSO`) is added.
- **Rules use the `@typespec/compiler` `createRule` API with `paramMessage`, not raw AST visitors.** — The `createRule` factory integrates with the TypeSpec diagnostic system for editor highlighting and CLI output. Raw visitors would bypass formatter and message-ID dispatch.

## Example: Adding a new warning rule that checks enum member names are UPPER_SNAKE_CASE

```
// rules/enum-casing.js
import { createRule, paramMessage } from '@typespec/compiler'
import { isSnakeCase } from './utils.js'

export const enumCasingRule = createRule({
  name: 'enum-casing',
  severity: 'warning',
  description: 'Enum members must use UPPER_SNAKE_CASE.',
  messages: {
    default: paramMessage`Enum member '${'name'}' must use UPPER_SNAKE_CASE`,
  },
  create: (context) => ({
    enumMember: (member) => {
      if (!isSnakeCase(member.name.toLowerCase())) {
        context.reportDiagnostic({
// ...
```

<!-- archie:ai-end -->
