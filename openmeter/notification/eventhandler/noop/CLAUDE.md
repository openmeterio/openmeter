# noop

<!-- archie:ai-start -->

> No-op implementation of notification.EventHandler used when the notification subsystem is disabled or unconfigured. All methods satisfy the interface contract without performing real dispatch, reconciliation, or lifecycle work, so Wire can inject a safe zero-value implementation instead of a nil pointer.

## Patterns

**Compile-time interface assertion** — handler.go must declare var _ notification.EventHandler = (*Handler)(nil) to catch interface drift; without it a new EventHandler method silently leaves the noop incomplete. (`var _ notification.EventHandler = (*Handler)(nil)`)
**Zero-value struct with value receivers** — Handler has no fields; all methods are value receivers returning nil, with no constructor params, state, or goroutines. (`type Handler struct{}
func (h Handler) Dispatch(_ context.Context, _ *notification.Event) error { return nil }
func (h Handler) Reconcile(_ context.Context) error { return nil }`)
**Two-value constructor matching Wire expectation** — New() returns (*Handler, error) even though it never fails, keeping the Wire provider signature consistent with the real eventhandler.New(). (`func New() (*Handler, error) { return &Handler{}, nil }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `handler.go` | Sole file: defines the Handler struct, the compile-time assertion, and all interface method implementations (Dispatch, Reconcile, Start, Close). | Never add fields, logging, metrics, or side-effects. Returning a non-nil error from any method breaks app/common's assumption that noop always succeeds. |

## Anti-Patterns

- Adding logging, metrics, or side-effects to any method — this package must remain intentionally inert.
- Removing the compile-time interface assertion — it is the only guard against interface drift.
- Using context.Background()/context.TODO() instead of blank-identifying the received context parameter.
- Storing any state on Handler — it must remain a zero-value struct with no fields.
- Returning a non-nil error from any method — callers in app/common assume noop always succeeds.

## Decisions

- **Separate noop sub-package rather than a nil check inside the real handler** — Consistent with the project-wide noop pattern (ledger/noop, notification/webhook noop) — callers receive a real interface and never need nil guards scattered through business logic.
- **Value receivers for all methods** — Handler has no pointer-receiver state (no channels, no atomics), so value receivers are idiomatic and prevent accidental mutation.

## Example: The complete noop EventHandler

```
package noop

import (
	"context"

	"github.com/openmeterio/openmeter/openmeter/notification"
)

var _ notification.EventHandler = (*Handler)(nil)

type Handler struct{}

func (h Handler) Dispatch(_ context.Context, _ *notification.Event) error { return nil }
func (h Handler) Reconcile(_ context.Context) error { return nil }
func (h Handler) Start() error { return nil }
// ...
```

<!-- archie:ai-end -->
