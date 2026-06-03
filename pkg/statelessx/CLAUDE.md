# statelessx

<!-- archie:ai-start -->

> Adapter utilities for the qmuntal/stateless state machine library — type-safe action and condition wrappers (WithParameters, AllOf, BoolFn, Not) that integrate with models.Validator so transitions validate inputs before executing.

## Patterns

**WithParameters wraps typed handlers with validation** — WithParameters[T models.Validator] adapts func(ctx, T) error into the stateless ...any signature. It asserts args[0] to T and calls T.Validate() before the inner function — panics if args is empty or the assertion fails. (`machine.Configure(stateGathering).OnEntry(statelessx.WithParameters[AdvanceInput](func(ctx context.Context, in AdvanceInput) error { return svc.advance(ctx, in) }))`)
**AllOf runs all actions regardless of failures** — AllOf(fn1, fn2, ...) runs every ActionFn even if earlier ones fail, then returns errors.Join of all failures. Use when multiple side effects (save + publish) must all be attempted. (`action := statelessx.AllOf(saveDB, publishEvent)`)
**BoolFn and Not adapt plain bool funcs** — BoolFn(fn func() bool) produces func(context.Context, ...any) bool for stateless guards; Not inverts a plain bool func. BoolFn discards ctx and args — not for guards needing per-trigger params. (`machine.Permit(trigger, state, statelessx.BoolFn(statelessx.Not(isDisabled)))`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `actions.go` | ActionFn type alias, AllOf combinator, WithParameters generic adapter for stateless entry/exit actions. | WithParameters expects args[0] to implement models.Validator; a non-Validator type or empty args panics. Do not call Validate() manually before passing — WithParameters already calls it. |
| `conditions.go` | BoolFn and Not helpers for guards that only need a bool, discarding ctx and trigger args. | BoolFn discards ctx and args — do not use for guards that need per-trigger parameter access. |

## Anti-Patterns

- Calling models.Validator.Validate() manually before WithParameters — it already calls Validate() internally.
- Using stateless ...any signatures directly instead of WithParameters when the trigger carries a typed argument — loses compile-time safety.
- Using BoolFn for guards that need access to the trigger's arguments — BoolFn ignores all args.

## Decisions

- **WithParameters constrains T to models.Validator rather than any.** — Forces callers to define Validate() on input types, catching structural errors at the state machine boundary before transitions commit side effects.

## Example: Register a transition with a validated typed parameter

```
import (
  "context"
  "github.com/openmeterio/openmeter/pkg/models"
  "github.com/openmeterio/openmeter/pkg/statelessx"
)

type AdvanceInput struct{ AsOf time.Time }
func (i AdvanceInput) Validate() error { if i.AsOf.IsZero() { return errors.New("AsOf required") }; return nil }

machine.Configure(stateGathering).Permit(triggerAdvance, stateIssued).OnEntry(statelessx.WithParameters[AdvanceInput](func(ctx context.Context, in AdvanceInput) error { return svc.advance(ctx, in) }))
```

<!-- archie:ai-end -->
