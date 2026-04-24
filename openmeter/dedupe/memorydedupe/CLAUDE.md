# memorydedupe

<!-- archie:ai-start -->

> In-memory LRU-backed deduplication for CloudEvents, used as the no-dependency fallback when Redis is unavailable. Implements the dedupe.Deduplicator interface using a fixed-size hashicorp/golang-lru cache keyed by dedupe.Item.Key().

## Patterns

**LRU-cache keyed by dedupe.Item.Key()** — All dedup checks use item.Key() (a composite of Namespace+ID+Source) as the LRU map key with nil values; never store event payloads. (`isContained, _ := d.store.ContainsOrAdd(item.Key(), nil)`)
**Implement all dedupe.Deduplicator methods** — Must implement IsUnique, CheckUnique, Set, Close, and CheckUniqueBatch to satisfy the dedupe.Deduplicator interface. (`func (d *Deduplicator) CheckUniqueBatch(ctx context.Context, items []dedupe.Item) (dedupe.CheckUniqueBatchResult, error)`)
**CheckUniqueBatch returns split result sets** — Always initialise both UniqueItems and AlreadyProcessedItems in CheckUniqueBatchResult with pre-allocated ItemSet maps before ranging over items. (`result := dedupe.CheckUniqueBatchResult{UniqueItems: make(dedupe.ItemSet, len(items)), AlreadyProcessedItems: make(dedupe.ItemSet, len(items))}`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `memorydedupe.go` | Single-file implementation of the in-memory Deduplicator; NewDeduplicator(size) is the only constructor. | defaultSize=1024 is the fallback when size<1; callers must choose a meaningful size to avoid premature eviction causing false-unique results. |
| `memorydedupe_test.go` | Validates IsUnique atomically adds to the cache (second call returns false) and that CheckUnique+Set are separate read/write operations. | Tests use context.Background() — acceptable in tests but not in production code. |

## Anti-Patterns

- Storing event payload data in the LRU cache (only nil values, keyed by item.Key())
- Skipping Close() implementation — even if it's a no-op it must exist to satisfy the interface
- Using context.Background() in production implementation code (only in tests)

## Decisions

- **LRU eviction over unbounded growth** — Bounded memory footprint for long-running sink workers; evicted entries can cause duplicate events to slip through, which is acceptable for the in-memory fallback.
- **IsUnique atomically checks-and-sets via ContainsOrAdd** — A single LRU call avoids a check-then-set race under concurrent ingest goroutines.

## Example: Implement a new Deduplicator backed by a different store

```
import (
	"context"
	"github.com/openmeterio/openmeter/openmeter/dedupe"
)

type MyDeduplicator struct{ /* store */ }

func (d *MyDeduplicator) IsUnique(ctx context.Context, namespace string, ev event.Event) (bool, error) {
	item := dedupe.Item{Namespace: namespace, ID: ev.ID(), Source: ev.Source()}
	// check-and-set item.Key()
}
func (d *MyDeduplicator) CheckUniqueBatch(ctx context.Context, items []dedupe.Item) (dedupe.CheckUniqueBatchResult, error) {
	result := dedupe.CheckUniqueBatchResult{
		UniqueItems:           make(dedupe.ItemSet, len(items)),
		AlreadyProcessedItems: make(dedupe.ItemSet, len(items)),
// ...
```

<!-- archie:ai-end -->
