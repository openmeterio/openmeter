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

package kafka

import (
	"context"
	"log/slog"
	"strconv"
	"time"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
)

type AutoProvisionTopic struct {
	Topic         string
	NumPartitions int32
	Retention     time.Duration
}

// provisionTopics creates the topics if they don't exist. This relies on the confluent kafka lib, as the sarama doesn't seem to
// properly support interacting with the confluent cloud.
// FIXME(chrisgacsal): use kafka.ProvisionTopics() from openmeter/pkg/kafka
func provisionTopics(ctx context.Context, logger *slog.Logger, config kafka.ConfigMap, topics []AutoProvisionTopic) error {
	// This is not supported on admin client, so we need to remove it
	delete(config, "go.logs.channel.enable")

	adminClient, err := kafka.NewAdminClient(&config)
	if err != nil {
		return err
	}

	defer adminClient.Close()

	for _, topic := range topics {
		topicConfig := map[string]string{}
		if topic.Retention > 0 {
			topicConfig["retention.ms"] = strconv.FormatInt(topic.Retention.Milliseconds(), 10)
		}
		result, err := adminClient.CreateTopics(ctx, []kafka.TopicSpecification{
			{
				Topic:         topic.Topic,
				NumPartitions: int(topic.NumPartitions),
				Config:        topicConfig,
			},
		})
		if err != nil {
			return err
		}

		for _, r := range result {
			code := r.Error.Code()

			if code == kafka.ErrTopicAlreadyExists {
				logger.Debug("topic already exists", slog.String("topic", topic.Topic))
			} else if code != kafka.ErrNoError {
				return r.Error
			}
		}
	}

	return nil
}
