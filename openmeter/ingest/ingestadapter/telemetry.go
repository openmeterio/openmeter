package ingestadapter

import (
	"context"
	"fmt"

	"github.com/cloudevents/sdk-go/v2/event"
	"go.opentelemetry.io/otel/attribute"
	otelcodes "go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"

	"github.com/openmeterio/openmeter/openmeter/ingest"
)

// collectorTelemetry emits metrics and spans for ingested events.
type collectorTelemetry struct {
	collector ingest.Collector
	tracer    trace.Tracer

	ingestEventsCounter metric.Int64Counter
	ingestErrorsCounter metric.Int64Counter
}

// WithTelemetry wraps an [ingest.Collector] and emits metrics and spans for ingested events.
func WithTelemetry(collector ingest.Collector, metricMeter metric.Meter, tracer trace.Tracer) (ingest.Collector, error) {
	ingestEventsCounter, err := metricMeter.Int64Counter(
		"openmeter.ingest.events",
		metric.WithDescription("Number of events ingested"),
		metric.WithUnit("{event}"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create events counter: %w", err)
	}

	ingestErrorsCounter, err := metricMeter.Int64Counter(
		"openmeter.ingest.errors",
		metric.WithDescription("Number of failed event ingests"),
		metric.WithUnit("{error}"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create errors counter: %w", err)
	}

	return &collectorTelemetry{
		collector: collector,
		tracer:    tracer,

		ingestEventsCounter: ingestEventsCounter,
		ingestErrorsCounter: ingestErrorsCounter,
	}, nil
}

// Ingest implements the [ingest.Collector] interface.
func (c *collectorTelemetry) Ingest(ctx context.Context, namespace string, ev event.Event) error {
	namespaceAttr := attribute.String("namespace", namespace)

	var err error

	ctx, span := c.tracer.Start(ctx, "openmeter.ingest.events", trace.WithAttributes(
		namespaceAttr,
		attribute.String("openmeter.event.id", ev.ID()),
	))
	defer span.End()

	err = c.collector.Ingest(ctx, namespace, ev)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(otelcodes.Error, err.Error())

		c.ingestErrorsCounter.Add(ctx, 1, metric.WithAttributes(namespaceAttr))

		return err
	}

	c.ingestEventsCounter.Add(ctx, 1, metric.WithAttributes(namespaceAttr))

	return nil
}

// Close implements the [ingest.Collector] interface.
func (c *collectorTelemetry) Close() {
	c.collector.Close()
}
