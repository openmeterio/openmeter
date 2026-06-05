# marshaler

<!-- archie:ai-start -->

> CloudEvents (1.0) <-> Watermill message codec. Defines the Event interface (EventName/EventMetadata/Validate) every published event implements and serializes events as JSON CloudEvents with ce_* metadata headers.

## Patterns

**Event interface contract** — Anything published must implement Event: EventName() string, EventMetadata() metadata.EventMetadata, Validate() error. Marshal/Unmarshal type-assert to Event and fail with 'invalid event type' otherwise. (`type Event interface { EventName() string; EventMetadata() metadata.EventMetadata; Validate() error }`)
**Validate runs on both marshal and unmarshal** — NewCloudEvent calls ev.Validate() before SetData; Unmarshal calls ev.Validate() after json.Unmarshal. Invalid events never cross the bus in either direction. (`if err := ev.Validate(); err != nil { return cloudevents.Event{}, err }`)
**ce_* metadata headers carry routing/identity** — Marshal sets CloudEventsHeaderType/Time/Source and (if non-empty) Subject on msg.Metadata. NameFromMessage reads ce_type — this is what eventbus/grouphandler/router key on, not the payload. (`msg.Metadata.Set(CloudEventsHeaderType, ce.Type())`)
**Defaulting of ID and Time** — NewCloudEvent fills empty metadata.ID with ulid.Make() and zero metadata.Time with time.Now(); Source is mandatory (errors if empty). (`if metadata.ID == "" { cloudEvent.SetID(ulid.Make().String()) }`)
**WithSource decorator for late source binding** — WithSource(source, ev) wraps an Event, overriding EventMetadata().Source; its MarshalJSON marshals only the inner Event to avoid an embedded "Event" key in JSON. (`func WithSource(source string, ev Event) Event { return &eventWithSource{source: source, Event: ev} }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `marshaler.go` | Marshaler alias (cqrs.CommandEventMarshaler), Event interface, New, Marshal/Unmarshal, NewCloudEvent, Name, NameFromMessage; ce_* header + UnknownEventName constants. | Name() returns UnknownEventName ('io.openmeter.unknown') for non-Event values rather than erroring — a wrong type silently routes nowhere. TransformFunc, if set, runs last in Marshal and can rewrite the message. |
| `source.go` | eventWithSource decorator and WithSource helper for injecting CloudEvents source. | Custom MarshalJSON delegates to e.Event; if you embed an Event pointer elsewhere expecting normal struct marshaling you'll get the wrapper's behavior. |
| `source_test.go` | Round-trip test verifying source header set and unmarshal equality. | Reference example of a minimal Event implementation for tests. |

## Anti-Patterns

- Publishing a struct that doesn't implement Event — Marshal returns 'invalid event type' and Name silently degrades to UnknownEventName.
- Relying on the JSON payload for event type/routing instead of the ce_type metadata header.
- Returning a zero/empty Source from EventMetadata — NewCloudEvent rejects it.
- Putting routing/validation logic in Marshal callers instead of the event's Validate().

## Decisions

- **CloudEvents as the wire format with metadata mirrored into Watermill headers.** — Lets routing/metrics layers (eventbus, grouphandler, router) inspect ce_type without decoding the full JSON payload.

<!-- archie:ai-end -->
