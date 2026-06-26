# Enforcement: codegen (4 rules)

Topic file. Loaded on demand when an agent works on something in the `codegen` area. The pre-edit hook reads `.archie/rules.json` directly — this file is for browsing/context only.

## Decision Violations (block)

### `dec-codegen-001` — Author API changes in TypeSpec and regenerate; never hand-edit generated OpenAPI, server stubs, or SDKs

*source: `deep_scan`*

**Why:** The API is authored once in TypeSpec under api/spec (legacy + aip packages); make gen-api compiles it to OpenAPI, then oapi-codegen generates api/api.gen.go and api/v3/api.gen.go, orval the JS SDK, poetry the Python SDK, and goverter/goderive the converters. Two server surfaces and three published SDKs cannot be kept consistent by hand. Editing api/openapi.yaml, api.gen.go, convert.gen.go, or SDK client methods directly breaks the single source of truth and is overwritten on the next generate.

## Mechanical Violations (block)

### `mech-gen-001` — Never hand-edit files carrying a `// Code generated ... DO NOT EDIT.` header

*source: `deep_scan`* · *check: `forbidden_content`*

**Why:** openmeter/ent/db/, **/wire_gen.go, **/convert.gen.go, billing/derived.gen.go, api/api.gen.go, api/v3/api.gen.go, and api/client/go/client.gen.go are all generated and carry the canonical DO-NOT-EDIT header. Hand-edits are silently overwritten by make generate / make gen-api, and CI fails if generated artifacts drift from source.

**Path glob:** `**/*.gen.go`, `**/wire_gen.go`, `openmeter/ent/db/**`, `api/api.gen.go`, `api/v3/api.gen.go`, `api/client/go/client.gen.go`

### `name-gen-file-001` — Generated Go files use *.gen.go / wire_gen.go names and carry the DO-NOT-EDIT header

*source: `deep_scan`*

**Why:** All generated Go carries the canonical `// Code generated ... DO NOT EDIT` header and a *.gen.go (or wire_gen.go / ent/db/) name (api/api.gen.go, openmeter/billing/derived.gen.go, openmeter/billing/service/convert.gen.go). Confirmed across api/ and billing/.

## Pattern Divergence (inform)

### `infra-gendrift-001` — Run make generate-all and commit before pushing so generated artifacts do not drift

*source: `deep_scan`*

**Why:** CI fails if `make update-openapi`, `make generate-javascript-sdk`, or `go generate ./...` produce any git diff or untracked files. The chi-middleware oapi-codegen template is patched via make patch-oapi-templates (run automatically by make generate). Run make generate-all and commit before pushing.
