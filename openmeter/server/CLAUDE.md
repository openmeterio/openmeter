# server

<!-- archie:ai-start -->

> Chi-based HTTP server package that assembles the v1 and v3 REST APIs behind a shared middleware stack (auth, OpenAPI validation, CORS, logging). The server/ root owns the Chi router construction, middleware wiring, and dual-version route registration; the router/ sub-package is the pure v1 endpoint delegation layer.

## Patterns

**Dual-version route registration in NewServer** — NewServer mounts the v3 API via v3server.NewServer + RegisterRoutes first in its own Chi Group, then the v1 API with api.HandlerWithOptions in a separate Chi Group so each version has its own middleware chain. (`v3API.RegisterRoutes(r); api.HandlerWithOptions(impl, api.ChiServerOptions{BaseRouter: r, Middlewares: middlewares})`)
**Config aggregates every service reference** — server.Config embeds router.Config which holds every domain service interface; all fields are validated via Config.Validate() before the router is created. (`config.RouterConfig.BillingService, config.RouterConfig.CustomerService, ...`)
**StaticNamespaceDecoder injected universally** — namespacedriver.StaticNamespaceDecoder(defaultNS) is injected into both v1 router.Config and v3server.Config — never resolve namespace inside a handler. (`NamespaceDecoder: namespacedriver.StaticNamespaceDecoder(config.RouterConfig.NamespaceManager.GetDefaultNamespace())`)
**RFC 7807 error responses via models.NewStatusProblem** — All non-handler errors (404, 405, param decode failures) call models.NewStatusProblem(ctx, err, status).Respond(w); the errorHandlerReply function maps oapi-codegen error types to status codes. (`models.NewStatusProblem(r.Context(), nil, http.StatusNotFound).Respond(w)`)
**OapiRequestValidatorWithOptions with NoopAuthenticationFunc** — OpenAPI schema validation runs with NoopAuthenticationFunc so the validator never rejects auth-related requests; actual auth is handled by the authenticator middleware. (`oapimiddleware.OapiRequestValidatorWithOptions(swagger, &Options{Options: openapi3filter.Options{AuthenticationFunc: openapi3filter.NoopAuthenticationFunc}})`)
**corsHandler with AllowedPaths filter** — CORS is gated to specific path prefixes using corsOptions.AllowedPaths; requests to other paths skip CORS middleware entirely. (`corsHandler(corsOptions{AllowedPaths: []string{"/api/v1/portal/meters"}, Options: cors.Options{...}})`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `openmeter/server/server.go` | NewServer: constructs the Chi router, wires middleware stacks for v1 and v3, and mounts both API versions. | Adding business logic or domain service calls directly here; middleware order matters — auth must precede OapiRequestValidator. |
| `openmeter/server/cors.go` | corsHandler with path-prefix filtering; only routes matching AllowedPaths receive CORS headers. | Setting AllowedPaths to nil/empty enables CORS for all paths — intentional for portal use case but dangerous elsewhere. |
| `openmeter/server/server_test.go` | Integration smoke test for all v1 and v3 routes using an in-memory test server constructed from mock services. | All new endpoints must have a corresponding test case here to verify routing and basic response shapes. |
| `openmeter/server/framework_test.go` | Unit tests for httptransport error encoding (ValidationIssue -> HTTP status mapping). | Any change to commonhttp.HandleIssueIfHTTPStatusKnown must be reflected here. |

## Anti-Patterns

- Adding business logic, DB calls, or domain service calls directly in server.go — all logic belongs in domain httpdriver packages
- Registering v3 routes inside the v1 Chi Group — each version must have its own group and middleware chain
- Skipping models.NewStatusProblem for error responses — all errors must render as application/problem+json
- Adding middleware that runs before the authenticator for auth-sensitive paths without updating the PostAuthMiddlewares extension point
- Hand-editing api/api.gen.go or api/v3/api.gen.go to add routes — always regenerate from TypeSpec via make gen-api

## Decisions

- **v3 server mounted in a separate Chi Group before the v1 group** — Each API version needs its own middleware chain (v3 uses oasmiddleware.ValidateRequest; v1 uses kin-openapi OapiRequestValidatorWithOptions); sharing a group would mix validator instances.
- **ExcludeReadOnlyValidations: true in OpenAPI filter options** — Go models translate required+readOnly fields to non-nil zero values; excluding read-only validation prevents false rejections on fields that SHOULD NOT appear in requests per the spec.

## Example: Adding a new v1 endpoint delegation in the router sub-package (after gen-api + generate)

```
// In openmeter/server/router/<domain>.go
func (a *Router) ListFoos(w http.ResponseWriter, r *http.Request, params api.ListFoosParams) {
	a.config.FooHandler.With(
		a.config.NamespaceDecoder,
		a.config.ErrorHandler,
	).ListFoos().ServeHTTP(w, r)
}
```

<!-- archie:ai-end -->
