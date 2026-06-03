# nopublisher

<!-- archie:ai-start -->

> Provides two pure adapter functions converting between Watermill's NoPublishHandlerFunc and HandlerFunc signatures, used when a handler is logically a consumer but must fit a router slot expecting a different signature.

## Patterns

**NoPublisherHandlerToHandlerFunc for consumer-only handlers** — Wraps a NoPublishHandlerFunc into a HandlerFunc that always returns (nil, err); use when the router requires HandlerFunc but the handler produces no messages. (`router.AddHandler("name", topic, sub, pub, nopublisher.NoPublisherHandlerToHandlerFunc(myConsumer))`)
**HandlerFuncToNoPublisherHandler with production guard** — Converts a HandlerFunc into a NoPublishHandlerFunc; returns ErrMessagesProduced if the inner handler emits any messages. Only wrap truly side-effect-only handlers. (`nopublisher.HandlerFuncToNoPublisherHandler(h) // errors if h returns messages`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `nopublisher.go` | Two pure adapter functions and one sentinel error (ErrMessagesProduced); no state. | ErrMessagesProduced is a hard error — a wrapped HandlerFunc returning messages causes infinite retries. Only wrap handlers guaranteed to return nil messages. |

## Anti-Patterns

- Using HandlerFuncToNoPublisherHandler on a handler that conditionally produces messages — ErrMessagesProduced causes infinite retries.
- Reimplementing these adapters inline in worker code instead of importing this package.

<!-- archie:ai-end -->
