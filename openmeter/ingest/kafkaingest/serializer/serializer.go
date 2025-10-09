package serializer

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/cloudevents/sdk-go/v2/event"
)

type Serializer interface {
	SerializeKey(topic string, namespace string, ev event.Event) ([]byte, error)
	SerializeValue(topic string, ev event.Event) ([]byte, error)
	GetFormat() string
	GetKeySchemaId() int
	GetValueSchemaId() int
}

type CloudEventsKafkaPayload struct {
	Id      string `json:"id"`
	Type    string `json:"type"`
	Source  string `json:"source"`
	Subject string `json:"subject"`
	// Note: By converting to unix timestamp we loose timezone information.
	Time int64  `json:"time"`
	Data string `json:"data"`
}

// ToCloudEventsKafkaPayload serializes a CloudEvent to a CloudEventsKafkaPayload.
func toCloudEventsKafkaPayload(ev event.Event) (CloudEventsKafkaPayload, error) {
	payload := CloudEventsKafkaPayload{
		Id:      ev.ID(),
		Type:    ev.Type(),
		Source:  ev.Source(),
		Subject: ev.Subject(),
		Time:    ev.Time().Unix(),
	}

	// Data is optional in CloudEvents.
	if len(ev.Data()) > 0 {
		// Parse CloudEvents data.
		var data interface{}
		err := ev.DataAs(&data)
		if err != nil {
			return payload, errors.New("cannot unmarshal cloudevents data")
		}

		// Serialize data to JSON.
		// We only support JSON seralizable data for now.
		payloadData, err := json.Marshal(data)
		if err != nil {
			return payload, errors.New("cannot json serialize cloudevents data")
		}

		payload.Data = string(payloadData)
	}

	return payload, nil
}

// FromKafkaPayloadToCloudEvents deserialized a CloudEventsKafkaPayload to a CloudEvent.
func FromKafkaPayloadToCloudEvents(payload CloudEventsKafkaPayload) (event.Event, error) {
	ev := event.New()

	ev.SetID(payload.Id)
	ev.SetType(payload.Type)
	ev.SetSource(payload.Source)
	ev.SetSubject(payload.Subject)
	ev.SetTime(time.Unix(payload.Time, 0))

	// Data is optional in CloudEvents.
	if payload.Data != "" {
		var data interface{}

		err := json.Unmarshal([]byte(payload.Data), &data)
		if err != nil {
			return event.Event{}, fmt.Errorf("cannot parse kafka payload data as json: %w", err)
		}

		err = ev.SetData(event.ApplicationJSON, data)
		if err != nil {
			return event.Event{}, fmt.Errorf("cannot set cloudevents data: %w", err)
		}
	}

	return ev, nil
}

func ValidateKafkaPayloadToCloudEvent(ce CloudEventsKafkaPayload) error {
	var errs []error

	if ce.Id == "" {
		errs = append(errs, errors.New("id is empty"))
	}

	if ce.Type == "" {
		errs = append(errs, errors.New("type is empty"))
	}

	if ce.Source == "" {
		errs = append(errs, errors.New("source is empty"))
	}

	if ce.Subject == "" {
		errs = append(errs, errors.New("subject is empty"))
	}

	if ce.Time <= 0 {
		errs = append(errs, errors.New("time is zero"))
	}

	return errors.Join(errs...)
}
