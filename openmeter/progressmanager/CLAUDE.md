# progressmanager

<!-- archie:ai-start -->

> Tracks progress of long-running async operations (e.g. ClickHouse export jobs) via a Redis-backed key-value store with TTL. Root package defines Service and Adapter interfaces; entity/ sub-package owns pure domain types; adapter/ sub-package provides Redis, noop, and mock implementations; httpdriver/ exposes a single GetProgress HTTP endpoint.

## Patterns

**Validate() on every input and entity type** — GetProgressInput, UpsertProgressInput, and Progress all implement Validate() using errors.Join to collect all validation errors. The adapter calls Validate() before any Redis operation. (`if err := input.Validate(); err != nil { return nil, err }`)
**Namespaced Redis key format** — Redis keys include namespace and operation ID (e.g. progress:<namespace>:<id>) to prevent multi-tenant key collisions. Key format is defined in adapter/progress.go getKey() — do not change without a migration plan. (`func getKey(input entity.GetProgressInput) string { return fmt.Sprintf("progress:%s:%s", input.Namespace, input.ID) }`)
**redis.Nil mapped to GenericNotFoundError** — When Redis returns redis.Nil (key not found), the adapter maps it to models.GenericNotFoundError — enabling correct HTTP 404 mapping via GenericErrorEncoder. (`if errors.Is(err, redis.Nil) { return nil, models.NewGenericNotFoundError("progress", input.ID) }`)
**Noop UpsertProgress is a safe silent no-op** — adapterNoop.UpsertProgress always returns nil. Callers rely on this for test/disabled-feature wiring — the noop must never return an error from UpsertProgress. (`func (a *adapterNoop) UpsertProgress(_ context.Context, _ entity.UpsertProgressInput) error { return nil }`)
**Service and Adapter interfaces mirror each other** — Service embeds ProgressManagerService; Adapter embeds ProgressManagerAdapter — both have identical method signatures. Service is the outward-facing interface; Adapter is the storage contract. Keep them in sync. (`type Service interface { ProgressManagerService }; type Adapter interface { ProgressManagerAdapter }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `service.go` | Defines the outward-facing Service interface (GetProgress, UpsertProgress). Must stay identical to Adapter method signatures. | Adding a method to Service without adding it to Adapter (and vice versa) breaks the compile-time interface chain. |
| `adapter.go` | Defines the Adapter interface (storage contract). Deliberately mirrors Service — do not diverge. | If Service and Adapter signatures diverge, the Wire provider that injects Adapter as Service will fail to compile. |
| `entity/progressmanager.go` | All domain value types (ProgressID, Progress, GetProgressInput, UpsertProgressInput) and their Validate() methods. Pure types only — no Redis or Ent imports. | Adding persistence logic or external imports here breaks entity/adapter separation and creates import cycles. |
| `adapter/adapter.go` | Redis implementation, adapterNoop, and MockProgressManager (testify mock for Service). Key format, TTL, and redis.Nil mapping live here. | Changing getKey() format orphans all in-flight progress records. Making adapterNoop.UpsertProgress return an error breaks callers that rely on noop writes being safe. |
| `httpdriver/progress.go` | Single GetProgress HTTP endpoint using httptransport.NewHandlerWithArgs; namespace resolved via namespacedriver.NamespaceDecoder. | Do not call the domain service from the decoder function — decode maps request fields only; the operation func calls the service. |

## Anti-Patterns

- Storing progress keys without namespace prefix — breaks multi-tenant isolation
- Returning raw redis errors instead of models.GenericNotFoundError for redis.Nil — breaks HTTP 404 mapping
- Changing the Redis key format in getKey() without a migration plan — orphans all in-flight progress records
- Adding persistence, Redis, or HTTP logic to entity/ types — must be pure domain types only
- Making adapterNoop.UpsertProgress return an error — callers rely on noop writes being safe no-ops

## Decisions

- **Redis (not PostgreSQL/Ent) as the backing store** — Progress records are ephemeral, TTL-scoped, and high-write; a Redis hash with TTL avoids Postgres table bloat and eliminates the need for a periodic cleanup job.
- **Entity types in a separate entity/ sub-package** — Keeps the root package as a thin interface definition layer; entity types can be imported by adapter/ without creating a circular import with the root package.

## Example: Upsert then get progress — standard lifecycle showing validation and error mapping

```
// Upsert
upsertInput := entity.UpsertProgressInput{
    Namespace: ns,
    ID:        opID,
    Progress:  entity.Progress{Total: 100, Completed: 50, UpdatedAt: time.Now()},
}
if err := upsertInput.Validate(); err != nil {
    return fmt.Errorf("invalid upsert input: %w", err)
}
if err := svc.UpsertProgress(ctx, upsertInput); err != nil {
    return err
}

// Get
getInput := entity.GetProgressInput{Namespace: ns, ID: opID}
// ...
```

<!-- archie:ai-end -->
