# adapter

<!-- archie:ai-start -->

> Redis-backed implementation of progressmanager.Adapter that stores/retrieves progress data as JSON with TTL expiration. Also provides a noop implementation (adapterNoop) and a testify-based mock (MockProgressManager) in the same package.

## Patterns

**Config struct with Validate() before construction** — All required fields (Redis client, Logger, Expiration > 0) are validated in Config.Validate() before the adapter is constructed. New() returns (Adapter, error) and calls Validate() first. (`func New(config Config) (progressmanager.Adapter, error) { if err := config.Validate(); err != nil { return nil, err } ... }`)
**Interface compliance assertion** — Both adapter structs assert interface compliance at compile time via blank identifier assignment. (`var _ progressmanager.Adapter = (*adapter)(nil)
var _ progressmanager.Adapter = (*adapterNoop)(nil)`)
**Namespaced Redis key format** — Keys follow the format '<keyPrefix>:progress:<namespace>:<id>' (or 'progress:<namespace>:<id>' when keyPrefix is empty). Never store progress without namespace isolation. (`fmt.Sprintf("%s:%s:%s:%s", a.keyPrefix, staticKeyPrefix, id.Namespace, id.ID)`)
**redis.Nil sentinel mapped to GenericNotFoundError** — When redis.Get returns redis.Nil, return models.NewGenericNotFoundError(...) — not a raw error — so the HTTP layer maps it to 404. (`if cmd.Err() == redis.Nil { return nil, models.NewGenericNotFoundError(...) }`)
**Input validation before Redis ops** — Both GetProgress and UpsertProgress call input.Validate() and wrap the error before touching Redis. (`if err := input.Validate(); err != nil { return nil, fmt.Errorf("validate get progress input: %w", err) }`)
**Noop UpsertProgress silently succeeds** — adapterNoop.UpsertProgress returns nil (no-op), but adapterNoop.GetProgress returns GenericNotFoundError. This asymmetry is intentional — writes are discarded, reads report not-found. (`func (a *adapterNoop) UpsertProgress(...) error { return nil }`)
**Mock implements Service not Adapter** — MockProgressManager implements progressmanager.Service (not Adapter), covering the full service surface including DeleteProgressByRuntimeID. (`var _ progressmanager.Service = &MockProgressManager{}`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `adapter.go` | Defines Config struct, New() constructor, adapter struct, and adapterNoop stub. Entry point for wiring. | Config.Validate() must reject nil Redis client and zero Expiration — never relax these guards. |
| `progress.go` | Implements GetProgress and UpsertProgress on the real adapter using Redis GET/SET with JSON marshaling. | getKey() is the sole key-format authority; if key format changes, all existing Redis data becomes unreachable. |
| `noop.go` | No-op implementation for disabled progress tracking; GetProgress always returns not-found, UpsertProgress is silent. | Do not make noop methods return non-nil errors for UpsertProgress — callers treat the noop as safe to ignore. |
| `mock.go` | Testify mock for progressmanager.Service used in unit tests. | Mock implements Service, not Adapter — keep in sync if Service interface gains new methods. |

## Anti-Patterns

- Storing progress keys without namespace prefix — breaks multi-tenant isolation
- Returning raw redis errors instead of models.GenericNotFoundError for redis.Nil — breaks HTTP 404 mapping
- Calling redis operations without first calling input.Validate() — bypasses invariant checks
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

func ProvideProgressAdapter(redis *redis.Client, logger *slog.Logger) (progressmanager.Adapter, error) {
    return adapter.New(adapter.Config{
        Expiration: 24 * time.Hour,
        Redis:      redis,
        Logger:     logger,
        KeyPrefix:  "om",
    })
}
```

<!-- archie:ai-end -->
