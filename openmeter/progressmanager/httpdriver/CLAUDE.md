# httpdriver

<!-- archie:ai-start -->

> HTTP handler layer for the progressmanager domain, implementing a single GetProgress endpoint. Follows the httptransport.Handler[Req, Resp] pattern with namespace resolution from context.

## Patterns

**Handler interface composes sub-interfaces** — The top-level Handler interface embeds ProgressHandler (which exposes GetProgress()). New types of endpoints add a new sub-interface and embed it in Handler. (`type Handler interface { ProgressHandler }
type ProgressHandler interface { GetProgress() GetProgressHandler }`)
**Compile-time interface assertion on handler struct** — var _ Handler = (*handler)(nil) ensures the concrete struct satisfies the Handler interface at compile time. (`var _ Handler = (*handler)(nil)`)
**Namespace resolved via namespacedriver.NamespaceDecoder** — All handlers call h.resolveNamespace(ctx) which reads the namespace from the request context via NamespaceDecoder. Returns HTTP 500 if namespace is missing. (`ns, ok := h.namespaceDecoder.GetNamespace(ctx)
if !ok { return "", commonhttp.NewHTTPError(http.StatusInternalServerError, ...) }`)
**httptransport.NewHandlerWithArgs for path-parameter endpoints** — Endpoints with URL path arguments use httptransport.NewHandlerWithArgs[Req, Resp, Arg]. The first func decodes (ctx, *http.Request, arg) -> (Req, error); the second is the operation. (`return httptransport.NewHandlerWithArgs(
    func(ctx context.Context, r *http.Request, progressID string) (GetProgressRequest, error) { ... },
    func(ctx context.Context, request GetProgressRequest) (GetProgressResponse, error) { ... },
    commonhttp.JSONResponseEncoderWithStatus[GetProgressResponse](http.StatusOK),
    ...options,
)`)
**Response type aliased from api package** — GetProgressResponse is aliased to api.Progress (the generated OpenAPI type), not a local struct. Conversion from domain type is done via a local progressToAPI() helper. (`type GetProgressResponse = api.Progress
func progressToAPI(p entity.Progress) api.Progress { ... }`)
**HandlerOption propagation with httptransport.AppendOptions** — Handler-level options (e.g., error encoders) are propagated to each endpoint via httptransport.AppendOptions(h.options, httptransport.WithOperationName("...")). (`httptransport.AppendOptions(h.options, httptransport.WithOperationName("getProgress"))...`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `handler.go` | Defines the Handler/ProgressHandler interfaces, the handler struct, New() constructor, and resolveNamespace helper. | resolveNamespace returns HTTP 500 (not 401/403) when namespace is missing — this is intentional for self-hosted deployments using StaticNamespaceDecoder. |
| `progress.go` | Implements GetProgress endpoint: decoding, operation (calls service.GetProgress), and response encoding. Contains the progressToAPI() conversion helper. | nil progress after a successful service call is treated as an internal error (not not-found) — the guard prevents silent data loss. |

## Anti-Patterns

- Calling the domain service directly from the decoder function — decode must only extract request fields, the operation func calls the service
- Returning domain entity types directly as HTTP responses — always convert via a local toAPI() helper using the generated api package types
- Hardcoding the namespace string instead of using h.resolveNamespace(ctx)
- Adding business logic to progressToAPI() — it must be a pure field mapping with no validation or computation

## Decisions

- **GetProgressResponse aliased to api.Progress rather than a new struct** — The generated api.Progress is the contract-stable API type; aliasing avoids a redundant intermediate struct and ensures the HTTP response always matches the OpenAPI spec.

## Example: Register the handler in the v1 router

```
import (
    "github.com/openmeterio/openmeter/openmeter/progressmanager/httpdriver"
)

handler := httpdriver.New(namespaceDecoder, progressSvc, options...)
router.Get("/api/v1/progress/{progressID}", handler.GetProgress().ServeHTTP)
```

<!-- archie:ai-end -->
