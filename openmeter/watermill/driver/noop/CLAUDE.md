# noop

<!-- archie:ai-start -->

> A no-op Watermill message.Publisher implementation that silently discards all published messages. Used by openmeter/watermill/eventbus as a null/disabled publisher when event publishing should be a no-op.

## Patterns

**Compile-time interface assertion** — Assert the type satisfies message.Publisher via a blank identifier var, so a signature drift breaks the build. (`var _ message.Publisher = (*Publisher)(nil)`)
**Value-receiver no-op methods returning nil** — Publish and Close are value-receiver methods on an empty struct that do nothing and return nil error. (`func (Publisher) Publish(topic string, messages ...*message.Message) error { return nil }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `publisher.go` | Defines the empty Publisher struct and its no-op Publish/Close methods implementing watermill message.Publisher. | Keep Publisher zero-state (struct{}); do not add fields or side effects. Both methods must return nil and never block — this is the disabled-publishing path. |

## Anti-Patterns

- Adding real publishing logic, buffering, or error returns — this driver exists specifically to discard messages.
- Removing the `var _ message.Publisher = (*Publisher)(nil)` assertion, which guards against Watermill interface changes.
- Giving Publisher state/fields, breaking its trivial value-receiver semantics.

## Decisions

- **Empty struct with value receivers instead of a pointer-based stateful type.** — A no-op publisher needs no state; value receivers make zero-value usage and copying safe and free.

## Example: Implementing the no-op Watermill publisher

```
package noop

import "github.com/ThreeDotsLabs/watermill/message"

type Publisher struct{}

var _ message.Publisher = (*Publisher)(nil)

func (Publisher) Publish(topic string, messages ...*message.Message) error {
	return nil
}

func (Publisher) Close() error {
	return nil
}
```

<!-- archie:ai-end -->
