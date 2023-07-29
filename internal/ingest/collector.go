package ingest

import "github.com/cloudevents/sdk-go/v2/event"

// Collector is a receiver of events that handles sending those events to some downstream broker.
type Collector interface {
	Ingest(ev event.Event, namespace string) error
	Close()
}
