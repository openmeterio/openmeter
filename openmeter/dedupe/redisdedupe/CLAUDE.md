# redisdedupe

<!-- archie:ai-start -->

> Redis-backed implementation of openmeter/dedupe.Deduplicator for distributed, TTL-bounded event deduplication. Wired via app/config; supports three keying modes including a live migration from raw keys to hashed keys.

## Patterns

**Mode switch on every key operation** — Every method switches on d.Mode (DedupeModeRawKey / DedupeModeKeyHash / DedupeModeKeyHashMigration) to decide whether to use item.Key() raw or GetKeyHash(item.Key()). Migration mode checks both. (`case DedupeModeKeyHash: keyHash := GetKeyHash(item.Key()); return d.setKey(ctx, keyHash)`)
**SET NX with TTL as the uniqueness primitive** — setKey uses Redis SetArgs{TTL: d.Expiration, Mode: "nx"}; status "OK" => unique, "" => duplicate. redis.Nil is treated as non-error. (`status, err := d.Redis.SetArgs(ctx, key, "", redis.SetArgs{TTL: d.Expiration, Mode: "nx"}).Result()`)
**Hash keys via GetKeyHash (xxh3-128 + base64 RawURL)** — keyhash.go hashes item.Key() with xxh3 HashString128 and base64.RawURLEncoding to shrink Redis memory ~57% vs raw keys; non-cryptographic but collision-safe at expected volume. (`b64 := base64.RawURLEncoding.EncodeToString(hashBytes[:])`)
**Migration mode double-checks the old raw key** — DedupeModeKeyHashMigration sets the hashed key, then if it appears unique also probes Redis.Exists(item.Key()) to detect pre-migration raw keys before declaring uniqueness. (`isSet, err := d.Redis.Exists(ctx, item.Key()).Result(); keyExists := isSet == 1; return !keyExists, nil`)
**Batch ops use Pipelined / MGet and tolerate redis.Nil** — Set pipelines SetArgs NX and collects existing items from per-command redis.Nil errors; CheckUniqueBatch MGets all keys and partitions by nil result. Both guard with errors.Is(err, redis.Nil). (`cmds, err := d.Redis.Pipelined(ctx, func(pipe redis.Pipeliner) error {...})`)
**Validate mode and guard nil client** — DedupeMode.Validate() rejects unknown modes; IsUnique returns an error if d.Redis is nil. CheckUniqueBatch returns ErrNoDedupItems on empty input. (`if d.Redis == nil { return false, errors.New("redis client not initialized") }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `redisdedupe.go` | Deduplicator value type (Redis client, Expiration, Mode) implementing the dedupe interface, plus DedupeMode constants/Validate and ErrNoDedupItems. | Methods are value receivers, not pointer. Set's pipeline Mode is "NX" (uppercase) while setKey uses "nx" (lowercase) — Redis treats both the same, but keep the distinction intentional. Set returns existing (duplicate) items, opposite-sense from CheckUniqueBatch's UniqueItems. |
| `keyhash.go` | GetKeyHash: deterministic xxh3-128 + base64 RawURL encoding of an item key, with a long comment block justifying the hash choice and keyspace/collision math. | Changing the hash function or encoding silently invalidates all existing Redis keys — only safe behind DedupeModeKeyHashMigration. RawURLEncoding (no padding) is required to keep Lua-safe keys. |

## Anti-Patterns

- Adding a new key-format without a corresponding migration mode — switching modes in place orphans existing keys and breaks dedup correctness.
- Treating redis.Nil as a fatal error in batch/set paths; it signals NX-skip or missing key and must be tolerated.
- Calling methods assuming pointer receivers or assuming Set returns unique items (it returns existing/duplicate items).
- Hand-formatting keys instead of going through item.Key() then GetKeyHash.
- Omitting TTL on SetArgs — keys must expire via d.Expiration to bound the dedup window and Redis memory.

## Decisions

- **Hash keys to base64(xxh3-128) instead of storing raw orgId-source-id strings.** — Raw keys average ~77 chars; xxh3-128 base64 is ~22 chars (~57% Redis memory saving) with ~1e-30 collision probability at 300M events, and base64 avoids binary-key handling in Lua.
- **Provide an explicit DedupeModeKeyHashMigration that checks both old and new key formats.** — Allows live cutover from raw to hashed keys without losing dedup coverage for already-recorded events during the TTL overlap window.

## Example: Unique-check that atomically records the event under the configured key mode

```
import (
	"github.com/redis/go-redis/v9"
	"github.com/openmeterio/openmeter/openmeter/dedupe"
)

func (d Deduplicator) setKey(ctx context.Context, key string) (bool, error) {
	status, err := d.Redis.SetArgs(ctx, key, "", redis.SetArgs{TTL: d.Expiration, Mode: "nx"}).Result()
	if err != nil && err != redis.Nil {
		return false, err
	}
	if status == "" { // existed -> duplicate
		return false, nil
	}
	if status == "OK" { // newly set -> unique
		return true, nil
// ...
```

<!-- archie:ai-end -->
