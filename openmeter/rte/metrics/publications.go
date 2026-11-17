package metrics

import (
	"context"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"

	"github.com/openmeterio/openmeter/openmeter/rte"
)

type PublicationOutcome string

const (
	PublicationOutcomeInvalid    PublicationOutcome = "invalid"
	PublicationOutcomeSampledOut PublicationOutcome = "sampled_out"
	PublicationOutcomeStopped    PublicationOutcome = "stopped"
	PublicationOutcomeQueueFull  PublicationOutcome = "queue_full"
	PublicationOutcomeExpired    PublicationOutcome = "expired"
	PublicationOutcomeFailed     PublicationOutcome = "failed"
	PublicationOutcomePublished  PublicationOutcome = "published"
)

// Publications records producer-side sampling, drops and publication results.
// Successful publication does not imply the worker has handled the event.
type Publications struct {
	attempts metric.Int64Counter
}

func NewPublications(meter metric.Meter) (*Publications, error) {
	attempts, err := meter.Int64Counter("rte.comparison.publications", metric.WithDescription("Observation and source event delivery attempts by outcome"))
	if err != nil {
		return nil, err
	}

	return &Publications{attempts: attempts}, nil
}

// Record counts observations by kind and operation. Source facts have neither;
// one fact can affect multiple kinds and is recorded with event=source.
func (m *Publications) Record(ctx context.Context, kind rte.CalculationKind, operation rte.ComparisonOperation, outcome PublicationOutcome) {
	event := "observation"
	if operation == "" {
		event = "source"
	}

	m.attempts.Add(ctx, 1, metric.WithAttributes(attribute.String("event", event),
		attribute.String("kind", string(kind)), attribute.String("operation", string(operation)),
		attribute.String("outcome", string(outcome))))
}
