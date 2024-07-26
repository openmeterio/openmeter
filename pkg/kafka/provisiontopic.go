package kafka

import (
	"context"
	"log/slog"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
)

func ProvisionTopic(ctx context.Context, adminClient *kafka.AdminClient, logger *slog.Logger, topic string, partitions int) error {
	result, err := adminClient.CreateTopics(ctx, []kafka.TopicSpecification{
		{
			Topic:         topic,
			NumPartitions: partitions,
		},
	})
	if err != nil {
		return err
	}

	for _, r := range result {
		code := r.Error.Code()

		if code == kafka.ErrTopicAlreadyExists {
			logger.Debug("topic already exists", slog.String("topic", topic))
		} else if code != kafka.ErrNoError {
			return r.Error
		}
	}

	return nil
}
