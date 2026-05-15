# nopublisher

<!-- archie:ai-start -->

> Provides two pure adapter functions for converting between Watermill's NoPublishHandlerFunc and HandlerFunc signatures, used when a handler is logically a consumer but must be wired into a router slot expecting a different function signature.

## Patterns

**NoPublisherHandlerToHandlerFunc for consumer-only handlers** — Wraps a NoPublishHandlerFunc into a HandlerFunc that always returns (nil, err). Use when the Watermill router API requires HandlerFunc but the handler never produces output messages. (`router.AddHandler("name", topic, sub, pub, nopublisher.NoPublisherHandlerToHandlerFunc(myConsumer))`)
**HandlerFuncToNoPublisherHandler with production guard** — Converts a HandlerFunc into a NoPublishHandlerFunc; returns ErrMessagesProduced if the inner handler emits any messages. Only wrap handlers that are truly side-effect-only — any returned message causes an error and infinite retries. (`nopublisher.HandlerFuncToNoPublisherHandler(h) // errors if h returns messages`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `nopublisher.go` | Two pure adapter functions and one sentinel error (ErrMessagesProduced). No state. | ErrMessagesProduced is a hard error — if a wrapped HandlerFunc ever returns messages, the consumer will error and retry indefinitely. Only wrap handlers that are guaranteed to return nil messages. |

## Anti-Patterns

- Using HandlerFuncToNoPublisherHandler on a handler that conditionally produces messages — ErrMessagesProduced will cause infinite retries.
- Reimplementing these adapters inline in worker code instead of importing this package.

<!-- archie:ai-end -->
