package metrics

import (
	"context"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"

	"github.com/openmeterio/openmeter/openmeter/rte"
)

type ReadOutcome string

const (
	ReadOutcomeHit     ReadOutcome = "hit"
	ReadOutcomeMissing ReadOutcome = "missing"
	ReadOutcomeFailed  ReadOutcome = "failed"
)

// StateReads records stored-state access without requiring calculation instruments.
// Both the database-backed client and the calculation service use this recorder.
type StateReads struct {
	kind         rte.CalculationKind
	reads        metric.Int64Counter
	readDuration metric.Float64Histogram
}

func NewStateReads(meter metric.Meter, kind rte.CalculationKind) (StateReads, error) {
	m := StateReads{kind: kind}
	var err error
	m.reads, err = meter.Int64Counter("rte.state.reads", metric.WithDescription("Stored balance reads, by outcome"))
	if err != nil {
		return m, err
	}

	m.readDuration, err = meter.Float64Histogram("rte.state.read.duration", durationBuckets(), metric.WithUnit("s"), metric.WithDescription("Elapsed time to read stored state"))

	return m, err
}

// Read records one stored-state access through a client or calculation service.
func (m StateReads) Read(ctx context.Context, start time.Time, outcome ReadOutcome) {
	labels := metric.WithAttributes(attribute.String("kind", string(m.kind)), attribute.String("outcome", string(outcome)))
	m.reads.Add(ctx, 1, labels)
	m.readDuration.Record(ctx, time.Since(start).Seconds(), labels)
}
