# aip

<!-- archie:ai-start -->

> Root package for the v3 AIP TypeSpec spec: compiles domain operation interfaces into api/v3/openapi.yaml via two root namespace files (openmeter.tsp, konnect.tsp) plus post-processing scripts, and enforces naming/structural constraints via a custom linter in lib/. All route binding, tag metadata, and security schemes live here; domain model definitions do not.

## Patterns

**Two-step compile pipeline** — tsp compile produces OpenAPI YAML, then scripts/flatten-allof.mjs normalises allOf schemas and seal-object-schemas.mjs closes object schemas. Both run via `pnpm generate` (package.json). Never hand-edit the output. (`pnpm generate runs: tsp compile --config tspconfig.yaml ./src && node ./scripts/flatten-allof.mjs ... && node ./scripts/seal-object-schemas.mjs ...`)
**Route and tag binding at root only** — @route and @tag are declared exclusively in src/openmeter.tsp and src/konnect.tsp, never inside domain sub-folder operation files. Adding @route inside a domain operations.tsp silently duplicates routes. (`src/openmeter.tsp: interface MeterRoutes extends MeterOperations { @route("/api/v1/meters") @tag("Meters") ... }`)
**Domain ops imported via index.tsp barrels** — Each domain sub-folder exposes an index.tsp barrel; root files import only the barrel, not individual operation files. This prevents accidental double-imports. (`src/openmeter.tsp: import "../meters/index.tsp";`)
**Security via @useRef to external YAML** — Security schemes are TypeSpec model stubs with @useRef pointing to external YAML fragments in common/definitions/, not inline credential definitions. (`@useRef("../common/definitions/security.yaml") model CloudTokenAuth {}`)
**Custom linter via createRule factory** — All lint rules in lib/rules/ use the TypeSpec createRule factory and are registered in lib/index.js $linter export. Error severity blocks gen-api; warning does not. New rules have zero effect until registered in lib/index.js. (`lib/rules/casing.js: export const casingRule = createRule({ name: 'casing', severity: 'warning', ... }); registered in lib/index.js rules array.`)
**omit-unreachable-types: true in tspconfig** — tspconfig.yaml sets omit-unreachable-types: true so only reachable schemas appear in emitted OpenAPI. Also sets seal-object-schemas: true. Both must remain set. (`tspconfig.yaml options: '@typespec/openapi3': omit-unreachable-types: true; seal-object-schemas: true`)
**PascalCase acronym exceptions in utils.js** — Accepted uppercase acronyms (OAuth2, URL, API, UI, ID) live exclusively in lib/rules/utils.js pascalCaseExceptions. Never add exceptions inline in individual rule files. (`lib/rules/utils.js: const pascalCaseExceptions = ['OAuth2', 'URL', 'API', 'UI', 'ID'];`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `src/main.tsp` | Compilation entry point; imports openmeter.tsp and konnect.tsp to produce two distinct OpenAPI outputs. | Missing import here silently drops a namespace from all output artifacts. |
| `src/openmeter.tsp` | Binds all domain interfaces to @route paths and @tag metadata for the OpenMeter self-hosted variant. | Adding a new domain import without a matching @tagMetadata declaration silently omits the tag description from generated OpenAPI. |
| `src/konnect.tsp` | Konnect variant root; overrides security schemes and some sub-routes relative to openmeter.tsp. | Inline security scheme bodies here duplicate config; always use @useRef to common/definitions/security.yaml instead. |
| `scripts/flatten-allof.mjs` | Post-processes emitted YAML to move sibling properties into allOf branches so SDK generators can consume them. | Removing the x-flatten-allOf idempotency marker check causes infinite re-run loops; changing YAML_OPTIONS.lineWidth from 0 creates spurious line-wrap diffs in committed openapi.yaml. |
| `scripts/seal-object-schemas.mjs` | Rewrites additionalProperties: {not:{}} to false to preserve closed-schema semantics after TypeSpec compilation. | Do not remove — without this step object schemas are not properly sealed and SDK generators may generate overly permissive types. |
| `lib/index.js` | Registers all linter rules via defineLinter/$linter export consumed by TypeSpec compiler during gen-api. | A new rule file has zero effect until its export is added to the rules array here. |
| `lib/rules/utils.js` | Centralised casing helpers and PascalCase acronym exception list shared across all lint rules. | Duplicating regex or exception lists in individual rule files causes divergence from the canonical set. |
| `tspconfig.yaml` | Compiler config: emitter output dir, omit-unreachable-types, seal-object-schemas, linter extends. | omit-unreachable-types and seal-object-schemas must stay true; linter extends must reference @openmeter/api-spec-aip/all. |

## Anti-Patterns

- Declaring @route or @tag inside a domain sub-folder's operations.tsp — routing is bound only in src/openmeter.tsp and src/konnect.tsp
- Hand-editing api/v3/openapi.yaml or api/v3/api.gen.go — always regenerate via `make gen-api` then `make generate`
- Defining a lint rule without registering it in lib/index.js — the rule is compiled but has zero effect at runtime
- Adding new accepted PascalCase acronyms anywhere other than pascalCaseExceptions in lib/rules/utils.js
- Adding a domain import in src/openmeter.tsp without a matching @tagMetadata declaration — tag description is silently absent from generated OpenAPI

## Decisions

- **Two root namespace files (openmeter.tsp, konnect.tsp) instead of one parameterised file** — The two variants differ in security schemes and some sub-route overrides; TypeSpec has no preprocessor so conditional logic would require complex emitter workarounds. Separate files keep the variants explicit.
- **Post-processing scripts run after tsp compile, not as TypeSpec emitter plugins** — TypeSpec emitter produces allOf schemas with sibling properties that confuse SDK generators, and additionalProperties: {not:{}} that need rewriting. These are output-level fixups that cannot be done inside the TypeSpec compiler without patching upstream.
- **Error vs warning severity split in the linter** — Only structurally breaking rules (friendlyName for SDK-visible names, composition-over-inheritance) use error severity to block gen-api. Stylistic rules (casing, docs, field-prefix) use warning to surface guidance without blocking developer flow.

## Example: Adding a new domain resource to the v3 spec

```
// 1. Create domain ops: src/widgets/operations.tsp
import "@typespec/http";
using TypeSpec.Http;

interface WidgetOperations {
  @get
  @doc("List widgets.")
  @operationId("listWidgets")
  list(@query namespace: string): Widget[] | OpenMeterError;
}

// 2. Create barrel: src/widgets/index.tsp
import "./operations.tsp";

// 3. In src/openmeter.tsp — import barrel, bind route+tag, add @tagMetadata:
// ...
```

<!-- archie:ai-end -->
