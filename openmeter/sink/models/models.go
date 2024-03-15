// Copyright © 2024 Tailfin Cloud Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package models

import (
	"fmt"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"

	"github.com/openmeterio/openmeter/openmeter/ingest/kafkaingest/serializer"
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
