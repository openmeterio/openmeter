package kafkaingest

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"

	"github.com/openmeterio/openmeter/openmeter/ingest/kafkaingest/topicresolver"
	pkgkafka "github.com/openmeterio/openmeter/pkg/kafka"
)

// NamespaceHandler is a namespace handler for Kafka ingest topics.
type NamespaceHandler struct {
	AdminClient   *kafka.AdminClient
	TopicResolver topicresolver.Resolver

	Partitions int

	Logger *slog.Logger
}

// CreateNamespace implements the namespace handler interface.
func (h NamespaceHandler) CreateNamespace(ctx context.Context, namespace string) error {
	if h.TopicResolver == nil {
		return errors.New("topic name resolver must not be nil")
	}

	topicName, err := h.TopicResolver.Resolve(ctx, namespace)
	if err != nil {
		return fmt.Errorf("failed to resolve namespace to topic name: %w", err)
	}

	err = pkgkafka.ProvisionTopics(ctx, h.AdminClient, pkgkafka.TopicConfig{
		Name:       topicName,
		Partitions: h.Partitions,
	})
	if err != nil {
		return fmt.Errorf("failed to provision topic %s: %s", topicName, err)
	}

	return nil
}

// DeleteNamespace implements the namespace handler interface.
func (h NamespaceHandler) DeleteNamespace(ctx context.Context, namespace string) error {
	if h.TopicResolver == nil {
		return errors.New("topic name resolver must not be nil")
	}

	topicName, err := h.TopicResolver.Resolve(ctx, namespace)
	if err != nil {
		return fmt.Errorf("failed to resolve namespace to topic name: %w", err)
	}

	result, err := h.AdminClient.DeleteTopics(ctx, []string{topicName})
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
