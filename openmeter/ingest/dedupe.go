package ingest

import (
	"context"
	"errors"
	"fmt"
	"time"

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
	item := dedupe.Item{Namespace: namespace, ID: ev.ID(), Source: ev.Source()}
	claim, isUnique, err := d.Deduplicator.Claim(ctx, item)
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
	cleanupCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
	defer cancel()
	if rerr := d.Deduplicator.Release(cleanupCtx, claim); rerr != nil {
		return errors.Join(
			fmt.Errorf("ingesting event: %w", err),
			fmt.Errorf("releasing dedupe claim: %w", rerr),
		)
	}

	return err
}
