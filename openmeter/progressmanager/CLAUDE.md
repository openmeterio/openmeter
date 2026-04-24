# progressmanager

<!-- archie:ai-start -->

> Tracks progress of long-running async operations (e.g. ClickHouse export jobs) via Redis-backed key-value store with TTL. Root package defines Service and Adapter interfaces; entity/ sub-package owns domain types; adapter/ provides Redis, noop, and mock implementations.

## Patterns

**Validate() on every input type** — GetProgressInput, UpsertProgressInput, and Progress all implement Validate() with errors.Join. Adapter calls Validate() before any Redis operation. (`if err := input.Validate(); err != nil { return nil, err }`)
**Namespaced Redis key format** — Redis keys include namespace and operation ID to prevent multi-tenant collisions. Key format defined in adapter/progress.go getKey(). (`func getKey(input entity.GetProgressInput) string { return fmt.Sprintf("progress:%s:%s", input.Namespace, input.ID) }`)
**redis.Nil mapped to GenericNotFoundError** — When Redis returns redis.Nil (key not found), the adapter maps it to models.GenericNotFoundError — enabling correct HTTP 404 mapping. (`if errors.Is(err, redis.Nil) { return nil, models.NewGenericNotFoundError("progress", input.ID) }`)
**Noop UpsertProgress is a safe silent no-op** — adapterNoop.UpsertProgress always returns nil. Callers rely on this for test/disabled-feature wiring. (`func (a *adapterNoop) UpsertProgress(_ context.Context, _ entity.UpsertProgressInput) error { return nil }`)
**Service and Adapter interfaces mirror each other** — Service embeds ProgressManagerService; Adapter embeds ProgressManagerAdapter — both have identical method signatures. Service is the outward-facing interface; Adapter is the storage contract. (`type Service interface { ProgressManagerService }; type Adapter interface { ProgressManagerAdapter }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `service.go` | Defines outward-facing Service interface. Must stay identical to Adapter method signatures. | Adding a method to Service without adding it to Adapter (and vice versa) breaks the compile-time interface chain. |
| `adapter.go` | Defines Adapter interface — storage contract for the Redis adapter. | Service and Adapter are deliberately identical here; do not diverge them. |
| `entity/progressmanager.go` | All domain value types and Validate() methods. Pure types only — no Redis or Ent imports. | Adding persistence logic here breaks the entity/adapter separation. |
| `adapter/adapter.go` | Redis implementation + noop + mock. Key format, TTL, and redis.Nil mapping live here. | Changing getKey() format orphans all in-flight progress records with no migration path. |

## Anti-Patterns

- Storing progress keys without namespace prefix — breaks multi-tenant isolation
- Returning raw redis errors instead of models.GenericNotFoundError for redis.Nil
- Changing the Redis key format in getKey() without a migration plan
- Adding Ent/DB dependencies to entity/ — it must be pure domain types
- Making adapterNoop.UpsertProgress return an error — callers rely on noop writes being safe

## Decisions

- **Redis (not PostgreSQL/Ent) as the backing store** — Progress records are ephemeral, TTL-scoped, and high-write; a Redis hash with TTL avoids Postgres table bloat and eliminates the need for a cleanup job.
- **Entity types in a separate entity/ sub-package** — Keeps the root package as a thin interface definition layer; entity types can be imported by adapter/ without creating a circular import with the root package.

<!-- archie:ai-end -->
