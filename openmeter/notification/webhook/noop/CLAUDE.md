# noop

<!-- archie:ai-start -->

> No-op implementation of webhook.Handler (Svix webhook delivery interface) wired by app/common when webhook delivery is disabled or unconfigured. Every method logs the call and returns webhook.ErrNotImplemented (or an empty slice) so the system stays operable without a real Svix backend.

## Patterns

**Compile-time interface assertion** — Assert the no-op satisfies webhook.Handler so the contract stays in sync as methods are added/removed. (`var _ webhook.Handler = (*Handler)(nil)`)
**Implement every webhook.Handler method** — Each interface method (RegisterEventTypes, CreateWebhook, UpdateWebhook, UpdateWebhookChannels, DeleteWebhook, GetWebhook, ListWebhooks, SendMessage, GetMessage, ResendMessage) must be present; adding a method to webhook.Handler requires a stub here or compilation fails. (`func (h Handler) CreateWebhook(ctx context.Context, params webhook.CreateWebhookInput) (*webhook.Webhook, error) { ... return nil, webhook.ErrNotImplemented }`)
**Return webhook.ErrNotImplemented (never panic)** — Methods returning error must return webhook.ErrNotImplemented; pointer returns use nil; ListWebhooks returns an empty []webhook.Webhook{} not nil. (`return []webhook.Webhook{}, webhook.ErrNotImplemented`)
**Logger tagged at construction** — New(logger) wraps the injected logger with a stable attribute so noop log lines are distinguishable; do not fall back to slog.Default(). (`logger: logger.With(slog.String("webhook_handler", "noop"))`)
**InfoContext logging on every call** — Each method logs intent with the request context before returning, giving operators visibility into calls that hit the disabled handler. (`h.logger.InfoContext(ctx, "sending message", "params", params)`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `noop.go` | Sole file: Handler struct (logger only) plus New(logger) constructor and stub implementations of all 10 webhook.Handler methods. | Keep the var _ webhook.Handler assertion; keep ListWebhooks returning an empty slice (callers may range over it); never substitute panic or slog.Default(). |

## Anti-Patterns

- Removing or stubbing only part of webhook.Handler — the compile-time assertion will break the build.
- Returning nil error from a write/delete method, which would falsely signal success to callers expecting ErrNotImplemented.
- Using slog.Default() instead of the injected, tagged logger.
- Adding real Svix logic here — that belongs in openmeter/notification/webhook/svix.

## Decisions

- **Provide a no-op Handler rather than a nil interface.** — Lets app/common wire notification without a configured webhook backend; callers can invoke methods safely and branch on webhook.ErrNotImplemented instead of nil-checking the handler.

<!-- archie:ai-end -->
