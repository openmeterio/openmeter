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

package sink

import (
	"fmt"
	"sync"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"

	sinkmodels "github.com/openmeterio/openmeter/openmeter/sink/models"
)

type SinkBuffer struct {
	mu   sync.Mutex
	data map[string]sinkmodels.SinkMessage
}

func NewSinkBuffer() *SinkBuffer {
	return &SinkBuffer{
		data: map[string]sinkmodels.SinkMessage{},
	}
}

func (b *SinkBuffer) Size() int {
	b.mu.Lock()
	defer b.mu.Unlock()
	return len(b.data)
}

func (b *SinkBuffer) Add(message sinkmodels.SinkMessage) {
	b.mu.Lock()
	defer b.mu.Unlock()
	// Unique identifier for each message (topic + partition + offset)
	key := message.KafkaMessage.String()
	b.data[key] = message
}

func (b *SinkBuffer) Dequeue() []sinkmodels.SinkMessage {
	b.mu.Lock()
	defer b.mu.Unlock()
	list := []sinkmodels.SinkMessage{}
	for key, message := range b.data {
		list = append(list, message)
		delete(b.data, key)
	}
	return list
}

// RemoveByPartitions removes messages from the buffer by partitions
// Useful when partitions are revoked.
func (b *SinkBuffer) RemoveByPartitions(partitions []kafka.TopicPartition) {
	b.mu.Lock()
	defer b.mu.Unlock()

	partitionMap := map[string]bool{}
	for _, topicPartition := range partitions {
		key := topicPartitionKey(topicPartition)
		partitionMap[key] = true
	}

	for key, message := range b.data {
		topicKey := topicPartitionKey(message.KafkaMessage.TopicPartition)

		if partitionMap[topicKey] {
			delete(b.data, key)
		}
	}
}

func topicPartitionKey(partition kafka.TopicPartition) string {
	var topic string
	if partition.Topic != nil {
		topic = *partition.Topic
	}
	return partitionKey(topic, partition.Partition)
}

func partitionKey(topic string, partition int32) string {
	return fmt.Sprintf("%s-%d", topic, partition)
}
