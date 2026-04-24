# redisdedupe

<!-- archie:ai-start -->

> Redis-backed deduplication for CloudEvents in the sink worker. Supports three key-format modes (rawkey, keyhash, keyhash-migration) and exposes the same dedupe.Deduplicator interface as memorydedupe, but with TTL-based expiry and pipeline batch operations.

## Patterns

**Mode-switch on every key operation** — Every method (IsUnique, CheckUnique, Set, CheckUniqueBatch) switches on d.Mode to determine the Redis key format; never assume a single format. (`switch d.Mode {
case DedupeModeRawKey: keys = append(keys, item.Key())
case DedupeModeKeyHash, DedupeModeKeyHashMigration: keys = append(keys, GetKeyHash(item.Key()))
}`)
**SetArgs with NX + TTL for atomic set-if-not-exists** — Use redis.SetArgs{TTL: d.Expiration, Mode: "nx"} for all uniqueness writes; interpret redis.Nil return as duplicate (key already existed). (`status, err := d.Redis.SetArgs(ctx, key, "", redis.SetArgs{TTL: d.Expiration, Mode: "nx"}).Result()`)
**Pipelined batch writes** — Set() pipelines multiple SetArgs calls; inspect per-command errors (redis.Nil = already existed) to build the returned existingItems slice. (`cmds, err := d.Redis.Pipelined(ctx, func(pipe redis.Pipeliner) error { ... })`)
**xxh3 128-bit hash → base64url key** — GetKeyHash() uses xxh3.HashString128 + base64.RawURLEncoding to shrink keys by ~57% in Redis memory versus the raw composite key. (`hashBytes := xxh3.HashString128(itemKey).Bytes(); b64 := base64.RawURLEncoding.EncodeToString(hashBytes[:])`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `redisdedupe.go` | Full Redis Deduplicator implementation; DedupeMode type and its three constants are the API extension point for key-format migration. | keyhash-migration mode checks both old rawkey and new keyhash Redis keys to prevent false-unique events during rolling deploy; do not simplify without coordinating a cutover. |
| `keyhash.go` | Standalone hashing helper; must not be changed without re-hashing all existing Redis keys. | Changing the hash algorithm or encoding invalidates all keys currently in Redis and will cause a flood of duplicate events. |

## Anti-Patterns

- Calling Redis without checking d.Redis != nil (IsUnique already guards this, replicate the guard in new methods)
- Using SET without NX — allows overwriting existing keys and defeats deduplication
- Adding a new DedupeMode without updating every method's switch statement
- Ignoring redis.Nil in pipeline results — it signals a pre-existing key, not an error
- Changing GetKeyHash algorithm without a migration mode and Redis key rotation plan

## Decisions

- **Three-mode key format with explicit migration mode** — Allows a rolling upgrade from rawkey to keyhash without a big-bang cutover; migration mode checks both formats so no events are incorrectly classified as unique during the transition window.
- **xxh3-128 + base64url over SHA-224 or raw composite strings** — 22-char base64url keys use 57% less Redis memory than the ~77-char raw composite key while maintaining collision probability ~1e-30 at 300M events/day.
- **TTL-based expiry instead of unbounded growth** — Ingest events are only deduplicated within a recent window; expired keys free memory automatically without a separate purge job.

<!-- archie:ai-end -->
