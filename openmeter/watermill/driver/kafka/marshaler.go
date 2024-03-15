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

package kafka

import (
	"github.com/IBM/sarama"
	"github.com/ThreeDotsLabs/watermill-kafka/v3/pkg/kafka"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/cloudevents/sdk-go/v2/event"
	"github.com/samber/lo"
)

const (
	PartitionKeyMetadataKey = "x-kafka-partition-key"
)

type marshalerWithPartitionKey struct {
	kafka.DefaultMarshaler
}

func (m marshalerWithPartitionKey) Marshal(topic string, msg *message.Message) (*sarama.ProducerMessage, error) {
	kafkaMsg, err := m.DefaultMarshaler.Marshal(topic, msg)
	if err != nil {
		return nil, err
	}

	partitionKey := msg.Metadata.Get(PartitionKeyMetadataKey)
	if partitionKey != "" {
		kafkaMsg.Key = sarama.ByteEncoder(partitionKey)
		kafkaMsg.Headers = lo.Filter(kafkaMsg.Headers, func(header sarama.RecordHeader, _ int) bool {
			return string(header.Key) != PartitionKeyMetadataKey
		})
	}

	return kafkaMsg, nil
}

// AddPartitionKeyFromSubject adds partition key to the message based on the CloudEvent subject.
func AddPartitionKeyFromSubject(watermillIn *message.Message, cloudEvent event.Event) (*message.Message, error) {
	if cloudEvent.Subject() != "" {
		watermillIn.Metadata[PartitionKeyMetadataKey] = cloudEvent.Subject()
	}
	return watermillIn, nil
}
