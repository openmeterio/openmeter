# serializer

<!-- archie:ai-start -->

> Defines the Serializer interface and its JSON implementation that translate CloudEvents into the Kafka wire format (CloudEventsKafkaPayload) for the ingest pipeline. It is the single boundary where event-time usage events become Kafka key/value bytes and back.

## Patterns

**Serializer interface contract** — Every serializer implements Serializer (SerializeKey, SerializeValue, GetFormat, GetKeySchemaId, GetValueSchemaId). Schema-less formats return -1 from the GetKeySchemaId/GetValueSchemaId methods. (`type Serializer interface { SerializeKey(topic, namespace string, ev event.Event) ([]byte, error); ... }`)
**Dedupe-derived Kafka key** — SerializeKey builds the Kafka partition key from dedupe.Item{Namespace, ID, Source}.Key() so that duplicate events route to the same partition. Do not hand-format keys. (`dedupe.Item{Namespace: namespace, ID: ev.ID(), Source: ev.Source()}.Key()`)
**Flat CloudEventsKafkaPayload struct** — The wire value is the flat CloudEventsKafkaPayload (Id, Type, Source, Subject, Time int64, Data string). Time is a unix int64 (timezone is intentionally lost); Data is the event payload re-marshalled to a JSON string. (`payload.Time = ev.Time().Unix(); payload.Data = string(payloadData)`)
**Symmetric encode/decode helpers** — toCloudEventsKafkaPayload (encode) and FromKafkaPayloadToCloudEvents (decode) are inverse functions; sink-side consumers call the exported From... helper to reconstruct event.Event. Keep them in lockstep when adding fields. (`ev, err := FromKafkaPayloadToCloudEvents(payload)`)
**Optional-data guard** — CloudEvents data is optional: encode only marshals when len(ev.Data()) > 0, decode only sets data when payload.Data != "". Preserve this so empty-data events round-trip cleanly. (`if len(ev.Data()) > 0 { ... }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `serializer.go` | Defines the Serializer interface, the CloudEventsKafkaPayload wire struct, the encode/decode helpers, and ValidateKafkaPayloadToCloudEvent. | Time is stored as Unix seconds — sub-second precision and timezone are dropped. Validation requires Id/Type/Source/Subject non-empty and Time > 0; relaxing these affects all consumers. |
| `json.go` | JSONSerializer: the default schema-less implementation. NewJSONSerializer returns a value (not pointer); GetFormat returns "JSON"; schema id getters return -1. | Only JSON-serializable CloudEvents data is supported (toCloudEventsKafkaPayload errors otherwise). A new format (e.g. Avro/Protobuf) needs its own Serializer impl returning real schema ids. |
| `serializer_test.go` | Table-driven tests for round-tripping payloads and the dedupe key format (e.g. SerializeKey yields "namespace-source-id"). | Tests assert exact key string "test-namespace-test-source-test-id" — changing dedupe.Item.Key() format breaks them and downstream partition routing. |

## Anti-Patterns

- Constructing Kafka keys manually instead of via dedupe.Item.Key() — breaks dedupe partition routing.
- Adding a field to encode (toCloudEventsKafkaPayload) without updating decode (FromKafkaPayloadToCloudEvents) — breaks round-trip symmetry.
- Storing Time as anything other than Unix seconds int64 — diverges from the established wire contract.
- Returning a real schema id (not -1) from JSONSerializer — implies a schema registry that does not exist for JSON.

## Decisions

- **Kafka value is a flat CloudEventsKafkaPayload with Data as a JSON string rather than the raw CloudEvents envelope.** — Gives a stable, ClickHouse-friendly columnar shape and avoids nested/binary CloudEvents encoding on the wire.
- **Event time is serialized as Unix seconds, explicitly dropping timezone.** — Metering operates on instants; a single int64 is compact and unambiguous for downstream aggregation.

## Example: Serialize a CloudEvent to a Kafka key and value

```
import (
	"github.com/cloudevents/sdk-go/v2/event"
	"github.com/openmeterio/openmeter/openmeter/ingest/kafkaingest/serializer"
)

s := serializer.NewJSONSerializer()
key, err := s.SerializeKey(topic, namespace, ev)   // "namespace-source-id"
val, err := s.SerializeValue(topic, ev)             // JSON CloudEventsKafkaPayload
```

<!-- archie:ai-end -->
