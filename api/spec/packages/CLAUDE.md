# packages

<!-- archie:ai-start -->

> Organisational root for the TypeSpec source that is the single source of truth for the entire OpenMeter HTTP surface, split into two independently compiled packages: aip/ owns the v3 (AIP-style) API (-> api/v3/openapi.yaml + Konnect variant) and legacy/ owns the v1/v2 + Cloud API (-> api/openapi.yaml, api/openapi.cloud.yaml). All OpenAPI YAMLs and Go/JS/Python SDKs are downstream artefacts regenerated exclusively by make gen-api — never edited directly.

## Patterns

**Two-package version split (aip vs legacy)** — v3/AIP changes live in aip/; v1/v2 + Cloud changes live in legacy/. The two compile to separate output files with different structural rules, so content must never cross between them. (`aip/src/openmeter.tsp (v3 routes); legacy/src/main.tsp (v1 entry point)`)
**make gen-api is the only regeneration path** — Both packages are consumed by make gen-api (tsp compile + post-processing); partial runs or manual edits to the emitted YAMLs cause silent drift between spec, stubs, and SDKs. (`make gen-api  # regenerates all OpenAPI YAMLs + SDKs from both packages`)
**Route and tag binding only at root namespace files** — @route and @tag are bound only in each package's root composition files (aip: openmeter.tsp/konnect.tsp; legacy: src/main.tsp or cloud/main.tsp). Domain sub-folder .tsp files declare operations without routing. (`aip/src/openmeter.tsp: interface BillingRoutes extends billing.Routes {}`)
**Per-package custom linters registered in lib/index.js** — Each package ships createRule-based linter rules (e.g. pascalCase, friendlyName, operationId) that block gen-api on violation; a rule not exported from lib/index.js has zero runtime effect. (`aip/lib/index.js: export { pascalCaseRule, friendlyNameRule }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `CLAUDE.md` | The only direct file at this level; documents the aip-vs-legacy split, gen-api regeneration contract, and root-only routing convention. | Treating this as live config — it is descriptive intent context, not a compiled artefact. |
| `aip/` | v3 (AIP) package: domain index.tsp barrels imported into root openmeter.tsp/konnect.tsp, two-step compile with flatten-allof.mjs / seal-object-schemas.mjs post-processing. | omit-unreachable-types: true silently drops schemas not reachable from a declared operation; new domain import needs a matching @tagMetadata. |
| `legacy/` | v1/v2 + Cloud package with dual entry points (src/main.tsp for Python SDK, src/cloud/main.tsp for Cloud OpenAPI) and shared primitives in src/types.tsp. | A new sub-domain file must be registered in BOTH src/main.tsp and src/cloud/main.tsp or it is excluded from the cloud variant. |

## Anti-Patterns

- Hand-editing api/openapi.yaml, api/openapi.cloud.yaml, or api/v3/openapi.yaml — always regenerate via make gen-api
- Declaring @route or @tag inside a domain sub-folder operations.tsp instead of the root namespace files
- Adding v3 content into legacy/ or v1/v2 content into aip/ — they compile to separate targets and mix-ins break both
- Adding a legacy/ sub-domain file without registering it in both src/main.tsp and src/cloud/main.tsp
- Re-declaring primitive types (ULID, DateTime, Key, Resource) in sub-domain files — they live in legacy/src/types.tsp

## Decisions

- **Two separate packages (aip/ and legacy/) with independent tspconfig files and entry points** — v1 and v3 follow different structural rules (REST-CRUD vs AIP), emit to different output files, and use different SDK generators; one package would force conditional compilation and cross-version coupling.
- **Post-processing (flatten-allof.mjs) in aip/ rather than altering TypeSpec source** — TypeSpec's allOf output is correct but incompatible with some SDK generators; post-processing keeps the source idiomatic while producing generator-friendly YAML.
- **Custom per-package linters instead of relying solely on the upstream TypeSpec linter** — Project conventions (PascalCase acronym exceptions, @friendlyName, @operationId enforcement) cannot be expressed upstream; custom error-severity rules block gen-api on violations.

## Example: Binding a domain's operations to a route at the v3 root (only valid place for @route)

```
// aip/src/openmeter.tsp
import "./billing";  // barrel: billing/index.tsp

@tagMetadata("Billing", { description: "Billing operations" })
@route("/api/v3/billing")
interface BillingRoutes extends billing.Routes {}
```

<!-- archie:ai-end -->
