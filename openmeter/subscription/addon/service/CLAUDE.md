# service

<!-- archie:ai-start -->

> Service layer (package service) implementing subscriptionaddon.Service: Create, Get, List, ChangeQuantity for subscription addons. It validates compatibility (plan linkage, phase availability, instance type, max quantity) before persisting through the repos and publishing watermill events.

## Patterns

**Config struct with Validate + NewService** — Dependencies (SubAddRepo, SubAddQtyRepo, Publisher, TxManager, AddonService, PlanAddonService, SubService, Logger) are passed via a Config whose Validate() checks each non-nil; NewService(cfg) returns subscriptionaddon.Service and errors on invalid config. (`func NewService(cfg Config) (subscriptionaddon.Service, error) { if err := cfg.Validate(); err != nil { return nil, err } ... }`)
**Validate-then-check-compatibility-then-transact** — Create/ChangeQuantity first input.Validate() (wrapped in models.NewGenericValidationError), then fetch addon + subscription view, run compatibility checks, and only then transaction.Run(ctx, TxManager, ...) to write + publish. (`return transaction.Run(ctx, s.cfg.TxManager, func(ctx context.Context) (...) { ... })`)
**GenericValidationError for business rules** — Domain violations (single-instance qty != 1, addon not linked to plan, phase too early, max quantity exceeded, custom subscription) return models.NewGenericValidationError with a formatted message; tests assert via ErrorAs(&GenericValidationError{}). (`return nil, models.NewGenericValidationError(fmt.Errorf("addon %s@%d can be added a maximum of %d times", add.Key, add.Version, *compatibility.MaxQuantity))`)
**Phase-availability scan against PlanAddon compatibility** — Both Create and ChangeQuantity look up planaddon.GetPlanAddon, then iterate sView.Phases stopping at compatibility.FromPlanPhase; if the addon's start phase is reached first it errors that the addon can only start from FromPlanPhase. (`for _, phase := range sView.Phases { if phase.SubscriptionPhase.Key == compatibility.FromPlanPhase { break }; if phase...Key == phaseAtAddonStart.PhaseKey { return ...validation error } }`)
**Publish customer-scoped events in-transaction** — After persisting, the service re-fetches the addon and publishes subscriptionaddon.NewCreatedEvent / NewChangeQuantityEvent with sView.Customer via the eventbus Publisher inside the same transaction. (`err = s.cfg.Publisher.Publish(ctx, subscriptionaddon.NewCreatedEvent(ctx, sView.Customer, *subscriptionAddon))`)
**Create initial quantity alongside addon** — Create writes the SubscriptionAddon row then the initial CreateSubscriptionAddonQuantityRepositoryInput, then re-Get to return a fully-loaded aggregate; ChangeQuantity reuses CreateSubscriptionAddonQuantityRepositoryInput(input). (`s.cfg.SubAddQtyRepo.Create(ctx, *subAdd, subscriptionaddon.CreateSubscriptionAddonQuantityRepositoryInput{ActiveFrom: input.InitialQuantity.ActiveFrom, Quantity: input.InitialQuantity.Quantity})`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `service.go` | Config/Validate/NewService and Create/Get/List/ChangeQuantity implementations | Compatibility-check logic is duplicated between Create and ChangeQuantity; both re-Get after write and publish in-tx; Create blocks PlanRef==nil (custom subscriptions) |
| `create_test.go` | Service-backed Create tests (invalid input, missing addon/subscription, single-instance qty, plan linkage, phase, max quantity) | Uses subscriptiontestutils withDeps + createPlanWithAddon; asserts exact error strings like 'addon %s@%d is not linked to the plan' |
| `change_test.go` | ChangeQuantity tests incl. max-quantity-after-purchase and single-instance guard | createExampleSubscriptionAddon helper builds plan+addon+subscription+subAddon; ChangeQuantity appends a quantity segment |
| `list_test.go` | Get/List tests including pagination and name/description inheritance from addon | List returns all when Page is zero; SubscriptionAddonsEqual used for comparison |

## Anti-Patterns

- Persisting before running input.Validate() and plan/phase/quantity compatibility checks
- Returning plain errors for business-rule violations instead of models.NewGenericValidationError
- Writing without transaction.Run, or publishing events outside the transaction
- Allowing addon purchase on a subscription with nil PlanRef (custom subscription)
- Skipping the re-Get before returning/publishing (event and return value must carry the full aggregate)

## Decisions

- **Quantity changes are modeled as new quantity segments, not in-place edits** — Mirrors the append-only repo timeline so an addon's quantity history is preserved; ChangeQuantity simply Creates another segment
- **Compatibility (plan linkage, FromPlanPhase, MaxQuantity, instance type) is enforced in the service, not the repo** — These rules depend on PlanAddon and SubscriptionView state the repo doesn't see; service is the only place with all dependencies
- **Events published inside the same transaction as the write** — Ensures created/change-quantity events are not emitted for rolled-back changes

## Example: Validate, check compatibility, then persist + publish in a transaction

```
func (s *service) Create(ctx context.Context, ns string, input subscriptionaddon.CreateSubscriptionAddonInput) (*subscriptionaddon.SubscriptionAddon, error) {
	if err := input.Validate(); err != nil {
		return nil, models.NewGenericValidationError(fmt.Errorf("invalid input: %w", err))
	}
	// ... fetch addon, sView, plan-addon compatibility, phase + max-quantity checks ...
	return transaction.Run(ctx, s.cfg.TxManager, func(ctx context.Context) (*subscriptionaddon.SubscriptionAddon, error) {
		subAdd, err := s.cfg.SubAddRepo.Create(ctx, ns, subscriptionaddon.CreateSubscriptionAddonRepositoryInput{
			MetadataModel: input.MetadataModel, AddonID: input.AddonID, SubscriptionID: input.SubscriptionID,
		})
		if err != nil { return nil, err }
		if _, err = s.cfg.SubAddQtyRepo.Create(ctx, *subAdd, subscriptionaddon.CreateSubscriptionAddonQuantityRepositoryInput{
			ActiveFrom: input.InitialQuantity.ActiveFrom, Quantity: input.InitialQuantity.Quantity,
		}); err != nil { return nil, err }
		subscriptionAddon, err := s.cfg.SubAddRepo.Get(ctx, subscriptionaddon.GetSubscriptionAddonInput{NamespacedID: *subAdd})
		if err != nil { return nil, err }
// ...
```

<!-- archie:ai-end -->
