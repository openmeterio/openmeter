package sink

import (
	"fmt"
	"sort"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
)

func prettyPartitions(partitions []kafka.TopicPartition) []string {
	out := []string{}

	for i := range partitions {
		topicName := ""
		if partitions[i].Topic != nil {
			topicName = *partitions[i].Topic
		}
		out = append(out, fmt.Sprintf("%s-%d", topicName, partitions[i].Partition))
	}

	sort.Strings(out)
	return out
}
