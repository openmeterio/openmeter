package ingest

import (
	"context"
	"fmt"

	"github.com/cloudevents/sdk-go/v2/event"
)

type Deduplicator interface {
	IsUnique(ctx context.Context, namespace string, ev event.Event) (bool, error)
	Set(ctx context.Context, namespace string, ev event.Event) (bool, error)
}

// CloudEvents are unique based on the source and id
func GetEventKey(namespace string, ev event.Event) string {
	return fmt.Sprintf("%s-%s-%s", namespace, ev.Source(), ev.ID())
}
