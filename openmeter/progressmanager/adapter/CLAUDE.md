# adapter

<!-- archie:ai-start -->

> Persistence layer for the progressmanager domain: a Redis-backed implementation of the progressmanager.Adapter interface that stores per-operation progress counters as JSON under namespaced keys with TTL expiration. Also ships noop and testify-mock variants.

## Patterns

**Config-validated constructor** — New(Config) validates via config.Validate() (Expiration>0, Redis!=nil, Logger!=nil) before constructing the unexported *adapter; never construct the struct directly. (`func New(config Config) (progressmanager.Adapter, error) { if err := config.Validate(); err != nil { return nil, err } ... }`)
**Interface compile-time assertion** — Every concrete type asserts the parent interface: `var _ progressmanager.Adapter = (*adapter)(nil)` and `(*adapterNoop)(nil)`. Add the same line for any new variant. (`var _ progressmanager.Adapter = (*adapter)(nil)`)
**Validate input before I/O** — Each adapter method calls input.Validate() first and wraps the error (fmt.Errorf("validate get progress input: %w", err)) before touching Redis. (`if err := input.Validate(); err != nil { return nil, fmt.Errorf("validate get progress input: %w", err) }`)
**redis.Nil -> domain not-found** — Reads translate redis.Nil into models.NewGenericNotFoundError; other Redis errors are wrapped verbatim. Never leak redis.Nil upward. (`if cmd.Err() == redis.Nil { return nil, models.NewGenericNotFoundError(fmt.Errorf("progress not found for id: %s", input.ProgressID.ID)) }`)
**Namespaced JSON value storage** — UpsertProgress json.Marshal's the entity.Progress and SETs it with a.expiration TTL; GetProgress json.Unmarshal's it back. Keys built only via a.getKey(ProgressID). (`a.redis.Set(ctx, a.getKey(input.ProgressID), data, a.expiration)`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `adapter.go` | Config struct + Validate, New() Redis adapter constructor, NewNoop() constructor, struct definitions and interface assertions. | KeyPrefix is optional (empty allowed); Expiration/Redis/Logger are required. Don't add slog.Default() fallbacks — Logger is injected. |
| `progress.go` | Redis Get/Upsert implementations plus getKey() key builder. | getKey concatenates staticKeyPrefix (already ends in ':') with another ':' producing 'progress::ns:id' / '<prefix>:progress::ns:id'. Both branches must stay in sync if the key scheme changes. |
| `noop.go` | adapterNoop methods: GetProgress always returns NotFound, UpsertProgress is a silent success. | Imports entity via the bare 'entity' alias here while progress.go uses the same path — keep noop's GetProgress returning NotFound so disabled progress tracking degrades gracefully. |
| `mock.go` | MockProgressManager (testify mock) implementing progressmanager.Service for service-layer tests. | Mocks the SERVICE interface (not Adapter) and declares DeleteProgressByRuntimeID, which is NOT on the current progressmanager.Service interface — drift; verify against service.go before relying on it. |

## Anti-Patterns

- Constructing &adapter{...} directly, bypassing Config.Validate().
- Returning redis.Nil to callers instead of models.NewGenericNotFoundError.
- Skipping input.Validate() before Redis access.
- Building Redis keys inline instead of through getKey().
- Falling back to slog.Default() instead of requiring Config.Logger.

## Decisions

- **Redis (not Postgres/Ent) backs progress.** — Progress is ephemeral, high-churn operation telemetry; TTL expiration auto-evicts stale runs without migrations.
- **Noop and mock variants live alongside the real adapter.** — Lets DI wire a no-op when progress tracking is disabled and lets service tests use testify without Redis.

## Example: Construct the Redis-backed adapter

```
import (
  "github.com/redis/go-redis/v9"
  "github.com/openmeterio/openmeter/openmeter/progressmanager/adapter"
)

adp, err := adapter.New(adapter.Config{
  Expiration: 24 * time.Hour,
  Redis:      redisClient,
  Logger:     logger,
})
if err != nil { return err }
```

<!-- archie:ai-end -->
