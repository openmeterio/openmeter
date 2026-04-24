# aip

<!-- archie:ai-start -->

> Root package for the v3 AIP TypeSpec spec: compiles domain operation interfaces into api/v3/openapi.yaml via two root namespace files (openmeter.tsp, konnect.tsp), runs post-processing (flatten-allof.mjs), and enforces naming/structural constraints via a custom linter (lib/). All route binding, tag metadata, and security schemes live here; domain definitions do not.

## Patterns

**Two-step compile pipeline** — tsp compile produces OpenAPI YAML, then scripts/flatten-allof.mjs normalises allOf schemas. Both steps run via `pnpm compile` (package.json). Never hand-edit the output. (`pnpm compile in api/spec/packages/aip runs both steps sequentially`)
**Route and tag binding at root only** — @route and @tag are declared exclusively in src/openmeter.tsp and src/konnect.tsp, never inside domain sub-folder operation files. (`openmeter.tsp: interface MeterRoutes extends MeterOperations { @route("/api/v1/meters") @tag("Meters") ... }`)
**Security via @useRef to external YAML** — Security schemes are TypeSpec model stubs with @useRef pointing to external YAML fragments, not inline credential definitions. (`@useRef("../common/definitions/security.yaml") model CloudTokenAuth {}`)
**Domain ops imported via index.tsp barrels** — Each domain sub-folder exposes an index.tsp; root files import only the barrel, not individual operation files. (`import "../meters/index.tsp";`)
**Custom linter via createRule factory** — All lint rules in lib/rules/ use the createRule factory and are registered in lib/index.js $linter. Error severity blocks gen-api; warning does not. (`lib/rules/casing.js exports createRule({ name: 'casing', ... }); registered in lib/index.js`)
**omit-unreachable-types: true in tspconfig** — tspconfig.yaml sets omit-unreachable-types: true so only reachable schemas appear in the emitted OpenAPI. (`options: '@typespec/openapi3': omit-unreachable-types: true`)
**PascalCase acronym exceptions in utils.js** — Accepted uppercase acronyms (OAuth2, URL, API, UI, ID) live exclusively in lib/rules/utils.js pascalCaseExceptions; never add them inline in rule files. (`const pascalCaseExceptions = ['OAuth2', 'URL', 'API', 'UI', 'ID'];`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `src/main.tsp` | Compilation entry point; imports openmeter.tsp and konnect.tsp | Missing import here silently drops a namespace from output |
| `src/openmeter.tsp` | Binds all domain interfaces to routes and tags for the OpenMeter variant | Adding @tagMetadata for a new domain import is mandatory or the tag description is absent from generated OpenAPI |
| `src/konnect.tsp` | Konnect variant root; overrides security and some sub-routes | Inline security scheme bodies here duplicate config; use @useRef instead |
| `scripts/flatten-allof.mjs` | Post-processes emitted YAML to move sibling properties into allOf branches | Removing x-flatten-allOf marker check causes infinite re-run loops; changing YAML_OPTIONS.lineWidth from 0 creates spurious diffs |
| `lib/index.js` | Registers all linter rules via defineLinter/$linter export | A new rule file has zero effect until its export is added to the rules array here |
| `lib/rules/utils.js` | Centralised casing helpers and acronym exception list | Duplicating regex or exceptions in individual rule files causes divergence |
| `tspconfig.yaml` | Compiler config: emitter output dir, omit-unreachable-types, linter extends | omit-unreachable-types must stay true; linter extends must reference @openmeter/api-spec-aip/all |

## Anti-Patterns

- Declaring @route or @tag inside a domain sub-folder's operations.tsp — routing is bound only in openmeter.tsp and konnect.tsp
- Hand-editing api/v3/openapi.yaml — always regenerate via `make gen-api`
- Defining a lint rule without registering it in lib/index.js — the rule has no effect
- Adding new accepted PascalCase acronyms anywhere other than pascalCaseExceptions in lib/rules/utils.js
- Adding a domain import in openmeter.tsp without a matching @tagMetadata declaration

## Decisions

- **Two root namespace files (openmeter.tsp, konnect.tsp) instead of one parameterised file** — The two variants differ in security schemes and some sub-route overrides; keeping them as separate files avoids conditional logic in TypeSpec which has no preprocessor
- **Post-processing flatten-allof.mjs runs after tsp compile** — TypeSpec emitter produces allOf schemas with sibling properties that confuse SDK generators; normalisation must happen outside the compiler to avoid patching the upstream emitter
- **Custom linter with error vs warning severity split** — Only friendlyName (SDK-visible names) and structurally breaking rules use error severity to block gen-api; stylistic rules use warning to surface guidance without blocking developer flow

## Example: Adding a new domain to the v3 spec

```
// 1. Create domain ops file: src/widgets/operations.tsp
import "@typespec/http";
using TypeSpec.Http;

interface WidgetOperations {
  @get
  @doc("List widgets")
  @operationId("listWidgets")
  list(@query namespace: string): Widget[] | OpenMeterError;
}

// 2. Create barrel: src/widgets/index.tsp
import "./operations.tsp";

// 3. In src/openmeter.tsp — import barrel, bind route+tag, add tagMetadata:
// ...
```

<!-- archie:ai-end -->
