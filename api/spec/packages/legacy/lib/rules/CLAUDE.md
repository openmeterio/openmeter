# rules

<!-- archie:ai-start -->

> TypeSpec linting rules for the v1 (legacy) API spec, enforcing naming, documentation, and structural constraints on TypeSpec models, properties, enums, unions, interfaces, and operations. Rules run in the TypeSpec compiler lint pipeline (make lint-api-spec) and block or warn before OpenAPI generation.

## Patterns

**createRule factory** — Every rule is a named const from createRule({ name, severity, description, messages, create }). create returns a visitor object keyed by TypeSpec node type (model, modelProperty, enum, union, interface, operation). (`export const casingRule = createRule({ name: 'casing', severity: 'warning', messages: {...}, create: (context) => ({ model: (model) => {...} }) })`)
**paramMessage for diagnostics referencing node names** — Diagnostic templates use paramMessage with ${'placeholder'} syntax; plain strings are not used for messages that reference node names. (`messages: { default: paramMessage`The ${'type'} ${'name'} must have a summary decorator.` }`)
**context.reportDiagnostic with AST target** — Violations are reported only via context.reportDiagnostic({ messageId, format, target }); target is always the offending AST/decorator node so editors can highlight it. (`context.reportDiagnostic({ format: { type: 'Model', casing: 'PascalCase' }, target: model, messageId: 'name' })`)
**Casing helpers centralised in utils.js** — casingRule enforces PascalCase (isPascalCaseNoAcronyms) on model names and camelCase (isCamelCaseNoAcronyms) on property names; '_' is exempt, and a property that is PascalCase also fails. Acronym exceptions (OAuth2, URL, API, UI, ID) live only in utils.js pascalCaseExceptions. (`if ((property.name !== '_' && !isCamelCaseNoAcronyms(property.name)) || isPascalCaseNoAcronyms(property.name)) { ... }`)
**@friendlyName required at error severity** — friendlyNameRule is severity 'error' (blocks compilation) and checks every named interface, model, enum, and union has a decorator whose decorator.name === '$friendlyName'. (`const hasFriendlyName = node.decorators.some((d) => d.decorator.name === '$friendlyName')`)
**Doc/summary/composition checks via program helpers** — docDecoratorRule uses getDoc(context.program, target) (exempting *Response models at property level and '_'/'contentType' props); operationSummaryRule checks for '$summary'; compositionOverInheritanceRule uses getDiscriminator and SyntaxKind.ModelStatement from @typespec/compiler/ast. (`if (model.baseModel && model.node?.kind === SyntaxKind.ModelStatement && model.node.extends && getDiscriminator(context.program, model.baseModel) === undefined) { ... }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `utils.js` | Exports isPascalCaseNoAcronyms, isCamelCaseNoAcronyms, isSnakeCase, isKebabCase. Pascal regex is derived at runtime from the pascalCaseExceptions array ['OAuth2','URL','API','UI','ID']. | Adding acronym exceptions requires updating pascalCaseExceptions here; never duplicate casing logic in individual rule files. |
| `casing.js` | PascalCase on model names, camelCase on property names. | The dual condition catches a property that accidentally becomes PascalCase even when the first camelCase check passes. |
| `friendly-name.js` | Error-severity rule requiring @friendlyName on every named model, enum, union, and interface. | Severity is 'error' not 'warning' — violations block compilation. Do not downgrade; missing @friendlyName breaks SDK code generation. |
| `docs.js` | Checks @doc presence on models, enums, unions, and model properties; skips property checks for *Response model names. | 'contentType' and '_' property exemptions are hardcoded; new structural exempt properties must be added to the exclusion list. |
| `composition-over-inheritance.js` | Warns on model inheritance without @discriminator on the base; distinguishes template-instance inheritance for message selection. | SyntaxKind.ModelStatement is imported from @typespec/compiler/ast — import path is /ast, not the root package. |
| `operation-summary.js` | Warns when an operation lacks a @summary decorator. | Checks for the internal '$summary' decorator name, not the user-facing @summary string. |

## Anti-Patterns

- Adding a rule without exporting it as a named const — the parent linter config imports named exports
- Using plain string messages instead of paramMessage for diagnostics that reference node names
- Reporting diagnostics with a target that is not the offending AST node — breaks editor highlighting
- Downgrading friendlyNameRule from error to warning — it is intentionally blocking
- Duplicating casing logic inline instead of importing from utils.js

## Decisions

- **Rules use the @typespec/compiler createRule API rather than raw AST visitors** — Keeps rules composable with the compiler diagnostic pipeline, enabling IDE integration and make lint-api-spec CI enforcement.
- **friendlyNameRule is error-severity while doc/casing/summary/composition rules are warnings** — Missing @friendlyName breaks SDK code generation (generated type names depend on it); missing docs only degrade API quality.
- **Casing helpers centralised in utils.js with a hardcoded acronym exception list** — Ensures PascalCase checks for OAuth2/URL/API/UI/ID are consistent across all rules without per-rule special-casing.

## Example: Adding a new lint rule that enforces a custom decorator on all enums

```
import { createRule, paramMessage } from '@typespec/compiler'

export const myDecoratorRule = createRule({
  name: 'my-decorator',
  severity: 'warning',
  description: 'Ensure @myDecorator on enums.',
  messages: { default: paramMessage`Enum '${'name'}' is missing @myDecorator.` },
  create: (context) => ({
    enum: (node) => {
      if (node.name && !node.decorators.some((d) => d.decorator.name === '$myDecorator')) {
        context.reportDiagnostic({ format: { name: node.name }, target: node, messageId: 'default' })
      }
    },
  }),
})
```

<!-- archie:ai-end -->
