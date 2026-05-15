# noop

<!-- archie:ai-start -->

> No-op implementation of webhook.Handler used when Svix is unconfigured. Every method logs the attempted call and returns webhook.ErrNotImplemented, giving callers a defined error instead of a nil panic.

## Patterns

**Compile-time interface assertion** — File opens with `var _ webhook.Handler = (*Handler)(nil)` to catch missing methods at compile time. If webhook.Handler gains a new method, this file must implement it or the build fails. (`var _ webhook.Handler = (*Handler)(nil)`)
**Log-then-return-ErrNotImplemented** — Every method calls h.logger.InfoContext with operation name and params, then returns webhook.ErrNotImplemented (or nil, ErrNotImplemented for pointer returns). Never return nil error — callers must distinguish noop from success. (`h.logger.InfoContext(ctx, "sending message", "params", params)
return nil, webhook.ErrNotImplemented`)
**Structured logger tagging in constructor** — New(logger *slog.Logger) tags the logger with slog.String("webhook_handler", "noop") so all log lines are attributable to this handler without additional fields at each call site. (`logger.With(slog.String("webhook_handler", "noop"))`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `noop.go` | Complete no-op implementation of webhook.Handler; single file, no other files in this package. | If webhook.Handler gains a new method, add it here or the compile-time assertion at the top will fail the build. |

## Anti-Patterns

- Returning nil error from any method — callers (notification.Service) must know the noop is not functional
- Omitting the compile-time interface assertion `var _ webhook.Handler = (*Handler)(nil)`
- Adding real business logic or state — this is intentionally a stub with no side-effects beyond logging
- Panicking instead of returning ErrNotImplemented — the noop must be safe to use in production when Svix is unconfigured

## Decisions

- **Return webhook.ErrNotImplemented instead of nil** — Callers (notification.Service) distinguish noop from real failures; a nil error would silently swallow webhook operations when Svix is not configured, causing invisible data loss.
- **Single-file package with no sub-packages** — The noop handler has no state or dependencies beyond slog; splitting it would add navigation overhead with no benefit.

## Example: Add a new method to satisfy an updated webhook.Handler interface

```
func (h Handler) NewMethod(ctx context.Context, params webhook.NewMethodInput) (*webhook.Result, error) {
	h.logger.InfoContext(ctx, "new method called", "params", params)
	return nil, webhook.ErrNotImplemented
}
```

<!-- archie:ai-end -->
