# statemachine

<!-- archie:ai-start -->

> Generic, type-parameterized charge lifecycle state machine built on qmuntal/stateless. Each per-charge-type service (flatfee/usagebased/creditpurchase) instantiates Machine[CHARGE,BASE,STATUS] to drive status transitions and accumulate invoiceupdater.Patch side effects; this package holds no domain rules itself.

## Patterns

**Generic ChargeLike contract** — CHARGE must satisfy ChargeLike[CHARGE,BASE,STATUS] (GetChargeID/GetStatus/WithStatus/GetBase/WithBase) and STATUS must be a Status (~string with Validate()); the machine mutates the charge through these accessors only. (`type Machine[CHARGE ChargeLike[CHARGE, BASE, STATUS], BASE any, STATUS Status] struct {...}`)
**External-storage stateless machine** — New() builds stateless.NewStateMachineWithExternalStorage reading status from Charge.GetStatus() and writing back via Charge.WithStatus after newStatus.Validate(); uses FiringImmediate. (`stateless.NewStateMachineWithExternalStorage(accessor, mutator, stateless.FiringImmediate)`)
**FireAndActivate guards + persists** — FireAndActivate checks CanFire (returns ErrUnsupportedOperation if not), fires + activates the trigger, then persists via config.Persistence.UpdateBase and folds the result back with WithBase. (`if !canFire { return fmt.Errorf("%w: %s [status=%s,id=%s]", ErrUnsupportedOperation, trigger, ...) }`)
**Drive to stable state with TriggerNext** — AdvanceUntilStateStable loops firing meta.TriggerNext until CanFire is false, returning nil when no transition happened or a pointer to the advanced charge otherwise. (`for { canFire, _ := m.CanFire(ctx, meta.TriggerNext); if !canFire { ... } m.FireAndActivate(ctx, meta.TriggerNext) }`)
**Patch accumulation buffer** — Transitions append invoiceupdater.Patch via AddInvoicePatch; callers read InvoicePatches() (clone) or DrainInvoicePatches() (consume+clear). (`func (m *Machine...) DrainInvoicePatches() []invoiceupdater.Patch { patches := m.invoicePatches; m.invoicePatches = nil; return patches }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `machine.go` | Status/ChargeLike/Persistence/Config/StateMachine interfaces, Machine generic struct, New, Configure, CanFire, FireAndActivate, AdvanceUntilStateStable, RefetchCharge, patch buffer. | Config.Validate requires Persistence.UpdateBase and Refetch; status mutator re-validates every new status; ErrUnsupportedOperation is a models.NewGenericPreConditionFailedError wrapping CanFire=false. |
| `machine_test.go` | Unit tests over the generic machine using a fake meta-typed charge. | Tests don't touch Postgres; they exercise transition/guard logic in isolation. |

## Anti-Patterns

- Mutating charge status/base directly instead of through WithStatus/WithBase accessors.
- Firing triggers without CanFire guarding (bypassing ErrUnsupportedOperation).
- Embedding charge-type-specific business rules here instead of in the per-type service's state configuration.
- Reading invoicePatches without DrainInvoicePatches when the caller intends to consume them.

## Decisions

- **One generic Machine parameterized over CHARGE/BASE/STATUS instead of three hand-written machines.** — Flat-fee, usage-based, and credit-purchase share lifecycle plumbing (fire, activate, persist, accumulate patches); only the state graph differs, configured per type.
- **External-storage stateless with Persistence callbacks rather than in-memory state.** — Charge status lives in the DB; the machine reads/writes through caller-supplied UpdateBase/Refetch so persistence stays in the owning adapter.

<!-- archie:ai-end -->
