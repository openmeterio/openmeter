# oasmiddleware

<!-- archie:ai-start -->

> Wraps kin-openapi (openapi3filter) into Chi-compatible middleware for v3 request/response validation against the OpenAPI spec, plus the AIP error-mapping layer converting openapi3.MultiError validation failures into apierrors.InvalidParameters.

## Patterns

**ValidateRequest middleware with hook functions** — Mount ValidateRequest(router, ValidateRequestOption{RouteNotFoundHook, RouteValidationErrorHook}) before the generated server handler. OasRouteNotFoundErrorHook returns 404; OasValidationErrorHook converts OAS errors to AIP 400/404 and returns true to stop the request. (`r.Use(oasmiddleware.ValidateRequest(validationRouter, oasmiddleware.ValidateRequestOption{RouteNotFoundHook: oasmiddleware.OasRouteNotFoundErrorHook, RouteValidationErrorHook: func(err error, w http.ResponseWriter, r *http.Request) bool { return oasmiddleware.OasValidationErrorHook(r.Context(), err, w, r) }}))`)
**NewValidationRouter for spec-backed route matching** — Call NewValidationRouter(ctx, doc, &ValidationRouterOpts{DeleteServers: true}) once at startup to build the gorillamux router from the parsed openapi3.T; DeleteServers removes server entries so paths match regardless of host prefix. (`router, err := oasmiddleware.NewValidationRouter(ctx, doc, nil)`)
**ToAipError for OAS -> InvalidParameter conversion** — Convert openapi3.MultiError to []apierrors.InvalidParameter via ToAipError; OAS schema fields map to AIP rule names via oasRuleToAip (minLength->min_length). Path-param errors map to 404, body/query to 400. (`params := oasmiddleware.ToAipError(multiErr); apierrors.NewBadRequestError(ctx, err, params).HandleAPIError(w, r)`)
**SanitizeSensitiveFieldValues scrubs x-sensitive fields** — Before returning an OAS validation error, wrap it with SanitizeSensitiveFieldValues to replace .Value of any schema field marked x-sensitive: true with '********'. (`apierrors.NewBadRequestError(ctx, oasmiddleware.SanitizeSensitiveFieldValues(err), params).HandleAPIError(w, r)`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `validator.go` | ValidateRequest and ValidateResponse Chi middleware factories; hook functions decide whether to short-circuit (return true to stop). | If RouteNotFoundHook returns false the request continues to the handler — 404 hooks must always return true. |
| `hook.go` | OasRouteNotFoundErrorHook (404 on missing route) and OasValidationErrorHook (400/404 on parameter/body violations). | OasValidationErrorHook checks Source==InvalidParamSourcePath to return 404 instead of 400 — critical for REST semantics. |
| `error.go` | aipMapper recursively converts openapi3.MultiError trees to []InvalidParameter; unwrapOriginError handles nested oneOf errors; oasRuleToAip maps rule names. | The recursive aipMapper shares a parent *InvalidParameter across children — mutations inside the loop affect subsequent iterations. |
| `router.go` | NewValidationRouter wraps openapi3.T doc validation and gorillamux router construction; build once at startup and reuse. | DeleteServers mutates doc.Servers in place; strip servers before the first call if the doc is shared. |
| `decoder.go` | JsonBodyDecoder registers json.NewDecoder with UseNumber() for custom vendor content types so numeric fields aren't truncated to float64. | Must be registered via openapi3filter.RegisterBodyDecoder per custom content-type at startup or numeric precision is lost. |

## Anti-Patterns

- Mounting ValidateRequest after the generated Chi server handler — validation must precede the handler
- Using OasValidationErrorHook without SanitizeSensitiveFieldValues — raw OAS errors may leak x-sensitive values
- Building a new validation router per request — build once at startup and reuse
- Ignoring the bool return of hook functions — returning false lets the request proceed past a validation failure

## Decisions

- **Path-parameter OAS validation errors mapped to 404 rather than 400** — AIP semantics: a malformed/non-existent path ID reads as 'resource not found', not 'bad request', matching REST client expectations.

## Example: Wiring OAS request validation middleware in the v3 Chi router

```
import "github.com/openmeterio/openmeter/api/v3/oasmiddleware"

validationRouter, err := oasmiddleware.NewValidationRouter(ctx, doc, nil)
if err != nil { return err }

r.Use(oasmiddleware.ValidateRequest(validationRouter, oasmiddleware.ValidateRequestOption{
    RouteNotFoundHook: oasmiddleware.OasRouteNotFoundErrorHook,
    RouteValidationErrorHook: func(err error, w http.ResponseWriter, r *http.Request) bool {
        return oasmiddleware.OasValidationErrorHook(r.Context(), err, w, r)
    },
    FilterOptions: &openapi3filter.Options{AuthenticationFunc: openapi3filter.NoopAuthenticationFunc},
}))
```

<!-- archie:ai-end -->
