# httpdriver

<!-- archie:ai-start -->

> HTTP driver (transport layer) for the debug domain. Exposes a single read-only endpoint that returns per-subject ingested-event counts in OpenMetrics/Prometheus text format by delegating to debug.DebugConnector.GetDebugMetrics.

## Patterns

**Handler interface + private struct + constructor** — Expose a DebugHandler interface whose methods return typed httptransport handlers; back it with an unexported debugHandler struct holding decoder, connector, and []httptransport.HandlerOption; construct via NewDebugHandler. (`type DebugHandler interface { GetMetrics() GetMetricsHandler }; func NewDebugHandler(namespaceDecoder, debugConnector, options...) DebugHandler`)
**Three-stage httptransport.NewHandlerWithArgs** — Each endpoint is built as decode (request -> typed Request) + business (Request -> Response, delegates to connector) + response encoder, wired through httptransport.NewHandlerWithArgs with type params making Request/Response/Params explicit. (`httptransport.NewHandlerWithArgs[GetMetricsHandlerRequest, string, GetMetricsHandlerParams](decode, exec, commonhttp.PlainTextResponseEncoder[string], opts...)`)
**Per-endpoint Request/Params/Response type aliases** — Define GetMetricsHandlerRequest (wrapping a params struct), GetMetricsHandlerParams, GetMetricsHandlerResponse, and a GetMetricsHandler alias over httptransport.HandlerWithArgs[...] so the router gets a named handler type. (`type GetMetricsHandler httptransport.HandlerWithArgs[GetMetricsHandlerRequest, GetMetricsHandlerResponse, GetMetricsHandlerParams]`)
**Namespace resolved from context decoder, not request body** — Multi-tenancy namespace comes from namespacedriver.NamespaceDecoder.GetNamespace(ctx); the decode stage calls h.resolveNamespace(ctx) and fails with a 500 HTTPError when absent. (`ns, ok := h.namespaceDecoder.GetNamespace(ctx); if !ok { return commonhttp.NewHTTPError(http.StatusInternalServerError, ...) }`)
**Custom error encoder maps validation errors to 400** — Attach a WithErrorEncoder via httptransport.AppendOptions that converts models.IsGenericValidationError(err) into a 400 commonhttp.NewHTTPError and returns true; return false to fall through to default handling. (`httptransport.WithErrorEncoder(func(...) bool { if models.IsGenericValidationError(err) { commonhttp.NewHTTPError(http.StatusBadRequest, err).EncodeError(ctx, w); return true }; return false })`)
**Plain-text (OpenMetrics) response encoding** — Response is a raw string of OpenMetrics text, so it uses commonhttp.PlainTextResponseEncoder[string] rather than a JSON encoder. (`commonhttp.PlainTextResponseEncoder[string]`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `metrics.go` | Sole file: defines DebugHandler interface, debugHandler struct, NewDebugHandler constructor, GetMetrics handler, and resolveNamespace helper. | Business stage must only delegate to h.debugConnector.GetDebugMetrics; no querying logic here. resolveNamespace returns 500 (not 400) when namespace missing because that signals a wiring/middleware failure, not bad user input. |

## Anti-Patterns

- Putting metric-counting/streaming query logic in the handler instead of delegating to debug.DebugConnector.
- Reading namespace from a query param or body instead of namespacedriver.NamespaceDecoder.GetNamespace(ctx).
- Returning a JSON response encoder for this endpoint; output is OpenMetrics plain text via PlainTextResponseEncoder.
- Bypassing httptransport.NewHandlerWithArgs and writing directly to http.ResponseWriter in the business stage.
- Dropping the WithErrorEncoder validation-to-400 mapping, causing validation errors to surface as 500s.

## Decisions

- **Transport layer is split from domain (httpdriver vs parent debug package).** — Keeps HTTP concerns (decoding, namespace resolution, error encoding) separate from the OpenMetrics/streaming logic in debug.DebugConnector, matching OpenMeter's service/adapter/driver layering.
- **Response is emitted as OpenMetrics text, not the project's usual JSON.** — The debug endpoint is meant to be scraped/monitored Prometheus-style to observe ingested event counts per subject, so PlainTextResponseEncoder is used.

## Example: Adding a new debug endpoint following the existing handler pattern

```
func (h *debugHandler) GetMetrics() GetMetricsHandler {
	return httptransport.NewHandlerWithArgs[GetMetricsHandlerRequest, string, GetMetricsHandlerParams](
		func(ctx context.Context, r *http.Request, params GetMetricsHandlerParams) (GetMetricsHandlerRequest, error) {
			ns, err := h.resolveNamespace(ctx)
			if err != nil {
				return GetMetricsHandlerRequest{}, err
			}
			return GetMetricsHandlerRequest{params: GetMetricsHandlerRequestParams{Namespace: ns}}, nil
		},
		func(ctx context.Context, request GetMetricsHandlerRequest) (string, error) {
			return h.debugConnector.GetDebugMetrics(ctx, request.params.Namespace)
		},
		commonhttp.PlainTextResponseEncoder[string],
		httptransport.AppendOptions(h.options, httptransport.WithErrorEncoder(func(ctx context.Context, err error, w http.ResponseWriter, _ *http.Request) bool {
			if models.IsGenericValidationError(err) {
// ...
```

<!-- archie:ai-end -->
