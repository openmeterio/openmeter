# statemachine

<!-- archie:ai-start -->

> Generic charge state machine library backed by qmuntal/stateless: Machine[CHARGE, BASE, STATUS] wraps a stateless.StateMachine with external storage bound to in-memory CHARGE, persisting via injected Persistence.UpdateBase after each successful transition. Shared by flatfee, usagebased, and creditpurchase charge types.

## Patterns

**ChargeLike constraint for type parameter** — CHARGE must implement ChargeLike[CHARGE, BASE, STATUS]: GetChargeID, GetStatus, WithStatus(STATUS) CHARGE (pure copy), GetBase, WithBase(BASE) CHARGE (pure copy). WithStatus/WithBase must be value receivers returning new copies — pointer mutation breaks external storage. (`func (c MyCharge) WithStatus(s MyStatus) MyCharge { c.Status = s; return c }
func (c MyCharge) WithBase(b MyBase) MyCharge { c.Base = b; return c }`)
**FireAndActivate for externally triggered transitions** — FireAndActivate checks CanFire, fires the trigger, calls Activate (OnActive callbacks), then persists BASE. Callers must not call m.stateMachine.FireCtx directly — that skips CanFire check, Activate, and persistence. (`if err := machine.FireAndActivate(ctx, meta.TriggerCollectionCompleted); err != nil { return err }`)
**AdvanceUntilStateStable for auto-advancement** — AdvanceUntilStateStable loops FireAndActivate(TriggerNext) until no TriggerNext is available; returns nil when no transitions were fired (not an error). Callers must handle the nil return. (`charge, err := machine.AdvanceUntilStateStable(ctx)
if err != nil { return nil, err }
if charge != nil { /* process advanced charge */ }`)
**Persistence stores BASE, Refetch reloads full CHARGE** — UpdateBase persists only the BASE fields (status-bearing struct) after a transition; Refetch is called when full CHARGE (including edges) must be reloaded after the transaction commits. (`Persistence[MyCharge, MyBase]{
    UpdateBase: func(ctx context.Context, base MyBase) (MyBase, error) { return adapter.UpdateCharge(ctx, base) },
    Refetch:    func(ctx context.Context, id meta.ChargeID) (MyCharge, error) { return adapter.GetByID(ctx, id) },
}`)
**STATUS Validate() called on every state setter** — The external state setter validates the incoming status via STATUS.Validate() on every transition. New status types must implement exhaustive Validate() — invalid statuses returned from transitions are caught at runtime. (`func (s MyStatus) Validate() error {
    if !slices.Contains(s.Values(), string(s)) { return models.NewGenericValidationError(...) }
    return nil
}`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `machine.go` | Machine[CHARGE,BASE,STATUS], Config, Persistence, ChargeLike constraint, StateMachine interface, New() constructor, FireAndActivate, AdvanceUntilStateStable, RefetchCharge, InvoicePatches/DrainInvoicePatches. | FireAndActivate calls Activate after Fire to trigger OnActive callbacks before persisting; if Activate errors, persistence is skipped — the in-memory charge may have advanced status but no DB write occurred. |
| `machine_test.go` | Reference implementations of ChargeLike (fakeCharge/fakeBase/fakeStatus) and test patterns for FireAndActivate, AdvanceUntilStateStable, and RefetchCharge. | Tests use t.Context() not context.Background(); use fakeCharge as the canonical template when implementing ChargeLike for a new charge type. |

## Anti-Patterns

- Implementing WithStatus or WithBase as pointer receivers that mutate in place — they must return new value copies because Machine.Charge is updated by assignment
- Calling m.stateMachine.FireCtx directly instead of FireAndActivate — skips the CanFire guard, Activate call, and persistence step
- Omitting OnActive callbacks when a state requires side effects on entry — activation runs between Fire and Persist
- Returning an invalid STATUS value from a transition action — the external state setter Validate() catches it at runtime, causing a confusing transition error
- Ignoring the nil return from AdvanceUntilStateStable — nil means no transitions were fired, not an error

## Decisions

- **Generic Machine[CHARGE, BASE, STATUS] instead of per-type state machines** — Three charge types share identical transition mechanics (fire, activate, persist-base, refetch); generics eliminate duplication while keeping type safety without reflection.
- **External storage pattern (stateless.NewStateMachineWithExternalStorage)** — Charge status is stored in the domain struct, not inside the stateless library; this keeps the machine stateless and allows refetch without re-creating the machine.
- **FireAndActivate persists BASE (not full CHARGE)** — Base fields (status, schedule fields) are what must be durable after a transition; realizations and edges are loaded on demand via Refetch, keeping persistence writes minimal.

## Example: Configure and drive a charge state machine for a new charge type

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
