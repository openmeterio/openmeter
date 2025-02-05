package ingestadapter

import (
	"context"
	"fmt"

	"github.com/cloudevents/sdk-go/v2/event"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"

	"github.com/openmeterio/openmeter/openmeter/ingest"
)

// collectorMetrics emits metrics for ingested events.
type collectorMetrics struct {
	collector ingest.Collector

	ingestEventsCounter metric.Int64Counter
	ingestErrorsCounter metric.Int64Counter

	// TODO: remove after deprecation period
	ingestEventsCounterOld metric.Int64Counter
	ingestErrorsCounterOld metric.Int64Counter
}

// WithMetrics wraps an [ingest.Collector] and emits metrics for ingested events.
func WithMetrics(collector ingest.Collector, metricMeter metric.Meter) (ingest.Collector, error) {
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

	ingestEventsCounterOld, err := metricMeter.Int64Counter(
		"ingest.events",
		metric.WithDescription("Number of events ingested"),
		metric.WithUnit("{event}"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create events counter: %w", err)
	}

	ingestErrorsCounterOld, err := metricMeter.Int64Counter(
		"ingest.errors",
		metric.WithDescription("Number of failed event ingests"),
		metric.WithUnit("{error}"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create errors counter: %w", err)
	}

	return collectorMetrics{
		collector: collector,

		ingestEventsCounter: ingestEventsCounter,
		ingestErrorsCounter: ingestErrorsCounter,

		ingestEventsCounterOld: ingestEventsCounterOld,
		ingestErrorsCounterOld: ingestErrorsCounterOld,
	}, nil
}

// Ingest implements the [ingest.Collector] interface.
func (c collectorMetrics) Ingest(ctx context.Context, namespace string, ev event.Event) error {
	namespaceAttr := attribute.String("namespace", namespace)

	err := c.collector.Ingest(ctx, namespace, ev)
	if err != nil {
		c.ingestErrorsCounter.Add(ctx, 1, metric.WithAttributes(namespaceAttr))
		c.ingestErrorsCounterOld.Add(ctx, 1, metric.WithAttributes(namespaceAttr))

		return err
	}

	c.ingestEventsCounter.Add(ctx, 1, metric.WithAttributes(namespaceAttr))
	c.ingestEventsCounterOld.Add(ctx, 1, metric.WithAttributes(namespaceAttr))

	return nil
}

// Close implements the [ingest.Collector] interface.
func (c collectorMetrics) Close() {
	c.collector.Close()
}
