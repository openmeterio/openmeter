# retry

<!-- archie:ai-start -->

> Decorator that wraps any streaming.Connector with automatic retry for transient ClickHouse errors (EOF, connection-acquire timeout, cluster-reshard code). BatchInsert and ValidateJSONPath are intentionally NOT retried to preserve idempotency.

## Patterns

**Decorator / pass-through wrapper** — Connector holds only downstreamConnector + retry config. Read methods wrap the downstream call in withRetry; mutation/lifecycle methods (BatchInsert, ValidateJSONPath, CreateNamespace, DeleteNamespace) delegate directly without retry. (`func (c *Connector) QueryMeter(...) { return withRetry(ctx, c, func() ([]meter.MeterQueryRow, error) { return c.downstreamConnector.QueryMeter(ctx, namespace, m, params) }) }`)
**Config.Validate() enforced by New()** — New() runs config.Validate() and errors on failure. Tests bypassing validation construct the Connector struct directly. (`c, err := New(Config{DownstreamConnector: ..., Logger: slog.Default(), RetryWaitDuration: 100*time.Millisecond, MaxTries: 3})`)
**Selective retry via RetryIf predicate** — Only io.ErrUnexpectedEOF, io.EOF, clickhouse.ErrAcquireConnTimeout, and chproto.ErrAllConnectionTriesFailed are retryable; all else fails immediately. New retryable errors go in the RetryIf closure. (`retry.RetryIf(func(err error) bool { if errors.Is(err, io.ErrUnexpectedEOF) { return true }; ... })`)
**CombineDelay(BackOffDelay, RandomDelay) with MaxDelay cap** — avast/retry-go with backoff + random jitter to avoid thundering-herd reconnects; MaxDelay caps exponential growth. (`retry.DelayType(retry.CombineDelay(retry.BackOffDelay, retry.RandomDelay)), retry.Delay(c.retryWaitDuration), retry.MaxDelay(c.maxDelay)`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `retry.go` | Entire package: Config, Connector, New, per-method wrappers, and generic withRetry[T any] using avast/retry-go with backoff+jitter. | BatchInsert is intentionally NOT wrapped — retrying inserts duplicates events. CreateNamespace/DeleteNamespace/ValidateJSONPath also bypass retry intentionally. |
| `retry_test.go` | Unit tests for Config validation, retry behaviour, context cancellation stopping retries, and backoff producing increasing delays. | Tests construct Connector directly (not via New()) to skip config validation when isolating withRetry — intentional. |

## Anti-Patterns

- Wrapping BatchInsert with withRetry — duplicate inserts break idempotency.
- Adding application errors (e.g. MeterNotFoundError) to RetryIf — only transient infrastructure errors retry.
- Relying on MaxDelay=0 — Validate() accepts it but retry-go treats 0 as no cap, allowing very long backoff.
- Constructing Connector without New() in production — skips config validation.

## Decisions

- **Backoff + random jitter (CombineDelay) with configurable MaxDelay.** — ClickHouse rolling restarts cause transient connection errors; jittered backoff avoids thundering-herd reconnect storms across concurrent workers.
- **BatchInsert not retried.** — Retrying inserts would duplicate rows; the sink worker's three-phase flush handles retry at a higher level.

## Example: Wrap a new read-only Connector method with retry

```
func (c *Connector) NewReadMethod(ctx context.Context, params streaming.NewParams) ([]SomeResult, error) {
    return withRetry(ctx, c, func() ([]SomeResult, error) {
        return c.downstreamConnector.NewReadMethod(ctx, params)
    })
}
// Mutation methods (BatchInsert) must NOT use withRetry.
```

<!-- archie:ai-end -->
