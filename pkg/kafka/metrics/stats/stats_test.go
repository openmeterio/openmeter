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

package stats

import (
	_ "embed"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
)

//go:embed testdata/stats.json
var statsJSON []byte

func TestStats(t *testing.T) {
	var stats Stats

	err := json.Unmarshal(statsJSON, &stats)
	if err != nil {
		t.Errorf("faield to unmarshal stats JSON: %v", err)
	}
	assert.Nil(t, err)

	assert.Equal(t, "rdkafka", stats.ClientID)
	assert.Equal(t, int64(22710), stats.MessageCount)

	brokerStats := []BrokerStats{
		{
			Name:              "localhost:9092/2",
			RequestsSent:      320,
			ResponsesReceived: 320,
			TopicPartitions: map[string]BrokerTopicPartition{
				"test-1": {
					Topic:     "test",
					Partition: 1,
				},
			},
		},
		{
			Name:              "localhost:9093/3",
			RequestsSent:      310,
			ResponsesReceived: 310,
			TopicPartitions: map[string]BrokerTopicPartition{
				"test-0": {
					Topic:     "test",
					Partition: 0,
				},
			},
		},
		{
			Name:              "localhost:9094/4",
			RequestsSent:      1,
			ResponsesReceived: 1,
			TopicPartitions:   map[string]BrokerTopicPartition{},
		},
	}
	for _, brokerStat := range brokerStats {
		s, ok := stats.Brokers[brokerStat.Name]
		assert.True(t, ok)
		assert.Equal(t, brokerStat.TopicPartitions, s.TopicPartitions)
		assert.Equal(t, brokerStat.RequestsSent, s.RequestsSent)
		assert.Equal(t, brokerStat.ResponsesReceived, s.ResponsesReceived)
	}

	expectedTopicStats := []TopicStats{
		{
			Topic:       "test",
			MetadataAge: 9060,
			Partitions: map[string]Partition{
				"0": {
					Partition:          0,
					Broker:             3,
					MessagesInQueue:    1,
					StoredOffset:       -1001,
					CommittedOffset:    -1001,
					ConsumerLag:        -1,
					TotalNumOfMessages: 2160510,
				},
				"1": {
					Partition:          1,
					Broker:             2,
					MessagesInQueue:    0,
					StoredOffset:       -1001,
					CommittedOffset:    -1001,
					ConsumerLag:        -1,
					TotalNumOfMessages: 2159735,
				},
			},
		},
	}
	for _, expectedTopicStat := range expectedTopicStats {
		topicStat, ok := stats.Topics[expectedTopicStat.Topic]
		assert.True(t, ok)
		assert.Equal(t, expectedTopicStat.MetadataAge, topicStat.MetadataAge)

		for expectedPartID, expectedPart := range expectedTopicStat.Partitions {
			topicPartition, ok := topicStat.Partitions[expectedPartID]
			assert.True(t, ok)

			assert.Equal(t, expectedPart.Partition, topicPartition.Partition)
			assert.Equal(t, expectedPart.Broker, topicPartition.Broker)
			assert.Equal(t, expectedPart.MessagesInQueue, topicPartition.MessagesInQueue)
			assert.Equal(t, expectedPart.StoredOffset, topicPartition.StoredOffset)
			assert.Equal(t, expectedPart.CommittedOffset, topicPartition.CommittedOffset)
			assert.Equal(t, expectedPart.ConsumerLag, topicPartition.ConsumerLag)
			assert.Equal(t, expectedPart.TotalNumOfMessages, topicPartition.TotalNumOfMessages)
		}
	}
}
