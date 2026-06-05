# service

<!-- archie:ai-start -->

> Business-logic layer for plan-addon assignments: validates plan/addon state and compatibility, orchestrates the adapter, and publishes lifecycle events. Implements planaddon.Service.

## Patterns

**transaction.Run around mutations** — Create/Update/Delete wrap their fn in transaction.Run / transaction.RunWithNoValue(ctx, s.adapter, fn); read-only Get/List just invoke fn(ctx). (`return transaction.Run(ctx, s.adapter, fn)`)
**Validate plan/addon state before write** — Plan must pass IsPlanDeleted(clock.Now()) + HasPlanStatus(Draft, Scheduled); addon must pass IsAddonDeleted + HasAddonStatus(Active). Failures become models.NewGenericValidationError. (`p.ValidateWith(plan.IsPlanDeleted(clock.Now()), plan.HasPlanStatus(productcatalog.PlanStatusDraft, productcatalog.PlanStatusScheduled))`)
**Conflict check on create** — CreatePlanAddon pre-fetches by plan+addon and returns models.NewGenericConflictError if an assignment already exists. (`if planAddon != nil && planAddon.Plan.ID == params.PlanID && planAddon.Addon.ID == params.AddonID { return nil, models.NewGenericConflictError(...) }`)
**Map typed errors to generic models errors** — errors.As against plan.NotFoundError / addon.NotFoundError / planaddon.NotFoundError, then return models.NewGenericNotFoundError so the HTTP layer renders a 404. (`if notFound := &(planaddon.NotFoundError{}); errors.As(err, &notFound) { return nil, models.NewGenericNotFoundError(err) }`)
**Patch-merge mirroring the adapter** — Update reconstructs a productcatalog.PlanAddon by merging params over existing (MaxQuantity always replaced, FromPlanPhase/Metadata/Annotations kept when nil) and validates before persisting — must stay in sync with adapter UpdatePlanAddon. (`fromPlanPhase := planAddon.FromPlanPhase; if params.FromPlanPhase != nil { fromPlanPhase = *params.FromPlanPhase }`)
**Publish lifecycle events** — After each successful mutation, publish via eventbus.Publisher: NewPlanAddonCreateEvent / NewPlanAddonUpdateEvent / NewPlanAddonDeleteEvent; a publish failure fails the whole transaction. (`event := planaddon.NewPlanAddonCreateEvent(ctx, planAddon); s.publisher.Publish(ctx, event)`)
**Scoped structured logging** — Each method derives a logger via s.logger.With("operation", ..., "namespace", ..., ids...) and logs Debug at phase boundaries. (`logger := s.logger.With("operation", "create", "namespace", params.Namespace, "plan.id", params.PlanID, "addon.id", params.AddonID)`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `service.go` | Config{Adapter,Plan,Addon,Logger,Publisher}, New() with explicit non-nil checks, and the service struct. | All five deps are required — New errors if any is nil; service depends on plan.Service and addon.Service (not just the repo) to validate state. |
| `planaddon.go` | The five Service methods. Holds the load-bearing comment block documenting patch-field merge rules that mirror the adapter. | Update's merge rules must match adapter/planaddon.go UpdatePlanAddon exactly (MaxQuantity replace vs FromPlanPhase/Metadata/Annotations keep-when-nil). Delete validates the plan is still Draft/Scheduled before deleting and is idempotent (returns nil if already DeletedAt). |
| `service_test.go` | TestPlanAddonService Postgres integration test via pctestutils.NewTestEnv; full plan+addon+feature lifecycle including the 'PublishedPlan' negative case. | Asserts create fails once the plan is published/active; drives through env.PlanAddon (service) vs env.PlanAddonRepository (adapter). |

## Anti-Patterns

- Mutating without wrapping in transaction.Run/RunWithNoValue (publish + write must be atomic).
- Letting the service's Update merge logic drift from the adapter's SetOrClearMaxQuantity / Set*-when-non-nil behavior.
- Returning typed *NotFoundError up the stack instead of models.NewGenericNotFoundError.
- Skipping plan/addon state validation (Draft/Scheduled, Active) before creating or updating an assignment.
- Swallowing publisher errors instead of failing the transaction.

## Decisions

- **Service depends on plan.Service and addon.Service, not just the planaddon repository.** — Assignment validity depends on live plan/addon status and compatibility, which only those services can resolve.
- **Update rebuilds and Validate()s a productcatalog.PlanAddon before persisting.** — Catches incompatible patches (e.g. bad FromPlanPhase) before they reach the DB, keeping invalid assignments out of the table.
- **Events publish inside the transaction.** — Guarantees no event is emitted for a write that later rolls back.

## Example: Service mutation: validate, persist via adapter, publish, all inside a transaction

```
func (s service) CreatePlanAddon(ctx context.Context, params planaddon.CreatePlanAddonInput) (*planaddon.PlanAddon, error) {
	fn := func(ctx context.Context) (*planaddon.PlanAddon, error) {
		if err := params.Validate(); err != nil { return nil, fmt.Errorf("invalid ...: %w", err) }
		p, err := s.plan.GetPlan(ctx, plan.GetPlanInput{NamespacedID: models.NamespacedID{Namespace: params.Namespace, ID: params.PlanID}})
		if err != nil { /* map plan.NotFoundError -> models.NewGenericNotFoundError */ }
		if err = p.ValidateWith(plan.IsPlanDeleted(clock.Now()), plan.HasPlanStatus(productcatalog.PlanStatusDraft, productcatalog.PlanStatusScheduled)); err != nil {
			return nil, models.NewGenericValidationError(err)
		}
		planAddon, err := s.adapter.CreatePlanAddon(ctx, params)
		if err != nil { return nil, err }
		if err = s.publisher.Publish(ctx, planaddon.NewPlanAddonCreateEvent(ctx, planAddon)); err != nil { return nil, err }
		return planAddon, nil
	}
	return transaction.Run(ctx, s.adapter, fn)
}
```

<!-- archie:ai-end -->
