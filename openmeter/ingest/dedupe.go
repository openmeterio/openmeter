package ingest

import (
	"context"
	"fmt"

	"github.com/cloudevents/sdk-go/v2/event"

	"github.com/openmeterio/openmeter/openmeter/dedupe"
)

// DeduplicatingCollector implements event deduplication at event ingestion.
type DeduplicatingCollector struct {
	Collector

	Deduplicator dedupe.Deduplicator
}

// Ingest implements the {Collector} interface wrapping an existing {Collector} and deduplicating events.
func (d DeduplicatingCollector) Ingest(ctx context.Context, namespace string, ev event.Event) error {
	isUnique, err := d.Deduplicator.IsUnique(ctx, namespace, ev)
	if err != nil {
		return fmt.Errorf("checking event uniqueness: %w", err)
	}

	if isUnique {
		return d.Collector.Ingest(ctx, namespace, ev)
	}

	return nil
}
