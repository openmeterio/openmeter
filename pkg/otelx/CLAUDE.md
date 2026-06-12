# otelx

<!-- archie:ai-start -->

> OpenTelemetry helper for serializing a trace.SpanContext to/from bytes (base64-encoded JSON) so span context can ride along on Kafka messages; used by ingest/kafkaingest for trace propagation.

## Patterns

**SpanContext serialized as base64(JSON)** — `SerializeSpanContext` marshals the SpanContext to JSON then base64 std-encodes it; `DeserializeSpanContext` reverses this and rebuilds via trace.NewSpanContext. Both round-trip through `trace.SpanContextConfig`. (`b, _ := c.MarshalJSON(); s := base64.StdEncoding.EncodeToString(b)`)
**Stable propagation key constant** — `OTelSpanContextKey = "otel.span.context"` is the canonical header/metadata key for carrying serialized span context across message boundaries. (`const OTelSpanContextKey = "otel.span.context"`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `spancontext.go` | SerializeSpanContext / DeserializeSpanContext plus the OTelSpanContextKey constant. | Deserialize reuses the `b` variable for the decoded bytes; format is base64-over-JSON on both sides — keep encode/decode symmetric or trace context will be lost silently. |

## Anti-Patterns

- Changing the serialization format on one side only (e.g. dropping base64), breaking span propagation between producer and consumer.
- Hard-coding the message key string instead of referencing OTelSpanContextKey.

## Decisions

- **base64-encode the JSON span context.** — Produces an ASCII-safe payload that can be carried in Kafka message headers/metadata without binary-encoding concerns.

## Example: Attach span context to a Kafka message

```
import "github.com/openmeterio/openmeter/pkg/otelx"

b, err := otelx.SerializeSpanContext(span.SpanContext())
if err != nil {
	return err
}
headers[otelx.OTelSpanContextKey] = b
```

<!-- archie:ai-end -->
