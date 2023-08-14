package ingest

import (
	"context"
	"fmt"

	"github.com/cloudevents/sdk-go/v2/event"
)

// Deduplicator checks if an event is unique.
type Deduplicator interface {
	// IsUnique checks if an event is unique AND adds it to the deduplication index.
	IsUnique(ctx context.Context, namespace string, ev event.Event) (bool, error)
}

// DeduplicatingCollector implements event deduplication at event ingestion.
type DeduplicatingCollector struct {
	Collector

	Deduplicator Deduplicator
}

// Ingest implements the {Collector} interface wrapping an existing {Collector} and deduplicating events.
func (d DeduplicatingCollector) Ingest(ev event.Event, namespace string) error {
	isUnique, err := d.Deduplicator.IsUnique(context.TODO(), namespace, ev)
	if err != nil {
		return fmt.Errorf("checking event uniqueness: %w", err)
	}

	if isUnique {
		return d.Collector.Ingest(ev, namespace)
	}

	return nil
}
