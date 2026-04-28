package models

import (
	"fmt"
	"time"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"

	"github.com/openmeterio/openmeter/openmeter/dedupe"
	"github.com/openmeterio/openmeter/openmeter/ingest/kafkaingest/serializer"
	"github.com/openmeterio/openmeter/openmeter/meter"
)

type SinkMessage struct {
	Namespace    string
	KafkaMessage *kafka.Message
	Serialized   *serializer.CloudEventsKafkaPayload
	Status       ProcessingStatus
	// Meters contains the list of meters this message affects
	Meters []*meter.Meter

	// IngestedAt is the time this message was ingested
	IngestedAt *time.Time
	// StoredAt is the time this message was stored
	StoredAt *time.Time
}

func (m SinkMessage) GetDedupeItem() dedupe.Item {
	return dedupe.Item{
		Namespace: m.Namespace,
		ID:        m.Serialized.Id,
		Source:    m.Serialized.Source,
	}
}

type ProcessingState int8

func (c ProcessingState) String() string {
	var state string
	switch c {
	case OK:
		state = "ok"
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
)

type ProcessingStatus struct {
	State     ProcessingState
	DropError error
}
