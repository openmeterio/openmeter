# rules

<!-- archie:ai-start -->

> Custom TypeSpec linter rules enforcing AIP naming conventions and structural constraints for the v3 API spec. Rules run during `make gen-api` (tsp compile) and block or warn on violations before any OpenAPI or SDK artifact is generated.

## Patterns

**createRule factory for every rule** — Every rule must be exported via `createRule` from `@typespec/compiler`, returning a visitor-callback object. Never define rules as raw functions or classes. (`export const myRule = createRule({ name: 'my-rule', severity: 'warning', messages: { default: paramMessage`...` }, create: (context) => ({ model: (node) => { ... } }) })`)
**Two-tier severity: error blocks generation, warning informs** — Rules whose violations corrupt wire format or SDK method names (enum values snake_case, operationId kebab-case, @friendlyName presence) use `severity: 'error'`. Style/doc rules use `severity: 'warning'`. (`casingErrorsRule (enum member values) → severity: 'error'. docDecoratorRule (missing @doc) → severity: 'warning'.`)
**Casing contracts by node type** — Model names: PascalCase. Non-path model properties: snake_case. Path parameters ($path decorator): camelCase. Enum names/member names: PascalCase. Enum member values: snake_case. @operationId values: kebab-case. All verified via utils.js helpers. (`casing.js checks `property.decorators.find(d => d.decorator.name === '$path')` before switching between camelCase and snake_case checks.`)
**@friendlyName required on named types (inverted for *Endpoints/*Operations interfaces)** — All named models, enums, unions, and interfaces must have @friendlyName EXCEPT interfaces ending in `Endpoints` or `Operations`, which must NOT have it. Violation is severity: 'error'. (`friendly-name.js: `isEndpointsOrOperations && hasFriendlyName` → reports 'avoid'. `!isEndpointsOrOperations && !hasFriendlyName` → reports 'default'.`)
**Casing utilities live exclusively in utils.js** — isPascalCaseNoAcronyms, isCamelCaseNoAcronyms, isSnakeCase, isKebabCase are exported from utils.js with PascalCase exceptions list (OAuth2, URL, API, UI, ID). All rule files import from utils.js; never inline regexes in rule files. (`import { isPascalCaseNoAcronyms, isSnakeCase } from './utils.js'`)
**field-prefix EXCLUDED_PREFIXES governs exemptions** — When 2+ fields share the same underscore prefix (e.g. billing_id, billing_status), a warning fires urging grouping under a nested object. AIP-standard prefixes (is, enable, disable, allow, etc.) are exempt via EXCLUDED_PREFIXES array in field-prefix.js. (`New prefixes that should be globally exempt must be added to EXCLUDED_PREFIXES in field-prefix.js; otherwise all existing uses of that prefix trigger warnings.`)
**Composition over inheritance: spread or `model is` preferred** — Using `extends` without `@discriminator` on the base model triggers a warning. composition-over-inheritance.js distinguishes template instances (different message) from plain model extension using isTemplateInstance(). (`model Foo { ...BaseFields; }  // OK. model Foo extends Base { }  // warning unless @discriminator on Base.`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `utils.js` | Single source for all casing helpers (isPascalCaseNoAcronyms, isCamelCaseNoAcronyms, isSnakeCase, isKebabCase) plus doc-comment string manipulation helpers (extractMarkdownFromDocComment, wrapMarkdownAsDocComment, detectNewline, getIndentBefore). PascalCase exceptions list maintained here. | Adding a new casing helper must go here; inlining regex in rule files breaks consistency. PascalCase exceptions (OAuth2, URL, API, UI, ID) are hardcoded — add new acronyms here if the regex incorrectly flags them. |
| `casing.js` | Two rules: `casing` (warning) for structural name casing; `casing-aip-errors` (error) for enum member values (must be snake_case). The modelProperty visitor explicitly checks for `$path` decorator before choosing camelCase vs snake_case. | Path parameter properties use camelCase — the `$path` decorator check is critical. Forgetting it causes false positives on path params. |
| `friendly-name.js` | Error-severity rule requiring @friendlyName on all named models, enums, unions, and interfaces. The interface logic is INVERTED: *Endpoints/*Operations must NOT have @friendlyName; all others must. | Getting the inversion wrong (flagging Endpoints interfaces for missing @friendlyName) is the most common mistake. Check `isEndpointsOrOperations` logic carefully. |
| `docs.js` | Two rules: `doc-decorator` (warning on missing @doc for models/enums/unions/properties) and `doc-format` (async warning that Prettier-formats doc comment bodies). Models ending in `Response` are exempt from property-level @doc checks. Properties named `_` or `contentType` are always exempt. | doc-format is async (Prettier 3.x); the rule declares `async: true` and uses an `exit()` hook to batch-process all collected DocNodes after all visitors run. Do not make async rules sync. |
| `field-prefix.js` | Warns when 2+ fields share the same underscore prefix within a model. EXCLUDED_PREFIXES exempts AIP-standard boolean-flag prefixes. | New prefixes that should be globally exempt (e.g. AIP-defined boolean prefixes) must be added to EXCLUDED_PREFIXES or they trigger warnings on all existing and future uses. |
| `no-nullable.js` | Warns against `field: T | null` — use `field?: T` instead. Only fires on non-optional union properties that include a null variant. | Only checks `!property.node?.optional` — optional properties with `| null` are not flagged. The rule is narrowly scoped. |
| `composition-over-inheritance.js` | Warning when `extends` is used without `@discriminator` on the base. Uses isTemplateInstance() to give a different message for template-based extension vs plain model extension. | isTemplateInstance check distinguishes template bases from plain bases — the messageId differs ('instance' vs 'default'). Ensure both message variants remain in sync. |

## Anti-Patterns

- Defining rules as raw functions instead of using createRule — TypeSpec rule registration requires createRule for proper diagnostic reporting and registration
- Using `| null` on model properties instead of the optional `?` modifier — triggers no-nullable warning
- Adding a field-prefix exemption inline in the rule logic instead of in EXCLUDED_PREFIXES — breaks the single-source pattern
- Using `extends` for composition without @discriminator on the base model — triggers composition-over-inheritance warning
- Omitting @friendlyName, @doc, or @summary on new named types/operations — these are errors or warnings that surface in `make gen-api` and block CI lint

## Decisions

- **Error severity for wire-format and SDK-method-name constraints; warning severity for stylistic consistency** — Enum member values must be snake_case (JSON wire format) and @operationId must be kebab-case (SDK method name generation) — violations corrupt generated artifacts. Doc/composition rules are warnings to allow incremental adoption.
- **PascalCase exceptions list (OAuth2, URL, API, UI, ID) codified in utils.js** — AIP naming conventions acknowledge common industry acronyms; without the exceptions list, names like OAuthToken or APIKey would be incorrectly flagged as non-PascalCase.
- **field-prefix EXCLUDED_PREFIXES exempts AIP-standard boolean-flag prefixes (is, enable, disable, allow, etc.)** — AIP itself defines standard boolean field prefixes (is_active, enable_x, disable_y) that intentionally repeat; grouping them under a nested object would violate AIP guidance.

## Example: Adding a new custom lint rule that warns when an operation lacks a @tag decorator

```
import { createRule, paramMessage } from '@typespec/compiler'

export const operationTagRule = createRule({
  name: 'operation-tag',
  severity: 'warning',
  description: 'Ensure operations have a @tag decorator.',
  messages: {
    default: paramMessage`Operation '${'name'}' must have a @tag decorator.`,
  },
  create: (context) => ({
    operation: (node) => {
      if (node.name && !node.decorators.some((d) => d.decorator.name === '$tag')) {
        context.reportDiagnostic({
          format: { name: node.name },
          target: node,
// ...
```

<!-- archie:ai-end -->
