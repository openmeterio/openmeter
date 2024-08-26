package marshaler

import (
	"encoding/json"
	"errors"

	"github.com/openmeterio/openmeter/openmeter/event/metadata"
)

type eventWithSource struct {
	Event `json:",inline"`

	source string `json:"-"`
}

// WithSource can be used to add the CloudEvents source field to an event.
func WithSource(source string, ev Event) Event {
	return &eventWithSource{
		source: source,
		Event:  ev,
	}
}

func (e *eventWithSource) EventMetadata() metadata.EventMetadata {
	metadata := e.Event.EventMetadata()
	metadata.Source = e.source

	return metadata
}

func (e *eventWithSource) Validate() error {
	if err := e.Event.Validate(); err != nil {
		return err
	}

	if e.source == "" {
		return errors.New("source must be set")
	}

	return nil
}

func (e *eventWithSource) EventName() string {
	return e.Event.EventName()
}

// MarshalJSON marshals the event only, as JSON library embeds the Event name into the output,
// if the composed object is a pointer to an interface. (e.g. we would get "Event": {} in the payload)
func (e *eventWithSource) MarshalJSON() ([]byte, error) {
	return json.Marshal(e.Event)
}
