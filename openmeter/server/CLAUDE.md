# server

<!-- archie:ai-start -->

> Chi-based HTTP server assembling the v1 and v3 REST APIs behind a shared middleware stack (auth, OpenAPI validation, CORS, logging). server/ root owns Chi router construction, dual-version mounting, and RFC 7807 error mapping; the router/ sub-package is the pure v1 endpoint delegation layer implementing api.ServerInterface.

## Patterns

**Dual-version route registration in NewServer** — NewServer mounts v3 via v3server.NewServer + RegisterRoutes in its own Chi Group first, then v1 via api.HandlerWithOptions in a separate Group so each version has its own middleware chain and validator. (`r.Group(func(r chi.Router){ v3RegisterErr = v3API.RegisterRoutes(r) }); api.HandlerWithOptions(impl, api.ChiServerOptions{BaseRouter: r, Middlewares: middlewares})`)
**router.Config aggregates every domain service; validated before routing** — config.RouterConfig holds ~40 domain service interface fields; router.NewRouter validates them before the server is constructed. v3 wiring reads the same fields from RouterConfig. (`v3server.NewServer(&v3server.Config{BillingService: config.RouterConfig.Billing, ...})`)
**StaticNamespaceDecoder injected universally** — namespacedriver.StaticNamespaceDecoder(defaultNS) is passed into both v1 router.Config and v3server.Config — namespace is never resolved inside a handler. (`NamespaceDecoder: namespacedriver.StaticNamespaceDecoder(config.RouterConfig.NamespaceManager.GetDefaultNamespace())`)
**RFC 7807 error responses via models.NewStatusProblem** — NotFound, MethodNotAllowed, and the errorHandlerReply oapi-codegen error switch all render application/problem+json via models.NewStatusProblem. (`models.NewStatusProblem(r.Context(), nil, http.StatusNotFound).Respond(w)`)
**v1 validator runs with NoopAuthenticationFunc + ExcludeReadOnlyValidations** — OapiRequestValidatorWithOptions uses NoopAuthenticationFunc (auth handled by the authenticator middleware) and ExcludeReadOnlyValidations so read-only zero-values are not rejected. (`openapi3filter.Options{AuthenticationFunc: openapi3filter.NoopAuthenticationFunc, ExcludeReadOnlyValidations: true}`)
**corsHandler with AllowedPaths prefix gate** — CORS applies only to path prefixes in corsOptions.AllowedPaths (e.g. /api/v1/portal/meters); empty AllowedPaths means CORS for all paths. (`corsHandler(corsOptions{AllowedPaths: []string{"/api/v1/portal/meters"}, Options: cors.Options{...}})`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `openmeter/server/server.go` | NewServer: builds the Chi router, wires v1+v3 middleware stacks, mounts both API versions, and defines errorHandlerReply. | Middleware order — authenticator must precede OapiRequestValidator; no business/domain logic here. |
| `openmeter/server/cors.go` | corsHandler with path-prefix filtering. | Empty/nil AllowedPaths enables CORS for all paths — intentional only for the portal use case. |
| `openmeter/server/server_test.go` | TestRoutes smoke test exercising v1 and v3 routes against an in-memory server built from mocks. | New endpoints should add a case here to verify routing and response shape. |
| `openmeter/server/framework_test.go` | Unit test for ValidationIssue -> HTTP status mapping through httptransport/commonhttp. | Changes to commonhttp.HandleIssueIfHTTPStatusKnown must be reflected here. |

## Anti-Patterns

- Adding business logic, DB calls, or domain service calls in server.go — logic belongs in domain httpdriver packages and the router delegation layer
- Registering v3 routes inside the v1 Chi Group — each version needs its own group and middleware chain
- Skipping models.NewStatusProblem for error responses — all errors must render as application/problem+json
- Hand-editing api/api.gen.go or api/v3/api.gen.go to add routes — regenerate from TypeSpec via make gen-api

## Decisions

- **v3 mounted in a separate Chi Group before the v1 group** — Each version needs its own validator (v3 oasmiddleware.ValidateRequest; v1 kin-openapi OapiRequestValidatorWithOptions); a shared group would mix validator instances.
- **ExcludeReadOnlyValidations: true in the v1 OpenAPI filter** — Go models translate required+readOnly fields to non-nil zero values; excluding read-only validation prevents false rejections on fields that should not appear in requests.

<!-- archie:ai-end -->
