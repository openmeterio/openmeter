# adapter

<!-- archie:ai-start -->

> Redis-backed implementation of progressmanager.Adapter that stores/retrieves TTL-bound progress data as JSON; also provides adapterNoop (silent no-op) and MockProgressManager (testify mock for Service) in the same package.

## Patterns

**Config.Validate() before construction** — New() calls config.Validate() first and returns (Adapter, error). Required fields: non-nil Redis client, non-nil Logger, Expiration > 0. Never relax these guards. (`func New(config Config) (progressmanager.Adapter, error) { if err := config.Validate(); err != nil { return nil, err } ... }`)
**Compile-time interface assertion** — Both adapter and adapterNoop assert interface compliance via blank-identifier assignment at package level. (`var _ progressmanager.Adapter = (*adapter)(nil)
var _ progressmanager.Adapter = (*adapterNoop)(nil)`)
**Namespaced Redis key via getKey()** — All Redis operations must route through getKey(id ProgressID) which produces '<keyPrefix>:progress:<namespace>:<id>' (or without prefix when keyPrefix is empty). Never construct keys inline. (`fmt.Sprintf("%s:%s:%s:%s", a.keyPrefix, staticKeyPrefix, id.Namespace, id.ID)`)
**redis.Nil mapped to GenericNotFoundError** — When redis.Get returns redis.Nil, return models.NewGenericNotFoundError(...) — not the raw redis error — so the HTTP layer maps it to 404. (`if cmd.Err() == redis.Nil { return nil, models.NewGenericNotFoundError(fmt.Errorf("progress not found for id: %s", input.ProgressID.ID)) }`)
**Input validation before Redis operations** — Both GetProgress and UpsertProgress call input.Validate() and wrap the error before any Redis call. (`if err := input.Validate(); err != nil { return nil, fmt.Errorf("validate get progress input: %w", err) }`)
**Noop asymmetry: writes succeed, reads return not-found** — adapterNoop.UpsertProgress returns nil (writes discarded silently); adapterNoop.GetProgress returns GenericNotFoundError. This asymmetry is intentional — callers treat noop writes as safe to ignore. (`func (a *adapterNoop) UpsertProgress(...) error { return nil }`)
**Mock implements Service not Adapter** — MockProgressManager satisfies progressmanager.Service (including DeleteProgressByRuntimeID), not progressmanager.Adapter. Keep in sync when Service interface gains new methods. (`var _ progressmanager.Service = &MockProgressManager{}`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `adapter.go` | Config struct, New() constructor, adapter and adapterNoop type definitions. Entry point for wiring in app/common. | Config.Validate() must reject nil Redis client and zero Expiration — never relax these guards. |
| `progress.go` | GetProgress and UpsertProgress implementations using Redis GET/SET with JSON marshaling and TTL. | getKey() is the sole key-format authority; changing the format orphans all existing in-flight Redis records. |
| `noop.go` | No-op adapter for disabled progress tracking. GetProgress always returns not-found; UpsertProgress is silent. | Do not make UpsertProgress return a non-nil error — callers rely on noop writes being safe no-ops. |
| `mock.go` | Testify mock for progressmanager.Service used in unit tests. | Mock is for Service (not Adapter); must be updated when Service interface adds new methods. |

## Anti-Patterns

- Storing progress keys without namespace prefix — breaks multi-tenant isolation
- Returning raw redis errors instead of models.GenericNotFoundError for redis.Nil — breaks HTTP 404 mapping
- Calling Redis operations without first calling input.Validate() — bypasses domain invariant checks
- Editing the key format in getKey() without a migration plan — orphans all existing in-flight progress records
- Making adapterNoop.UpsertProgress return an error — callers rely on noop writes being safe no-ops

## Decisions

- **Redis (not PostgreSQL/Ent) as the backing store** — Progress data is transient and TTL-bound; Redis SET with expiration avoids schema migrations and naturally garbage-collects stale progress without a cleanup job.
- **Noop adapter co-located with the real adapter in the same package** — Keeps the disabled-feature code path trivially discoverable without a separate package; follows the broader project pattern of noop implementations per optional feature.

## Example: Wire the Redis adapter in app/common

```
import (
    "github.com/openmeterio/openmeter/openmeter/progressmanager/adapter"
)

func ProvideProgressAdapter(redisClient *redis.Client, logger *slog.Logger) (progressmanager.Adapter, error) {
    return adapter.New(adapter.Config{
        Expiration: 24 * time.Hour,
        Redis:      redisClient,
        Logger:     logger,
        KeyPrefix:  "om",
    })
}
```

<!-- archie:ai-end -->
