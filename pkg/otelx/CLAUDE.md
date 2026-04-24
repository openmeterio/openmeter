# otelx

<!-- archie:ai-start -->

> Minimal OpenTelemetry span-context serialization helpers for cross-process propagation (e.g. via Kafka message headers). Serializes trace.SpanContext to/from base64-encoded JSON using the OTelSpanContextKey constant.

## Patterns

**Base64-over-JSON encoding** — SpanContext is marshalled to JSON via trace.SpanContext.MarshalJSON() and then base64-encoded; reversal uses base64.DecodeString then json.Unmarshal into trace.SpanContextConfig. (`s := base64.StdEncoding.EncodeToString(b)`)
**OTelSpanContextKey constant for header/attribute name** — Always use the OTelSpanContextKey constant ("otel.span.context") as the key when storing or reading the serialized span context — do not hardcode the string. (`const OTelSpanContextKey = "otel.span.context"`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `spancontext.go` | Only file; exposes SerializeSpanContext and DeserializeSpanContext plus the key constant. | DeserializeSpanContext returns *trace.SpanContext (pointer); callers must nil-check before dereferencing. |

## Anti-Patterns

- Storing the serialized span context under a key other than OTelSpanContextKey.
- Using W3C TraceContext or B3 propagators here — this package uses its own JSON/base64 scheme for non-HTTP transports (e.g. Kafka headers).

## Decisions

- **Custom base64+JSON serialization instead of standard W3C propagator** — Kafka message headers carry opaque byte slices; W3C propagator expects HTTP carrier semantics. Base64 over JSON is self-contained and transport-agnostic.

<!-- archie:ai-end -->
