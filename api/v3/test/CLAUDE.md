# test

<!-- archie:ai-start -->

> Integration test fixtures and tests for the v3 API filter parsing and OAS validation pipeline. Provides an embedded test OpenAPI spec (openapi.test.yaml) and end-to-end tests that spin up a real Chi+oasmiddleware server to verify every filter type is correctly validated and parsed by filters.Parse.

## Patterns

**Embedded test spec via //go:embed** — openapi.test.yaml is embedded as []byte via embed.go and loaded by tests through openapi3.NewLoader().LoadFromData(apiv3test.OpenAPITestSpec). Tests never read the file from disk at runtime. (`//go:embed openapi.test.yaml
var OpenAPITestSpec []byte`)
**newTestServer helper bootstraps full validation stack** — Test helpers construct a real Chi router with oasmiddleware.ValidateRequest middleware and httptest.NewServer. Tests exercise the full request path including OAS schema validation before reaching the handler. (`srv := newTestServer(t, noopHandler) // spins up Chi+oasmiddleware with embedded spec`)
**Two-phase test structure: validation then parse** — TestFieldFilterValidation asserts OAS schema acceptance/rejection (204/400). TestFieldFilterParse additionally runs filters.Parse and asserts the resulting typed Go struct. Both must pass for a valid filter. (`// noopHandler for validation tests; parseHandler for parse tests`)
**t.Context() for all test contexts** — All context values in tests use t.Context() (tied to test lifecycle), never context.Background(). (`validationRouter, err := oasmiddleware.NewValidationRouter(t.Context(), doc, ...)`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `embed.go` | Embeds openapi.test.yaml as OpenAPITestSpec []byte for use in tests. | Package name is 'test' (not 'test_test'); tests in filters_test.go use package 'test_test' to test as external consumer. |
| `openapi.test.yaml` | Minimal OpenAPI 3.0 spec defining the FieldFilters schema with all supported filter types (BooleanFieldFilter, NumericFieldFilter, StringFieldFilter, etc.) mapped to x-go-type from api/v3/filters package. | x-go-type-import and x-go-type annotations on each schema drive which Go type oapi-codegen generates. Adding a new filter type here requires a matching type in api/v3/filters. |
| `filters_test.go` | End-to-end filter validation and parse tests covering all filter types and operators in both short form (filter[field]=value) and object form (filter[field][op]=value). | ULID and DateTime pattern/format violations surface as anyOf-collapse errors (rule: 'anyOf'), not per-branch errors, due to kin-openapi behaviour. Labels bare scalar (non-map) is rejected at validator level. |

## Anti-Patterns

- Reading openapi.test.yaml from the filesystem at test time — always use the embedded OpenAPITestSpec
- Using context.Background() in tests — use t.Context() throughout
- Adding production API endpoints to openapi.test.yaml — it is a minimal fixture for filter type testing only

## Decisions

- **Separate test spec (openapi.test.yaml) rather than reusing the production spec** — A minimal spec isolates filter type validation tests from the full API surface, making tests faster to load and focused on filter behaviour only.

<!-- archie:ai-end -->
