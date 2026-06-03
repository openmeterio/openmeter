# noop

<!-- archie:ai-start -->

> Zero-value no-op implementation of message.Publisher for disabled-feature Wire paths (e.g. credits.enabled=false). Satisfies the compiler with Publish and Close returning nil; no messages are ever sent.

## Patterns

**Compile-time interface assertion** — A package-level var ensures the struct always satisfies the Watermill message.Publisher interface, catching drift at compile time. (`var _ message.Publisher = (*Publisher)(nil)`)
**Zero-value struct with no constructor** — noop.Publisher has no fields; instantiate directly with &noop.Publisher{}. No constructor is needed or should be added. (`return &noop.Publisher{}`)
**Both methods return nil** — Publish and Close must always return nil — callers treat non-nil errors as real failures even when the feature is disabled. (`func (Publisher) Publish(topic string, messages ...*message.Message) error { return nil }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `publisher.go` | Entire package: the Publisher struct, the compile-time interface assertion, and no-op Publish/Close methods. | Both Publish and Close must return nil. Do not add fields, dependencies, logging, or metrics — this defeats the purpose of a disabled-feature noop. |

## Anti-Patterns

- Adding logging or metrics to Publisher — adds side-effects in disabled paths.
- Returning non-nil errors from Publish or Close — callers treat errors as real failures.
- Adding constructor functions that accept dependencies — the zero-value struct is the intended usage.
- Growing this package beyond the single Publisher type — other noop concerns belong in their own packages.
- Using this publisher in non-disabled-feature paths — only use it in Wire providers gated by a feature flag.

## Decisions

- **Zero-value struct with no constructor.** — Maximally simple; Wire providers return &noop.Publisher{} inline with no setup, making disabled-feature paths trivially safe and dependency-free.
- **Package scoped to a single type.** — One type (Publisher) keeps the package trivially auditable and prevents it from accumulating hidden side-effects.

## Example: Wire provider returning a noop publisher when a feature is disabled

```
import "github.com/openmeterio/openmeter/openmeter/watermill/driver/noop"

func NewEventPublisher(cfg config.CreditsConfiguration) message.Publisher {
	if !cfg.Enabled {
		return &noop.Publisher{}
	}
	return buildRealKafkaPublisher(cfg)
}
```

<!-- archie:ai-end -->
