# retry

<!-- archie:ai-start -->

> Decorator that wraps any streaming.Connector with automatic retry logic for transient ClickHouse errors (EOF, connection-acquire timeout, cluster-reshard error code). BatchInsert and ValidateJSONPath are intentionally not retried.

## Patterns

**Decorator / pass-through wrapper** — Connector embeds no state beyond the downstreamConnector and retry config. Every streaming.Connector method either calls withRetry wrapping the downstream call, or delegates directly (BatchInsert, ValidateJSONPath, CreateNamespace, DeleteNamespace do not retry). (`func (c *Connector) QueryMeter(...) ([]meter.MeterQueryRow, error) { return withRetry(ctx, c, func() ([]meter.MeterQueryRow, error) { return c.downstreamConnector.QueryMeter(ctx, namespace, m, params) }) }`)
**Config.Validate() before New()** — New() calls config.Validate() and returns an error if validation fails. Tests must call New() with a valid config or construct the Connector struct directly for low-level tests. (`c, err := New(Config{DownstreamConnector: ..., Logger: slog.Default(), RetryWaitDuration: 100*time.Millisecond, MaxTries: 3})`)
**Selective retry via RetryIf predicate** — Only io.ErrUnexpectedEOF, io.EOF, clickhouse.ErrAcquireConnTimeout, and chproto.ErrAllConnectionTriesFailed are retryable. All other errors fail immediately. Add new retryable errors to the RetryIf closure in withRetry. (`retry.RetryIf(func(err error) bool { if errors.Is(err, io.ErrUnexpectedEOF) { return true }; ... return false })`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `retry.go` | Entire package: Config struct, Connector struct, New constructor, per-method wrappers, and the generic withRetry helper using avast/retry-go with backoff+jitter. | BatchInsert is not wrapped in withRetry — retrying inserts could cause duplicates. CreateNamespace/DeleteNamespace and ValidateJSONPath also bypass retry intentionally. |
| `retry_test.go` | Unit tests for Config validation, retry behaviour, context cancellation stopping retries, and backoff producing increasing delays. | Tests construct Connector directly (not via New()) to skip config validation when testing withRetry in isolation — this is intentional. |

## Anti-Patterns

- Wrapping BatchInsert with withRetry — duplicate event inserts on retry break idempotency guarantees.
- Adding application-level errors (e.g. MeterNotFoundError) to the RetryIf predicate — only transient infrastructure errors should be retried.
- Setting MaxDelay to 0 — the Config.Validate() accepts 0 as valid (non-negative) but retry-go interprets 0 as no cap, potentially causing very long delays under exponential backoff.

## Decisions

- **Backoff + random jitter (CombineDelay) with configurable MaxDelay** — ClickHouse cluster restarts during rolling updates cause transient connection errors; backoff with jitter avoids thundering-herd reconnect storms from multiple concurrent workers.

<!-- archie:ai-end -->
