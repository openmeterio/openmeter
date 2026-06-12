# memorydedupe

<!-- archie:ai-start -->

> In-memory implementation of the openmeter/dedupe.Deduplicator interface, backed by a fixed-size hashicorp golang-lru cache. Used by openmeter/ingest for single-process event deduplication where no shared Redis is configured.

## Patterns

**Implement full dedupe.Deduplicator interface** — Deduplicator must implement every method of openmeter/dedupe.Deduplicator: IsUnique, CheckUnique, Set, CheckUniqueBatch, Close. Missing one breaks app/config wiring. (`func (d *Deduplicator) IsUnique(ctx context.Context, namespace string, ev event.Event) (bool, error)`)
**Key derived only via dedupe.Item.Key()** — Build a dedupe.Item{Namespace, ID, Source} and call item.Key() for the cache key — never hand-format the namespace-source-id string. (`item := dedupe.Item{Namespace: namespace, ID: ev.ID(), Source: ev.Source()}; d.store.ContainsOrAdd(item.Key(), nil)`)
**IsUnique mutates as it checks** — IsUnique both tests uniqueness AND inserts the key atomically via store.ContainsOrAdd, returning !isContained. CheckUnique is read-only (store.Contains). (`isContained, _ := d.store.ContainsOrAdd(item.Key(), nil); return !isContained, nil`)
**Constructor clamps invalid size to defaultSize** — NewDeduplicator(size) replaces size<1 with defaultSize (1024) before calling lru.New, and returns the lru error unwrapped. (`if size < 1 { size = defaultSize }`)
**Cache stores nil values, keys carry all signal** — Only presence of the key matters; values are always nil/struct{}{}. CheckUniqueBatch partitions items into UniqueItems / AlreadyProcessedItems ItemSets. (`result.AlreadyProcessedItems[item] = struct{}{}`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `memorydedupe.go` | The entire implementation: Deduplicator struct wrapping *lru.Cache[string, any] plus NewDeduplicator constructor and all interface methods. | Set returns (nil, nil) — it never reports already-existing items unlike redisdedupe.Set; callers needing existing-item detection must use CheckUniqueBatch. Close is a no-op. |
| `memorydedupe_test.go` | Black-box tests (package memorydedupe_test) verifying IsUnique flips true->false on repeat and Set+CheckUnique interaction. | Tests construct dedupe.Item literals directly; keep field order Namespace/ID/Source consistent with the interface. |

## Anti-Patterns

- Persisting or sharing state across processes — this is per-process only; use redisdedupe for distributed dedup.
- Returning non-nil from Set's existing-items slice (the in-memory impl intentionally returns nil).
- Constructing the cache key by string-formatting instead of dedupe.Item.Key().
- Adding eviction-sensitive correctness assumptions — LRU silently evicts old keys, so dedup is best-effort within cache size.

## Decisions

- **Use hashicorp golang-lru with a bounded default size (1024).** — Bounds memory for an in-process cache while accepting that very old event IDs may be evicted; dedup is a best-effort safeguard, not a guarantee.

<!-- archie:ai-end -->
