# statelessx

<!-- archie:ai-start -->

> Adapter utilities for the stateless state machine library — provides type-safe action and condition wrappers that integrate with models.Validator so state machine transitions can validate their parameters before executing.

## Patterns

**WithParameters wraps typed handlers** — Use WithParameters[T models.Validator] to adapt a func(ctx, T) error into the stateless ...any signature; it validates the argument with T.Validate() before calling the inner function. (`stateless.Machine.Fire(trigger, statelessx.WithParameters[MyInput](func(ctx, in MyInput) error { ... }))`)
**AllOf joins action errors without short-circuit** — AllOf(fn1, fn2, ...) runs all ActionFn callbacks even if earlier ones fail, then returns errors.Join of all failures. Use when multiple side effects must all be attempted. (`action := statelessx.AllOf(saveDB, publishEvent)`)
**BoolFn and Not adapt plain booleans to stateless guard signatures** — BoolFn(fn func() bool) produces a func(context.Context, ...any) bool accepted by stateless guards; Not inverts a plain bool func. (`machine.Permit(trigger, state, statelessx.BoolFn(statelessx.Not(isDisabled)))`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `actions.go` | Defines ActionFn type alias, AllOf combinator, and WithParameters generic adapter | WithParameters expects args[0] to implement models.Validator; passing a non-Validator type panics at runtime with a type assertion failure |
| `conditions.go` | Provides BoolFn and Not helpers for stateless guard functions | BoolFn discards both ctx and args — not suitable for guards that need per-trigger parameter access |

## Anti-Patterns

- Calling models.Validator.Validate() manually before passing to WithParameters — WithParameters already calls it
- Using stateless ...any directly instead of WithParameters when the trigger carries a typed argument — loses compile-time safety

## Decisions

- **WithParameters constrains T to models.Validator rather than any** — Forces callers to define Validate() on input types, catching structural errors at the boundary before state transitions commit any side effects

## Example: Register a state machine transition with a validated typed parameter

```
import (
	"context"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/statelessx"
)

type AdvanceInput struct{ AsOf time.Time }
func (i AdvanceInput) Validate() error { if i.AsOf.IsZero() { return errors.New("AsOf required") }; return nil }

machine.Configure(stateGathering).
	Permit(triggerAdvance, stateIssued).
	OnEntry(statelessx.WithParameters[AdvanceInput](func(ctx context.Context, in AdvanceInput) error {
		return svc.advance(ctx, in)
	}))
```

<!-- archie:ai-end -->
