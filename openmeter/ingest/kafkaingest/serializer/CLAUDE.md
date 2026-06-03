# serializer

<!-- archie:ai-start -->

> Defines the Serializer interface and its JSON implementation that convert CloudEvents into Kafka key/value byte pairs. The key encodes a dedupe.Item (namespace+source+id) for sink-worker Redis deduplication; the value encodes a flat CloudEventsKafkaPayload with unix-timestamp time.

## Patterns

**Serializer interface contract** — Every serializer implements SerializeKey(topic, namespace, ev), SerializeValue(topic, ev), GetFormat(), GetKeySchemaId(), GetValueSchemaId(). JSONSerializer returns -1 for both schema IDs (no registry). (`var _ Serializer = JSONSerializer{}`)
**Key encodes dedupe.Item** — SerializeKey must produce []byte(dedupe.Item{Namespace, ID, Source}.Key()). The sink worker's Redis dedup depends on this exact 'namespace-source-id' format — changing it silently breaks deduplication. (`dedupeItem := dedupe.Item{Namespace: namespace, ID: ev.ID(), Source: ev.Source()}; return []byte(dedupeItem.Key()), nil`)
**CloudEventsKafkaPayload wire format** — SerializeValue goes through toCloudEventsKafkaPayload then json.Marshal. Time is unix int64 (seconds, timezone lost); Data is a JSON string. FromKafkaPayloadToCloudEvents is the inverse, consumed by the sink worker. (`payload.Time = ev.Time().Unix()`)
**Optional data field — error, never drop** — When ev.Data() is empty, Data is left as empty string. When present, parse via ev.DataAs(&data) then re-marshal to a JSON string; parse failures must return an error, never silently drop. (`if len(ev.Data()) > 0 { if err := ev.DataAs(&data); err != nil { return payload, errors.New("cannot unmarshal cloudevents data") } }`)
**Compile-time interface assertion** — var _ Serializer = ConcreteType{} at package level asserts implementation at compile time. (`var _ Serializer = JSONSerializer{}`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `serializer.go` | Defines Serializer interface, CloudEventsKafkaPayload, toCloudEventsKafkaPayload, FromKafkaPayloadToCloudEvents, and ValidateKafkaPayloadToCloudEvent. Shared by producer and sink consumer paths. | FromKafkaPayloadToCloudEvents is consumed by the sink worker — changing field names or unix-timestamp semantics breaks consumer-side deserialization. |
| `json.go` | Concrete JSONSerializer: no schema registry, GetKeySchemaId/GetValueSchemaId return -1, key derived from dedupe.Item. | Key format must match dedupe.Item.Key() output exactly; the sink worker Redis dedup depends on it. |
| `serializer_test.go` | Table-driven round-trip tests for toCloudEventsKafkaPayload, FromKafkaPayloadToCloudEvents, and SerializeKey format. | TestSerializeKey asserts key = 'test-namespace-test-source-test-id'; any dedupe.Item.Key() format change must be reflected here. |

## Anti-Patterns

- Adding schema-registry-aware serializers without the full Serializer interface (GetKeySchemaId/GetValueSchemaId must return real IDs)
- Storing timezone-aware time in CloudEventsKafkaPayload.Time — it is unix int64, timezone intentionally dropped
- Embedding namespace in the value payload — namespace belongs only in the key via dedupe.Item
- Silently dropping data parse errors in toCloudEventsKafkaPayload
- Bypassing toCloudEventsKafkaPayload in SerializeValue — the sink worker expects exactly this struct

## Decisions

- **Key derived from dedupe.Item (namespace+source+id) rather than event ID alone** — Guarantees global uniqueness across namespaces for Redis-backed deduplication in the sink worker.
- **Data stored as a JSON string inside CloudEventsKafkaPayload rather than raw bytes** — CloudEvents data is JSON-only here; string encoding avoids base64 overhead and keeps payloads human-readable in Kafka.

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
```

<!-- archie:ai-end -->
