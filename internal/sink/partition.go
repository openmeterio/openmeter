package sink

import (
	"fmt"
	"sort"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
)

func prettyPartitions(partitions []kafka.TopicPartition) []string {
	out := make([]string, 0, len(partitions))

	for _, partition := range partitions {
		var topicName string

		if partition.Topic != nil {
			topicName = *partition.Topic
		}

		out = append(out, fmt.Sprintf("%s-%d", topicName, partition.Partition))
	}

	sort.Strings(out)

	return out
}
