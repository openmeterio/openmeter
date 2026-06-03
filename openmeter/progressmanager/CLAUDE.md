# progressmanager

<!-- archie:ai-start -->

> Tracks progress of long-running async operations (e.g. ClickHouse export jobs) via a Redis-backed key-value store with TTL. Root package defines mirrored Service and Adapter interfaces; entity/ owns pure domain types; adapter/ provides Redis, noop, and mock implementations; httpdriver/ exposes a single GetProgress endpoint.

## Patterns

**Validate() on every input and entity type** — GetProgressInput, UpsertProgressInput, and Progress implement Validate() via errors.Join; the adapter calls Validate() before any Redis operation. (`if err := input.Validate(); err != nil { return nil, err }`)
**Namespaced Redis key format** — Redis keys include namespace and operation ID to prevent multi-tenant collisions; the format lives in adapter/progress.go getKey() — do not change without a migration plan. (`func getKey(input entity.GetProgressInput) string { return fmt.Sprintf("progress:%s:%s", input.Namespace, input.ID) }`)
**redis.Nil mapped to GenericNotFoundError** — Key-not-found from Redis is mapped to models.GenericNotFoundError so GenericErrorEncoder produces HTTP 404. (`if errors.Is(err, redis.Nil) { return nil, models.NewGenericNotFoundError("progress", input.ID) }`)
**Noop UpsertProgress is a safe silent no-op** — adapterNoop.UpsertProgress always returns nil; callers rely on this for test/disabled-feature wiring. (`func (a *adapterNoop) UpsertProgress(_ context.Context, _ entity.UpsertProgressInput) error { return nil }`)
**Service and Adapter interfaces mirror each other** — Service embeds ProgressManagerService; Adapter embeds ProgressManagerAdapter — identical signatures so the Wire provider can inject Adapter as Service. (`type Service interface { ProgressManagerService }; type Adapter interface { ProgressManagerAdapter }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `service.go` | Outward-facing Service interface (GetProgress, UpsertProgress); must stay identical to Adapter signatures. | Adding a method to Service without adding it to Adapter (and vice versa) breaks the compile-time interface chain. |
| `adapter.go` | Adapter interface (storage contract), deliberately mirroring Service. | Diverging Service/Adapter signatures makes the Wire provider that injects Adapter as Service fail to compile. |
| `entity/progressmanager.go` | All domain value types (ProgressID, Progress, GetProgressInput, UpsertProgressInput) and their Validate(). Pure types only. | Adding persistence logic or external imports here breaks entity/adapter separation and creates import cycles. |
| `adapter/adapter.go` | Redis implementation, adapterNoop, and MockProgressManager (testify mock for Service); key format, TTL, and redis.Nil mapping live here. | Changing getKey() orphans all in-flight progress records; making adapterNoop.UpsertProgress return an error breaks callers relying on safe noop writes. |
| `httpdriver/progress.go` | Single GetProgress endpoint via httptransport.NewHandlerWithArgs; namespace resolved via namespacedriver.NamespaceDecoder. | Do not call the domain service from the decoder — decode maps request fields only; the operation func calls the service. Convert responses via progressToAPI(). |

## Anti-Patterns

- Storing progress keys without a namespace prefix — breaks multi-tenant isolation
- Returning raw redis errors instead of models.GenericNotFoundError for redis.Nil — breaks HTTP 404 mapping
- Changing the Redis key format in getKey() without a migration plan — orphans all in-flight progress records
- Adding persistence, Redis, or HTTP logic to entity/ types — must be pure domain types only
- Making adapterNoop.UpsertProgress return an error — callers rely on noop writes being safe no-ops

## Decisions

- **Redis (not PostgreSQL/Ent) as the backing store** — Progress records are ephemeral, TTL-scoped, and high-write; a Redis hash with TTL avoids Postgres table bloat and a cleanup job.
- **Entity types in a separate entity/ sub-package** — Keeps the root package as a thin interface layer; entity types can be imported by adapter/ without a circular import.

<!-- archie:ai-end -->
