# oasmiddleware

<!-- archie:ai-start -->

> Wraps kin-openapi (openapi3filter) into Chi-compatible middleware for v3 request and response validation against the OpenAPI spec. Also provides the AIP error mapping layer that converts openapi3.MultiError validation failures into apierrors.InvalidParameters.

## Patterns

**ValidateRequest middleware with hook functions** — Instantiate ValidateRequest(router, ValidateRequestOption{RouteNotFoundHook: OasRouteNotFoundErrorHook, RouteValidationErrorHook: adapted hook}) and mount it in the Chi middleware stack before the generated server handler. OasValidationErrorHook converts OAS errors to AIP 400/404 responses. (`r.Use(oasmiddleware.ValidateRequest(validationRouter, oasmiddleware.ValidateRequestOption{
    RouteNotFoundHook: oasmiddleware.OasRouteNotFoundErrorHook,
    RouteValidationErrorHook: func(err error, w http.ResponseWriter, r *http.Request) bool {
        return oasmiddleware.OasValidationErrorHook(r.Context(), err, w, r)
    },
}))`)
**NewValidationRouter for spec-backed route matching** — Call NewValidationRouter(ctx, doc, &ValidationRouterOpts{DeleteServers: true}) to build the kin-openapi gorillamux router from the parsed openapi3.T. DeleteServers=true removes server entries so the router matches paths regardless of host prefix. (`router, err := oasmiddleware.NewValidationRouter(ctx, doc, nil)`)
**ToAipError for OAS → InvalidParameter conversion** — Convert openapi3.MultiError to []apierrors.InvalidParameter using ToAipError. OAS schema fields are mapped to AIP rule names via oasRuleToAip (e.g. 'minLength' → 'min_length'). Path parameter errors map to 404, body/query errors map to 400. (`params := oasmiddleware.ToAipError(multiErr)
apierrors.NewBadRequestError(ctx, err, params).HandleAPIError(w, r)`)
**SanitizeSensitiveFieldValues scrubs x-sensitive fields** — Before passing an OAS validation error to the client, wrap it with SanitizeSensitiveFieldValues to replace the .Value of any schema field marked 'x-sensitive: true' with '********'. (`apierrors.NewBadRequestError(ctx, oasmiddleware.SanitizeSensitiveFieldValues(err), params).HandleAPIError(w, r)`)
**ResponseWriterWrapper for response body capture** — Use NewResponseWriterWrapper(w) in ValidateResponse to buffer the written body for post-handler OAS response validation without breaking the normal write path. (`rww := oasmiddleware.NewResponseWriterWrapper(w)
h.ServeHTTP(rww, r)
// rww.Body() contains the full response body`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `validator.go` | ValidateRequest and ValidateResponse Chi middleware factories. The hook functions receive errors and decide whether to short-circuit the handler. | If RouteNotFoundHook returns false, the request continues to the handler — ensure 404 hooks always return true. |
| `hook.go` | OasRouteNotFoundErrorHook and OasValidationErrorHook: the two standard hooks for path-not-found (404) and validation failure (400/404 depending on parameter source). | OasValidationErrorHook maps path-parameter validation errors to 404 (not 400) — this matches AIP semantics where a bad path ID means resource not found. |
| `error.go` | aipMapper recursively converts openapi3.MultiError trees to []InvalidParameter. unwrapOriginError handles nested oneOf schema errors. | The recursive aipMapper shares a parent *InvalidParameter across children — mutations to parent inside the loop affect subsequent iterations. |
| `router.go` | NewValidationRouter: wraps openapi3.T doc validation and gorillamux router construction. Must be called once at startup. | DeleteServers mutates doc.Servers in place; if the doc is shared across multiple router instantiations, strip servers before the first call. |
| `decoder.go` | JsonBodyDecoder: registers json.NewDecoder with UseNumber() for custom vendor content types so numeric fields are not silently truncated to float64. | Must be registered via openapi3filter.RegisterBodyDecoder for each custom content-type at startup. |

## Anti-Patterns

- Mounting ValidateRequest after the generated Chi server handler — validation must precede the handler in the middleware chain
- Using OasValidationErrorHook without SanitizeSensitiveFieldValues — raw OAS errors may leak x-sensitive field values
- Building a new validation router per request — it is expensive; build once at startup and reuse
- Ignoring the bool return of hook functions — returning false allows the request to proceed past a validation failure

## Decisions

- **Path-parameter OAS validation errors mapped to 404 rather than 400** — AIP semantics: an invalid or non-existent path parameter (e.g. malformed ID) should read as 'resource not found', not 'bad request', matching client expectations for REST resource paths.

## Example: Wiring OAS request validation middleware in the v3 Chi router

```
import (
    "github.com/openmeterio/openmeter/api/v3/oasmiddleware"
)

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
