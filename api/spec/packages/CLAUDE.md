# packages

<!-- archie:ai-start -->

> Organisational root for TypeSpec source packages: aip/ owns the v3 AIP spec (compiles to api/v3/openapi.yaml) and legacy/ owns the v1/v2 spec (compiles to api/openapi.yaml and api/openapi.cloud.yaml). All SDKs and Go server stubs are downstream artefacts — never edit them directly.

## Patterns

**Two-package split: aip vs legacy** — v3 domain changes go in aip/; v1/v2 changes go in legacy/. Cross-contamination (e.g. adding a v3 route in legacy/) breaks the separate compile targets and OpenAPI output files. (`aip/src/openmeter.tsp — v3 route binding; legacy/src/main.tsp — v1 entry point`)
**make gen-api is the only regeneration path** — Both packages are consumed by `make gen-api` (tsp compile + flatten-allof post-processing). Partial runs or manual edits to api/openapi.yaml / api/v3/openapi.yaml produce silent drift. (`make gen-api  # regenerates all OpenAPI YAMLs + SDKs from both packages`)
**Route/tag binding at root namespace files only** — In aip/, @route and @tag belong only in openmeter.tsp / konnect.tsp. In legacy/, they belong only in src/main.tsp or cloud/main.tsp. Domain sub-folder .tsp files declare operations without routing. (`aip/src/openmeter.tsp: interface BillingRoutes extends billing.Routes {}`)
**Custom linter rules must be registered in lib/index.js** — Each sub-package has its own lib/index.js that exports createRule-based rules. A rule defined in lib/rules/ but not exported from lib/index.js has no effect. (`aip/lib/index.js: export { pascalCaseRule, friendlyNameRule }`)

## Key Files

| File                            | Role                                                                                             | Watch For                                                                                                                        |
| ------------------------------- | ------------------------------------------------------------------------------------------------ | -------------------------------------------------------------------------------------------------------------------------------- |
| `aip/src/openmeter.tsp`         | v3 composition root: imports domain index barrels, binds @route/@tag, declares @tagMetadata      | Every new domain import in openmeter.tsp needs a matching @tagMetadata declaration; missing it causes undeclared tag lint errors |
| `aip/src/konnect.tsp`           | Second v3 entry point for the Konnect variant; same routing conventions as openmeter.tsp         | New domain ops added to openmeter.tsp may need mirroring in konnect.tsp if they belong in the Konnect surface                    |
| `aip/scripts/flatten-allof.mjs` | Post-compile OpenAPI transformation that flattens allOf schemas for SDK generator compatibility  | Runs after tsp compile; if it fails silently the output YAML contains nested allOf that breaks some SDK generators               |
| `legacy/src/main.tsp`           | v1 composition root: imports all domain sub-folders and registers operations                     | New sub-domain .tsp files must be imported here AND in legacy/src/cloud/main.tsp or they are excluded from the cloud variant     |
| `legacy/src/types.tsp`          | Shared primitive types (ULID, DateTime, Key, Resource base models) for the entire legacy package | Re-declaring these primitives in sub-domain files causes duplicate schema definitions and inconsistent SDK types                 |
| `legacy/src/errors.tsp`         | Defines the OpenMeterError union used on every operation's error branch                          | Using `extends Error` without @error decorator breaks OpenAPI error schema generation                                            |
| `aip/lib/rules/utils.js`        | Shared utilities for aip linter rules including pascalCaseExceptions list                        | New accepted PascalCase acronym exceptions must be added here, not inline in individual rules                                    |
| `aip/tspconfig.yaml`            | TypeSpec compiler config for the v3 package; sets omit-unreachable-types: true and output paths  | omit-unreachable-types: true means types not reachable from any declared operation are silently dropped from the output YAML     |

## Anti-Patterns

- Hand-editing api/openapi.yaml, api/openapi.cloud.yaml, or api/v3/openapi.yaml — always regenerate via `make gen-api`
- Declaring @route or @tag inside domain sub-folder operation files — routing is bound only in the root namespace files
- Adding a new sub-domain file in legacy/ without registering it in both src/main.tsp and src/cloud/main.tsp
- Re-declaring primitive types (ULID, DateTime, Resource) in sub-domain files — they are already in legacy/src/types.tsp
- Adding v3 domain content into legacy/ or v1 content into aip/ — the packages compile to separate output files and mix-ins break both

## Decisions

- **Two separate packages (aip/ and legacy/) with independent tspconfig files and entry points** — v1 and v3 APIs have different structural rules (AIP vs REST-CRUD), different output targets, and different SDK generators; a single package would require conditional compilation logic and increase cross-version coupling
- **Post-processing step (flatten-allof.mjs) in aip/ rather than upstream TypeSpec changes** — TypeSpec's allOf output is structurally correct but incompatible with some SDK generators; post-processing keeps the TypeSpec source idiomatic while producing generator-friendly output
- **Custom per-package linters (lib/index.js + lib/rules/) instead of relying solely on upstream TypeSpec linter** — Project-specific conventions (PascalCase acronym exceptions, @friendlyName requirement, @operationId enforcement) cannot be expressed in upstream linter rules; custom rules block gen-api on violations

<!-- archie:ai-end -->
