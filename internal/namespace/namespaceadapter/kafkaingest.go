package namespaceadapter

import (
	"context"
	"fmt"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"golang.org/x/exp/slog"
)

// KafkaIngestHandler is a namespace handler for Kafka ingest topics.
type KafkaIngestHandler struct {
	AdminClient *kafka.AdminClient

	DefaultTopic string

	// NamespacedTopicTemplate needs to contain at least one string parameter passed to fmt.Sprintf.
	// For example: "om_%s_events"
	NamespacedTopicTemplate string

	Partitions int

	Logger *slog.Logger
}

// CreateNamespace implements the namespace handler interface.
func (h KafkaIngestHandler) CreateNamespace(ctx context.Context, name string) error {
	topic := h.DefaultTopic

	if name != "" {
		topic = fmt.Sprintf(h.NamespacedTopicTemplate, name)
	}

	result, err := h.AdminClient.CreateTopics(ctx, []kafka.TopicSpecification{
		{
			Topic:         topic,
			NumPartitions: h.Partitions,
		},
	})
	if err != nil {
		return err
	}

	for _, r := range result {
		code := r.Error.Code()

		if code == kafka.ErrTopicAlreadyExists {
			h.Logger.Debug("topic already exists", slog.String("topic", topic))
		} else if code != kafka.ErrNoError {
			return r.Error
		}
	}

	return nil
}
