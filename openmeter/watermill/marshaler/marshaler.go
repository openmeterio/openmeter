package marshaler

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/ThreeDotsLabs/watermill/components/cqrs"
	"github.com/ThreeDotsLabs/watermill/message"
	cloudevents "github.com/cloudevents/sdk-go/v2/event"
	"github.com/oklog/ulid/v2"
	"github.com/openmeterio/openmeter/internal/event/spec"
)

const (
	CloudEventsHeaderType    = "ce_type"
	CloudEventsHeaderTime    = "ce_time"
	CloudEventsHeaderSource  = "ce_source"
	CloudEventsHeaderSubject = "ce_subject"
)

type TransformFunc func(watermillIn *message.Message, cloudEvent cloudevents.Event) (*message.Message, error)

type event interface {
	EventName() string
	EventMetadata() spec.EventMetadata
	Validate() error
}

type marshaler struct{}

func New() cqrs.CommandEventMarshaler {
	return &marshaler{}
}

func (m *marshaler) Marshal(v interface{}) (*message.Message, error) {
	ev, ok := v.(event)
	if !ok {
		return nil, errors.New("invalid event type")
	}

	// cloud events object
	ce, err := NewCloudEvent(ev)
	if err != nil {
		return nil, err
	}

	ceBytes, err := ce.MarshalJSON()
	if err != nil {
		return nil, err
	}

	// watermill message
	msg := message.NewMessage(ce.ID(), ceBytes)

	msg.Metadata.Set(CloudEventsHeaderType, ce.Type())
	msg.Metadata.Set(CloudEventsHeaderTime, ce.Time().In(time.UTC).Format(time.RFC3339))
	msg.Metadata.Set(CloudEventsHeaderSource, ce.Source())
	if ce.Subject() != "" {
		msg.Metadata.Set(CloudEventsHeaderSubject, ce.Subject())
	}

	/*
		// TODO!
			if m.transform != nil {
				msg, err = m.transform(msg, event)
				if err != nil {
					return nil, err
				}
			}*/

	return msg, nil
}

func NewCloudEvent(ev event) (cloudevents.Event, error) {
	metadata := ev.EventMetadata()
	// Mandatory cloud events fields
	if metadata.Source == "" {
		return cloudevents.Event{}, errors.New("source is required")
	}

	cloudEvent := cloudevents.New()
	cloudEvent.SetType(ev.EventName())
	cloudEvent.SetSpecVersion("1.0")

	if metadata.Time.IsZero() {
		cloudEvent.SetTime(time.Now())
	} else {
		cloudEvent.SetTime(metadata.Time)
	}

	if metadata.ID == "" {
		cloudEvent.SetID(ulid.Make().String())
	} else {
		cloudEvent.SetID(metadata.ID)
	}

	cloudEvent.SetSource(metadata.Source)

	cloudEvent.SetSubject(metadata.Subject)

	if err := ev.Validate(); err != nil {
		return cloudevents.Event{}, err
	}

	if err := cloudEvent.SetData("application/json", ev); err != nil {
		return cloudevents.Event{}, err
	}
	return cloudEvent, nil
}

func (m *marshaler) Unmarshal(msg *message.Message, v interface{}) error {
	cloudEvent := cloudevents.Event{}
	if err := cloudEvent.UnmarshalJSON(msg.Payload); err != nil {
		return fmt.Errorf("failed to unmarshal CloudEvent: %w", err)
	}

	return json.Unmarshal(cloudEvent.Data(), v)
}

func (m *marshaler) Name(v interface{}) string {
	ev, ok := v.(event)
	if !ok {
		// TODO: how to report error
		return "TODO"
	}

	return ev.EventName()
}

func (m *marshaler) NameFromMessage(msg *message.Message) string {
	return msg.Metadata.Get(CloudEventsHeaderType)
}
