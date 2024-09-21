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
}

// WithMetrics wraps an [ingest.Collector] and emits metrics for ingested events.
func WithMetrics(collector ingest.Collector, metricMeter metric.Meter) (ingest.Collector, error) {
	ingestEventsCounter, err := metricMeter.Int64Counter(
		"ingest.events",
		metric.WithDescription("Number of events ingested"),
		metric.WithUnit("{event}"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create events counter: %w", err)
	}

	ingestErrorsCounter, err := metricMeter.Int64Counter(
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
	}, nil
}

// Ingest implements the [ingest.Collector] interface.
func (c collectorMetrics) Ingest(ctx context.Context, namespace string, ev event.Event) error {
	namespaceAttr := attribute.String("namespace", namespace)

	err := c.collector.Ingest(ctx, namespace, ev)
	if err != nil {
		c.ingestErrorsCounter.Add(ctx, 1, metric.WithAttributes(namespaceAttr))

		return err
	}

	c.ingestEventsCounter.Add(ctx, 1, metric.WithAttributes(namespaceAttr))

	return nil
}

// Close implements the [ingest.Collector] interface.
func (c collectorMetrics) Close() {
	c.collector.Close()
}
