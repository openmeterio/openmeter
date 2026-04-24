# lib

<!-- archie:ai-start -->

> TypeSpec linter plugin for the v3 AIP spec — exports a $linter via index.js that registers all custom rules enforcing AIP naming conventions, documentation requirements, and structural constraints. Rules run during `make gen-api` (tsp compile) and block or warn on violations before any OpenAPI output is generated.

## Patterns

**createRule factory for all rules** — Every rule is defined via `createRule` from '@typespec/compiler'. Raw functions that call context.report directly are not registered by the TypeSpec rule engine. (`import { createRule } from '@typespec/compiler'
export const myRule = createRule({ name: 'my-rule', severity: 'error', messages: { default: 'msg' }, create: (context) => ({ model: (m) => { /* ... */ } }) })`)
**Two-severity model: error vs warning** — Rules that block compilation (naming errors, missing @friendlyName, missing @doc, enum value casing, inheritance without discriminator error path) use severity: 'error'. Style preferences (model/enum name casing, field prefixes, composition suggestion) use severity: 'warning'. (`// error: blocks gen-api
export const friendlyNameRule = createRule({ severity: 'error', ... })
// warning: reported but does not block
export const casingRule = createRule({ severity: 'warning', ... })`)
**Casing contracts: models+enums PascalCase, properties snake_case, path params camelCase, enum values snake_case** — isPascalCaseNoAcronyms allows OAuth2/URL/API/UI/ID exceptions. isSnakeCase enforces property names. @path-decorated properties use isCamelCaseNoAcronyms. Enum member string values are checked by casingErrorsRule (severity: error) for snake_case. (`// utils.js exceptions list:
const pascalCaseExceptions = ['OAuth2', 'URL', 'API', 'UI', 'ID']`)
**@friendlyName required on all non-Endpoints/non-Operations interfaces and all models/enums/unions** — friendlyNameRule fires severity: error if any interface (that does not end in 'Endpoints' or 'Operations'), model, enum, or union lacks a @friendlyName decorator. Interfaces ending in 'Endpoints'/'Operations' must NOT have @friendlyName. (`@friendlyName("MyResource")
model MyResource { ... }`)
**Composition over inheritance: use spread or 'model is' instead of extends without @discriminator** — compositionOverInheritanceRule warns whenever a model extends a base without a @discriminator decorator. To compose, use spread (...) or 'model is'. For polymorphism, add @discriminator to the base. (`// Bad (warning): model Derived extends Base { ... }
// Good (composition): model Derived { ...Base; extra_field: string }
// Good (polymorphism): @discriminator("kind") model Base { kind: string }`)
**Repeated field prefix grouping rule** — repeatedPrefixGroupingRule warns when 2+ properties share the same prefix_ within a model (e.g. billing_name, billing_id). EXCLUDED_PREFIXES (is_, enable_, disable_, allow_, custom_, default_, disable_, include_in_, initial_, last_, primary_) are exempt. (`// Warning: { billing_name: string, billing_id: string } → group under billing: { name, id }
// Exempt: { is_active: boolean, is_default: boolean } → no warning`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `index.js` | Exports $linter via defineLinter — the single entry point that registers all rules. Must import every new rule here for it to take effect. | Adding a new rule file but forgetting to import and include it in the rules array — the rule will be silently ignored. |
| `rules/utils.js` | Provides isPascalCaseNoAcronyms, isCamelCaseNoAcronyms, isSnakeCase, isKebabCase helpers. Holds the pascalCaseExceptions list (OAuth2, URL, API, UI, ID). | Acronyms not in pascalCaseExceptions will be flagged by casingRule — add new accepted acronyms here, not in individual rule files. |
| `rules/casing.js` | casingRule (warning): models/enums PascalCase, non-path properties snake_case, path properties camelCase, enum member names PascalCase. casingErrorsRule (error): enum member string values must be snake_case. | Path parameters decorated with @path are checked for camelCase, not snake_case — do not rename path params to snake_case to fix a lint warning. |
| `rules/friendly-name.js` | Enforces @friendlyName on all models, enums, unions, and non-Endpoints/Operations interfaces. severity: error. | Every new TypeSpec model, enum, union, and route interface needs @friendlyName — missing it blocks gen-api. |
| `rules/field-prefix.js` | Warns when 2+ model properties share a common prefix_ and should be grouped into a sub-object. EXCLUDED_PREFIXES exempts AIP standard boolean-flag prefixes. | Adding new boolean-flag prefixes without extending EXCLUDED_PREFIXES will generate spurious warnings for is_*, enable_*, etc. |
| `rules/composition-over-inheritance.js` | Warns when a model extends a base without @discriminator. Uses getDiscriminator from '@typespec/compiler' and SyntaxKind from '@typespec/compiler/ast'. | Using 'extends' for mixins without @discriminator will produce warnings — prefer spread (...BaseModel) or 'model is BaseModel'. |

## Anti-Patterns

- Defining a rule as a raw function calling context.report directly instead of using createRule — TypeSpec rule registration requires createRule
- Using severity: 'error' for stylistic preferences (casing, prefix grouping) — errors block gen-api; use 'warning' for non-blocking guidance
- Adding new accepted PascalCase acronyms (e.g. SDK, RPC) anywhere other than pascalCaseExceptions in utils.js
- Forgetting to import a new rule in index.js — the rule file is valid JS but has zero effect until registered
- Adding underscore-prefixed boolean-flag field groups without adding the prefix to EXCLUDED_PREFIXES in field-prefix.js

## Decisions

- **Two-severity model: error blocks compilation, warning surfaces in lint output but does not fail build.** — Hard naming contracts (friendlyName, enum value casing) must be enforced before code generation can succeed; stylistic preferences (model casing, prefix grouping) should guide authors without halting the build on every iteration.
- **PascalCase exceptions list (OAuth2, URL, API, UI, ID) codified in utils.js rather than inline regex.** — Centralising exceptions makes them reviewable and extensible without touching each rule that uses isPascalCaseNoAcronyms.
- **EXCLUDED_PREFIXES in field-prefix.js exempts AIP-standard boolean-flag prefixes (is_, enable_, disable_, etc.).** — AIP guidelines explicitly allow these prefix patterns for boolean fields; grouping them into sub-objects would violate AIP conventions.

## Example: Adding a new AIP linter rule that enforces a required decorator on all operations

```
// rules/my-rule.js
import { createRule, paramMessage } from '@typespec/compiler'

export const myRule = createRule({
  name: 'my-rule',
  severity: 'error',
  description: 'Ensure @myDecorator on all operations.',
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
