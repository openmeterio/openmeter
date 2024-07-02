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
	"log/slog"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"go.opentelemetry.io/otel/metric"

	"github.com/openmeterio/openmeter/internal/ingest/kafkaingest"
	"github.com/openmeterio/openmeter/openmeter/ingest/kafkaingest/serializer"
	kafkametrics "github.com/openmeterio/openmeter/pkg/kafka/metrics"
)

// Collector is a receiver of events that handles sending those events to a downstream Kafka broker.
type Collector = kafkaingest.Collector

func KafkaProducerGroup(ctx context.Context, producer *kafka.Producer, logger *slog.Logger, kafkaMetrics *kafkametrics.Metrics) (execute func() error, interrupt func(error)) {
	return kafkaingest.KafkaProducerGroup(ctx, producer, logger, kafkaMetrics)
}

func NewCollector(
	producer *kafka.Producer,
	serializer serializer.Serializer,
	namespacedTopicTemplate string,
	metricMeter metric.Meter,
) (*Collector, error) {
	return kafkaingest.NewCollector(producer, serializer, namespacedTopicTemplate, metricMeter)
}
