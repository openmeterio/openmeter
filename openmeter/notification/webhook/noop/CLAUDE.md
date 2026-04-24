# noop

<!-- archie:ai-start -->

> No-op implementation of webhook.Handler used when Svix is unconfigured. Every method logs the attempted call and returns webhook.ErrNotImplemented, ensuring callers get a defined error rather than a nil panic.

## Patterns

**Interface compliance assertion** — File opens with `var _ webhook.Handler = (*Handler)(nil)` to enforce compile-time interface compliance. (`var _ webhook.Handler = (*Handler)(nil)`)
**Log-then-return-ErrNotImplemented** — Every method calls h.logger.InfoContext with the operation name and params, then returns webhook.ErrNotImplemented (or nil, ErrNotImplemented for pointer returns). (`h.logger.InfoContext(ctx, "sending message", "params", params)
return nil, webhook.ErrNotImplemented`)
**Structured logger injection** — Constructor New(logger *slog.Logger) tags the logger with slog.String("webhook_handler", "noop") so all log lines are attributable. (`logger.With(slog.String("webhook_handler", "noop"))`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `noop.go` | Complete no-op implementation of webhook.Handler; single file, no other files in this package. | If webhook.Handler gains a new method, this file must add the method or it will fail the compile-time assertion at the top. |

## Anti-Patterns

- Returning nil error from any method — callers must know the noop is not functional, so always return webhook.ErrNotImplemented
- Omitting the compile-time interface assertion var _ webhook.Handler = (*Handler)(nil)
- Adding real business logic or state — this is intentionally a stub

## Decisions

- **Return webhook.ErrNotImplemented instead of nil** — Callers (notification.Service) distinguish noop from real failures; a nil error would silently swallow operations when Svix is not configured.

<!-- archie:ai-end -->
