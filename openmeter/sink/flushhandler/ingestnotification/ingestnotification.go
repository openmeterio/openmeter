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

package ingestnotification

import (
	"log/slog"

	"go.opentelemetry.io/otel/metric"

	"github.com/openmeterio/openmeter/internal/sink/flushhandler/ingestnotification"
	"github.com/openmeterio/openmeter/openmeter/event/publisher"
	"github.com/openmeterio/openmeter/openmeter/sink/flushhandler"
)

// Event types
const (
	EventSubsystem = ingestnotification.EventSubsystem
)

type (
	IngestEventData    = ingestnotification.IngestEventData
	EventBatchedIngest = ingestnotification.EventBatchedIngest
	HandlerConfig      = ingestnotification.HandlerConfig
)

// Ingest notification handler
func NewHandler(logger *slog.Logger, metricMeter metric.Meter, publisher publisher.TopicPublisher, config ingestnotification.HandlerConfig) (flushhandler.FlushEventHandler, error) {
	return ingestnotification.NewHandler(logger, metricMeter, publisher, config)
}
