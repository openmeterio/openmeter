// Package ingest implements event ingestion.
package ingest

import (
	"context"

	"github.com/cloudevents/sdk-go/v2/event"
)

// Collector is a receiver of events that handles sending those events to some downstream broker.
type Collector interface {
	Ingest(ctx context.Context, namespace string, ev event.Event) error
	Close()
}
