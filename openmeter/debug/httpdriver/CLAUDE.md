# httpdriver

<!-- archie:ai-start -->

> HTTP handler layer for the debug domain. Bridges debug.DebugConnector to HTTP using the httptransport pipeline. Single current endpoint: GetMetrics, returning plain-text ClickHouse metrics resolved from the namespace context.

## Patterns

**Handler interface + private struct** — Expose a DebugHandler interface with method GetMetrics(); back it with a private debugHandler struct. Consumers depend on the interface. (`type DebugHandler interface { GetMetrics() GetMetricsHandler }`)
**httptransport.NewHandlerWithArgs for every endpoint** — Each endpoint is returned as an httptransport.HandlerWithArgs[Request, Response, Params] value — decode, operation, encode wired inline. Never implement http.Handler directly. (`return httptransport.NewHandlerWithArgs[GetMetricsHandlerRequest, string, GetMetricsHandlerParams](decodeFn, opFn, commonhttp.PlainTextResponseEncoder[string], opts...)`)
**Namespace from NamespaceDecoder only** — Namespace is always resolved via h.namespaceDecoder.GetNamespace(ctx). Never read from URL params or headers. Missing namespace returns 500. (`ns, ok := h.namespaceDecoder.GetNamespace(ctx); if !ok { return "", commonhttp.NewHTTPError(http.StatusInternalServerError, ...) }`)
**Per-handler error encoder via AppendOptions** — Domain-specific error mapping is added per-handler by appending httptransport.WithErrorEncoder to shared options. Never replace h.options wholesale. (`httptransport.AppendOptions(h.options, httptransport.WithErrorEncoder(func(...) bool { ... }))`)
**Constructor accepts variadic HandlerOptions** — NewDebugHandler takes ...httptransport.HandlerOption and stores them on the struct so all handlers inherit shared options (auth, tracing) from the router layer. (`func NewDebugHandler(dec namespacedriver.NamespaceDecoder, conn debug.DebugConnector, options ...httptransport.HandlerOption) DebugHandler`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `metrics.go` | Sole file. Defines DebugHandler interface, private struct, constructor, request/response/params types, and the GetMetrics handler factory. | GetMetricsHandlerResponse is a type alias for string — uses PlainTextResponseEncoder, not JSON. New endpoints must follow the same HandlerWithArgs shape; do not return raw http.HandlerFunc. |

## Anti-Patterns

- Implementing http.Handler directly instead of using httptransport.NewHandlerWithArgs
- Reading namespace from URL path or query params instead of namespacedriver.NamespaceDecoder
- Replacing h.options instead of appending with httptransport.AppendOptions when adding error encoders
- Putting business logic (metrics computation) inside decode or encode functions — delegate to debugConnector
- Returning JSON from a plain-text endpoint by swapping the encoder without updating Content-Type handling

## Decisions

- **Handlers return httptransport.HandlerWithArgs, not http.Handler** — httptransport provides a uniform decode/encode/error pipeline with OTel and error-encoder chaining consistent with all other domain httpdriver packages.
- **Namespace resolved from context via NamespaceDecoder, not from request** — Static namespace injection (self-hosted) is handled at the router middleware layer; re-parsing in the handler would bypass that and break multi-tenant isolation.

## Example: Adding a second debug endpoint (e.g. GetStatus)

```
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
            httptransport.WithErrorEncoder(func(ctx context.Context, err error, w http.ResponseWriter, _ *http.Request) bool {
// ...
```

<!-- archie:ai-end -->
