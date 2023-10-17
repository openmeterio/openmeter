// Package dedupe implements in-process event deduplication.
package dedupe

import (
	"context"
	"fmt"

	"github.com/cloudevents/sdk-go/v2/event"
)

// Deduplicator checks if an event is unique.
type Deduplicator interface {
	// IsUnique checks if an event is unique AND adds it to the deduplication index.
	// TODO: deprecate or rename IsUnique
	IsUnique(ctx context.Context, namespace string, ev event.Event) (bool, error)
	// CheckUnique checks if an item is unique.
	CheckUnique(ctx context.Context, item Item) (bool, error)
	// Set adds the item(s) to the deduplicator
	Set(ctx context.Context, events ...Item) error
}

type Item struct {
	Namespace string
	ID        string
	Source    string
}

func (i Item) Key() string {
	return fmt.Sprintf("%s-%s-%s", i.Namespace, i.Source, i.ID)
}
