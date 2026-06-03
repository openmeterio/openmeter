# aip

<!-- archie:ai-start -->

> Root TypeSpec package for the v3 (AIP-style) OpenMeter API. Compiles domain operation interfaces into api/v3/openapi.yaml (and the Konnect variant) via two root namespace files, then normalises the emitted YAML with post-processing scripts and enforces naming/structural constraints with a custom linter. Primary constraint: all route binding, tag metadata, and security schemes live in src/; domain model and operation definitions live in domain sub-folders that are barrel-imported here.

## Patterns

**Route and tag binding at root only** — @route and @tag are declared exclusively in src/openmeter.tsp and src/konnect.tsp, never inside a domain sub-folder's operations.tsp. Domain interfaces are mounted by extending them in a root interface body. (`src/openmeter.tsp: interface MeterRoutes extends MeterOperations { @route("/api/v1/meters") @tag("Meters") ... }`)
**Two-step compile pipeline** — pnpm generate runs tsp compile, then scripts/flatten-allof.mjs and scripts/seal-object-schemas.mjs post-process the emitted YAML. Never hand-edit the output; always regenerate via make gen-api then make generate. (`package.json generate: tsp compile --config tspconfig.yaml ./src && node ./scripts/flatten-allof.mjs ... && node ./scripts/seal-object-schemas.mjs ...`)
**Domain ops imported via index.tsp barrels** — Each domain sub-folder exposes an index.tsp barrel; root files import only the barrel, never individual operation files. Importing a domain both directly and via its barrel causes duplicate symbol errors. (`src/openmeter.tsp: import "../meters/index.tsp";`)
**Two-namespace compilation (OpenMeter vs Konnect)** — main.tsp imports both openmeter.tsp and konnect.tsp to emit two distinct OpenAPI outputs from one shared domain library; the variants differ in security schemes and some sub-route overrides. (`src/main.tsp imports openmeter.tsp, konnect.tsp, test.tsp`)
**Security via @useRef to external YAML** — Security schemes are TypeSpec model stubs with @useRef pointing to common/definitions/*.yaml fragments, never inline credential definitions. (`@useRef("../common/definitions/security.yaml") model CloudTokenAuth {}`)
**Custom linter via createRule, registered in lib/index.js** — Every lint rule in lib/rules/ uses the createRule factory and must be registered in the lib/index.js $linter rules array. Error severity blocks gen-api; warning surfaces guidance only. An unregistered rule file has zero runtime effect. (`lib/index.js: export const $linter = defineLinter({ rules: [casingRule, friendlyNameRule, ...] })`)
**Sealed, reachable-only emitted schemas** — tspconfig.yaml must keep omit-unreachable-types: true and seal-object-schemas: true; seal-object-schemas.mjs further rewrites additionalProperties:{not:{}} to false to preserve closed-schema semantics. (`tspconfig.yaml options['@typespec/openapi3']: omit-unreachable-types: true; seal-object-schemas: true`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `src/main.tsp` | Compilation entry point importing openmeter.tsp and konnect.tsp to produce two OpenAPI variants. | A missing import here silently drops a whole namespace from every generated artifact. |
| `src/openmeter.tsp` | Binds all domain interfaces to @route paths and @tag metadata for the self-hosted OpenMeter variant. | Adding a domain import without a matching @tagMetadata declaration silently omits the tag description from generated OpenAPI. |
| `src/konnect.tsp` | Konnect variant root; overrides security schemes and some sub-routes relative to openmeter.tsp. | Inline security scheme bodies duplicate config — use @useRef to common/definitions/security.yaml. |
| `scripts/flatten-allof.mjs` | Post-processes emitted YAML to move sibling properties into allOf branches for SDK generators; idempotent via the x-flatten-allOf marker. | Removing the x-flatten-allOf marker check causes infinite re-run loops; changing YAML_OPTIONS.lineWidth from 0 creates spurious line-wrap diffs. |
| `scripts/seal-object-schemas.mjs` | Rewrites additionalProperties:{not:{}} to false to keep object schemas closed after compilation. | Removing it lets SDK generators emit overly permissive types. |
| `lib/index.js` | Registers all linter rules via defineLinter/$linter, consumed by the compiler during gen-api. | A new rule file is inert until its export is added to the rules array here — no compile error signals the omission. |
| `lib/rules/utils.js` | Centralised casing helpers and the pascalCaseExceptions acronym list shared by all rules. | Adding accepted acronyms inline in individual rule files instead of here causes divergence from the canonical set. |
| `tspconfig.yaml` | Emitter config: output dir, omit-unreachable-types, seal-object-schemas, linter extends @openmeter/api-spec-aip/all. | omit-unreachable-types and seal-object-schemas must stay true; emitter-output-dir must match Makefile/CI expectations. |

## Anti-Patterns

- Declaring @route or @tag inside a domain sub-folder's operations.tsp — routing is bound only in src/openmeter.tsp and src/konnect.tsp
- Hand-editing api/v3/openapi.yaml or api/v3/api.gen.go — always regenerate via make gen-api then make generate
- Defining a lint rule without registering it in lib/index.js — the rule compiles but has zero runtime effect
- Adding accepted PascalCase acronyms anywhere other than pascalCaseExceptions in lib/rules/utils.js
- Importing a domain twice (directly and via its index.tsp barrel) — causes duplicate-symbol compilation errors

## Decisions

- **Two root namespace files (openmeter.tsp, konnect.tsp) instead of one parameterised file** — The variants differ in security schemes and some sub-route overrides, and TypeSpec has no preprocessor; separate files keep the variants explicit rather than buried in emitter workarounds.
- **Post-processing scripts run after tsp compile rather than as emitter plugins** — The emitter produces allOf-with-siblings schemas and additionalProperties:{not:{}} that confuse SDK generators; these are output-level fixups that cannot be done inside the compiler without patching upstream.
- **Error-vs-warning severity split in the linter** — Only structurally breaking rules (friendlyName for SDK-visible names, composition-over-inheritance) block gen-api; stylistic rules (casing, docs, field-prefix) warn so they guide without blocking developer flow.

## Example: Adding a new domain resource to the v3 spec

```
// 1. src/widgets/operations.tsp
import "@typespec/http";
using TypeSpec.Http;
interface WidgetOperations {
  @get @doc("List widgets.") @operationId("listWidgets")
  list(@query namespace: string): Widget[] | OpenMeterError;
}
// 2. src/widgets/index.tsp
import "./operations.tsp";
// 3. src/openmeter.tsp — import the barrel, bind route+tag, add @tagMetadata:
import "../widgets/index.tsp";
interface WidgetRoutes extends WidgetOperations { @route("/api/v1/widgets") @tag("Widgets") }
```

<!-- archie:ai-end -->
