package ingest

import (
	"context"
	"errors"
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

	if !isUnique {
		return nil
	}

	err = d.Collector.Ingest(ctx, namespace, ev)
	if err == nil {
		return nil
	}

	// The event was not ingested, so the deduplication claim must be released.
	// Otherwise the client retry documented for event ingestion would be dropped
	// as a duplicate and the event would never be metered. The release is
	// best-effort: if it fails, surface both failures instead of failing silently.
	if rerr := d.Deduplicator.Remove(ctx, dedupe.Item{
		Namespace: namespace,
		ID:        ev.ID(),
		Source:    ev.Source(),
	}); rerr != nil {
		return errors.Join(
			fmt.Errorf("ingesting event: %w", err),
			fmt.Errorf("releasing dedupe claim: %w", rerr),
		)
	}

	return err
}
