# noop

<!-- archie:ai-start -->

> No-op implementation of notification.EventHandler used when notification event dispatch/reconciliation is disabled or unwired. Satisfies the full interface contract so DI can always supply a handler without running Kafka/Svix delivery side effects.

## Patterns

**Compile-time interface assertion** — Assert the no-op type satisfies notification.EventHandler at compile time so interface drift breaks the build, not runtime. (`var _ notification.EventHandler = (*Handler)(nil)`)
**All methods return nil / no side effects** — Every EventHandler method (Dispatch, Reconcile, Start, Close) must be a pure no-op returning nil. No state, no goroutines, no I/O. (`func (h Handler) Dispatch(_ context.Context, _ *notification.Event) error { return nil }`)
**Value-receiver empty struct** — Handler is an empty struct{} with value receivers; it holds no dependencies (no logger, no repo, no webhook.Handler) unlike the real eventhandler/service implementation. (`type Handler struct{}`)
**Fallible constructor signature parity** — New() returns (*Handler, error) to mirror the real handler constructor so callers/DI can swap implementations without changing call sites, even though error is always nil. (`func New() (*Handler, error) { return &Handler{}, nil }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `handler.go` | Sole file; defines Handler{} and its four EventHandler methods plus New(). The complete no-op surface. | Keep method set in lockstep with notification.EventHandler in openmeter/notification/eventhandler.go (Dispatch, Reconcile via embedded EventDispatcher/EventReconciler, Start, Close). Adding a method there silently breaks the var _ assertion here until updated. |

## Anti-Patterns

- Adding fields, loggers, or goroutines to Handler — the whole point is zero side effects.
- Returning a non-nil error from any method or from New(); callers assume the no-op never fails.
- Importing webhook/Svix, Kafka, or repository packages here — that recreates the real handler and defeats the no-op purpose.
- Letting the method set drift from notification.EventHandler; the contract is defined upstream, not here.

## Decisions

- **Provide a no-op EventHandler in its own package rather than nil.** — DI/wiring (app/common, cmd/notification-service) can always inject a valid handler; disabling notification delivery becomes a constructor swap, avoiding nil-checks at every call site.

## Example: Full no-op handler satisfying notification.EventHandler

```
package noop

import (
	"context"

	"github.com/openmeterio/openmeter/openmeter/notification"
)

var _ notification.EventHandler = (*Handler)(nil)

type Handler struct{}

func (h Handler) Dispatch(_ context.Context, _ *notification.Event) error { return nil }
func (h Handler) Reconcile(_ context.Context) error                       { return nil }
func (h Handler) Start() error                                           { return nil }
// ...
```

<!-- archie:ai-end -->
