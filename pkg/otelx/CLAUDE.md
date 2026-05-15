# otelx

<!-- archie:ai-start -->

> Minimal OpenTelemetry span-context serialization helpers for cross-process propagation over non-HTTP transports (e.g. Kafka message headers). Serializes trace.SpanContext to/from base64-encoded JSON using the OTelSpanContextKey constant.

## Patterns

**OTelSpanContextKey constant as the canonical header/attribute name** — Always use the OTelSpanContextKey constant ("otel.span.context") when storing or reading the serialized span context in message headers or attributes. Never hardcode the string. (`headers[otelx.OTelSpanContextKey] = serialized`)
**Base64-over-JSON encoding for transport-agnostic propagation** — SpanContext is marshalled to JSON via trace.SpanContext.MarshalJSON() and then base64-encoded. Reversal uses base64.DecodeString then json.Unmarshal into trace.SpanContextConfig then trace.NewSpanContext. (`b, _ := c.MarshalJSON()
s := base64.StdEncoding.EncodeToString(b)`)
**Nil-check DeserializeSpanContext return value** — DeserializeSpanContext returns *trace.SpanContext (pointer). Callers must nil-check before dereferencing even when err == nil, because a non-nil error short-circuits via early return. (`sc, err := otelx.DeserializeSpanContext(headerVal)
if err != nil || sc == nil { return }
span := sc.TraceID()`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `spancontext.go` | Only file; exports SerializeSpanContext, DeserializeSpanContext, and OTelSpanContextKey constant. | DeserializeSpanContext returns *trace.SpanContext — callers must nil-check. The function does not validate that the resulting SpanContext is valid (IsValid()); add that check at the call site if needed. |

## Anti-Patterns

- Storing the serialized span context under any key other than OTelSpanContextKey
- Using W3C TraceContext or B3 propagators here — this package uses its own JSON/base64 scheme for non-HTTP transports
- Dereferencing the *trace.SpanContext return from DeserializeSpanContext without a nil check

## Decisions

- **Custom base64+JSON serialization instead of standard W3C propagator** — Kafka message headers carry opaque byte slices; the W3C TraceContext propagator expects an HTTP carrier interface. Base64-over-JSON is self-contained, transport-agnostic, and avoids pulling the full otel propagation API into Kafka message handling code.

<!-- archie:ai-end -->
