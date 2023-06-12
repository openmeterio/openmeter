package kafka_connector

import (
	"github.com/cloudevents/sdk-go/v2/event"
)

type CloudEventsKafkaPayload struct {
	Id      *string `json:"ID"`
	Type    *string `json:"TYPE"`
	Source  *string `json:"SOURCE"`
	Subject string  `json:"SUBJECT"`
	Time    *string `json:"TIME"`
	Data    *string `json:"DATA"`
}

func ToCloudEventsKafkaPayload(event event.Event) CloudEventsKafkaPayload {
	eid := event.ID()
	ety := event.Type()
	eso := event.Source()
	esu := event.Subject()
	eti := event.Time().String()
	ed := string(event.Data())

	payload := CloudEventsKafkaPayload{
		Id:      &eid,
		Type:    &ety,
		Source:  &eso,
		Subject: esu,
		Time:    &eti,
		Data:    &ed,
	}

	return payload
}
