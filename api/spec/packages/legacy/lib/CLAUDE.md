# lib

<!-- archie:ai-start -->

> TypeSpec linter package for the v1 (legacy) API spec: index.js assembles all lint rules via defineLinter into the $linter export the TypeSpec compiler picks up during make gen-api, enforcing naming, documentation and structural constraints before OpenAPI/SDKs are generated. The rules/ child holds each individual rule implementation.

## Patterns

**Rule registration via defineLinter** — Every rule must be imported and added to the rules array in index.js, then passed to defineLinter({ rules }). A rule omitted from this array is silently never executed. (`import { casingRule } from './rules/casing.js'; const rules = [casingRule, ...]; export const $linter = defineLinter({ rules })`)
**createRule factory with paramMessage** — Rules use createRule from @typespec/compiler with a messages map of paramMessage templates; plain string messages break type-safe interpolation. (`export const casingRule = createRule({ name: 'casing', severity: 'warning', messages: { name: paramMessage`...${'type'}...` }, create: (context) => ({ ... }) })`)
**Report diagnostics against the offending AST node** — context.reportDiagnostic must set target to the offending model/property/decorator node so editor tooling highlights the right location. (`context.reportDiagnostic({ format: { type: 'Model', casing: 'PascalCase' }, target: model, messageId: 'name' })`)
**Centralised casing helpers in utils.js** — isPascalCaseNoAcronyms/isCamelCaseNoAcronyms/isSnakeCase and the pascalCaseExceptions acronym list live only in rules/utils.js; rule files import them, never redefine casing logic inline. (`import { isCamelCaseNoAcronyms, isPascalCaseNoAcronyms } from './utils.js'`)
**friendlyName is the only error-severity rule** — friendlyNameRule uses severity: 'error' and blocks make gen-api; all other rules are warnings. Do not lower it — a missing @friendlyName breaks SDK code generation across Go/JS/Python. (`export const friendlyNameRule = createRule({ name: 'friendlyName', severity: 'error', ... })`)
**Doc-check naming exemptions** — docDecoratorRule skips properties named '_' or 'contentType' and skips all property checks on models whose names end with 'Response'; new response models must use that suffix. (`if (!['_', 'contentType'].includes(name) && !getDoc(context.program, property)) { context.reportDiagnostic(...) }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `index.js` | Linter entry point exporting $linter recognised by @typespec/compiler; imports and lists every rule in the rules array. | Forgetting to add a new rule import + array entry means the rule is silently never applied during make gen-api. |
| `rules/utils.js` | Shared casing helpers and the canonical pascalCaseExceptions acronym list. | Adding acronym exceptions anywhere else causes inconsistent behaviour across all casing rules. |
| `rules/friendly-name.js` | Error-severity rule requiring @friendlyName on interfaces/models/enums/unions; blocks make gen-api on violation. | Lowering severity to warning silently removes the compilation block — this is the only error-level rule. |
| `rules/casing.js` | Warns when model names aren't PascalCase or properties aren't camelCase; imports helpers from utils.js. | The '_' property is exempt via a name guard — do not remove it. |
| `rules/docs.js` | Warns on missing @doc for models/enums/unions and non-exempt properties. | Models ending in 'Response' skip property doc checks; a non-Response suffix triggers warnings on all properties. |
| `rules/composition-over-inheritance.js` | Warns when a model extends a base without @discriminator, encouraging spread / model is composition. | Imports SyntaxKind from the @typespec/compiler/ast sub-path — verify after compiler upgrades. |
| `rules/operation-summary.js` | Warns when an operation lacks a @summary decorator. | The internal decorator name is $summary (with the $ prefix), not summary. |

## Anti-Patterns

- Adding a rule file without exporting a named const and registering it in the rules array in index.js — silently never executed.
- Duplicating casing regex or acronym exceptions inline instead of importing from rules/utils.js — the exception list diverges.
- Using a plain string in messages instead of paramMessage — loses type-safe interpolation and breaks diagnostic formatting.
- Reporting a diagnostic with target set to something other than the offending AST node — editor highlights the wrong location.
- Changing friendlyNameRule severity from error to warning — removes the SDK-name enforcement block across Go/JS/Python.

## Decisions

- **friendlyNameRule is the only error-severity rule; all others are warnings.** — @friendlyName directly affects generated SDK type names; a missing decorator is a silent, hard contract break, while casing/doc issues are fixable post-hoc.
- **Casing helpers and the acronym exception list are centralised in utils.js.** — Multiple rules check the same conventions; one source of truth for pascalCaseExceptions prevents divergence when a new acronym is added.
- **Rules use createRule with paramMessage rather than raw AST visitors.** — createRule integrates with the TypeSpec diagnostic system for editor highlighting and CLI output; raw visitors bypass the formatter and message-ID dispatch.

## Example: Add a new warning rule and wire it into the linter

```
// rules/enum-casing.js
import { createRule, paramMessage } from '@typespec/compiler'
import { isSnakeCase } from './utils.js'
export const enumCasingRule = createRule({
  name: 'enum-casing', severity: 'warning',
  messages: { default: paramMessage`Enum member '${'name'}' must use UPPER_SNAKE_CASE` },
  create: (context) => ({ enumMember: (member) => {
    if (!isSnakeCase(member.name.toUpperCase())) {
      context.reportDiagnostic({ messageId: 'default', format: { name: member.name }, target: member })
    }
  } }),
})
// then import enumCasingRule and add it to the rules array in index.js
```

<!-- archie:ai-end -->
