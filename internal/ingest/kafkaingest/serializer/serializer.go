package serializer

import (
	_ "embed"
	"encoding/json"

	"github.com/cloudevents/sdk-go/v2/event"
)

type Serializer interface {
	SerializeKey(topic string, ev event.Event) ([]byte, error)
	SerializeValue(topic string, ev event.Event) ([]byte, error)
	GetKeySchemaId() *int
	GetValueSchemaId() *int
}

type CloudEventsKafkaPayload struct {
	Id      string      `json:"id"`
	Type    string      `json:"type"`
	Source  string      `json:"source"`
	Subject string      `json:"subject"`
	Time    int64       `json:"time"`
	Data    interface{} `json:"data"`
}

func toCloudEventsKafkaPayload(ev event.Event) (CloudEventsKafkaPayload, error) {
	payload := CloudEventsKafkaPayload{
		Id:      ev.ID(),
		Type:    ev.Type(),
		Source:  ev.Source(),
		Subject: ev.Subject(),
		Time:    ev.Time().UnixMilli(),
	}

	// We try to parse data as JSON.
	// CloudEvents data can be other than JSON but currently only support JSON data.
	var data interface{}
	err := json.Unmarshal(ev.Data(), &data)
	if err != nil {
		return payload, err
	}
	payload.Data = data

	return payload, nil
}
