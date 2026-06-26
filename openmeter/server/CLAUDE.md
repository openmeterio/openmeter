# server

<!-- archie:ai-start -->

> HTTP server assembly: NewServer wires the chi router, OpenAPI validation/auth middleware, the v3 API (api/v3/server) and the legacy v1/v2 Router (server/router) into one *Server. It owns transport-level concerns — middleware stacks, CORS, error mapping — never domain logic.

## Patterns

**Two route groups under one chi router** — NewServer registers v3 routes via v3server.NewServer(...).RegisterRoutes in one chi.Group and the legacy api.HandlerWithOptions(impl,...) in another, both fed the same materialized hook middlewares. (`hookMiddlewares := collectMiddlewareHooks(config.RouterHooks.Middlewares)`)
**Hook middlewares materialized once** — MiddlewareHooks are run a single time into a flat slice via collectMiddlewareHooks/middlewareCollector and reused for both groups; running hooks per-group would execute bodies twice. (`v3Middlewares := append([]server.MiddlewareFunc{}, hookMiddlewares...)`)
**OpenAPI-driven validation + auth middleware** — Legacy group uses authenticator.NewAuthenticator(...) + oapimiddleware.OapiRequestValidatorWithOptions(swagger,...) with ExcludeReadOnlyValidations and a NoopAuthenticationFunc. (`swagger.Servers = nil  // skip server-name validation`)
**Typed param-error mapping** — errorHandlerReply switches on api.*ParamError types and maps each to models.NewStatusProblem with the right HTTP status. (`case *api.RequiredParamError: ... http.StatusBadRequest`)
**Default namespace via StaticNamespaceDecoder** — Both the v3 server and the legacy Router resolve namespace from NamespaceManager.GetDefaultNamespace() through namespacedriver.StaticNamespaceDecoder. (`NamespaceDecoder: namespacedriver.StaticNamespaceDecoder(...)`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `server.go` | NewServer: assembles chi router, middleware stacks, v3 + legacy route groups, OpenAPI validation, and error handling. | v3 and legacy groups must receive the same hookMiddlewares slice; re-invoking hooks per group double-runs side effects. Every RouterConfig.* field passed to v3server.Config must be wired by the caller. |
| `cors.go` | corsHandler wraps go-chi/cors to apply CORS only to AllowedPaths prefixes (else pass through). | Empty AllowedPaths applies CORS to ALL paths. |
| `router/` | Implements the generated legacy api.ServerInterface as the *Router passed as impl to api.HandlerWithOptions. | Router methods must match the generated ServerInterface signatures and contain no business logic. |

## Anti-Patterns

- Putting domain logic, DB queries, or validation in server.go — it is transport assembly only.
- Calling RouterHooks.Middlewares hooks separately per route group instead of once via collectMiddlewareHooks.
- Bypassing models.NewStatusProblem / errorHandlerReply when emitting HTTP error responses.
- Leaving swagger.Servers populated (re-enables server-name validation that breaks unknown deployments).

## Decisions

- **Mount v3 and legacy v1/v2 APIs as separate chi groups within one server.** — Each version has its own generated handler set and validation, but must share the same OTEL/logging middleware stack.
- **Materialize MiddlewareHooks once before mounting either group.** — Avoids double-running hook bodies, which is unsafe for hooks with construction side effects.

## Example: Mapping generated OpenAPI param errors to HTTP problems

```
func errorHandlerReply(w http.ResponseWriter, r *http.Request, err error) {
	switch e := err.(type) {
	case *api.RequiredParamError:
		err := fmt.Errorf("required param missing %s: %w", e.ParamName, err)
		models.NewStatusProblem(r.Context(), err, http.StatusBadRequest).Respond(w)
	default:
		err := fmt.Errorf("unhandled server error: %w", err)
		models.NewStatusProblem(r.Context(), err, http.StatusInternalServerError).Respond(w)
	}
}
```

<!-- archie:ai-end -->
