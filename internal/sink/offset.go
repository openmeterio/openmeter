package sink

import "github.com/confluentinc/confluent-kafka-go/v2/kafka"

// OffsetStore helps to determinate the next offset to commit
type OffsetStore struct {
	topics map[string]*PartitionOffsets
}

type PartitionOffsets struct {
	partitions map[int32]*Offset
}

type Offset struct {
	Offset int64
}

func NewOffsetStore() *OffsetStore {
	return &OffsetStore{
		topics: map[string]*PartitionOffsets{},
	}
}

func (o *OffsetStore) Add(topicPartition kafka.TopicPartition) {
	topic := *topicPartition.Topic
	partition := topicPartition.Partition
	offset := int64(topicPartition.Offset)

	if o.topics[topic] == nil {
		o.topics[topic] = &PartitionOffsets{
			partitions: map[int32]*Offset{},
		}
	}
	if o.topics[topic].partitions[partition] == nil {
		o.topics[topic].partitions[partition] = &Offset{Offset: offset}
	}

	if o.topics[topic].partitions[partition].Offset < offset {
		o.topics[topic].partitions[partition] = &Offset{Offset: offset}
	}
}

func (o *OffsetStore) Get() []kafka.TopicPartition {
	offsets := []kafka.TopicPartition{}
	for topic, t := range o.topics {
		for partition, p := range t.partitions {
			metadata := ""
			offsets = append(offsets, kafka.TopicPartition{
				Topic:     &topic,
				Partition: partition,
				Metadata:  &metadata,
				// We increase latest offset by one
				Offset: kafka.Offset(p.Offset + 1),
			})
		}
	}
	return offsets
}
