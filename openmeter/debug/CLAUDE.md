# debug

<!-- archie:ai-start -->

> Exposes a single ClickHouse-backed debug metrics endpoint returning per-subject event counts in OpenMetrics (Prometheus) format. Minimal two-file domain: debug.go owns the DebugConnector wrapping streaming.Connector; httpdriver/ translates HTTP to domain calls.

## Patterns

**DebugConnector as the only service interface** — Business logic lives in debugConnector wrapping streaming.Connector; httpdriver delegates to DebugConnector, never to streaming.Connector directly. (`connector := debug.NewDebugConnector(streamingConnector); handler := httpdriver.NewHandler(connector, ...)`)
**httptransport.NewHandlerWithArgs for every endpoint** — Handlers use httptransport.NewHandlerWithArgs (not http.Handler); namespace resolved via namespacedriver.NamespaceDecoder. (`httptransport.NewHandlerWithArgs(func(ctx, r) (string, error) { return namespaceDecoder(ctx) }, operationFn, encodeResponse)`)
**Per-handler error encoder via AppendOptions** — Error encoders are appended with httptransport.AppendOptions rather than replacing h.options. (`opts := httptransport.AppendOptions(h.options, httptransport.WithErrorEncoder(myEncoder))`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `debug.go` | DebugConnector interface and debugConnector implementation that queries streaming.CountEvents and formats Prometheus OpenMetrics output (openmeter_events_total counter). | GetDebugMetrics queries from start-of-day UTC (hardcoded window); document the query window for any new metrics method. |
| `httpdriver/metrics.go` | GetMetrics HTTP handler via httptransport.NewHandlerWithArgs. | Returns plain-text OpenMetrics, not JSON; do not swap to a JSON encoder without updating Content-Type handling. |

## Anti-Patterns

- Implementing http.Handler directly instead of httptransport.NewHandlerWithArgs
- Reading namespace from URL path/query instead of namespacedriver.NamespaceDecoder
- Replacing h.options instead of appending with httptransport.AppendOptions
- Putting metrics computation inside the decode/encode functions — delegate to DebugConnector
- Adding streaming.Connector as a direct dependency in httpdriver — always go through DebugConnector

## Decisions

- **DebugConnector wraps streaming.Connector rather than exposing streaming directly** — Keeps httpdriver decoupled from ClickHouse query details; DebugConnector owns the Prometheus format conversion.

<!-- archie:ai-end -->
