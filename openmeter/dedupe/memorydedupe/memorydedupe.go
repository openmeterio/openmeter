// Package memorydedupe implements in-memory event deduplication.
package memorydedupe

import (
	"context"
	"sync"

	"github.com/cloudevents/sdk-go/v2/event"
	"github.com/google/uuid"
	lru "github.com/hashicorp/golang-lru/v2"

	"github.com/openmeterio/openmeter/openmeter/dedupe"
)

const defaultSize = 1024

// Deduplicator implements in-memory event deduplication.
type Deduplicator struct {
	mu    sync.Mutex
	store *lru.Cache[string, string]
}

// NewDeduplicator returns a new {Deduplicator}.
func NewDeduplicator(size int) (*Deduplicator, error) {
	if size < 1 {
		size = defaultSize
	}

	store, err := lru.New[string, string](size)
	if err != nil {
		return nil, err
	}

	return &Deduplicator{store: store}, nil
}

func (d *Deduplicator) IsUnique(ctx context.Context, namespace string, ev event.Event) (bool, error) {
	_, unique, err := d.Claim(ctx, dedupe.Item{Namespace: namespace, ID: ev.ID(), Source: ev.Source()})
	return unique, err
}

func (d *Deduplicator) Claim(_ context.Context, item dedupe.Item) (dedupe.Claim, bool, error) {
	claim := dedupe.Claim{Item: item, Token: uuid.NewString()}
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.store.Contains(item.Key()) {
		return dedupe.Claim{}, false, nil
	}
	d.store.Add(item.Key(), claim.Token)
	return claim, true, nil
}

func (d *Deduplicator) CheckUnique(_ context.Context, item dedupe.Item) (bool, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	return !d.store.Contains(item.Key()), nil
}

func (d *Deduplicator) Set(_ context.Context, items ...dedupe.Item) ([]dedupe.Item, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	for _, item := range items {
		d.store.Add(item.Key(), "")
	}
	return nil, nil
}

func (d *Deduplicator) Release(_ context.Context, claim dedupe.Claim) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	if token, ok := d.store.Peek(claim.Item.Key()); ok && token == claim.Token {
		d.store.Remove(claim.Item.Key())
	}
	return nil
}

func (d *Deduplicator) Close() error { return nil }

func (d *Deduplicator) CheckUniqueBatch(_ context.Context, items []dedupe.Item) (dedupe.CheckUniqueBatchResult, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	result := dedupe.CheckUniqueBatchResult{UniqueItems: make(dedupe.ItemSet, len(items)), AlreadyProcessedItems: make(dedupe.ItemSet, len(items))}
	for _, item := range items {
		if d.store.Contains(item.Key()) {
			result.AlreadyProcessedItems[item] = struct{}{}
		} else {
			result.UniqueItems[item] = struct{}{}
		}
	}
	return result, nil
}

