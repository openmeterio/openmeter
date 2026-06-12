# nopublisher

<!-- archie:ai-start -->

> Tiny adapter pair between Watermill's HandlerFunc (returns messages) and NoPublishHandlerFunc (returns none), used when a consumer must not emit messages.

## Patterns

**Bidirectional handler adapters** — NoPublisherHandlerToHandlerFunc wraps a no-publish handler as a HandlerFunc (returns nil messages); HandlerFuncToNoPublisherHandler does the reverse and treats any produced message as the error ErrMessagesProduced. (`if len(outMessages) > 0 { return ErrMessagesProduced }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `nopublisher.go` | ErrMessagesProduced sentinel plus the two conversion functions. | Wrapping a HandlerFunc that legitimately returns messages will fail at runtime with ErrMessagesProduced rather than dropping them — only use on handlers expected to be side-effect-only. |

## Anti-Patterns

- Using HandlerFuncToNoPublisherHandler on a handler that intentionally produces output messages.

<!-- archie:ai-end -->
