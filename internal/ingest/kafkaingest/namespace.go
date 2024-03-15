package kafkaingest

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
)

// NamespaceHandler is a namespace handler for Kafka ingest topics.
type NamespaceHandler struct {
	AdminClient *kafka.AdminClient

	// NamespacedTopicTemplate needs to contain at least one string parameter passed to fmt.Sprintf.
	// For example: "om_%s_events"
	NamespacedTopicTemplate string

	Partitions int

	Logger *slog.Logger
}

// CreateNamespace implements the namespace handler interface.
func (h NamespaceHandler) CreateNamespace(ctx context.Context, namespace string) error {
	topic := h.getTopicName(namespace)
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
			h.Logger.Debug(fmt.Sprintf("topic already exists: %s", topic))
		} else if code != kafka.ErrNoError {
			return r.Error
		}
	}

	return nil
}

// DeleteNamespace implements the namespace handler interface.
func (h NamespaceHandler) DeleteNamespace(ctx context.Context, namespace string) error {
	topic := h.getTopicName(namespace)
	result, err := h.AdminClient.DeleteTopics(ctx, []string{topic})
	if err != nil {
		return err
	}
	for _, r := range result {
		if r.Error.Code() != kafka.ErrNoError {
			return r.Error
		}
	}

	return nil
}

func (h NamespaceHandler) getTopicName(namespace string) string {
	topic := fmt.Sprintf(h.NamespacedTopicTemplate, namespace)
	return topic
}
