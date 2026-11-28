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
//
// IsUnique atomically reserves the event key before forwarding to the downstream
// collector so concurrent requests for the same event cannot both publish. If
// downstream delivery fails (and was not canceled, in which case an async
// producer may still deliver the queued message), the reservation is released so
// retries can succeed.
func (d DeduplicatingCollector) Ingest(ctx context.Context, namespace string, ev event.Event) error {
	isUnique, err := d.Deduplicator.IsUnique(ctx, namespace, ev)
	if err != nil {
		return fmt.Errorf("checking event uniqueness: %w", err)
	}

	if !isUnique {
		return nil
	}

	if err := d.Collector.Ingest(ctx, namespace, ev); err != nil {
		if !errors.Is(err, context.Canceled) && !errors.Is(err, context.DeadlineExceeded) {
			item := dedupe.Item{
				Namespace: namespace,
				ID:        ev.ID(),
				Source:    ev.Source(),
			}
			if delErr := d.Deduplicator.Delete(ctx, item); delErr != nil {
				return errors.Join(err, fmt.Errorf("releasing event deduplication key: %w", delErr))
			}
		}

		return err
	}

	return nil
}
