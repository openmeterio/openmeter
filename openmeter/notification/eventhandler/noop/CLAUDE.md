# noop

<!-- archie:ai-start -->

> No-op implementation of notification.EventHandler used when the notification subsystem is disabled or unconfigured. All methods satisfy the interface contract without performing any real dispatch, reconciliation, or lifecycle work, enabling Wire to inject a safe zero-value implementation instead of a nil pointer.

## Patterns

**Compile-time interface assertion** — handler.go must declare `var _ notification.EventHandler = (*Handler)(nil)` to catch interface drift. Without it, a new method on notification.EventHandler silently leaves the noop broken. (`var _ notification.EventHandler = (*Handler)(nil)`)
**Zero-value struct with value receivers** — Handler has no fields. All methods are value receivers returning nil. No constructor parameters, no state, no goroutines. (`type Handler struct{}
func (Handler) Start() error { return nil }
func (Handler) Close() error { return nil }
func (Handler) Dispatch(_ context.Context, _ *notification.Event) error { return nil }`)
**Two-value constructor matching Wire expectation** — New() returns (*Handler, error) even though it never fails, so Wire provider function signatures remain consistent with the real eventhandler.New(). (`func New() (*Handler, error) { return &Handler{}, nil }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `handler.go` | Sole file: defines Handler struct, compile-time assertion, and all interface method implementations. | Never add fields, logging, metrics, or side-effects. Returning non-nil errors from any method breaks app/common assumptions that noop always succeeds. |

## Anti-Patterns

- Adding logging, metrics, or side-effects to any method — this package must remain intentionally inert.
- Removing the compile-time interface assertion — it is the only guard against interface drift.
- Using context.Background() or context.TODO() instead of blank-identifying the received context parameter.
- Storing any state on Handler — it must remain a zero-value struct with no fields.
- Returning non-nil errors from any method — callers in app/common assume noop always succeeds.

## Decisions

- **Separate noop sub-package rather than a nil check inside the real handler** — Consistent with the project-wide noop pattern (ledger/noop, notification/webhook noop) — callers receive a real interface and never need nil guards scattered through business logic.
- **Value receivers for all methods** — Handler has no pointer-receiver state (no channels, no atomics), so value receivers are idiomatic and prevent accidental mutation.

<!-- archie:ai-end -->
