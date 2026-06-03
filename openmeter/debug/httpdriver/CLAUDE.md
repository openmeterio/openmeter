# httpdriver

<!-- archie:ai-start -->

> HTTP handler layer for the debug domain. Bridges debug.DebugConnector to HTTP via the httptransport pipeline. Single current endpoint: GetMetrics, returning plain-text ClickHouse metrics resolved from the namespace context.

## Patterns

**Handler interface + private struct** — Expose a DebugHandler interface with GetMetrics(); back it with a private debugHandler struct. Consumers depend on the interface. (`type DebugHandler interface { GetMetrics() GetMetricsHandler }`)
**httptransport.NewHandlerWithArgs for every endpoint** — Each endpoint returns httptransport.HandlerWithArgs[Request, Response, Params] with decode/operation/encode wired inline; never implement http.Handler directly. (`return httptransport.NewHandlerWithArgs[GetMetricsHandlerRequest, string, GetMetricsHandlerParams](decodeFn, opFn, commonhttp.PlainTextResponseEncoder[string], opts...)`)
**Namespace from NamespaceDecoder only** — Namespace is always resolved via h.namespaceDecoder.GetNamespace(ctx); never from URL params or headers. Missing namespace returns 500. (`ns, ok := h.namespaceDecoder.GetNamespace(ctx); if !ok { return "", commonhttp.NewHTTPError(http.StatusInternalServerError, ...) }`)
**Per-handler error encoder via AppendOptions** — Domain-specific error mapping is added per-handler by appending httptransport.WithErrorEncoder to shared options; never replace h.options wholesale. (`httptransport.AppendOptions(h.options, httptransport.WithErrorEncoder(func(...) bool { ... }))`)
**Constructor accepts variadic HandlerOptions** — NewDebugHandler takes ...httptransport.HandlerOption and stores them so all handlers inherit shared options (auth, tracing) from the router layer. (`func NewDebugHandler(dec namespacedriver.NamespaceDecoder, conn debug.DebugConnector, options ...httptransport.HandlerOption) DebugHandler`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `metrics.go` | Sole file: DebugHandler interface, private struct, constructor, request/response/params types, GetMetrics handler factory. | GetMetricsHandlerResponse is a type alias for string and uses PlainTextResponseEncoder, not JSON; new endpoints follow the same HandlerWithArgs shape and must not return raw http.HandlerFunc. |

## Anti-Patterns

- Implementing http.Handler directly instead of httptransport.NewHandlerWithArgs.
- Reading namespace from URL path or query params instead of namespacedriver.NamespaceDecoder.
- Replacing h.options instead of appending with httptransport.AppendOptions when adding error encoders.
- Putting business logic (metrics computation) inside decode/encode functions — delegate to debugConnector.
- Returning JSON from a plain-text endpoint by swapping the encoder without updating Content-Type handling.

## Decisions

- **Handlers return httptransport.HandlerWithArgs, not http.Handler.** — httptransport provides a uniform decode/encode/error pipeline with OTel and error-encoder chaining consistent with all other httpdriver packages.
- **Namespace resolved from context via NamespaceDecoder, not the request.** — Static namespace injection (self-hosted) is handled at router middleware; re-parsing in the handler would bypass it and break multi-tenant isolation.

## Example: Adding a second debug endpoint (GetStatus)

```
type GetStatusHandler httptransport.HandlerWithArgs[GetStatusRequest, GetStatusResponse, GetStatusParams]

func (h *debugHandler) GetStatus() GetStatusHandler {
	return httptransport.NewHandlerWithArgs[GetStatusRequest, GetStatusResponse, GetStatusParams](
		func(ctx context.Context, r *http.Request, _ GetStatusParams) (GetStatusRequest, error) {
			ns, ok := h.namespaceDecoder.GetNamespace(ctx)
			if !ok { return GetStatusRequest{}, commonhttp.NewHTTPError(http.StatusInternalServerError, errors.New("namespace")) }
			return GetStatusRequest{Namespace: ns}, nil
		},
		func(ctx context.Context, req GetStatusRequest) (GetStatusResponse, error) { return h.debugConnector.GetStatus(ctx, req.Namespace) },
		commonhttp.JSONResponseEncoder[GetStatusResponse],
		h.options...,
	)
}
```

<!-- archie:ai-end -->
