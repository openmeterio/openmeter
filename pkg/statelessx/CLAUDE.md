# statelessx

<!-- archie:ai-start -->

> Adapter utilities for the qmuntal/stateless state machine library — provides type-safe action and condition wrappers (WithParameters, AllOf, BoolFn, Not) that integrate with models.Validator so state machine transitions validate their inputs before executing.

## Patterns

**WithParameters wraps typed handlers with automatic validation** — WithParameters[T models.Validator] adapts a func(ctx, T) error into the stateless ...any signature. It asserts args[0] to T and calls T.Validate() before invoking the inner function — panics if args is empty or the type assertion fails. (`machine.Configure(stateGathering).OnEntry(statelessx.WithParameters[AdvanceInput](func(ctx context.Context, in AdvanceInput) error { return svc.advance(ctx, in) }))`)
**AllOf runs all actions regardless of individual failures** — AllOf(fn1, fn2, ...) runs every ActionFn even if earlier ones fail, then returns errors.Join of all failures. Use when multiple side effects (save + publish) must all be attempted. (`action := statelessx.AllOf(saveDB, publishEvent)`)
**BoolFn and Not adapt plain bool functions to stateless guard signatures** — BoolFn(fn func() bool) produces func(context.Context, ...any) bool accepted by stateless guards; Not inverts a plain bool func. BoolFn discards ctx and args — not suitable for guards needing per-trigger parameters. (`machine.Permit(trigger, state, statelessx.BoolFn(statelessx.Not(isDisabled)))`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `actions.go` | Defines ActionFn type alias, AllOf combinator, and WithParameters generic adapter for stateless entry/exit actions. | WithParameters expects args[0] to implement models.Validator; passing a non-Validator type or empty args panics at runtime with a type assertion failure. Callers must not call Validate() manually before passing — WithParameters already calls it. |
| `conditions.go` | BoolFn and Not helpers for stateless guard functions that only need a bool, discarding ctx and trigger args. | BoolFn discards both ctx and args — do not use for guards that need per-trigger parameter access. |

## Anti-Patterns

- Calling models.Validator.Validate() manually before passing to WithParameters — it already calls Validate() internally
- Using stateless ...any signatures directly instead of WithParameters when the trigger carries a typed argument — loses compile-time safety
- Using BoolFn for guards that need access to the trigger's arguments — BoolFn ignores all args

## Decisions

- **WithParameters constrains T to models.Validator rather than any** — Forces callers to define Validate() on input types, catching structural errors at the state machine boundary before transitions commit any side effects.

## Example: Register a state machine transition with a validated typed parameter

```
import (
    "context"
    "github.com/openmeterio/openmeter/pkg/models"
    "github.com/openmeterio/openmeter/pkg/statelessx"
)

type AdvanceInput struct{ AsOf time.Time }
func (i AdvanceInput) Validate() error {
    if i.AsOf.IsZero() { return errors.New("AsOf required") }
    return nil
}

machine.Configure(stateGathering).
    Permit(triggerAdvance, stateIssued).
    OnEntry(statelessx.WithParameters[AdvanceInput](func(ctx context.Context, in AdvanceInput) error {
// ...
```

<!-- archie:ai-end -->
