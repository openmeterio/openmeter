# service

<!-- archie:ai-start -->

> Business-logic layer implementing planaddon.Service — validates inputs, enforces plan/addon state constraints (status, deleted checks), delegates persistence to planaddon.Repository, and publishes domain events after every mutation. No Ent or SQL imports allowed.

## Patterns

**transaction.Run / RunWithNoValue for writes** — Mutating methods (CreatePlanAddon, DeletePlanAddon, UpdatePlanAddon) wrap their fn in transaction.Run(ctx, s.adapter, fn) for atomicity across adapter calls and event publishing. (`return transaction.Run(ctx, s.adapter, fn)`)
**Validate params first, then cross-entity checks** — Each method calls params.Validate() on entry, then fetches related entities (plan, addon) to enforce status/deleted rules before calling the adapter. (`if err := params.Validate(); err != nil { return nil, fmt.Errorf("invalid ...: %w", err) }`)
**Domain errors wrapped as models.Generic*** — plan/addon/planaddon NotFoundError caught via errors.As and re-wrapped as models.NewGenericNotFoundError; validation -> NewGenericValidationError; duplicates -> NewGenericConflictError. (`if notFound := &(plan.NotFoundError{}); errors.As(err, &notFound) { return nil, models.NewGenericNotFoundError(err) }`)
**Publish domain event after every mutation** — After successful create/update/delete, publish a typed event (NewPlanAddonCreateEvent/UpdateEvent/DeleteEvent) via s.publisher inside the transaction closure — publish failure rolls back the DB change. (`event := planaddon.NewPlanAddonCreateEvent(ctx, planAddon); if err = s.publisher.Publish(ctx, event); err != nil { return nil, err }`)
**Config nil-checks in New()** — service.New(Config) checks Adapter, Plan, Addon, Logger, Publisher for nil and errors, enforcing complete wiring. var _ planaddon.Service = (*service)(nil) enforces the interface. (`if config.Adapter == nil { return nil, errors.New("add-on assignment adapter is required") }`)
**Patch-field merge mirrored with adapter** — UpdatePlanAddon builds a merged draft PlanAddon for Validate() before calling the adapter. Merge rules (MaxQuantity always via SetOrClearMaxQuantity; FromPlanPhase/Metadata/Annotations only when non-nil) must match the adapter exactly. (`// MaxQuantity is always replaced via SetOrClearMaxQuantity, so nil clears the column.`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `service.go` | Config, New constructor, service struct. No method logic. | var _ planaddon.Service = (*service)(nil) compile-check — add interface methods here before implementing in planaddon.go. |
| `planaddon.go` | All service method implementations. Reads call fn(ctx) directly; writes use transaction.Run / RunWithNoValue. | DeletePlanAddon re-fetches after deletion for the DeletedAt timestamp in the event payload — do not skip. Always publish inside the transaction closure. |
| `service_test.go` | Integration tests of service+adapter+DB via pctestutils.NewTestEnv, covering invalid-state transitions (published plan rejects new plan-addon). | Tests create meters/features/plans/addons as prerequisites — reuse pctestutils helpers, don't duplicate fixtures. |

## Anti-Patterns

- Calling s.adapter outside a transaction.Run closure for writes — risks partial writes if publisher.Publish fails.
- Returning adapter-level errors (planaddon.NotFoundError) without wrapping in models.Generic* — breaks the HTTP error encoder.
- Checking plan/addon status inside the adapter — status validation belongs in the service.
- Omitting event publishing after a successful mutation — downstream systems depend on these events.
- Using context.Background() instead of the passed ctx — breaks tracing and cancellation.

## Decisions

- **transaction.Run wraps all write operations.** — Create/Update/Delete each do get-then-mutate and publish an event; all must be atomic so a publish failure rolls back the DB change.
- **Cross-entity validation in the service, not the adapter.** — Adapters are pure persistence; rules about which states allow assignment must be in the service to stay testable without a DB.
- **Patch-field merge mirrored between service and adapter.** — UpdatePlanAddon builds a draft merged state for Validate() that must match the adapter's SetOrClear/conditional-set behaviour to avoid silent divergence.

## Example: Add a new write method with transaction, validation, event publishing

```
func (s service) SomeMutation(ctx context.Context, params planaddon.SomeMutationInput) (*planaddon.PlanAddon, error) {
  fn := func(ctx context.Context) (*planaddon.PlanAddon, error) {
    if err := params.Validate(); err != nil { return nil, fmt.Errorf("invalid params: %w", err) }
    pa, err := s.adapter.SomeMutation(ctx, params)
    if err != nil {
      if notFound := &(planaddon.NotFoundError{}); errors.As(err, &notFound) { return nil, models.NewGenericNotFoundError(err) }
      return nil, err
    }
    if err = s.publisher.Publish(ctx, planaddon.NewPlanAddonUpdateEvent(ctx, pa)); err != nil { return nil, err }
    return pa, nil
  }
  return transaction.Run(ctx, s.adapter, fn)
}
```

<!-- archie:ai-end -->
