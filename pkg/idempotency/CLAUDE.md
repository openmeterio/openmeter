# idempotency

<!-- archie:ai-start -->

> Single-purpose helper for generating UUIDv7-based idempotency keys, used where outbound operations need a unique, time-ordered idempotency token (e.g. notification/webhook/svix).

## Patterns

**UUIDv7 idempotency keys** — Keys are generated via `uuid.NewV7()` and returned as strings; v7 gives time-ordered uniqueness suitable for idempotency tokens. (`u, err := uuid.NewV7(); return u.String(), nil`)
**Error-returning Key plus MustKey panic variant** — `Key() (string, error)` is the safe API; `MustKey() string` panics on failure and is intended only for setup/non-recoverable paths. (`func MustKey() string { k, err := Key(); if err != nil { panic(err) }; return k }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `key.go` | Provides Key() and MustKey() for UUIDv7 idempotency key generation. | MustKey panics — per project convention avoid panics in production code paths; prefer Key() and propagate the error in service code. |

## Anti-Patterns

- Using MustKey() in request/service code paths where a returned error can be propagated instead.
- Switching to UUIDv4 or random strings, losing the time-ordered property of v7.

## Decisions

- **UUIDv7 over v4 for idempotency keys.** — v7 embeds a timestamp giving monotonic, sortable keys while preserving uniqueness.

## Example: Generate an idempotency key for an outbound request

```
import "github.com/openmeterio/openmeter/pkg/idempotency"

key, err := idempotency.Key()
if err != nil {
	return fmt.Errorf("generate idempotency key: %w", err)
}
```

<!-- archie:ai-end -->
