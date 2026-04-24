# nopublisher

<!-- archie:ai-start -->

> Adapter utilities for converting between Watermill's NoPublishHandlerFunc and HandlerFunc signatures. Used when a handler is logically a consumer (no outgoing messages) but must be wired into a router slot that expects HandlerFunc, or vice versa.

## Patterns

**NoPublisherHandlerToHandlerFunc adapter** — Wraps a NoPublishHandlerFunc into a HandlerFunc that always returns (nil, err). Use when the router API requires HandlerFunc but the handler never produces output messages. (`router.AddHandler("name", topic, sub, pub, nopublisher.NoPublisherHandlerToHandlerFunc(myConsumer))`)
**HandlerFuncToNoPublisherHandler with production guard** — Converts a HandlerFunc into a NoPublishHandlerFunc; returns ErrMessagesProduced if the inner handler emits any messages. Use to enforce the no-publish contract at runtime. (`nopublisher.HandlerFuncToNoPublisherHandler(h) // panics-equivalent via error if messages returned`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `nopublisher.go` | Two pure adapter functions and one sentinel error. No state. | ErrMessagesProduced is a hard error — if a wrapped HandlerFunc ever returns messages, the consumer will error and retry. Only wrap handlers that are truly side-effect-only. |

## Anti-Patterns

- Using HandlerFuncToNoPublisherHandler on a handler that conditionally produces messages — ErrMessagesProduced will cause infinite retries.
- Reimplementing these adapters inline in worker code instead of importing this package.

<!-- archie:ai-end -->
