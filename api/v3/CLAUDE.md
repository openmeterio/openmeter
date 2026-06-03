# v3

<!-- archie:ai-start -->

> Infrastructure and generated-code layer for the v3 AIP-style HTTP API. api.gen.go is the sole ServerInterface source, generated from openapi.yaml via oapi-codegen (DO NOT EDIT); all business logic lives in sub-packages. Primary constraint: api/v3 itself is pure infrastructure — no domain calls, no business logic at this level.

## Patterns

**Generated stub is the sole ServerInterface source** — api.gen.go is generated from openapi.yaml via the //go:generate directive in api.go. Never hand-edit; all changes start in api/spec/ then run make gen-api. (`//go:generate go tool github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen --config=codegen.yaml ./openapi.yaml`)
**nullable-type for optional fields** — codegen.yaml sets nullable-type: true, so optional fields use nullable.Nullable[T] from oapi-codegen/nullable, never *T. Use NewNullableWithValue / NewNullNullable. (`Next nullable.Nullable[string] `json:"next"``)
**always-prefix-enum-values** — codegen.yaml sets always-prefix-enum-values: true, so every enum constant is TypeName+ValueName prefixed and has a Valid() method. Always reference the fully prefixed constant. (`BillingAppTypeSandbox BillingAppType = "sandbox"`)
**deepObject filter params routed through filters.Parse** — The custom chi-middleware.tmpl patch makes filter[field][op]=value params call filters.Parse instead of runtime.BindQueryParameterWithOptions. Any param with style:deepObject and name 'filter' gets this automatically. (`user-templates: chi/chi-middleware.tmpl: ./templates/chi-middleware.tmpl`)
**All error responses via apierrors** — v3 handlers use apierrors named constructors and route through GenericErrorEncoder. Direct w.WriteHeader / http.Error is forbidden. (`return apierrors.NewBadRequestProblemResponse(ctx, err)`)
**OAS request validation before the handler** — The Chi router in api/v3/server mounts oasmiddleware.ValidateRequest before the generated server handler. The validation router is built once at startup (NewValidationRouter is expensive). (`router.Use(oasmiddleware.ValidateRequest(spec, hooks...))`)
**Compile-time ServerInterface assertion** — The server keeps var _ api.ServerInterface = (*Server)(nil) so every generated route is verified against an implementation at compile time. (`var _ api.ServerInterface = (*Server)(nil)`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `api/v3/api.gen.go` | Generated Chi server stubs, all request/response types, prefixed enum constants with Valid() methods, and the embedded OpenAPI spec. | DO NOT EDIT. Enum constants use TypeName+ValueName prefix; optional fields use nullable.Nullable[T], not *T. |
| `api/v3/api.go` | Single //go:generate directive that regenerates api.gen.go. | Touch only to update generator flags; regen runs via make gen-api, not standalone go generate. |
| `api/v3/codegen.yaml` | oapi-codegen config: enables chi-server/models/embedded-spec; sets nullable-type, always-prefix-enum-values, and the custom chi-middleware template path. | Changing the user-templates path breaks deepObject filter handling. |
| `api/v3/openapi.yaml` | Generated OpenAPI spec compiled from TypeSpec in api/spec/; source of truth for all v3 routes, schemas, parameters. | DO NOT EDIT directly. filter params with style:deepObject are special-cased by the template patch. |

## Anti-Patterns

- Editing api.gen.go or openapi.yaml directly — both are generated; edit api/spec/ then run make gen-api.
- Bypassing the apierrors package with http.Error or w.WriteHeader — breaks RFC 7807 (application/problem+json) compliance.
- Building a new oasmiddleware validation router per request instead of once at startup.
- Using *string instead of nullable.Nullable[string] for optional JSON fields — produces wrong null-vs-absent distinction.
- Adding business logic at the api/v3 level (outside handlers/) — api/v3 is infrastructure only; all domain calls belong in handlers/* sub-packages.

## Decisions

- **oapi-codegen chi-server generates the ServerInterface and routing from openapi.yaml.** — A single TypeSpec → OpenAPI → Go-stub pipeline eliminates hand-rolled routing drift; routes are verified against the spec at compile time via var _ api.ServerInterface.
- **Custom chi-middleware.tmpl patch for deepObject filter params.** — Default oapi-codegen runtime cannot parse union filter[field][op]=value params; a targeted patch routes only ParamName=filter + Style=deepObject through filters.Parse without affecting standard params.
- **nullable-type: true and always-prefix-enum-values in codegen.yaml.** — nullable.Nullable[T] expresses explicit null vs absent in JSON; prefixed enum constants prevent name collisions across the large generated type surface.

## Example: Declare a new v3 list endpoint handler that returns a page-paginated response

```
// api/v3/handlers/foo/handler.go
package foo

import (
    v3 "github.com/openmeterio/openmeter/api/v3"
    "github.com/openmeterio/openmeter/pkg/framework/transport/httptransport"
)

type ListFoosHandler httptransport.HandlerWithArgs[ListFoosRequest, *v3.FooPagePaginatedResponse, v3.ListFoosParams]
```

<!-- archie:ai-end -->
