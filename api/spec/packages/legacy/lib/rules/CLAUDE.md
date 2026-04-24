# rules

<!-- archie:ai-start -->

> TypeSpec linting rules for the v1 (legacy) API spec, enforcing naming conventions, documentation, and structural constraints on TypeSpec models, properties, enums, unions, interfaces, and operations. These rules run via the TypeSpec compiler lint pipeline and block or warn on spec violations before OpenAPI is generated.

## Patterns

**createRule factory** — Every rule is exported as a named const created with `createRule({ name, severity, description, messages, create })` from `@typespec/compiler`. The `create` function returns a visitor object keyed by TypeSpec node type (model, modelProperty, enum, union, interface, operation). (`export const myRule = createRule({ name: 'my-rule', severity: 'warning', messages: { default: paramMessage`...` }, create: (context) => ({ model: (node) => { ... } }) })`)
**paramMessage for diagnostic messages** — All diagnostic message templates use `paramMessage` from `@typespec/compiler` with `${'placeholder'}` syntax. Plain strings are not used for messages that reference node names. (`messages: { default: paramMessage`The ${'type'} ${'name'} must have a summary decorator.` }`)
**context.reportDiagnostic for violations** — Rules report violations exclusively via `context.reportDiagnostic({ messageId, format, target })`. The `target` is always the offending AST node or decorator node. (`context.reportDiagnostic({ format: { type: 'Model', casing: 'PascalCase' }, target: model, messageId: 'name' })`)
**PascalCase for models, camelCase for properties** — `casingRule` enforces PascalCase (via `isPascalCaseNoAcronyms`) on model names and camelCase (via `isCamelCaseNoAcronyms`) on model property names. The `_` property name is exempt; property names that are PascalCase also fail. (`if (!isPascalCaseNoAcronyms(model.name)) { context.reportDiagnostic(...) }`)
**@friendlyName required on interfaces, models, enums, unions** — `friendlyNameRule` is severity `error` (not warning) and checks that every named interface, model, enum, and union has a decorator whose `decorator.name === '$friendlyName'`. (`const hasFriendlyName = node.decorators.some((d) => d.decorator.name === '$friendlyName')`)
**Doc decorator required on models and their properties** — `docDecoratorRule` calls `getDoc(context.program, target)` to check presence of `@doc`. Models named with `*Response` suffix are exempt from property-level doc checks. `_` and `contentType` properties are exempt. (`if (target.name && !getDoc(context.program, target)) { context.reportDiagnostic(...) }`)
**Composition over inheritance enforcement** — `compositionOverInheritanceRule` warns when a model extends another model (via `model.node.extends`) without a `@discriminator` on the base. Template-instance inheritance uses the `instance` message ID; plain inheritance uses `default`. (`if (model.baseModel && model.node?.kind === SyntaxKind.ModelStatement && model.node.extends && getDiscriminator(context.program, model.baseModel) === undefined) { ... }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `utils.js` | Exports `isPascalCaseNoAcronyms`, `isCamelCaseNoAcronyms`, `isSnakeCase`, `isKebabCase`. The Pascal regex has hardcoded exceptions: `['OAuth2', 'URL', 'API', 'UI', 'ID']`. | Adding new acronym exceptions requires updating `pascalCaseExceptions` in utils.js; the regex is derived from that array at runtime. |
| `casing.js` | Enforces PascalCase on model names and camelCase on property names. A property that IS PascalCase also triggers the warning. | The dual condition `(name !== '_' && !isCamelCase(name)) || isPascalCase(name)` means a property that accidentally becomes PascalCase is caught even if the first check passes. |
| `friendly-name.js` | Error-severity rule requiring `@friendlyName` on every named model, enum, union, and interface. | Severity is `error`, not `warning` — violations block compilation, not just lint. |
| `docs.js` | Checks `@doc` presence on models, enums, unions, and model properties. Skips property checks for `*Response` model names. | The `contentType` and `_` property exemptions are hardcoded; new structural properties that should be exempt need to be added to the exclusion list. |
| `composition-over-inheritance.js` | Detects model inheritance without `@discriminator` on the base; distinguishes template-instance inheritance from plain inheritance for message selection. | Uses `SyntaxKind.ModelStatement` from `@typespec/compiler/ast` — import path is `/ast`, not the root. |
| `operation-summary.js` | Warns when an operation lacks a `@summary` decorator, checked via `decorator.name === '$summary'`. | Checks for `$summary` (dollar-prefixed internal decorator name), not the user-facing `@summary` string. |

## Anti-Patterns

- Adding a rule without exporting it as a named const — the rule registry in the parent linter config imports named exports from this folder.
- Using plain string messages instead of `paramMessage` for diagnostics that reference node names.
- Reporting diagnostics with a `target` that is not the offending AST node — callers rely on the target for editor highlighting.
- Changing `friendlyNameRule` severity from `error` to `warning` — it is intentionally blocking.
- Duplicating casing logic inline instead of importing from utils.js — the Pascal exception list must stay in one place.

## Decisions

- **Rules use the `@typespec/compiler` `createRule` API rather than raw AST visitors.** — Keeps rules composable with the TypeSpec compiler's diagnostic pipeline, enabling IDE integration and `make lint-api-spec` CI enforcement.
- **`friendlyNameRule` is error-severity while doc/casing/summary/composition rules are warnings.** — Missing `@friendlyName` breaks SDK code generation (generated type names depend on it); missing docs only degrade API quality.
- **Casing helpers are centralised in utils.js with a hardcoded acronym exception list.** — Ensures PascalCase checks for `OAuth2`, `URL`, `API`, `UI`, `ID` are consistent across all rules without per-rule special-casing.

## Example: Adding a new lint rule that enforces a custom decorator on all enums

```
import { createRule, paramMessage } from '@typespec/compiler'

export const myDecoratorRule = createRule({
  name: 'my-decorator',
  severity: 'warning',
  description: 'Ensure @myDecorator on enums.',
  messages: {
    default: paramMessage`Enum '${'name'}' is missing @myDecorator.`,
  },
  create: (context) => ({
    enum: (node) => {
      if (node.name && !node.decorators.some((d) => d.decorator.name === '$myDecorator')) {
        context.reportDiagnostic({ format: { name: node.name }, target: node, messageId: 'default' })
      }
    },
// ...
```

<!-- archie:ai-end -->
