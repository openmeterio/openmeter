# marshaler

<!-- archie:ai-start -->

> Implements the cqrs.CommandEventMarshaler contract using CloudEvents 1.0 as the wire format. Every domain event is serialized to a CloudEvents JSON envelope with ce_type/ce_time/ce_source/ce_subject Watermill metadata headers; deserialization validates via Event.Validate(). WithSource wraps an Event to inject the source field without modifying the event struct.

## Patterns

**Event interface — EventName, EventMetadata, Validate** — Every publishable struct must implement marshaler.Event: EventName() string (used as ce_type and router dispatch key), EventMetadata() metadata.EventMetadata (source, subject, time, id), and Validate() error (called on both Marshal and Unmarshal paths). (`func (e *InvoiceCreated) EventName() string { return EventVersionSubsystem + ".invoice.created/v1" }`)
**WithSource wrapper to inject CloudEvents source** — Use marshaler.WithSource(sourceURI, ev) when the event struct does not carry the source itself. MarshalJSON delegates to the inner event to avoid adding an extra 'Event' wrapper key in the JSON payload. (`return marshaler.WithSource("openmeter/billing-worker", ev)`)
**ULID auto-ID and zero-time auto-now in NewCloudEvent** — NewCloudEvent auto-generates a ULID ID and sets time to now if EventMetadata returns empty values. Events needing deterministic IDs must set metadata.ID explicitly before publishing. (`if metadata.ID == "" { cloudEvent.SetID(ulid.Make().String()) }`)
**Validate called on both sides of the wire** — Marshal calls Validate before encoding; Unmarshal calls it after JSON decode. Consumers rely on this invariant — do not skip Validate in event structs, even if fields appear optional. (`return ev.Validate() // last line of Unmarshal`)
**UnknownEventName fallback for non-Event types** — Name() returns 'io.openmeter.unknown' for types not implementing the Event interface. This silently routes to SystemEventsTopic via eventbus prefix matching. All published structs must implement Event. (`if !ok { return UnknownEventName } // inside Name()`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `marshaler.go` | Core marshaler: Marshal (Event→CloudEvents JSON+headers), Unmarshal (CloudEvents JSON→Event+Validate), Name, NameFromMessage, NewCloudEvent helper. | Name() returns UnknownEventName for non-Event values — this silently routes to SystemEventsTopic; ensure all published structs implement Event. |
| `source.go` | eventWithSource wrapper that overrides EventMetadata().Source. MarshalJSON delegates to inner event to avoid extra 'Event' key in payload. | json.inline tag on embedded Event is decorative; actual marshaling goes through the explicit MarshalJSON override. Pass pointer events to WithSource — value receivers on the inner Event may not satisfy interface correctly. |
| `source_test.go` | Unit test for WithSource round-trip: confirms source header is set and payload unmarshals back to original struct. | Test uses package-internal event struct — real events must implement all three Event methods or Validate will fail during Unmarshal. |

## Anti-Patterns

- Implementing EventName() with an unstable string that changes between versions — event routing and consumer handler dispatch rely on stable names.
- Returning a non-pointer from WithSource when the inner Event uses pointer receivers — MarshalJSON override requires pointer semantics to delegate correctly.
- Setting CloudEventsHeaderType manually in Watermill message metadata — Marshal sets it from EventName(); overriding bypasses type-based routing.
- Skipping Validate() in event structs — Unmarshal calls it; invalid events surface as consumer errors and trigger retries.
- Embedding a non-Event type and passing it to Marshal — Name() returns UnknownEventName and routing silently falls to SystemEventsTopic.

## Decisions

- **CloudEvents 1.0 as the wire format** — Provides a standard envelope (type, source, subject, time, id) consumable by non-Go systems and inspectable in Kafka without custom schema knowledge, enabling cross-team debugging and future polyglot consumers.
- **Validate called on both producer and consumer sides** — Catches corrupt or partial events at both publish time and consumption time, preventing invalid data from propagating downstream and making schema violations immediately visible at the source.

## Example: Defining a new domain event struct that satisfies the marshaler.Event interface

```
import "github.com/openmeterio/openmeter/openmeter/event/metadata"

const EventVersionSubsystem = "io.openmeter.billing"

type InvoiceCreated struct {
    InvoiceID string `json:"invoiceId"`
}

func (e *InvoiceCreated) EventName() string {
    return EventVersionSubsystem + ".invoice.created/v1"
}

func (e *InvoiceCreated) EventMetadata() metadata.EventMetadata {
    return metadata.EventMetadata{Source: "openmeter/billing"}
}
// ...
```

<!-- archie:ai-end -->
