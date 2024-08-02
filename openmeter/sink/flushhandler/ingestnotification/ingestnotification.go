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
)

// Ingest notification handler
func NewHandler(logger *slog.Logger, metricMeter metric.Meter, publisher publisher.TopicPublisher, config ingestnotification.HandlerConfig) (flushhandler.FlushEventHandler, error) {
	return ingestnotification.NewHandler(logger, metricMeter, publisher, config)
}
