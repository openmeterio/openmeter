package spec

import (
	"errors"
	"fmt"
	"time"

	"github.com/cloudevents/sdk-go/v2/event"
	"github.com/oklog/ulid/v2"
)

type CloudEventsPayload interface {
	Spec() *EventTypeSpec
	Validate() error
}

// NewCloudEvent creates a new CloudEvent with the given event spec and payload
// example usage:
//
//	 ev, err := CreateCloudEvent(EventSpec{
//					ID:    "123",
//					Source: "test",
//	 }, IngestEvent{...})
func NewCloudEvent(eventSpec EventSpec, payload CloudEventsPayload) (event.Event, error) {
	// Mandatory cloud events fields
	if eventSpec.Source == "" {
		return event.Event{}, errors.New("source is required")
	}

	meta := payload.Spec()
	ev := newCloudEventFromSpec(meta, eventSpec)

	if err := payload.Validate(); err != nil {
		return event.Event{}, err
	}

	if err := ev.SetData("application/json", payload); err != nil {
		return event.Event{}, err
	}
	return ev, nil
}

// newCloudEventFromSpec generates a new cloudevents without data being set based on the event spec
func newCloudEventFromSpec(meta *EventTypeSpec, spec EventSpec) event.Event {
	ev := event.New()
	ev.SetType(meta.Type())
	ev.SetSpecVersion(string(meta.SpecVersion))

	if spec.Time.IsZero() {
		ev.SetTime(time.Now())
	} else {
		ev.SetTime(spec.Time)
	}

	if spec.ID == "" {
		ev.SetID(ulid.Make().String())
	} else {
		ev.SetID(spec.ID)
	}

	ev.SetSource(spec.Source)

	ev.SetSubject(spec.Subject)
	return ev
}

// ParseCloudEvent unmarshals a single CloudEvent into the given payload
// example usage:
// ingest, err := UnmarshalCloudEvent[schema.IngestEvent](ev)
func ParseCloudEvent[PayloadType CloudEventsPayload](ev event.Event) (PayloadType, error) {
	var payload PayloadType

	expectedType := payload.Spec().Type()
	if expectedType != ev.Type() {
		return payload, fmt.Errorf("cannot parse cloud event type %s as %s (expected by target payload)", ev.Type(), expectedType)
	}

	if err := ev.DataAs(&payload); err != nil {
		return payload, err
	}

	if err := payload.Validate(); err != nil {
		return payload, err
	}

	return payload, nil
}

type ParsedCloudEvent[PayloadType CloudEventsPayload] struct {
	Event   event.Event
	Payload PayloadType
}

func ParseCloudEventFromBytes[PayloadType CloudEventsPayload](data []byte) (*ParsedCloudEvent[PayloadType], error) {
	cloudEvent := event.Event{}
	if err := cloudEvent.UnmarshalJSON(data); err != nil {
		return nil, fmt.Errorf("failed to unmarshal CloudEvent: %w", err)
	}

	eventBody, err := ParseCloudEvent[PayloadType](cloudEvent)
	if err != nil {
		return nil, fmt.Errorf("failed to parse payload: %w", err)
	}

	return &ParsedCloudEvent[PayloadType]{Event: cloudEvent, Payload: eventBody}, nil
}
