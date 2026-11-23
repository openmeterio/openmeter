package metrics

import (
	"context"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"

	"github.com/openmeterio/openmeter/openmeter/rte"
)

type ProducerEventOutcome string

const (
	ProducerEventOutcomeInvalid    ProducerEventOutcome = "invalid"
	ProducerEventOutcomeSampledOut ProducerEventOutcome = "sampled_out"
	ProducerEventOutcomeStopped    ProducerEventOutcome = "stopped"
	ProducerEventOutcomeQueueFull  ProducerEventOutcome = "queue_full"
	ProducerEventOutcomeExpired    ProducerEventOutcome = "expired"
	ProducerEventOutcomeFailed     ProducerEventOutcome = "failed"
	ProducerEventOutcomePublished  ProducerEventOutcome = "published"
)

// ProducerEvents counts observations and source events by their outcome on the producer.
// Successful publication does not imply the worker has handled the event.
type ProducerEvents struct {
	attempts metric.Int64Counter
}

func NewProducerEvents(meter metric.Meter) (*ProducerEvents, error) {
	attempts, err := meter.Int64Counter("rte.comparison.publications", metric.WithDescription("Observation and source event delivery attempts by outcome"))
	if err != nil {
		return nil, err
	}

	return &ProducerEvents{attempts: attempts}, nil
}

// Record counts observations by kind and operation. Source facts have neither;
// one fact can affect multiple kinds and is recorded with event=source.
func (m *ProducerEvents) Record(ctx context.Context, kind rte.CalculationKind, operation rte.ComparisonOperation, outcome ProducerEventOutcome) {
	event := "observation"
	if operation == "" {
		event = "source"
	}

	m.attempts.Add(ctx, 1, metric.WithAttributes(attribute.String("event", event),
		attribute.String("kind", string(kind)), attribute.String("operation", string(operation)),
		attribute.String("outcome", string(outcome))))
}
