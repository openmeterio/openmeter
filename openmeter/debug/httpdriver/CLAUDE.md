# httpdriver

<!-- archie:ai-start -->

> HTTP handler layer for the debug domain, bridging the debug.DebugConnector service to HTTP using the httptransport pattern. Single endpoint: GetMetrics, which resolves namespace from context and returns plain-text ClickHouse metrics.

## Patterns

**Handler interface + struct separation** — Expose a DebugHandler interface with method GetMetrics(); back it with a private debugHandler struct. Consumers depend on the interface, not the concrete type. (`type DebugHandler interface { GetMetrics() GetMetricsHandler }`)
**httptransport.NewHandlerWithArgs for every endpoint** — Each endpoint is returned as an httptransport.HandlerWithArgs[Request, Response, Params] value — decode, operation, encode wired inline. Never implement http.Handler directly. (`return httptransport.NewHandlerWithArgs[GetMetricsHandlerRequest, string, GetMetricsHandlerParams](decodeFn, opFn, commonhttp.PlainTextResponseEncoder[string], opts...)`)
**Namespace resolved via namespacedriver.NamespaceDecoder** — Namespace is never read from URL params or headers directly; always use h.namespaceDecoder.GetNamespace(ctx). Missing namespace → 500 internal error. (`ns, ok := h.namespaceDecoder.GetNamespace(ctx); if !ok { return "", commonhttp.NewHTTPError(http.StatusInternalServerError, ...) }`)
**Per-handler error encoder appended via httptransport.AppendOptions** — Domain-specific error mapping (e.g. validation → 400) is added per-handler by appending httptransport.WithErrorEncoder to the shared options slice, not by replacing them. (`httptransport.AppendOptions(h.options, httptransport.WithErrorEncoder(func(...) bool { ... }))`)
**Constructor accepts variadic HandlerOptions** — NewDebugHandler takes ...httptransport.HandlerOption and stores them on the struct so all handlers inherit shared options (auth, tracing, etc.) from the router layer. (`func NewDebugHandler(dec namespacedriver.NamespaceDecoder, conn debug.DebugConnector, options ...httptransport.HandlerOption) DebugHandler`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `metrics.go` | Sole file; defines the DebugHandler interface, private struct, constructor, request/response types, and the GetMetrics handler factory. | GetMetricsHandlerResponse is a type alias for string — uses PlainTextResponseEncoder, not JSON. Adding a new endpoint must follow the same HandlerWithArgs shape; do not return raw http.HandlerFunc. |

## Anti-Patterns

- Implementing http.Handler directly instead of using httptransport.NewHandlerWithArgs
- Reading namespace from URL path or query params instead of namespacedriver.NamespaceDecoder
- Replacing h.options instead of appending with httptransport.AppendOptions when adding error encoders
- Putting business logic (metrics computation) inside the decode or encode functions — delegate to debugConnector
- Returning JSON from a plain-text endpoint by swapping the encoder without updating the Content-Type error path

## Decisions

- **Handler returns httptransport.HandlerWithArgs value, not http.Handler** — httptransport provides uniform decode/encode/error pipeline with OTel and error-encoder chaining across all domain httpdriver packages.
- **Namespace resolved from context via NamespaceDecoder, not from request** — Static namespace injection (self-hosted) is handled at the router middleware layer; the handler must not re-parse it, ensuring multi-tenant isolation is enforced once and consistently.

## Example: Adding a second debug endpoint (e.g. GetStatus)

```
// In metrics.go (same file) or a new file in the same package:
type GetStatusHandler httptransport.HandlerWithArgs[GetStatusRequest, GetStatusResponse, GetStatusParams]

func (h *debugHandler) GetStatus() GetStatusHandler {
    return httptransport.NewHandlerWithArgs[GetStatusRequest, GetStatusResponse, GetStatusParams](
        func(ctx context.Context, r *http.Request, _ GetStatusParams) (GetStatusRequest, error) {
            ns, err := h.resolveNamespace(ctx)
            if err != nil { return GetStatusRequest{}, err }
            return GetStatusRequest{Namespace: ns}, nil
        },
        func(ctx context.Context, req GetStatusRequest) (GetStatusResponse, error) {
            return h.debugConnector.GetStatus(ctx, req.Namespace)
        },
        commonhttp.JSONResponseEncoder[GetStatusResponse],
        httptransport.AppendOptions(h.options,
// ...
```

<!-- archie:ai-end -->
