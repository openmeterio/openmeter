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
//
// The event is only recorded in the deduplication index after it was ingested
// successfully. Marking it before ingestion would make a failed delivery
// (e.g. a Kafka delivery error) permanently invisible and retries of the same
// event would be silently skipped even though it was never delivered.
func (d DeduplicatingCollector) Ingest(ctx context.Context, namespace string, ev event.Event) error {
	item := dedupe.Item{
		Namespace: namespace,
		ID:        ev.ID(),
		Source:    ev.Source(),
	}

	isUnique, err := d.Deduplicator.CheckUnique(ctx, item)
	if err != nil {
		return fmt.Errorf("checking event uniqueness: %w", err)
	}

	if !isUnique {
		return nil
	}

	if err := d.Collector.Ingest(ctx, namespace, ev); err != nil {
		return err
	}

	if _, err := d.Deduplicator.Set(ctx, item); err != nil {
		return fmt.Errorf("recording event in deduplicator: %w", err)
	}

	return nil
}
