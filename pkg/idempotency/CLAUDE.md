# idempotency

<!-- archie:ai-start -->

> Single-purpose package that generates UUIDv7 idempotency keys for external API calls (e.g., Stripe). UUIDv7 provides monotonic, time-ordered keys that are safe to use as idempotency tokens without a shared counter.

## Patterns

**Key() in production, MustKey() in tests/init only** — Key() returns (string, error) and must be used in all application code paths. MustKey() panics on failure and is only appropriate in test setup or package-level init where panics are acceptable. (`key, err := idempotency.Key()
if err != nil {
    return fmt.Errorf("idempotency key: %w", err)
}`)
**One key per request — never reuse** — Each external API call that requires idempotency must call Key() to produce a fresh UUIDv7. Reusing the same key across requests causes the remote API to deduplicate what should be distinct operations. (`key, err := idempotency.Key()
// pass key to stripe.ChargeCreate or similar`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `key.go` | Entire package — exports Key() and MustKey(). uuid.NewV7() is the sole dependency. | uuid.NewV7() can fail in environments with broken entropy sources — always handle the error in production paths. MustKey() panics; never call it inside request handlers. |

## Anti-Patterns

- Using MustKey() in request-handling code paths — prefer Key() and propagate errors
- Reusing the same key across multiple requests — each call to Key() must produce a fresh key
- Hand-rolling UUID generation instead of using this package — breaks the centralized key strategy

## Decisions

- **UUIDv7 instead of UUIDv4 for idempotency keys** — UUIDv7 is time-ordered, making keys sortable and cache-friendly in database indexes while still being globally unique — important for Stripe and similar APIs that store idempotency keys in indexed tables.

## Example: Generate a fresh idempotency key before a Stripe charge call

```
import "github.com/openmeterio/openmeter/pkg/idempotency"

key, err := idempotency.Key()
if err != nil {
    return fmt.Errorf("idempotency key: %w", err)
}
// use key as stripe.ChargeCreateParams.IdempotencyKey
```

<!-- archie:ai-end -->
