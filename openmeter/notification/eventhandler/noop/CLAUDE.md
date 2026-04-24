# noop

<!-- archie:ai-start -->

> No-op implementation of notification.EventHandler used when the notification subsystem is disabled or unconfigured. All methods return nil, satisfying the interface contract without performing any real dispatch, reconciliation, or lifecycle work.

## Patterns

**Interface compliance assertion** — The file-level var _ notification.EventHandler = (*Handler)(nil) ensures compile-time verification that Handler fully implements the interface. (`var _ notification.EventHandler = (*Handler)(nil)`)
**No-op method bodies** — Every method accepts all parameters (using blank identifiers) and returns nil — never logs, never panics, never mutates state. (`func (h Handler) Dispatch(_ context.Context, _ *notification.Event) error { return nil }`)
**Constructor returns (*Handler, error)** — New() follows the standard two-value constructor signature matching the Wire provider expectation, even though no error can occur. (`func New() (*Handler, error) { return &Handler{}, nil }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `handler.go` | Single-file package that provides the noop EventHandler; wired by app/common when notification is disabled. | If notification.EventHandler gains new methods, this file will fail to compile — add matching no-op stubs immediately. |

## Anti-Patterns

- Adding logging, metrics, or side-effects to any method — this is intentionally inert.
- Removing the compile-time interface assertion (var _ ...) — it is the only guard against interface drift.
- Using context.Background() instead of the received context parameter — blank-identify it instead.
- Storing any state on Handler — it must remain a zero-value struct.

## Decisions

- **Value receiver on Handler, not pointer receiver for most methods** — Handler holds no state; value receivers avoid unnecessary heap allocation and communicate that the struct is safe to copy.
- **Separate noop sub-package rather than a nil check inside the real handler** — Matches the broader noop pattern used across openmeter (ledgernoop, webhooknoop) — Wire wires the concrete noop type so call sites need no nil guards at runtime.

## Example: Adding a new method required by notification.EventHandler after an interface change

```
// handler.go — add the method with blank-identifier params and nil return
func (h Handler) NewMethod(_ context.Context, _ *notification.SomeInput) error {
	return nil
}
```

<!-- archie:ai-end -->
