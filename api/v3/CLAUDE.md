# v3

<!-- archie:ai-start -->

> The v3 HTTP API layer: generated types and server stubs (api.gen.go via oapi-codegen from openapi.yaml), sub-packages for error handling, filtering, label conversion, request/response parsing, OAS validation middleware, and response rendering. Its primary constraint is that all business logic lives in handlers/* sub-packages — api/v3 itself is purely infrastructure and generated code.

## Patterns

**Generated stub is the sole ServerInterface source** — api.gen.go is generated from openapi.yaml via codegen.yaml; the //go:generate directive in api.go drives regeneration. Never edit api.gen.go directly — all changes start in api/spec/. (`//go:generate go tool github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen --config=codegen.yaml ./openapi.yaml`)
**nullable-type: true for optional fields** — codegen.yaml sets nullable-type: true, so optional response fields use nullable.Nullable[T] (from oapi-codegen/nullable), not *T. Use nullable.NewNullableWithValue / nullable.NewNullNullable in handlers. (`Next nullable.Nullable[string] `json:"next"``)
**always-prefix-enum-values** — codegen.yaml sets always-prefix-enum-values: true, so all enum constants are prefixed with the type name (e.g. BillingAppTypeSandbox, not just Sandbox). Always use the fully prefixed constant in handler code. (`BillingAppTypeSandbox BillingAppType = "sandbox"`)
**deepObject filter params routed through filters.Parse** — The chi-middleware.tmpl patch ensures filter[field][op]=value params call filters.Parse instead of runtime.BindQueryParameterWithOptions. Any new filter parameter with style: deepObject and name 'filter' gets this automatically. (`// From codegen.yaml: user-templates: chi/chi-middleware.tmpl: ./templates/chi-middleware.tmpl.patch`)
**All error responses via apierrors package** — v3 handlers must use apierrors named constructors (apierrors.NewBadRequestProblemResponse, etc.) and route through GenericErrorEncoder. Direct w.WriteHeader or http.Error calls are forbidden. (`return apierrors.NewBadRequestProblemResponse(ctx, err)`)
**OAS request validation via oasmiddleware.ValidateRequest** — The Chi router in api/v3/server mounts oasmiddleware.ValidateRequest before the generated server handler. This is spec-backed; it must be built once at startup (NewValidationRouter is expensive). (`router.Use(oasmiddleware.ValidateRequest(spec, hooks...))`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `api/v3/api.gen.go` | Generated Chi server stubs, all request/response types, enum constants with Valid() methods, and the embedded OpenAPI spec. DO NOT EDIT. | Enum constants use TypeName+ValueName prefix (e.g. BillingAppTypeSandbox). nullable.Nullable[T] used for optional fields, not *T. |
| `api/v3/api.go` | Single //go:generate directive that regenerates api.gen.go. Touch only to update generator flags. | Running make gen-api triggers this; never regenerate manually with go generate in isolation. |
| `api/v3/codegen.yaml` | oapi-codegen config: enables chi-server, models, embedded-spec; sets nullable-type, always-prefix-enum-values, custom chi-middleware template. | user-templates override injects filter parsing patch; changing the template path breaks deepObject filter param handling. |
| `api/v3/openapi.yaml` | Generated OpenAPI spec compiled from TypeSpec in api/spec/. Source of truth for all v3 routes, schemas, and parameters. DO NOT EDIT directly. | filter parameters with style: deepObject are special-cased by the template patch. |

## Anti-Patterns

- Editing api.gen.go or openapi.yaml directly — both are generated; edit api/spec/ then run make gen-api
- Bypassing the apierrors package for error responses — using http.Error or w.WriteHeader directly breaks RFC 7807 compliance
- Building a new oasmiddleware validation router per request — it must be constructed once at startup
- Using *string instead of nullable.Nullable[string] for optional JSON fields — produces incorrect null vs absent distinction
- Adding business logic inside api/v3 (outside handlers/) — api/v3 is infrastructure only; all domain calls belong in api/v3/handlers/* sub-packages

## Decisions

- **oapi-codegen with chi-server generates the ServerInterface and Chi routing from openapi.yaml** — Single TypeSpec → OpenAPI → Go stub pipeline eliminates hand-rolled routing drift; all routes are verifiable against the spec at compile time via var _ api.ServerInterface = (*Server)(nil).
- **Custom chi-middleware.tmpl patch for deepObject filter params** — Default oapi-codegen runtime cannot parse union filter[field][op]=value params; a targeted template patch routes only ParamName=filter + Style=deepObject through filters.Parse without affecting standard params.
- **nullable-type: true and always-prefix-enum-values in codegen.yaml** — nullable.Nullable[T] expresses explicit null vs absent in JSON; prefixed enum constants prevent name collisions across the large generated type surface.

## Example: Implement a new v3 list endpoint handler that returns a page-paginated response

```
// api/v3/handlers/foo/handler.go
package foo

import (
    "context"
    "net/http"

    v3 "github.com/openmeterio/openmeter/api/v3"
    "github.com/openmeterio/openmeter/api/v3/apierrors"
    "github.com/openmeterio/openmeter/api/v3/render"
    "github.com/openmeterio/openmeter/api/v3/response"
    "github.com/openmeterio/openmeter/pkg/framework/transport/httptransport"
)

type ListFoosHandler httptransport.HandlerWithArgs[ListFoosRequest, *v3.FooPagePaginatedResponse, v3.ListFoosParams]
// ...
```

<!-- archie:ai-end -->
