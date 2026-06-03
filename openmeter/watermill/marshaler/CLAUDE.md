# marshaler

<!-- archie:ai-start -->

> Implements the cqrs.CommandEventMarshaler contract using CloudEvents 1.0 as the wire format. Each domain event serializes to a CloudEvents JSON envelope with ce_type/ce_time/ce_source/ce_subject headers; deserialization validates via Event.Validate(). WithSource injects the source field without modifying the event struct.

## Patterns

**Event interface — EventName, EventMetadata, Validate** — Each publishable struct implements EventName() (ce_type and dispatch key), EventMetadata() (source/subject/time/id), and Validate() (called on both Marshal and Unmarshal). (`func (e *InvoiceCreated) EventName() string { return EventVersionSubsystem + ".invoice.created/v1" }`)
**WithSource wrapper to inject CloudEvents source** — Use marshaler.WithSource(sourceURI, ev) when the event struct lacks a source; MarshalJSON delegates to the inner event to avoid an extra 'Event' wrapper key. (`return marshaler.WithSource("openmeter/billing-worker", ev)`)
**ULID auto-ID and zero-time auto-now** — NewCloudEvent auto-generates a ULID ID and sets time to now if EventMetadata returns empty values; events needing deterministic IDs set metadata.ID explicitly. (`if metadata.ID == "" { cloudEvent.SetID(ulid.Make().String()) }`)
**Validate called on both sides of the wire** — Marshal calls Validate before encoding; Unmarshal calls it after JSON decode. Do not skip Validate even for optional-looking fields. (`return ev.Validate() // last line of Unmarshal`)
**UnknownEventName fallback for non-Event types** — Name() returns 'io.openmeter.unknown' for types not implementing Event, which silently routes to SystemEventsTopic. All published structs must implement Event. (`if !ok { return UnknownEventName }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `marshaler.go` | Marshal (Event->CloudEvents JSON+headers), Unmarshal (->Event+Validate), Name, NameFromMessage, NewCloudEvent helper. | Name() returns UnknownEventName for non-Event values, silently routing to SystemEventsTopic; ensure all published structs implement Event. |
| `source.go` | eventWithSource wrapper overriding EventMetadata().Source; MarshalJSON delegates to inner event. | Pass pointer events to WithSource — value receivers on the inner Event may not satisfy the interface. |
| `source_test.go` | WithSource round-trip test confirming source header and payload round-trip. | Real events must implement all three Event methods or Validate fails during Unmarshal. |

## Anti-Patterns

- Implementing EventName() with an unstable string that changes between versions — routing and dispatch rely on stable names.
- Returning a non-pointer from WithSource when the inner Event uses pointer receivers — MarshalJSON override needs pointer semantics.
- Setting CloudEventsHeaderType manually in message metadata — Marshal sets it from EventName().
- Skipping Validate() in event structs — Unmarshal calls it; invalid events trigger retries.
- Embedding a non-Event type and passing it to Marshal — Name() returns UnknownEventName and routing falls to SystemEventsTopic.

## Decisions

- **CloudEvents 1.0 as the wire format.** — Provides a standard envelope consumable by non-Go systems and inspectable in Kafka without custom schema knowledge, enabling cross-team debugging and polyglot consumers.
- **Validate called on both producer and consumer sides.** — Catches corrupt/partial events at publish and consume time, preventing invalid data from propagating and making schema violations visible at the source.

## Example: Defining a new domain event satisfying marshaler.Event

```
import "github.com/openmeterio/openmeter/openmeter/event/metadata"

const EventVersionSubsystem = "io.openmeter.billing"

type InvoiceCreated struct { InvoiceID string `json:"invoiceId"` }

func (e *InvoiceCreated) EventName() string { return EventVersionSubsystem + ".invoice.created/v1" }
func (e *InvoiceCreated) EventMetadata() metadata.EventMetadata { return metadata.EventMetadata{Source: "openmeter/billing"} }
func (e *InvoiceCreated) Validate() error { if e.InvoiceID == "" { return errors.New("invoice id required") }; return nil }
```

<!-- archie:ai-end -->
