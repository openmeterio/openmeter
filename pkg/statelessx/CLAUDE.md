# statelessx

<!-- archie:ai-start -->

> Adapter helpers that bridge typed, validated functions into the untyped func(context.Context, ...any) shape expected by a stateless state-machine/condition framework. Used by billing charge services (creditpurchase/flatfee/usagebased) and billing/service.

## Patterns

**Typed-to-untyped action adapters** — EntryFunc wraps a no-arg ActionFn; WithParameters[T models.Validator] type-asserts args[0] to T, calls T.Validate(), then invokes the typed function — all surfaced as func(ctx, ...any) error. (`h := statelessx.WithParameters(func(ctx context.Context, in CreateInput) error { ... })`)
**Error-joining composition** — AllOf(fns...) runs every ActionFn regardless of individual failures and joins errors with errors.Join — no short-circuit. (`action := statelessx.AllOf(stepA, stepB, stepC)`)
**Condition adapters** — BoolFn lifts func()bool to the framework's func(ctx, ...any) bool; Not negates a func()bool predicate. (`guard := statelessx.BoolFn(statelessx.Not(isClosed))`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `actions.go` | ActionFn type, EntryFunc, AllOf, WithParameters[T models.Validator]. | WithParameters validates input via T.Validate() before running — T MUST implement models.Validator; it errors (not panics) on missing/wrong-typed args[0]. |
| `conditions.go` | BoolFn and Not condition adapters. | Both ignore context/args entirely — only use for state-independent predicates. |

## Anti-Patterns

- Passing a type to WithParameters that doesn't implement models.Validator — won't compile / loses validation.
- Assuming AllOf stops on first error — it always runs all functions and joins errors.
- Using BoolFn/Not for predicates that actually need ctx or args.

## Decisions

- **Generic WithParameters enforces Validate() at the framework boundary.** — Guarantees every typed action validates its input (via models.Validator) before the stateless transition body runs.

## Example: Register a validated typed action in a stateless transition

```
import (
    "github.com/openmeterio/openmeter/pkg/statelessx"
)

entry := statelessx.WithParameters(func(ctx context.Context, in CreateChargeInput) error {
    return svc.create(ctx, in) // in.Validate() already ran
})
```

<!-- archie:ai-end -->
