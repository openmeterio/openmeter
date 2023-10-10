// Package dedupe implements in-process event deduplication.
package dedupe

import (
	"context"
	"fmt"
)

// Deduplicator checks if an event is unique.
type Deduplicator interface {
	IsUnique(ctx context.Context, item Item) (bool, error)
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
