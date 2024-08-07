package marshaler

import (
	"errors"

	"github.com/openmeterio/openmeter/internal/event/spec"
)

type eventWithSource struct {
	Event `json:",inline"`

	source string `json:"-"`
}

func WithSource(source string, ev Event) Event {
	return &eventWithSource{
		source: source,
		Event:  ev,
	}
}

func (e *eventWithSource) EventMetadata() spec.EventMetadata {
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
