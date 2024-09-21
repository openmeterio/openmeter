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

package kafkaingest

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"

	pkgkafka "github.com/openmeterio/openmeter/pkg/kafka"
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
	topicName := h.getTopicName(namespace)

	err := pkgkafka.ProvisionTopics(ctx, h.AdminClient, pkgkafka.TopicConfig{
		Name:       h.getTopicName(namespace),
		Partitions: h.Partitions,
	})
	if err != nil {
		return fmt.Errorf("failed to provision topic %s: %s", topicName, err)
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
