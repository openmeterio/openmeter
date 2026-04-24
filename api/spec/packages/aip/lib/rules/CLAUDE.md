# rules

<!-- archie:ai-start -->

> Custom TypeSpec linter rules enforcing AIP naming conventions and structural constraints for the v3 API spec. These rules run during `make gen-api` (tsp compile) and fail the build on errors or emit warnings on violations.

## Patterns

**createRule factory** — Every rule is defined using `createRule` from `@typespec/compiler`, returning an object of TypeSpec node visitor callbacks (model, modelProperty, enum, operation, interface, union). Never use ad-hoc functions. (`export const myRule = createRule({ name: '...', severity: 'warning'|'error', messages: {...}, create: (context) => ({ model: (node) => { ... } }) })`)
**severity: error vs warning** — Rules that must block code generation use `severity: 'error'` (casingErrorsRule, friendlyNameRule, operationIdKebabCaseRule). Style guidance uses `severity: 'warning'`. New rules must pick severity intentionally. (`casingErrorsRule: enum member values must be snake_case — error. casingRule: model names must be PascalCase — warning.`)
**casing contracts** — Model names: PascalCase (isPascalCaseNoAcronyms). Non-path model properties: snake_case (isSnakeCase). Path parameters: camelCase (isCamelCaseNoAcronyms). Enum names/members: PascalCase. Enum member values: snake_case. @operationId values: kebab-case. All checked via utils.js helpers. (`path param `subscriptionId` → camelCase OK. body field `plan_id` → snake_case OK. operationId `list-meters` → kebab-case OK.`)
**@friendlyName required on all named types** — All named models, enums, unions, and non-Endpoints/Operations interfaces must have `@friendlyName`. Interfaces ending in `Endpoints` or `Operations` must NOT have `@friendlyName`. This is an error-severity rule. (`@friendlyName("Meter") model MeterResource { ... }`)
**@summary required on operations** — Every operation must have a `@summary` decorator. Missing summary is a warning that surfaces in linting output. (`@summary("List meters") @get op listMeters(): MeterList;`)
**@doc required on models, enums, unions, and properties** — All named models, enums, unions, and their non-special properties (excluding `_` and `contentType`) require a `@doc` decorator. Models ending in `Response` are exempt from property-level doc checks. (`@doc("A usage meter") model Meter { @doc("Unique slug") slug: string; }`)
**Composition over inheritance** — Model inheritance via `extends` is flagged as a warning if the base model lacks `@discriminator`. Prefer spread (`...BaseModel`) or `model is BaseModel` for composition. (`model Foo { ...BaseFields; }  // OK. model Foo extends Base { }  // warning unless @discriminator on Base.`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `utils.js` | Exports isPascalCaseNoAcronyms, isCamelCaseNoAcronyms, isSnakeCase, isKebabCase — shared by all rule files. PascalCase exceptions: OAuth2, URL, API, UI, ID. | Adding a new casing utility must be added here and imported; do not inline regex in rule files. |
| `casing.js` | Enforces naming casing on models, properties, enums, enum members. Two rules: `casing` (warning) for structural names; `casing-aip-errors` (error) for enum member values. | Path parameters (`$path` decorator) use camelCase, not snake_case — the rule explicitly checks for `$path` decorator presence before switching casing check. |
| `friendly-name.js` | Error-severity rule requiring @friendlyName on all named models, enums, unions, and interfaces (except *Endpoints/*Operations interfaces which must NOT have it). | Interfaces: the rule inverts — *Endpoints/*Operations must NOT have @friendlyName; all others must. Getting this backwards produces an error. |
| `field-prefix.js` | Warns when 2+ fields share the same underscore prefix within a model (e.g. billing_id, billing_status → should be billing.id, billing.status). EXCLUDED_PREFIXES list exempts AIP-standard prefixes. | New prefixes that should be exempt must be added to EXCLUDED_PREFIXES or they will trigger warnings on all existing uses. |
| `no-nullable.js` | Warns against `field: T | null` — use `field?: T` instead for optional properties. | Only fires on non-optional (`!property.node?.optional`) union properties that include a null variant. |
| `operation-id.js` | Error rule enforcing kebab-case on explicit `@operationId` decorator values. | Only checks properties with explicit @operationId — auto-generated operation IDs are not checked. |
| `composition-over-inheritance.js` | Warning rule discouraging `extends` without `@discriminator` on base. Distinguishes template instances (different message) from plain model extension. | isTemplateInstance check — template base models get a different diagnostic message than plain model bases. |

## Anti-Patterns

- Defining rules outside `createRule` (e.g. raw functions that call context.report) — TypeSpec rule registration requires createRule
- Using `| null` on model properties instead of the optional `?` modifier
- Adding underscore-prefixed field groups without checking EXCLUDED_PREFIXES in field-prefix.js
- Using `extends` for composition without adding `@discriminator` to the base model
- Omitting `@friendlyName`, `@doc`, or `@summary` on new types/operations — these are linting errors/warnings that block CI

## Decisions

- **Two-severity model: errors block compilation, warnings surface in lint but don't fail build** — Structural invariants (enum values must be snake_case for JSON wire format, operationId must be kebab-case for SDK method names) are errors; stylistic consistency (docs, composition preference) are warnings to allow incremental adoption.
- **PascalCase exceptions list (OAuth2, URL, API, UI, ID) codified in utils.js** — AIP naming conventions acknowledge common industry acronyms; the regex would incorrectly flag OAuthToken or APIKey without the exceptions list.
- **field-prefix rule EXCLUDED_PREFIXES exempts AIP-standard boolean-flag prefixes (is_, enable_, disable_, etc.)** — AIP itself defines standard boolean field prefixes that intentionally repeat; grouping them under a nested object would violate AIP guidance.

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
      if (
        node.name &&
        !node.decorators.some((d) => d.decorator.name === '$tag')
      ) {
// ...
```

<!-- archie:ai-end -->
