package kafkaingest

import (
	"context"
	"errors"
	"fmt"

	"github.com/openmeterio/openmeter/openmeter/ingest/kafkaingest/topicresolver"
	pkgkafka "github.com/openmeterio/openmeter/pkg/kafka"
)

// NamespaceHandler is a namespace handler for Kafka ingest topics.
type NamespaceHandler struct {
	TopicResolver    topicresolver.Resolver
	TopicProvisioner pkgkafka.TopicProvisioner

	Partitions int
}

// CreateNamespace implements the namespace handler interface.
func (h NamespaceHandler) CreateNamespace(ctx context.Context, namespace string) error {
	if h.TopicResolver == nil {
		return errors.New("topic name resolver must not be nil")
	}

	topicName, err := h.TopicResolver.Resolve(ctx, namespace)
	if err != nil {
		return fmt.Errorf("failed to resolve namespace %q to topic name: %w", namespace, err)
	}

	err = h.TopicProvisioner.Provision(ctx, pkgkafka.TopicConfig{
		Name:       topicName,
		Partitions: h.Partitions,
	})
	if err != nil {
		return fmt.Errorf("failed to provision topic %q: %s", topicName, err)
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

	err = h.TopicProvisioner.DeProvision(ctx, topicName)
	if err != nil {
		return fmt.Errorf("failed to deprovision kafka topic %q: %w", topicName, err)
	}

	return nil
}
