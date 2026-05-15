# redisdedupe

<!-- archie:ai-start -->

> Redis-backed deduplication for CloudEvents in the sink worker, implementing dedupe.Deduplicator with TTL-based expiry, pipelined batch writes, and three key-format modes (rawkey, keyhash, keyhash-migration) to support zero-downtime migration of Redis key encoding.

## Patterns

**Mode-switch on every key operation** — Every method (IsUnique, CheckUnique, Set, CheckUniqueBatch) must switch on d.Mode to select the Redis key format. Never assume a single format; adding a new method without the switch breaks migration mode. (`switch d.Mode {
case DedupeModeRawKey: keys = append(keys, item.Key())
case DedupeModeKeyHash, DedupeModeKeyHashMigration: keys = append(keys, GetKeyHash(item.Key()))
}`)
**SetArgs with NX + TTL for atomic set-if-not-exists** — All uniqueness writes use redis.SetArgs{TTL: d.Expiration, Mode: "nx"}. Return redis.Nil means the key already existed (duplicate). An empty status string also signals duplicate. (`status, err := d.Redis.SetArgs(ctx, key, "", redis.SetArgs{TTL: d.Expiration, Mode: "nx"}).Result()`)
**Pipelined batch writes in Set()** — Set() pipelines multiple SetArgs calls via d.Redis.Pipelined. Inspect per-command errors individually — redis.Nil on a pipeline command means the key pre-existed (not a fatal error). (`cmds, err := d.Redis.Pipelined(ctx, func(pipe redis.Pipeliner) error { ... })
for i, cmd := range cmds { if errors.Is(cmd.Err(), redis.Nil) { existingItems = append(existingItems, items[i]) } }`)
**keyhash-migration checks both old and new key formats** — During DedupeModeKeyHashMigration, IsUnique sets the new hashed key first, then checks if the old rawkey exists in Redis to avoid false-unique results during rolling deploys. (`isUnique, err := d.setKey(ctx, keyHash); if isUnique { isSet, _ := d.Redis.Exists(ctx, item.Key()).Result(); return isSet == 0, nil }`)
**nil guard on d.Redis before any Redis call** — IsUnique guards d.Redis == nil and returns an error. Replicate this guard in any new method that calls d.Redis directly. (`if d.Redis == nil { return false, errors.New("redis client not initialized") }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `redisdedupe.go` | Full Deduplicator implementation; DedupeMode type and its three constants are the API extension point for key-format migration. | keyhash-migration mode reads both rawkey and keyhash formats to prevent false-unique events during rolling deploy — do not simplify without coordinating a full cutover. |
| `keyhash.go` | Standalone hashing helper using xxh3.HashString128 + base64.RawURLEncoding to shrink Redis keys by ~57% vs raw composite keys. | Changing the hash algorithm or encoding invalidates all existing Redis keys — causes a flood of duplicate events to be treated as unique. Any change requires a new migration mode. |

## Anti-Patterns

- Using SET without NX — allows overwriting existing keys and defeats deduplication semantics
- Adding a new DedupeMode without updating every method's switch statement
- Ignoring redis.Nil in pipeline results — it signals a pre-existing key, not a fatal error
- Calling d.Redis methods without the nil guard (replicate the guard from IsUnique in every new method)
- Changing GetKeyHash algorithm without adding a new migration mode and Redis key rotation plan

## Decisions

- **Three-mode key format with explicit migration mode** — Allows a rolling upgrade from rawkey to keyhash without a big-bang cutover; migration mode checks both formats so no events are incorrectly classified as unique during the transition window.
- **xxh3-128 + base64url over raw composite strings** — 22-char base64url keys use ~57% less Redis memory than the ~77-char raw composite key while maintaining collision probability ~1e-30 at 300M events/day.
- **TTL-based expiry instead of unbounded growth** — Ingest events are only deduplicated within a recent window; expired keys free memory automatically without a separate purge job.

## Example: Check uniqueness in DedupeModeKeyHash mode and write atomically with TTL

```
import (
	"github.com/redis/go-redis/v9"
	"github.com/openmeterio/openmeter/openmeter/dedupe"
)

func (d Deduplicator) setKey(ctx context.Context, key string) (bool, error) {
	status, err := d.Redis.SetArgs(ctx, key, "", redis.SetArgs{
		TTL:  d.Expiration,
		Mode: "nx",
	}).Result()
	if err != nil && err != redis.Nil {
		return false, err
	}
	if status == "" {
		return false, nil // duplicate: key already existed
// ...
```

<!-- archie:ai-end -->
