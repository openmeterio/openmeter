package types

import (
	"errors"
	"fmt"

	"github.com/cloudevents/sdk-go/v2/event"
)

type CloudEventsPayload interface {
	Spec() *EventTypeSpec
}

type PayloadWithValidation interface {
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

	if eventSpec.ID == "" {
		return event.Event{}, errors.New("id is required")
	}

	meta := payload.Spec()

	if validator, ok := payload.(PayloadWithValidation); ok {
		if err := validator.Validate(); err != nil {
			return event.Event{}, err
		}
	}

	ev := event.New()
	meta.FillEvent(ev)
	eventSpec.FillEvent(ev, meta.SubjectKind)

	if err := ev.SetData("application/json", payload); err != nil {
		return event.Event{}, err
	}
	return ev, nil
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

	// TODO: this is a hack to avoid having to add Validate() to all payloads
	// later we should add Validate() to all payloads or have a generator that solves it
	// for us
	var payloadAny any = payload
	if validator, ok := payloadAny.(PayloadWithValidation); ok {
		if err := validator.Validate(); err != nil {
			return payload, err
		}
	}

	return payload, nil
}
