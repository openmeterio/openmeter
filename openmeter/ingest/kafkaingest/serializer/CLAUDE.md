# serializer

<!-- archie:ai-start -->

> Defines the Serializer interface and its JSON implementation for converting CloudEvents into Kafka key/value byte pairs. The key encodes a dedupe.Item (namespace+source+id) to enable downstream deduplication; the value encodes a flat CloudEventsKafkaPayload struct with unix-timestamp time.

## Patterns

**Serializer interface contract** — Every serializer must implement SerializeKey(topic, namespace, event), SerializeValue(topic, event), GetFormat(), GetKeySchemaId(), GetValueSchemaId(). Schema IDs return -1 for JSON (no schema registry). (`var _ Serializer = JSONSerializer{}`)
**Key encodes dedupe.Item** — SerializeKey must produce []byte(dedupeItem.Key()) where dedupeItem = dedupe.Item{Namespace, ID, Source}. The sink worker relies on this exact key format for Redis deduplication. (`dedupeItem := dedupe.Item{Namespace: namespace, ID: ev.ID(), Source: ev.Source()}; return []byte(dedupeItem.Key()), nil`)
**CloudEventsKafkaPayload wire format** — Value serialization must go through toCloudEventsKafkaPayload → json.Marshal producing {id, type, source, subject, time (unix int64), data (JSON string)}. Time precision is intentionally truncated to seconds; timezone is lost. (`payload.Time = ev.Time().Unix()`)
**Optional data field handling** — When ev.Data() is empty, Data field is left as empty string. When present, data is parsed via ev.DataAs(&data) then re-marshaled to JSON string. A parse failure returns an error — never silently drops data. (`if len(ev.Data()) > 0 { ev.DataAs(&data); json.Marshal(data) }`)
**Compile-time interface assertion** — Use var _ Serializer = (*ConcreteType)(nil) to assert implementation at compile time. (`var _ Serializer = JSONSerializer{}`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `serializer.go` | Defines the Serializer interface, CloudEventsKafkaPayload wire struct, toCloudEventsKafkaPayload (event→payload), FromKafkaPayloadToCloudEvents (payload→event), and ValidateKafkaPayloadToCloudEvent. These are shared by both producer (serializer) and consumer (sink) paths. | FromKafkaPayloadToCloudEvents is consumed by the sink worker — changing field names or unix-timestamp semantics here breaks deserialization on the other side. |
| `json.go` | Concrete JSONSerializer implementation. No schema registry, GetKeySchemaId/GetValueSchemaId return -1. | Key format must match dedupe.Item.Key() output exactly — the sink worker's Redis deduplication depends on it. |
| `serializer_test.go` | Table-driven tests for round-trip fidelity (toCloudEventsKafkaPayload, FromKafkaPayloadToCloudEvents) and key format assertion. | TestSerializeKey asserts key = 'namespace-source-id'; any dedupe.Item.Key() format change must be reflected here. |

## Anti-Patterns

- Adding schema-registry-aware serializers without implementing the full Serializer interface (GetKeySchemaId/GetValueSchemaId must return real schema IDs)
- Storing timezone-aware time in CloudEventsKafkaPayload.Time — field is unix int64, timezone is intentionally dropped
- Embedding namespace in the value payload — namespace belongs only in the key via dedupe.Item
- Silently dropping data parse errors in toCloudEventsKafkaPayload — always return the error
- Bypassing toCloudEventsKafkaPayload in SerializeValue — the sink worker's FromKafkaPayloadToCloudEvents expects exactly this struct

## Decisions

- **Key is derived from dedupe.Item (namespace+source+id) rather than event ID alone** — Guarantees global uniqueness across namespaces for Redis-backed deduplication in the sink worker
- **Data is stored as a JSON string inside CloudEventsKafkaPayload rather than raw bytes** — CloudEvents data is JSON-only in this system; string encoding avoids base64 overhead and keeps the payload human-readable in Kafka

## Example: Implementing a new Serializer (e.g. Avro) following the existing pattern

```
package serializer

import (
	"github.com/cloudevents/sdk-go/v2/event"
	"github.com/openmeterio/openmeter/openmeter/dedupe"
)

var _ Serializer = AvroSerializer{}

type AvroSerializer struct{}

func (s AvroSerializer) SerializeKey(topic string, namespace string, ev event.Event) ([]byte, error) {
	item := dedupe.Item{Namespace: namespace, ID: ev.ID(), Source: ev.Source()}
	return []byte(item.Key()), nil
}
// ...
```

<!-- archie:ai-end -->
