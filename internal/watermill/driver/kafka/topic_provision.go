package kafka

import (
	"github.com/IBM/sarama"
	"github.com/openmeterio/openmeter/pkg/errorsx"
)

type AutoProvisionTopic struct {
	Topic         string
	NumPartitions int32
}

func provisionTopics(broker string, config *sarama.Config, topics []AutoProvisionTopic) error {
	admin, err := sarama.NewClusterAdmin([]string{broker}, config)
	if err != nil {
		return err
	}
	defer admin.Close()

	for _, topic := range topics {
		err := admin.CreateTopic(topic.Topic, &sarama.TopicDetail{
			NumPartitions:     topic.NumPartitions,
			ReplicationFactor: -1, // use default
		}, false)
		if err != nil {
			if topicError, ok := errorsx.ErrorAs[*sarama.TopicError](err); ok && topicError.Err == sarama.ErrTopicAlreadyExists {
				continue
			}

			return err
		}
	}

	return nil
}
