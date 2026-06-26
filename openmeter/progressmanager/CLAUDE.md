# progressmanager

<!-- archie:ai-start -->

> Small standalone domain that tracks per-operation progress counters (e.g. long-running ClickHouse queries). The root files declare the Adapter and Service interfaces; entity/ holds the I/O-free domain types, adapter/ is the Redis-backed store (with noop/mock variants), httpdriver/ exposes a read-only GetProgress endpoint.

## Patterns

**Interface-only root** — adapter.go and service.go declare Adapter/Service as aliases of ProgressManagerAdapter/ProgressManagerService — both expose the same GetProgress/UpsertProgress pair over entity input types. No logic lives at the root. (`type Service interface { ProgressManagerService } with GetProgress(ctx, entity.GetProgressInput) and UpsertProgress(ctx, entity.UpsertProgressInput)`)
**Dedicated method-input structs** — Operations take entity.GetProgressInput / entity.UpsertProgressInput wrappers (validated in entity/) rather than loose parameters. (`GetProgress(ctx context.Context, input entity.GetProgressInput) (*entity.Progress, error)`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `adapter.go` | Adapter interface (Redis-backed persistence contract) | Keep it a thin alias of ProgressManagerAdapter; the real store, noop, and mock live in adapter/ |
| `service.go` | Service interface mirroring the adapter surface | Service and Adapter expose the same two methods — keep them in sync when adding operations |

## Anti-Patterns

- Putting Redis access, key construction, or validation logic in the root interfaces instead of adapter/ and entity/
- Letting Service and Adapter method sets drift apart
- Backing progress with Postgres/Ent — this domain is intentionally Redis-only with TTL expiry

## Decisions

- **Root is interface-only; implementation, types, and transport are all in sub-packages** — Keeps the domain contract importable without pulling in Redis or HTTP dependencies, and lets app/common swap the real adapter for the noop variant

<!-- archie:ai-end -->
