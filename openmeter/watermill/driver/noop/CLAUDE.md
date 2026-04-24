# noop

<!-- archie:ai-start -->

> Zero-value no-op implementation of message.Publisher for use in disabled-feature Wire paths (e.g. credits.enabled=false). Satisfies the compiler with Publish and Close returning nil; no messages are ever sent.

## Patterns

**Compile-time interface assertion** — var _ message.Publisher = (*Publisher)(nil) at package level ensures the struct always satisfies the interface, catching drift at compile time. (`var _ message.Publisher = (*Publisher)(nil)`)
**Zero-value struct usage** — noop.Publisher has no fields; instantiate with &noop.Publisher{} or noop.Publisher{}. No constructor needed. (`return &noop.Publisher{}`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `publisher.go` | Entire package. Defines Publisher struct, compile-time assertion, and no-op Publish/Close methods. | Both Publish and Close must return nil — non-nil errors are treated as real failures by callers. |

## Anti-Patterns

- Adding logging or metrics to Publisher — defeats the purpose and adds side-effects in disabled paths
- Returning non-nil errors from Publish or Close — callers treat errors as real failures
- Adding constructor functions that accept dependencies — zero-value struct is the intended usage
- Growing this package beyond the single Publisher type — other noop concerns belong in their own packages

## Decisions

- **Zero-value struct with no constructor** — Maximally simple; Wire providers return &noop.Publisher{} inline without any setup, making disabled paths trivially safe.

<!-- archie:ai-end -->
