# marshaler

<!-- archie:ai-start -->

> Implements the cqrs.CommandEventMarshaler contract using CloudEvents 1.0 as the wire format. Every domain event is serialized to a CloudEvents JSON envelope with ce_type/ce_time/ce_source/ce_subject Watermill metadata headers; deserialization validates via Event.Validate(). WithSource wraps an Event to inject the source field without modifying the event struct.

## Patterns

**Event interface with EventName, EventMetadata, Validate** — Every publishable struct must implement marshaler.Event: EventName() string (used as ce_type and router key), EventMetadata() metadata.EventMetadata (source, subject, time, id), and Validate() error (called on both marshal and unmarshal). (`func (e *InvoiceCreated) EventName() string { return "io.openmeter.billing.invoice.created/v1" }`)
**WithSource wrapper to inject CloudEvents source** — Use marshaler.WithSource(sourceURI, ev) when the event struct does not carry the source itself. The wrapper overrides EventMetadata().Source and delegates MarshalJSON to the inner event to avoid double-wrapping in JSON. (`marshaler.WithSource("openmeter/billing-worker", ev)`)
**ULID auto-ID and zero-time auto-now** — NewCloudEvent auto-generates a ULID ID and sets time to now if EventMetadata returns empty values. Events that need deterministic IDs must set metadata.ID explicitly. (`if metadata.ID == "" { cloudEvent.SetID(ulid.Make().String()) }`)
**Unmarshal calls Validate after JSON decode** — After json.Unmarshal into the event struct, Unmarshal calls ev.Validate(). Consumers rely on this for invariant checking — do not skip Validate in event structs. (`return ev.Validate() // at end of Unmarshal`)
**NameFromMessage reads ce_type header** — The router uses NameFromMessage to dispatch to the correct handler. The ce_type metadata header is set from EventName() during Marshal; consumers must not rename this header. (`return msg.Metadata.Get(CloudEventsHeaderType)`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `marshaler.go` | Core marshaler: Marshal (Event->CloudEvents JSON+headers), Unmarshal (CloudEvents JSON->Event+Validate), Name, NameFromMessage, NewCloudEvent helper. | Name() returns UnknownEventName ('io.openmeter.unknown') for non-Event values — this silently routes to SystemEventsTopic; ensure all published structs implement Event. |
| `source.go` | eventWithSource wrapper that overrides EventMetadata().Source. MarshalJSON delegates to inner event to avoid extra 'Event' key in payload. | The inner json.inline tag is decorative; actual marshaling is via the explicit MarshalJSON override. If Event is a value (not pointer) receiver, WithSource may not work correctly — pass pointer events. |
| `source_test.go` | Unit test for WithSource round-trip: confirms source header is set and payload unmarshals back to original struct. | Test uses package-internal event struct — real events must implement all three Event methods or Validate will fail during Unmarshal. |

## Anti-Patterns

- Implementing EventName() with a string that does not end with a versioned suffix (e.g. /v1) — no technical constraint, but event routing and schema evolution rely on stable names.
- Returning a non-pointer from WithSource when the inner Event uses pointer receivers — MarshalJSON override requires pointer semantics.
- Setting CloudEventsHeaderType manually in Watermill message metadata — Marshal sets it from EventName(); overriding bypasses routing.
- Skipping Validate() in event structs — Unmarshal calls it; invalid events will surface as consumer errors.
- Embedding a non-Event type and passing it to Marshal — Name() returns UnknownEventName and routing silently falls through to SystemEventsTopic.

## Decisions

- **CloudEvents 1.0 as the wire format** — Provides a standard envelope (type, source, subject, time, id) that can be consumed by non-Go systems and inspected in Kafka without custom schema knowledge.
- **Validate called on both sides of the wire** — Catches corrupt or partial events at both producer and consumer, preventing invalid data from propagating through the system.

## Example: Defining a new domain event struct

```
import (
    "github.com/openmeterio/openmeter/openmeter/event/metadata"
)

const EventVersionSubsystem = "io.openmeter.billing"

type InvoiceCreated struct {
    InvoiceID string `json:"invoiceId"`
}

func (e *InvoiceCreated) EventName() string {
    return EventVersionSubsystem + ".invoice.created/v1"
}

func (e *InvoiceCreated) EventMetadata() metadata.EventMetadata {
// ...
```

<!-- archie:ai-end -->
