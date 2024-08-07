package ingestnotification

import (
	"log/slog"

	"go.opentelemetry.io/otel/metric"

	"github.com/openmeterio/openmeter/internal/sink/flushhandler/ingestnotification"
	"github.com/openmeterio/openmeter/openmeter/sink/flushhandler"
	"github.com/openmeterio/openmeter/openmeter/watermill/eventbus"
)

// Event types
type (
	HandlerConfig = ingestnotification.HandlerConfig
)

// Ingest notification handler
func NewHandler(logger *slog.Logger, metricMeter metric.Meter, publisher eventbus.Publisher, config ingestnotification.HandlerConfig) (flushhandler.FlushEventHandler, error) {
	return ingestnotification.NewHandler(logger, metricMeter, publisher, config)
}
