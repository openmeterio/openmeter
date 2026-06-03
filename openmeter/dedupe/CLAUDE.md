# dedupe

<!-- archie:ai-start -->

> Provides CloudEvent deduplication for the sink worker via the Deduplicator interface. Two implementations: memorydedupe/ (LRU, no-dependency fallback) and redisdedupe/ (TTL-based, pipelined batch, three key-format modes for zero-downtime migration). All callers depend on dedupe.Deduplicator; implementations are swapped by config.

## Patterns

**Deduplicator interface as the contract boundary** — All callers depend on dedupe.Deduplicator; implementations are swapped by config without callers knowing which backend is active. (`var d dedupe.Deduplicator = memorydedupe.New(size) // or redisdedupe.New(cfg)`)
**Item.Key() as the canonical dedup key** — All implementations key their stores on item.Key() = "namespace-source-id"; never use raw event IDs or namespaces alone. (`key := item.Key() // "myns-mysource-event123"`)
**SET NX + TTL for Redis atomic set-if-not-exists** — redisdedupe uses SetArgs{NX: true, TTL: d.ttl} to write-if-absent; never plain SET which overwrites. (`d.Redis.SetArgs(ctx, key, val, redis.SetArgs{NX: true, TTL: d.ttl})`)
**ContainsOrAdd for atomic check-and-set in memorydedupe** — IsUnique uses LRU.ContainsOrAdd rather than separate Contains + Add to avoid TOCTOU races under concurrent ingest. (`existed, _ := lru.ContainsOrAdd(item.Key(), nil)`)
**Three-mode key format with explicit migration mode** — redisdedupe supports rawkey, keyhash, keyhash-migration; every key-operation method must switch on d.mode. (`switch d.mode { case ModeRawKey: return item.Key(); case ModeKeyHash: return GetKeyHash(item); ... }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `dedupe.go` | Defines Deduplicator interface, Item, Item.Key(), CheckUniqueBatchResult. | IsUnique is marked TODO/deprecated in favor of CheckUnique + Set; prefer the newer methods in new code. |
| `memorydedupe/memorydedupe.go` | LRU-backed Deduplicator; IsUnique uses ContainsOrAdd for atomic check-and-set. | LRU stores nil values keyed by item.Key() — do not store event payload data. |
| `redisdedupe/redisdedupe.go` | Redis-backed Deduplicator with pipelined batch writes and NX TTL semantics. | redis.Nil in pipeline results signals a pre-existing key (already deduped), not an error; guard d.Redis for nil before any call. |
| `redisdedupe/keyhash.go` | GetKeyHash using xxh3-128 + base64url encoding. | Changing the hash algorithm requires a new migration mode and key rotation plan — never change silently. |

## Anti-Patterns

- Using Redis SET without NX — overwrites existing keys and defeats deduplication
- Adding a new DedupeMode without updating every method's switch statement in redisdedupe
- Ignoring redis.Nil in pipeline results — it signals a pre-existing key, not an error
- Storing event payload data in the LRU cache — only nil values, keyed by item.Key()
- Skipping Close() implementation on a new Deduplicator — must exist even as a no-op to satisfy the interface

## Decisions

- **LRU eviction in memorydedupe over unbounded map growth** — The sink worker processes high-volume events; unbounded growth would exhaust memory on long-running sinks.
- **Three-mode key format (rawkey, keyhash, keyhash-migration) in redisdedupe** — Changing the key format without a migration mode would leave pre-existing keys unreachable during rollout.
- **xxh3-128 + base64url over SHA-224 or raw composite strings** — xxh3 is much faster for high-throughput batch hashing; base64url keeps keys URL-safe and compact.

<!-- archie:ai-end -->
