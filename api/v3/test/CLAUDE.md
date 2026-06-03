# test

<!-- archie:ai-start -->

> Integration test fixtures and tests for the v3 API filter parsing and OAS validation pipeline. Provides an embedded minimal OpenAPI spec (openapi.test.yaml) and end-to-end tests that spin up a real Chi+oasmiddleware server to verify every filter type is correctly validated then parsed by filters.Parse.

## Patterns

**Embedded test spec via //go:embed** — openapi.test.yaml is embedded as []byte via embed.go (OpenAPITestSpec) and loaded through openapi3.NewLoader().LoadFromData. Tests never read the file from disk at runtime. (`//go:embed openapi.test.yaml
var OpenAPITestSpec []byte`)
**newTestServer bootstraps the full validation stack** — Test helpers build a real Chi router with oasmiddleware.ValidateRequest and httptest.NewServer so tests exercise OAS schema validation before reaching the handler. (`srv := newTestServer(t, noopHandler) // Chi + oasmiddleware over the embedded spec`)
**Two-phase test structure: validation then parse** — Validation tests assert OAS acceptance/rejection (204/400) with noopHandler; parse tests additionally run filters.Parse and assert the resulting typed Go struct with parseHandler. Both must pass for a valid filter. (`// TestFieldFilterValidation -> 204/400; TestFieldFilterParse -> typed struct`)
**t.Context() for all test contexts** — Every context value uses t.Context() tied to the test lifecycle, never context.Background(). (`validationRouter, err := oasmiddleware.NewValidationRouter(t.Context(), doc, ...)`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `embed.go` | Embeds openapi.test.yaml as OpenAPITestSpec []byte. Package name is 'test'. | filters_test.go uses package 'test_test' to test as an external consumer; embed.go itself stays in package 'test'. |
| `openapi.test.yaml` | Minimal OpenAPI 3.0 spec defining the FieldFilters schema with every supported filter type (BooleanFieldFilter, NumericFieldFilter, StringFieldFilter, etc.) mapped via x-go-type to the api/v3/filters package. | x-go-type-import/x-go-type annotations drive oapi-codegen output; a new filter type here needs a matching type in api/v3/filters. |
| `filters_test.go` | End-to-end filter validation and parse tests covering all filter types and operators in both short form (filter[field]=value) and object form (filter[field][op]=value). | ULID/DateTime pattern/format violations surface as anyOf-collapse errors (rule: 'anyOf'), not per-branch errors, due to kin-openapi behaviour; a bare scalar (non-map) Labels value is rejected at the validator level. |

## Anti-Patterns

- Reading openapi.test.yaml from the filesystem at test time instead of using the embedded OpenAPITestSpec
- Using context.Background() in tests instead of t.Context()
- Adding production API endpoints to openapi.test.yaml — it is a minimal fixture for filter type testing only

## Decisions

- **Separate minimal test spec (openapi.test.yaml) rather than reusing the production spec** — Isolates filter-type validation from the full API surface, making the spec faster to load and the tests focused on filter behaviour only.

<!-- archie:ai-end -->
