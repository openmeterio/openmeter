# idempotency

<!-- archie:ai-start -->

> Single-purpose package that generates UUIDv7 idempotency keys for external API calls (e.g., Stripe). UUIDv7 provides monotonic, time-ordered keys that are safe to use as idempotency tokens without a shared counter.

## Patterns

**Use Key() in production, MustKey() in tests/init** — Key() returns (string, error) and should be used in application code paths. MustKey() panics on failure and is only appropriate in test setup or package-level init. (`key, err := idempotency.Key()
if err != nil {
    return fmt.Errorf("idempotency key: %w", err)
}`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `key.go` | Exports Key() and MustKey(). The entire package is this one file. | uuid.NewV7() can fail in environments with broken entropy sources — always handle the error in production paths. |

## Anti-Patterns

- Using MustKey() in request-handling code paths — prefer Key() and propagate errors
- Reusing the same key across multiple requests — each call to Key() must produce a fresh key

## Decisions

- **UUIDv7 instead of UUIDv4 for idempotency keys.** — UUIDv7 is time-ordered, making keys sortable and slightly more cache-friendly in database indexes while still being globally unique.

<!-- archie:ai-end -->
