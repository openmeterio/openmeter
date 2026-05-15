# httpdriver

<!-- archie:ai-start -->

> HTTP handler layer for the progressmanager domain, implementing a single GetProgress endpoint using the httptransport.Handler pipeline with namespace resolution from request context.

## Patterns

**Handler interface composes sub-interfaces** — The top-level Handler interface embeds ProgressHandler (which exposes GetProgress()). New endpoint groups add a new sub-interface and embed it in Handler. (`type Handler interface { ProgressHandler }
type ProgressHandler interface { GetProgress() GetProgressHandler }`)
**Compile-time interface assertion** — var _ Handler = (*handler)(nil) ensures the concrete struct satisfies the Handler interface at package level. (`var _ Handler = (*handler)(nil)`)
**Namespace resolved via namespacedriver.NamespaceDecoder** — All handlers call h.resolveNamespace(ctx) which reads the namespace from request context via NamespaceDecoder. Returns HTTP 500 if namespace is missing — intentional for self-hosted single-namespace deployments. (`ns, ok := h.namespaceDecoder.GetNamespace(ctx)
if !ok { return "", commonhttp.NewHTTPError(http.StatusInternalServerError, errors.New("internal server error")) }`)
**httptransport.NewHandlerWithArgs for path-parameter endpoints** — Endpoints with URL path arguments use httptransport.NewHandlerWithArgs[Req, Resp, Arg]. The first func decodes (ctx, *http.Request, arg) -> (Req, error); the second func is the operation. (`httptransport.NewHandlerWithArgs(
    func(ctx context.Context, r *http.Request, progressID string) (GetProgressRequest, error) { ... },
    func(ctx context.Context, request GetProgressRequest) (GetProgressResponse, error) { ... },
    commonhttp.JSONResponseEncoderWithStatus[GetProgressResponse](http.StatusOK),
    ...options,
)`)
**Response type aliased from generated api package** — GetProgressResponse is aliased to api.Progress (the generated OpenAPI type), not a local struct. Conversion from domain type is done via a local progressToAPI() pure-mapping helper. (`type GetProgressResponse = api.Progress
func progressToAPI(p entity.Progress) api.Progress { ... }`)
**HandlerOption propagation via httptransport.AppendOptions** — Handler-level options are propagated to each endpoint via httptransport.AppendOptions(h.options, httptransport.WithOperationName("...")). (`httptransport.AppendOptions(h.options, httptransport.WithOperationName("getProgress"))...`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `handler.go` | Defines Handler/ProgressHandler interfaces, handler struct, New() constructor, and resolveNamespace helper. | resolveNamespace returns HTTP 500 (not 401/403) when namespace is missing — this is intentional for StaticNamespaceDecoder in self-hosted deployments. |
| `progress.go` | Implements GetProgress endpoint: decoding, operation (calls service.GetProgress), response encoding, and progressToAPI() conversion helper. | A nil progress after a successful service call is treated as an internal error (not not-found) — the nil guard prevents silent data loss to callers. |

## Anti-Patterns

- Calling the domain service from the decoder function — decode must only extract request fields; the operation func calls the service
- Returning domain entity types directly as HTTP responses — always convert via progressToAPI() using generated api package types
- Hardcoding the namespace string instead of calling h.resolveNamespace(ctx)
- Adding business logic to progressToAPI() — it must be a pure field mapping with no validation or computation

## Decisions

- **GetProgressResponse aliased to api.Progress rather than a new struct** — The generated api.Progress is the contract-stable OpenAPI type; aliasing avoids a redundant intermediate struct and ensures the HTTP response always matches the spec.

## Example: Register the GetProgress handler in the v1 router

```
import (
    "github.com/openmeterio/openmeter/openmeter/progressmanager/httpdriver"
)

handler := httpdriver.New(namespaceDecoder, progressSvc, options...)
router.Get("/api/v1/progress/{progressID}", handler.GetProgress().ServeHTTP)
```

<!-- archie:ai-end -->
