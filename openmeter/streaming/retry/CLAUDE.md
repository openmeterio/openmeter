# retry

<!-- archie:ai-start -->

> Decorator that wraps any streaming.Connector with retry-on-transient-failure behavior for ClickHouse reads. Package streamingretry; its Connector forwards every call to a downstream connector, retrying only read methods on retryable ClickHouse/network errors.

## Patterns

**Decorator over streaming.Connector** — Connector holds a downstreamConnector streaming.Connector and asserts `var _ streaming.Connector = (*Connector)(nil)`. Every interface method delegates to the downstream; read methods wrap the call in withRetry, mutating/namespace methods (BatchInsert, ValidateJSONPath, Create/DeleteNamespace) delegate directly without retry. (`func (c *Connector) ListEvents(ctx, ns, params) (...) { return withRetry(ctx, c, func() ([]streaming.RawEvent, error) { return c.downstreamConnector.ListEvents(ctx, ns, params) }) }`)
**Generic withRetry helper** — withRetry[T any](ctx, c, fn func() (T, error)) wraps retry.DoWithData (avast/retry-go/v4) with retry.Context(ctx), retry.Attempts(c.maxTries), LastErrorOnly, CombineDelay(BackOffDelay, RandomDelay), Delay/MaxDelay, OnRetry logging, and a RetryIf predicate. All retried methods route through it. (`return retry.DoWithData(fn, retry.Context(ctx), retry.Attempts(uint(c.maxTries)), retry.RetryIf(...))`)
**Explicit retryable-error allowlist** — RetryIf returns true only for io.EOF/io.ErrUnexpectedEOF, clickhouse.ErrAcquireConnTimeout, and *clickhouse.Exception with Code == chproto.ErrAllConnectionTriesFailed (cluster upscale/downscale, CH restarts). Everything else fails fast — do not broaden this to retry arbitrary errors. (`if errors.Is(err, io.ErrUnexpectedEOF) || errors.Is(err, io.EOF) { return true }`)
**Validated config constructor** — New(Config) calls Config.Validate() which collects errors via errors.Join: DownstreamConnector and Logger required, RetryWaitDuration > 0, MaxTries >= 1, MaxDelay >= 0. Config fields are copied into unexported Connector fields. (`if c.MaxTries < 1 { errs = append(errs, errors.New("max retries must be greater than or equal to 1")) }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `retry.go` | Config + Validate, Connector decorator, all delegating methods, generic withRetry, RetryIf allowlist | BatchInsert and ValidateJSONPath are deliberately NOT retried; adding retry there would risk duplicate inserts. New downstream interface methods must be added here too or they won't compile against streaming.Connector |
| `retry_test.go` | Config validation tests, constructor tests, withRetry behavior (success/retry/exhaustion/non-retryable), context-cancellation and backoff+jitter timing | noopConnector embeds streaming.Connector to satisfy the interface for config tests; backoff test asserts total elapsed exceeds flat-delay sum |

## Anti-Patterns

- Adding retry to BatchInsert or other write/mutation paths (non-idempotent — risks duplicate events)
- Broadening RetryIf to retry generic errors instead of the explicit EOF / conn-timeout / ErrAllConnectionTriesFailed allowlist
- Bypassing withRetry for a new read method, or calling the downstream connector directly without the generic helper
- Using slog.Default() as a fallback — Logger is a required config field and must be injected

## Decisions

- **Only read methods are retried; writes delegate straight through** — ClickHouse reads are idempotent and safe to repeat across CH restarts; BatchInsert is not, so retrying could double-insert events
- **Retry triggers limited to connection-class errors (EOF, acquire timeout, ErrAllConnectionTriesFailed)** — These specifically occur during cluster upscale/downscale and CH restarts where the connection pool neglects pings; query-logic errors should fail fast

## Example: Wrapping a downstream read with the generic retry helper and allowlist

```
func withRetry[T any](ctx context.Context, c *Connector, fn func() (T, error)) (T, error) {
	return retry.DoWithData(fn,
		retry.Context(ctx),
		retry.Attempts(uint(c.maxTries)),
		retry.LastErrorOnly(true),
		retry.DelayType(retry.CombineDelay(retry.BackOffDelay, retry.RandomDelay)),
		retry.Delay(c.retryWaitDuration),
		retry.MaxDelay(c.maxDelay),
		retry.RetryIf(func(err error) bool {
			if errors.Is(err, io.ErrUnexpectedEOF) || errors.Is(err, io.EOF) {
				return true
			}
			if errors.Is(err, clickhouse.ErrAcquireConnTimeout) {
				return true
			}
// ...
```

<!-- archie:ai-end -->
