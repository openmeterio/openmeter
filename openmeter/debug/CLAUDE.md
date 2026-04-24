# debug

<!-- archie:ai-start -->

> Exposes a single ClickHouse-backed debug metrics endpoint that returns per-subject event counts in OpenMetrics (Prometheus) format. Minimal domain: one connector interface, one httpdriver handler.

## Patterns

**DebugConnector as the only service interface** — Business logic lives in debugConnector wrapping streaming.Connector; the httpdriver delegates to DebugConnector, never to streaming.Connector directly. (`connector := debug.NewDebugConnector(streamingConnector); handler := httpdriver.NewHandler(connector, ...)`)
**httptransport.NewHandlerWithArgs for every HTTP endpoint** — Handlers use httptransport.NewHandlerWithArgs, not http.Handler directly; namespace is resolved via namespacedriver.NamespaceDecoder injected into the handler. (`httptransport.NewHandlerWithArgs(func(ctx context.Context, r *http.Request) (string, error) { return namespaceDecoder(ctx) }, operationFn, encodeResponse)`)
**Per-handler error encoder appended via httptransport.AppendOptions** — Error encoders are appended to shared options with httptransport.AppendOptions rather than replacing h.options. (`opts := httptransport.AppendOptions(h.options, httptransport.WithErrorEncoder(myEncoder))`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `openmeter/debug/debug.go` | Defines DebugConnector interface and debugConnector implementation that queries streaming.CountEvents and formats Prometheus OpenMetrics output. | GetDebugMetrics queries from start-of-day UTC — hardcoded time window; any new metrics methods should document their query window clearly. |
| `openmeter/debug/httpdriver/metrics.go` | HTTP handler for the GetMetrics endpoint using httptransport.NewHandlerWithArgs. | Handler returns plain text, not JSON; do not add JSON encoder without also updating the Content-Type handling. |

## Anti-Patterns

- Implementing http.Handler directly instead of using httptransport.NewHandlerWithArgs.
- Reading namespace from URL path or query params instead of namespacedriver.NamespaceDecoder.
- Replacing h.options instead of appending with httptransport.AppendOptions when adding error encoders.
- Putting metrics computation inside the decode or encode functions — delegate to DebugConnector.
- Adding streaming.Connector as a direct dependency in the httpdriver — always go through DebugConnector.

## Decisions

- **DebugConnector wraps streaming.Connector rather than exposing streaming directly.** — Keeps the httpdriver decoupled from ClickHouse query details; DebugConnector owns the Prometheus format conversion.

<!-- archie:ai-end -->
