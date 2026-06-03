# rules

<!-- archie:ai-start -->

> Custom TypeSpec linter rules enforcing AIP naming conventions and structural constraints for the v3 API spec. Rules run during make gen-api (tsp compile) and block (error) or warn before any OpenAPI/SDK artifact is generated.

## Patterns

**createRule factory for every rule** — Every rule is exported via createRule from @typespec/compiler returning a visitor-callback object; never raw functions or classes. (`export const myRule = createRule({ name: 'my-rule', severity: 'warning', messages: { default: paramMessage`...` }, create: (context) => ({ model: (node) => { ... } }) })`)
**Two-tier severity: error blocks, warning informs** — Rules whose violations corrupt wire format or SDK method names (enum values snake_case, @friendlyName presence) use severity:'error'; style/doc rules use 'warning'. (`casingErrorsRule (enum member values) -> 'error'; docDecoratorRule (missing @doc) -> 'warning'`)
**Casing contracts by node type** — Model names PascalCase; non-path properties snake_case; $path params camelCase; enum names/members PascalCase; enum member values snake_case; @operationId kebab-case. The $path decorator is checked before choosing camelCase vs snake_case. (`property.decorators.find(d => d.decorator.name === '$path') ? camelCase : snake_case`)
**@friendlyName required, inverted for *Endpoints/*Operations** — All named models/enums/unions/interfaces require @friendlyName EXCEPT interfaces ending in Endpoints or Operations, which must NOT have it. Violation severity is 'error'. (`isEndpointsOrOperations && hasFriendlyName -> report 'avoid'; !isEndpointsOrOperations && !hasFriendlyName -> report 'default'`)
**Casing helpers live only in utils.js** — isPascalCaseNoAcronyms/isCamelCaseNoAcronyms/isSnakeCase/isKebabCase are exported from utils.js with a PascalCase exceptions list (OAuth2, URL, API, UI, ID). Rule files import them; never inline regexes. (`import { isPascalCaseNoAcronyms, isSnakeCase } from './utils.js'`)
**field-prefix EXCLUDED_PREFIXES governs exemptions** — When 2+ fields share an underscore prefix a grouping warning fires; AIP-standard boolean prefixes (is, enable, disable, allow, etc.) are exempt via the EXCLUDED_PREFIXES array. (`EXCLUDED_PREFIXES in field-prefix.js — add new globally-exempt prefixes here`)
**Composition over inheritance** — extends without @discriminator on the base triggers a warning; composition-over-inheritance.js uses isTemplateInstance() to give a different message for template-based vs plain extension. (`model Foo { ...BaseFields; } // OK   vs   model Foo extends Base {} // warning unless @discriminator on Base`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `utils.js` | Single source for casing helpers and doc-comment string helpers (extractMarkdownFromDocComment, wrapMarkdownAsDocComment, detectNewline, getIndentBefore); holds the PascalCase exceptions list. | Add new acronyms (e.g. that the regex wrongly flags) to the exceptions list here; never inline regexes in rule files. |
| `casing.js` | Two rules: casing (warning) for name casing; casing-aip-errors (error) for enum member values being snake_case. The modelProperty visitor checks $path before camelCase vs snake_case. | Forgetting the $path check produces false positives on path params. |
| `friendly-name.js` | Error-severity rule requiring @friendlyName on named types; INVERTED for *Endpoints/*Operations interfaces. | Getting the inversion wrong (flagging Endpoints interfaces for missing @friendlyName) is the most common mistake. |
| `docs.js` | doc-decorator (warning on missing @doc) and doc-format (async Prettier formatting of doc bodies). Models ending in Response are exempt from property @doc; properties named _ or contentType always exempt. | doc-format is async (Prettier 3.x), declares async:true and batch-processes DocNodes in an exit() hook — do not make it sync. |
| `field-prefix.js` | Warns when 2+ fields share an underscore prefix; EXCLUDED_PREFIXES exempts AIP boolean-flag prefixes. | Globally-exempt prefixes must go in EXCLUDED_PREFIXES or they warn on all uses. |
| `no-nullable.js` | Warns against `field: T | null`; prefer `field?: T`. Only fires on non-optional union properties including null. | Checks !property.node?.optional — optional `| null` properties are intentionally not flagged. |
| `composition-over-inheritance.js` | Warns on extends without @discriminator; isTemplateInstance() distinguishes template vs plain bases (messageId 'instance' vs 'default'). | Keep both message variants in sync. |

## Anti-Patterns

- Defining rules as raw functions instead of createRule
- Using `| null` on properties instead of the optional `?` modifier
- Adding a field-prefix exemption inline instead of in EXCLUDED_PREFIXES
- Using extends for composition without @discriminator on the base
- Omitting @friendlyName/@doc/@summary on new named types/operations

## Decisions

- **Error severity for wire-format/SDK-name constraints; warning for stylistic** — Enum values must be snake_case (JSON wire) and @operationId kebab-case (SDK method names) — violations corrupt artifacts; doc/composition rules are warnings to allow incremental adoption.
- **PascalCase exceptions list (OAuth2, URL, API, UI, ID) in utils.js** — AIP acknowledges common industry acronyms; without the list, names like APIKey would be wrongly flagged.
- **field-prefix exempts AIP boolean-flag prefixes via EXCLUDED_PREFIXES** — AIP defines standard boolean prefixes (is_active, enable_x) that intentionally repeat; grouping them would violate AIP guidance.

<!-- archie:ai-end -->
