# service

<!-- archie:ai-start -->

> Business logic service for subscription addons: validates plan compatibility, instance-type constraints, and MaxQuantity limits before persisting via SubscriptionAddonRepository and publishing Watermill events. Implements subscriptionaddon.Service.

## Patterns

**Config struct with Validate() and fail-fast construction** — Constructor NewService accepts a Config struct and calls cfg.Validate() before returning — fail-fast at wiring time rather than runtime nil-pointer panics. (`func NewService(cfg Config) (subscriptionaddon.Service, error) {
	if err := cfg.Validate(); err != nil { return nil, err }
	return &service{cfg: cfg}, nil
}`)
**Pre-persist validation chain** — Create and ChangeQuantity both: (1) call input.Validate(), (2) fetch the addon to check InstanceType, (3) fetch the subscription view to check PlanRef and current phase, (4) fetch PlanAddon for FromPlanPhase and MaxQuantity. Only then enter transaction.Run. (`if err := input.Validate(); err != nil { return nil, err }
add, err := s.cfg.AddonService.GetAddon(ctx, ...)
if add.InstanceType == productcatalog.AddonInstanceTypeSingle && input.InitialQuantity.Quantity != 1 {
	return nil, models.NewGenericValidationError(...)
}`)
**transaction.Run for multi-step writes** — All writes use transaction.Run(ctx, s.cfg.TxManager, func) which coordinates the repo Tx/Self/WithTx chain. Repo calls inside the transaction func use the tx-bound ctx. (`return transaction.Run(ctx, s.cfg.TxManager, func(ctx context.Context) (*subscriptionaddon.SubscriptionAddon, error) {
	repo.Create(ctx, ns, ...)
	qtyRepo.Create(ctx, id, ...)
	add, _ := repo.Get(ctx, id)
	s.cfg.Publisher.Publish(ctx, subscriptionaddon.NewCreatedEvent(...))
	return add, nil
})`)
**Event publish inside transaction** — After persisting, Publisher.Publish is called inside the transaction.Run func so the event is tied to the same unit of work. NewCreatedEvent / NewChangeQuantityEvent are factory functions in the subscriptionaddon package. (`s.cfg.Publisher.Publish(ctx, subscriptionaddon.NewCreatedEvent(ctx, sView.Customer, *subscriptionAddon))`)
**Phase-ordered compatibility check** — To verify an addon can be added at a given time, the service iterates sView.Phases in order and checks whether compatibility.FromPlanPhase is reached before the phase containing the addon start time. (`for _, phase := range sView.Phases {
	if phase.Key == compatibility.FromPlanPhase { break }
	if phase.Key == phaseAtAddonStart.PhaseKey {
		return nil, models.NewGenericValidationError(...)
	}
}`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `service.go` | Full implementation of subscriptionaddon.Service: Create, Get, List, ChangeQuantity with all validation and transaction coordination. | Create and ChangeQuantity duplicate several validation steps (InstanceType, FromPlanPhase, MaxQuantity) — keep them in sync if validation rules change. Addons cannot be added to custom subscriptions (no PlanRef). |

## Anti-Patterns

- Calling SubAddRepo or SubAddQtyRepo outside of transaction.Run for writes — partial writes will occur on failure
- Bypassing cfg.Validate() by constructing the service struct directly instead of via NewService
- Publishing events outside the transaction.Run func — event and DB write will be inconsistent on rollback
- Adding addons to subscriptions without a PlanRef — the service explicitly rejects custom subscriptions

## Decisions

- **Service validates plan compatibility before any DB write** — Addon purchase must respect plan-level constraints (FromPlanPhase, MaxQuantity, InstanceType) that live in the product catalog, not in the subscription tables.
- **Separate SubscriptionAddonQuantityRepository for quantity records** — Quantity timeline is append-only and models a different entity; a dedicated repo keeps quantity writes isolated and avoids loading all quantities on every addon mutation.

## Example: Create a subscription addon with validation and transactional persistence

```
// From service.go — transaction.Run coordinates repo writes and event publish
return transaction.Run(ctx, s.cfg.TxManager, func(ctx context.Context) (*subscriptionaddon.SubscriptionAddon, error) {
	subAddID, err := s.cfg.SubAddRepo.Create(ctx, namespace, subscriptionaddon.CreateSubscriptionAddonRepositoryInput{
		AddonID: input.AddonID, SubscriptionID: input.SubscriptionID,
	})
	if err != nil { return nil, err }
	_, err = s.cfg.SubAddQtyRepo.Create(ctx, *subAddID, subscriptionaddon.CreateSubscriptionAddonQuantityRepositoryInput{
		ActiveFrom: input.InitialQuantity.ActiveFrom,
		Quantity:   input.InitialQuantity.Quantity,
	})
	if err != nil { return nil, err }
	subAdd, err := s.cfg.SubAddRepo.Get(ctx, *subAddID)
	if err != nil { return nil, err }
	s.cfg.Publisher.Publish(ctx, subscriptionaddon.NewCreatedEvent(ctx, sView.Customer, *subAdd))
	return subAdd, nil
// ...
```

<!-- archie:ai-end -->
