package models

import (
	"fmt"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"

	"github.com/openmeterio/openmeter/internal/ingest/kafkaingest/serializer"
	"github.com/openmeterio/openmeter/pkg/models"
)

type SinkMessage struct {
	Namespace    string
	KafkaMessage *kafka.Message
	Serialized   *serializer.CloudEventsKafkaPayload
	Status       ProcessingStatus
	// Meters contains the list of meters this message affects
	Meters []models.Meter
}

type ProcessingState int8

func (c ProcessingState) String() string {
	var state string
	switch c {
	case OK:
		state = "ok"
	case INVALID:
		state = "invalid"
	case DROP:
		state = "drop"
	default:
		state = fmt.Sprintf("unknown(%d)", c)
	}

	return state
}

const (
	OK ProcessingState = iota
	DROP
	INVALID
)

type ProcessingStatus struct {
	State ProcessingState
	Error error
}
