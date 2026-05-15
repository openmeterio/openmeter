# lib

<!-- archie:ai-start -->

> TypeSpec linter entry point for the v1 (legacy) API spec. Assembles all lint rules via `defineLinter` into the `$linter` export that the TypeSpec compiler picks up during `make gen-api` to enforce naming, documentation, and structural constraints before OpenAPI is generated.

## Patterns

**Rule registration via defineLinter** — All rules must be imported and added to the `rules` array in `index.js`, then passed to `defineLinter({ rules })`. A rule file that omits this step is silently never executed. (`import { casingRule } from './rules/casing.js'; const rules = [casingRule, ...]; export const $linter = defineLinter({ rules });`)
**createRule factory with paramMessage** — Every rule uses `createRule` from `@typespec/compiler` with a `messages` map of `paramMessage` templates. Plain string messages break type-safe interpolation and diagnostic formatting. (`export const casingRule = createRule({ name: 'casing', severity: 'warning', messages: { name: paramMessage`The names of ${'type'} types must use ${'casing'}` }, create: (context) => ({ model: (m) => { ... } }) })`)
**context.reportDiagnostic with AST node as target** — Diagnostics must set `target` to the offending AST node (model, property, decorator node) so editor tooling highlights the correct location. (`context.reportDiagnostic({ format: { type: 'Model', casing: 'PascalCase' }, target: model, messageId: 'name' })`)
**Casing helpers centralised in utils.js** — `isPascalCaseNoAcronyms`, `isCamelCaseNoAcronyms`, and the `pascalCaseExceptions` list live exclusively in `rules/utils.js`. Rule files import from there — never define casing logic inline. (`import { isCamelCaseNoAcronyms, isPascalCaseNoAcronyms } from './utils.js'`)
**friendlyName rule is error-severity (blocking)** — `friendlyNameRule` uses `severity: 'error'` which blocks `make gen-api`. All other rules use `severity: 'warning'`. Do not lower it to warning — missing `@friendlyName` breaks SDK code generation across Go, JS, and Python. (`export const friendlyNameRule = createRule({ name: 'friendlyName', severity: 'error', ... })`)
**Model property doc exemptions** — `docDecoratorRule` skips doc checks for properties named `_` or `contentType`, and skips all property checks for models whose names end with `Response`. New response models must follow this naming to avoid spurious warnings. (`if (!['_', 'contentType'].includes(name) && !getDoc(context.program, property)) { context.reportDiagnostic(...) }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `index.js` | Linter entry point; exports `$linter` recognised by `@typespec/compiler`. Must import and list every rule in the `rules` array. | Forgetting to add a new rule import and array entry here means the rule is silently never applied during `make gen-api`. |
| `rules/utils.js` | Shared casing helpers (`isPascalCaseNoAcronyms`, `isCamelCaseNoAcronyms`, `isSnakeCase`, `isKebabCase`) and the canonical `pascalCaseExceptions` list. | Adding acronym exceptions anywhere other than `pascalCaseExceptions` in this file causes inconsistent behaviour across all casing rules. |
| `rules/friendly-name.js` | Error-severity rule requiring `@friendlyName` on all interfaces, models, enums, and unions. Blocks `make gen-api` on violation. | Changing severity to `warning` silently removes the compilation block; this is the only error-level rule. |
| `rules/casing.js` | Warns when model names are not PascalCase or model properties are not camelCase. Imports helpers from utils.js. | The `_` property name is exempt from camelCase check via `property.name !== '_'` guard — do not remove this guard. |
| `rules/docs.js` | Warns on missing `@doc` decorator for models, enums, unions, and their non-exempt properties. | Models ending with `Response` skip property-level doc checks — a non-Response suffix on a response model triggers warnings on all its properties. |
| `rules/composition-over-inheritance.js` | Warns when a model extends a base model without a `@discriminator` decorator, encouraging spread or `model is` composition. | Imports `SyntaxKind` from `@typespec/compiler/ast` (sub-path) — verify this sub-path after compiler upgrades. |
| `rules/operation-summary.js` | Warns when an operation lacks a `@summary` decorator. Checks `node.decorators` for the compiler internal name `$summary`. | The internal decorator name is `$summary`, not `summary` — searching without the `$` prefix never matches. |

## Anti-Patterns

- Adding a rule file without exporting a named const and registering it in the `rules` array in `index.js` — the rule is silently never executed.
- Duplicating casing regex or acronym exceptions inline in a rule file instead of importing from `rules/utils.js` — the exception list diverges.
- Using a plain string in `messages` instead of `paramMessage` — loses type-safe interpolation and breaks diagnostic formatting.
- Reporting a diagnostic with `target` set to something other than the offending AST node — editor highlights the wrong location.
- Changing `friendlyNameRule` severity from `error` to `warning` — removes the compilation block that enforces SDK-visible names across Go, JS, and Python.

## Decisions

- **`friendlyNameRule` is the only error-severity rule; all others are warnings.** — `@friendlyName` directly affects generated SDK type names across Go, JS, and Python — a missing decorator causes silent SDK naming drift, which is a hard contract break. Casing and doc issues are fixable post-hoc without breaking SDK consumers.
- **Casing helpers and the acronym exception list are centralised in `utils.js`.** — Multiple rules check the same naming conventions. A single source of truth for `pascalCaseExceptions` prevents divergence when a new acronym is added.
- **Rules use the `@typespec/compiler` `createRule` API with `paramMessage`, not raw AST visitors.** — The `createRule` factory integrates with the TypeSpec diagnostic system for editor highlighting and CLI output. Raw visitors bypass the formatter and message-ID dispatch.

## Example: Adding a new warning rule and wiring it into the linter — full end-to-end pattern

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
      if (!isSnakeCase(member.name.toUpperCase())) {
        context.reportDiagnostic({ messageId: 'default', format: { name: member.name }, target: member })
// ...
```

<!-- archie:ai-end -->
