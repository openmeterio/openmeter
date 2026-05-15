# retry

<!-- archie:ai-start -->

> Decorator that wraps any streaming.Connector with automatic retry logic for transient ClickHouse errors (EOF, connection-acquire timeout, cluster-reshard error code). BatchInsert and ValidateJSONPath are intentionally not retried to preserve idempotency.

## Patterns

**Decorator / pass-through wrapper** — Connector embeds no state beyond the downstreamConnector and retry config. Every streaming.Connector method either calls withRetry wrapping the downstream call, or delegates directly (BatchInsert, ValidateJSONPath, CreateNamespace, DeleteNamespace do not retry). (`func (c *Connector) QueryMeter(...) ([]meter.MeterQueryRow, error) { return withRetry(ctx, c, func() ([]meter.MeterQueryRow, error) { return c.downstreamConnector.QueryMeter(ctx, namespace, m, params) }) }`)
**Config.Validate() enforced by New()** — New() calls config.Validate() and returns an error if validation fails. Tests that want to bypass config validation for withRetry isolation should construct the Connector struct directly. (`c, err := New(Config{DownstreamConnector: ..., Logger: slog.Default(), RetryWaitDuration: 100*time.Millisecond, MaxTries: 3})`)
**Selective retry via RetryIf predicate** — Only io.ErrUnexpectedEOF, io.EOF, clickhouse.ErrAcquireConnTimeout, and chproto.ErrAllConnectionTriesFailed are retryable. All other errors fail immediately. Add new retryable errors to the RetryIf closure in withRetry. (`retry.RetryIf(func(err error) bool { if errors.Is(err, io.ErrUnexpectedEOF) { return true }; ...; return false })`)
**CombineDelay(BackOffDelay, RandomDelay) with configurable MaxDelay** — avast/retry-go is used with backoff + random jitter to avoid thundering-herd reconnect storms. MaxDelay caps exponential growth; set it to a non-zero value to prevent unbounded delays. (`retry.DelayType(retry.CombineDelay(retry.BackOffDelay, retry.RandomDelay)), retry.Delay(c.retryWaitDuration), retry.MaxDelay(c.maxDelay)`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `retry.go` | Entire package: Config struct, Connector struct, New constructor, per-method wrappers, and the generic withRetry[T any] helper using avast/retry-go with backoff+jitter. | BatchInsert is intentionally NOT wrapped in withRetry — retrying inserts could cause duplicate event ingestion. CreateNamespace/DeleteNamespace and ValidateJSONPath also bypass retry intentionally. |
| `retry_test.go` | Unit tests for Config validation, retry behaviour, context cancellation stopping retries, and backoff producing increasing delays. | Tests construct Connector directly (not via New()) to skip config validation when testing withRetry in isolation — this is intentional, not a bug. |

## Anti-Patterns

- Wrapping BatchInsert with withRetry — duplicate event inserts on retry break idempotency guarantees.
- Adding application-level errors (e.g. MeterNotFoundError) to the RetryIf predicate — only transient infrastructure errors should be retried.
- Setting MaxDelay to 0 — Config.Validate() accepts 0 as valid (non-negative) but retry-go interprets 0 as no cap, potentially causing very long delays under exponential backoff.
- Constructing the Connector without calling New() in production code — always use New() to ensure config validation runs.

## Decisions

- **Backoff + random jitter (CombineDelay) with configurable MaxDelay** — ClickHouse cluster restarts during rolling updates cause transient connection errors; backoff with jitter avoids thundering-herd reconnect storms from multiple concurrent workers.
- **BatchInsert not retried** — Retrying inserts on transient errors would produce duplicate rows in the events table, breaking the exactly-once ingestion guarantee. The sink worker's three-phase flush ordering handles retry at a higher level.

## Example: Wrap a new streaming.Connector method with retry (read-only operations only)

```
// In retry.go, add a new method following the existing pattern:
func (c *Connector) NewReadMethod(ctx context.Context, params streaming.NewParams) ([]SomeResult, error) {
    return withRetry(ctx, c, func() ([]SomeResult, error) {
        return c.downstreamConnector.NewReadMethod(ctx, params)
    })
}
// Note: mutation methods (BatchInsert) must NOT use withRetry
```

<!-- archie:ai-end -->
