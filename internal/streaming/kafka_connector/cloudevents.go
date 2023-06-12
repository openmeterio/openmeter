package kafka_connector

import (
	"github.com/cloudevents/sdk-go/v2/event"
)

type CloudEventsKafkaPayload struct {
	Id      string `json:"ID"`
	Type    string `json:"TYPE"`
	Source  string `json:"SOURCE"`
	Subject string `json:"SUBJECT"`
	Time    string `json:"TIME"`
	Data    string `json:"DATA"`
}

func ToCloudEventsKafkaPayload(event event.Event) CloudEventsKafkaPayload {
	payload := CloudEventsKafkaPayload{
		Id:      event.ID(),
		Type:    event.Type(),
		Source:  event.Source(),
		Subject: event.Subject(),
		Time:    event.Time().String(),
		Data:    string(event.Data()),
	}

	return payload
}
