# service

<!-- archie:ai-start -->

> Business-logic layer implementing planaddon.Service — validates inputs, enforces plan/addon state constraints (status, deleted checks), delegates persistence to planaddon.Repository, and publishes domain events after every mutation. No Ent or SQL imports allowed here.

## Patterns

**transaction.Run / RunWithNoValue for write operations** — Mutating methods (CreatePlanAddon, DeletePlanAddon, UpdatePlanAddon) wrap their fn closure in transaction.Run(ctx, s.adapter, fn) or transaction.RunWithNoValue to ensure atomicity across adapter calls and event publishing. (`return transaction.Run(ctx, s.adapter, fn)`)
**Validate params first, then cross-entity checks** — Every method calls params.Validate() immediately on entry, then fetches related entities (plan, addon) to enforce business rules (status, deleted state) before calling the adapter. (`if err := params.Validate(); err != nil { return nil, fmt.Errorf("invalid ...: %w", err) }`)
**Domain-specific errors wrapped as models.Generic* errors** — plan.NotFoundError, addon.NotFoundError, planaddon.NotFoundError are caught via errors.As and re-wrapped as models.NewGenericNotFoundError. Validation failures use models.NewGenericValidationError. Duplicates use models.NewGenericConflictError. (`if notFound := &(plan.NotFoundError{}); errors.As(err, &notFound) { return nil, models.NewGenericNotFoundError(err) }`)
**Publish domain event after every successful mutation** — After successful create/update/delete, publish a typed event (planaddon.NewPlanAddonCreateEvent, UpdateEvent, DeleteEvent) via s.publisher inside the transaction closure — failure rolls back the DB change. (`event := planaddon.NewPlanAddonCreateEvent(ctx, planAddon); if err = s.publisher.Publish(ctx, event); err != nil { return nil, err }`)
**Config struct with required-field nil-checks in New()** — service.New(Config) checks Adapter, Plan, Addon, Logger, Publisher for nil and returns an error, enforcing complete wiring at construction time. var _ planaddon.Service = (*service)(nil) enforces compile-time interface satisfaction. (`if config.Adapter == nil { return nil, errors.New("add-on assignment adapter is required") }`)
**Patch-field merge rules mirrored and documented between service and adapter** — UpdatePlanAddon builds a merged draft PlanAddon for Validate() before calling the adapter. Merge rules (MaxQuantity always replaced via SetOrClearMaxQuantity; FromPlanPhase/Metadata/Annotations only written when non-nil) must exactly match the adapter's behaviour. (`// MaxQuantity is always replaced via SetOrClearMaxQuantity, so nil clears the column — use params.MaxQuantity as-is.`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `service.go` | Defines Config, New constructor, and service struct. No method logic lives here. | var _ planaddon.Service = (*service)(nil) compile-check — add new interface methods here before implementing them in planaddon.go. |
| `planaddon.go` | All planaddon.Service method implementations. Read methods call fn(ctx) directly; write methods use transaction.Run / RunWithNoValue. | DeletePlanAddon re-fetches the record after deletion to get the DeletedAt timestamp for the event payload — do not skip this refetch. Always publish the event inside the transaction closure. |
| `service_test.go` | Integration tests exercising the full service+adapter+DB stack via pctestutils.NewTestEnv. Covers invalid-state transitions (e.g. published plan rejects new plan-addon). | Tests create meters, features, plans, and addons as prerequisites — reuse pctestutils helpers rather than duplicating fixture code. |

## Anti-Patterns

- Calling s.adapter methods outside a transaction.Run closure for write paths — risks partial writes if publisher.Publish fails.
- Returning adapter-level errors (planaddon.NotFoundError) directly without wrapping in models.Generic* — breaks the HTTP error encoder chain.
- Checking plan/addon status inside the adapter — status validation belongs exclusively in the service layer.
- Omitting event publishing after a successful mutation — downstream systems depend on these events for sync.
- Using context.Background() instead of the passed ctx — breaks tracing and cancellation.

## Decisions

- **transaction.Run wraps all write operations** — Create/Update/Delete each call the adapter multiple times (get-then-mutate pattern) and publish an event; all must be atomic so a publish failure rolls back the DB change.
- **Cross-entity validation (plan status, addon status) in the service not the adapter** — Adapters are pure persistence; business rules about which states allow assignment creation/deletion must be in the service to stay testable without a DB.
- **Patch-field merge logic mirrored and documented between service and adapter** — UpdatePlanAddon builds a draft merged state for Validate() before calling the adapter; the merge rules must exactly match the adapter's SetOrClear/conditional-set behaviour to avoid silent divergence.

## Example: Add a new write service method with transaction, validation, and event publishing

```
func (s service) SomeMutation(ctx context.Context, params planaddon.SomeMutationInput) (*planaddon.PlanAddon, error) {
	fn := func(ctx context.Context) (*planaddon.PlanAddon, error) {
		if err := params.Validate(); err != nil {
			return nil, fmt.Errorf("invalid params: %w", err)
		}
		pa, err := s.adapter.SomeMutation(ctx, params)
		if err != nil {
			if notFound := &(planaddon.NotFoundError{}); errors.As(err, &notFound) {
				return nil, models.NewGenericNotFoundError(err)
			}
			return nil, fmt.Errorf("failed to mutate: %w", err)
		}
		event := planaddon.NewPlanAddonUpdateEvent(ctx, pa)
		if err = s.publisher.Publish(ctx, event); err != nil {
			return nil, fmt.Errorf("failed to publish event: %w", err)
// ...
```

<!-- archie:ai-end -->
