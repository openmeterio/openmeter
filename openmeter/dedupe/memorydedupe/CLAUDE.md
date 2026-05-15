# memorydedupe

<!-- archie:ai-start -->

> In-memory LRU-backed deduplication for CloudEvents, used as the no-dependency fallback when Redis is unavailable. Implements dedupe.Deduplicator using a fixed-size hashicorp/golang-lru cache keyed by dedupe.Item.Key() with nil values — never stores event payloads.

## Patterns

**LRU keyed by item.Key() with nil values** — All dedup checks use item.Key() (composite of Namespace+ID+Source) as the LRU map key, storing nil as the value. Never store event payload data. (`isContained, _ := d.store.ContainsOrAdd(item.Key(), nil)`)
**Atomic check-and-set via ContainsOrAdd for IsUnique** — IsUnique must use ContainsOrAdd (single LRU call) to avoid a check-then-set race under concurrent ingest goroutines. CheckUnique and Set are separate non-atomic operations. (`isContained, _ := d.store.ContainsOrAdd(item.Key(), nil); return !isContained, nil`)
**CheckUniqueBatch pre-allocates both result sets** — Always initialise UniqueItems and AlreadyProcessedItems with make(dedupe.ItemSet, len(items)) before ranging; never append to nil maps. (`result := dedupe.CheckUniqueBatchResult{UniqueItems: make(dedupe.ItemSet, len(items)), AlreadyProcessedItems: make(dedupe.ItemSet, len(items))}`)
**Implement all five Deduplicator methods** — Must satisfy dedupe.Deduplicator: IsUnique, CheckUnique, Set, Close, and CheckUniqueBatch. Close() is a no-op but must exist. (`func (d *Deduplicator) Close() error { return nil }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `memorydedupe.go` | Single-file implementation; NewDeduplicator(size) is the only constructor. defaultSize=1024 is used when size<1. | Callers must choose a meaningful cache size — too small causes premature LRU eviction, causing previously-seen events to appear unique again (false-unique bug). |
| `memorydedupe_test.go` | Validates atomic IsUnique (second call returns false) and that CheckUnique+Set are separate read/write operations. | Tests use context.Background() — acceptable in tests only; production code must propagate caller context. |

## Anti-Patterns

- Storing event payload data in the LRU cache — only nil values allowed, keyed by item.Key()
- Using separate Contains + Add calls instead of ContainsOrAdd in IsUnique — creates a TOCTOU race under concurrent ingest
- Skipping Close() — must exist even as a no-op to satisfy dedupe.Deduplicator interface
- Using context.Background() in production implementation code (tests only)
- Constructing CheckUniqueBatchResult without pre-allocated maps — nil map assignment panics

## Decisions

- **LRU eviction over unbounded growth** — Bounded memory footprint for long-running sink workers; evicted entries can cause duplicate events to slip through, which is acceptable for the in-memory fallback (Redis is preferred for production).
- **IsUnique atomically checks-and-sets via ContainsOrAdd** — A single LRU call avoids a check-then-set race under concurrent ingest goroutines without introducing a mutex around every operation.

## Example: Implement a new Deduplicator satisfying dedupe.Deduplicator

```
import (
	"context"
	"github.com/cloudevents/sdk-go/v2/event"
	"github.com/openmeterio/openmeter/openmeter/dedupe"
)

func (d *MyDeduplicator) IsUnique(ctx context.Context, namespace string, ev event.Event) (bool, error) {
	item := dedupe.Item{Namespace: namespace, ID: ev.ID(), Source: ev.Source()}
	// use atomic check-and-set: isContained, _ := d.store.ContainsOrAdd(item.Key(), nil)
	return !isContained, nil
}

func (d *MyDeduplicator) CheckUniqueBatch(ctx context.Context, items []dedupe.Item) (dedupe.CheckUniqueBatchResult, error) {
	result := dedupe.CheckUniqueBatchResult{
		UniqueItems:           make(dedupe.ItemSet, len(items)),
// ...
```

<!-- archie:ai-end -->
