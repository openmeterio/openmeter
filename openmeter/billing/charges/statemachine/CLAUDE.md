# statemachine

<!-- archie:ai-start -->

> Generic charge state machine backed by the stateless library. Machine[CHARGE, BASE, STATUS] wraps a stateless.StateMachine with external storage bound to in-memory CHARGE, persisting via injected Persistence.UpdateBase after each successful transition. FireAndActivate fires a trigger and persists; AdvanceUntilStateStable walks TriggerNext transitions until no more are available. Used by all charge types (flatfee, usagebased, creditpurchase).

## Patterns

**ChargeLike constraint for type parameter** — CHARGE must implement ChargeLike[CHARGE, BASE, STATUS]: GetChargeID, GetStatus, WithStatus (pure copy), GetBase, WithBase (pure copy). STATUS must implement Status (~string + Validate()). (`func (c MyCharge) WithStatus(s MyStatus) MyCharge { c.Status = s; return c }
func (c MyCharge) WithBase(b MyBase) MyCharge { c.Base = b; return c }`)
**FireAndActivate before AdvanceUntilStateStable** — Use FireAndActivate for externally triggered transitions (e.g. payment authorized). Use AdvanceUntilStateStable to auto-advance through all TriggerNext transitions after an external event. (`if err := machine.FireAndActivate(ctx, meta.TriggerCollectionCompleted); err != nil { return err }
charge, err := machine.AdvanceUntilStateStable(ctx)`)
**Persistence.UpdateBase stores BASE, not full CHARGE** — UpdateBase persists only the BASE fields (status-bearing struct); Refetch loads the full CHARGE when edges must be reloaded after a transition. (`Persistence[MyCharge, MyBase]{
    UpdateBase: func(ctx context.Context, base MyBase) (MyBase, error) { return adapter.UpdateCharge(ctx, base) },
    Refetch:    func(ctx context.Context, id meta.ChargeID) (MyCharge, error) { return adapter.GetByID(ctx, id) },
}`)
**AdvanceUntilStateStable returns nil when already stable** — When the machine has no TriggerNext transition from the current state, AdvanceUntilStateStable returns (nil, nil). Callers must handle the nil charge pointer. (`charge, err := machine.AdvanceUntilStateStable(ctx)
if err != nil { return err }
if charge != nil { // process advanced charge }`)
**STATUS Validate() called on every state setter** — The external state setter validates the incoming status via STATUS.Validate(). New status types must have exhaustive Validate() implementations — invalid statuses returned from transitions will be caught here. (`func (s MyStatus) Validate() error {
    if !slices.Contains(s.Values(), string(s)) { return models.NewGenericValidationError(...) }
    return nil
}`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `machine.go` | Machine[CHARGE,BASE,STATUS], Config, Persistence, ChargeLike constraint, StateMachine interface, New() constructor. | FireAndActivate calls Activate after Fire to trigger OnActive callbacks before persisting; activation errors skip persistence. |
| `machine_test.go` | Reference implementations of ChargeLike (fakeCharge/fakeBase/fakeStatus) and test patterns for FireAndActivate, AdvanceUntilStateStable, and RefetchCharge. | Tests use t.Context() not context.Background(); use these as templates for new charge type state machine tests. |

## Anti-Patterns

- Implementing WithStatus or WithBase as pointer receivers that mutate in place — they must return a new value copy because Machine.Charge is updated by assignment.
- Calling m.stateMachine.FireCtx directly instead of FireAndActivate — skips the CanFire guard, Activate call, and persistence step.
- Omitting OnActive callbacks when the state requires side effects on entry — activation runs between Fire and Persist.
- Returning an invalid STATUS value from a transition target — the external state setter's Validate() will catch it but only at runtime.
- Ignoring the nil return from AdvanceUntilStateStable — nil means no transitions were fired, not an error.

## Decisions

- **Generic Machine[CHARGE, BASE, STATUS] instead of per-type state machines** — Three charge types share identical transition mechanics (fire, activate, persist-base, refetch); generics eliminate duplication while keeping type safety.
- **External storage pattern (stateless.NewStateMachineWithExternalStorage)** — Charge status is stored in the domain struct, not inside the stateless library; this keeps the machine stateless and allows refetch without re-creating the machine.
- **FireAndActivate persists BASE (not full CHARGE)** — Base fields (status, schedule fields) are what must be durable after a transition; realizations and edges are loaded on demand via Refetch.

## Example: Configure and drive a charge state machine

```
import (
    "github.com/openmeterio/openmeter/openmeter/billing/charges/statemachine"
    "github.com/openmeterio/openmeter/openmeter/billing/charges/meta"
)

machine, err := statemachine.New(statemachine.Config[MyCharge, MyBase, MyStatus]{
    Charge: charge,
    Persistence: statemachine.Persistence[MyCharge, MyBase]{
        UpdateBase: func(ctx context.Context, base MyBase) (MyBase, error) {
            return adapter.UpdateCharge(ctx, base)
        },
        Refetch: func(ctx context.Context, id meta.ChargeID) (MyCharge, error) {
            return adapter.GetByID(ctx, id)
        },
    },
// ...
```

<!-- archie:ai-end -->
