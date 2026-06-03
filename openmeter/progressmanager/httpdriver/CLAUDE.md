# httpdriver

<!-- archie:ai-start -->

> HTTP handler layer for the progressmanager domain, implementing a single GetProgress endpoint via the httptransport.Handler pipeline with namespace resolution from request context.

## Patterns

**Handler interface composes sub-interfaces** — The top-level Handler embeds ProgressHandler (exposing GetProgress()). New endpoint groups add a new sub-interface and embed it. (`type Handler interface { ProgressHandler }; type ProgressHandler interface { GetProgress() GetProgressHandler }`)
**Compile-time interface assertion** — var _ Handler = (*handler)(nil) ensures the concrete struct satisfies the Handler interface. (`var _ Handler = (*handler)(nil)`)
**Namespace resolved via NamespaceDecoder** — Handlers call h.resolveNamespace(ctx) reading namespace from request context; returns HTTP 500 if missing — intentional for self-hosted single-namespace deployments. (`ns, ok := h.namespaceDecoder.GetNamespace(ctx); if !ok { return "", commonhttp.NewHTTPError(http.StatusInternalServerError, errors.New("internal server error")) }`)
**NewHandlerWithArgs for path-param endpoints** — Endpoints with URL path args use httptransport.NewHandlerWithArgs[Req, Resp, Arg]; the first func decodes (ctx, *http.Request, arg)->(Req, error), the second is the operation. (`httptransport.NewHandlerWithArgs(decodeFn, operationFn, commonhttp.JSONResponseEncoderWithStatus[GetProgressResponse](http.StatusOK), ...options)`)
**Response type aliased from generated api package** — GetProgressResponse is aliased to api.Progress (generated OpenAPI type); conversion from domain is via the pure progressToAPI() helper. (`type GetProgressResponse = api.Progress; func progressToAPI(p entity.Progress) api.Progress { ... }`)
**HandlerOption propagation via AppendOptions** — Handler-level options propagate to each endpoint via httptransport.AppendOptions(h.options, httptransport.WithOperationName("...")). (`httptransport.AppendOptions(h.options, httptransport.WithOperationName("getProgress"))...`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `handler.go` | Handler/ProgressHandler interfaces, handler struct, New() constructor, resolveNamespace helper. | resolveNamespace returns HTTP 500 (not 401/403) when namespace is missing — intentional for StaticNamespaceDecoder in self-hosted deployments. |
| `progress.go` | GetProgress endpoint: decode, operation (calls service.GetProgress), encode, and progressToAPI() conversion. | A nil progress after a successful service call is treated as an internal error (not not-found) — the nil guard prevents silent data loss. |

## Anti-Patterns

- Calling the domain service from the decoder function — decode must only extract request fields
- Returning domain entity types directly as HTTP responses — always convert via progressToAPI() using api types
- Hardcoding the namespace string instead of calling h.resolveNamespace(ctx)
- Adding business logic to progressToAPI() — it must be a pure field mapping

## Decisions

- **GetProgressResponse aliased to api.Progress rather than a new struct** — The generated api.Progress is the contract-stable OpenAPI type; aliasing avoids a redundant intermediate struct and ensures the response matches the spec.

## Example: Register the GetProgress handler in the v1 router

```
handler := httpdriver.New(namespaceDecoder, progressSvc, options...)
router.Get("/api/v1/progress/{progressID}", handler.GetProgress().ServeHTTP)
```

<!-- archie:ai-end -->
