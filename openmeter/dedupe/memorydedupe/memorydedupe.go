// Package memorydedupe implements in-memory event deduplication.
package memorydedupe

import (
	"context"

	"github.com/cloudevents/sdk-go/v2/event"
	lru "github.com/hashicorp/golang-lru/v2"

	"github.com/openmeterio/openmeter/openmeter/dedupe"
)

const defaultSize = 1024

// Deduplicator implements in-memory event deduplication.
type Deduplicator struct {
	store *lru.Cache[string, any]
}

// NewDeduplicator returns a new {Deduplicator}.
func NewDeduplicator(size int) (*Deduplicator, error) {
	if size < 1 {
		size = defaultSize
	}

	store, err := lru.New[string, any](size)
	if err != nil {
		return nil, err
	}

	return &Deduplicator{
		store: store,
	}, nil
}

func (d *Deduplicator) IsUnique(ctx context.Context, namespace string, ev event.Event) (bool, error) {
	item := dedupe.Item{
		Namespace: namespace,
		ID:        ev.ID(),
		Source:    ev.Source(),
	}
	isContained, _ := d.store.ContainsOrAdd(item.Key(), nil)

	return !isContained, nil
}

func (d *Deduplicator) CheckUnique(ctx context.Context, item dedupe.Item) (bool, error) {
	isContained := d.store.Contains(item.Key())

	return !isContained, nil
}

func (d *Deduplicator) Set(ctx context.Context, items ...dedupe.Item) error {
	for _, item := range items {
		_ = d.store.Add(item.Key(), nil)
	}

	return nil
}

func (d *Deduplicator) Close() error {
	return nil
}
