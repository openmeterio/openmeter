# service

<!-- archie:ai-start -->

> Concrete implementation of customer.Service (package customerservice). Orchestrates the customer CRUD + lifecycle flow: request validation, transactional adapter calls, service hooks, and event publishing. The high-fan-in root service consumed by billing, subscription, ledger, and the v3 customers handlers.

## Patterns

**Service struct holds adapter + validators + publisher + hook registry** — Service has exactly four fields: adapter customer.Adapter, requestValidatorRegistry customer.RequestValidatorRegistry, publisher eventbus.Publisher, hooks models.ServiceHookRegistry[customer.Customer]. No other dependencies are injected. (`type Service struct { adapter customer.Adapter; requestValidatorRegistry customer.RequestValidatorRegistry; publisher eventbus.Publisher; hooks models.ServiceHookRegistry[customer.Customer] }`)
**Config + Validate + New constructor** — Construct via New(Config) which calls config.Validate() (rejects nil Adapter/Publisher) before building Service. RequestValidatorRegistry and empty hook registry are created internally, not passed in. (`func New(config Config) (*Service, error) { if err := config.Validate(); err != nil { return nil, err }; return &Service{...}, nil }`)
**Interface assertion per file** — Each file asserts the interface it satisfies at package level: customer.go and service.go both have `var _ customer.Service = (*Service)(nil)`; requestvalidator.go has `var _ customer.RequestValidatorService = (*Service)(nil)`. (`var _ customer.Service = (*Service)(nil)`)
**Validate -> transaction.Run -> adapter -> hooks -> publish** — Every mutating method follows: requestValidatorRegistry.Validate*(ctx, input) wrapped in models.NewGenericValidationError, then transaction.Run/RunWithNoValue(ctx, s.adapter, ...), inside which the adapter mutates, hooks fire (PostCreate / Pre+PostUpdate / Pre+PostDelete), then publisher.Publish emits the domain event. (`return transaction.Run(ctx, s.adapter, func(ctx context.Context) (*customer.Customer, error) { c, err := s.adapter.CreateCustomer(ctx, input); ...; s.hooks.PostCreate(ctx, c); s.publisher.Publish(ctx, customer.NewCustomerCreateEvent(ctx, c)) })`)
**Read methods pass straight through to adapter** — GetCustomer, ListCustomers, GetCustomerByUsageAttribution, ListCustomerUsageAttributions are one-line delegations to s.adapter with no validation, transaction, hook, or event. Only mutations get the full pipeline. (`func (s *Service) GetCustomer(ctx context.Context, input customer.GetCustomerInput) (*customer.Customer, error) { return s.adapter.GetCustomer(ctx, input) }`)
**Deleted / precondition guards before mutate** — Delete refuses customers with active subscriptions (NewGenericPreConditionFailedError + NewErrDeletingCustomerWithActiveSubscriptions) and treats already-deleted/not-found as no-op; Update rejects soft-deleted customers (cus.IsDeleted() -> NewGenericPreConditionFailedError). Delete expands ExpandSubscriptions and errors if ActiveSubscriptionIDs.IsAbsent(). (`if len(cus.ActiveSubscriptionIDs.OrEmpty()) > 0 { return models.NewGenericPreConditionFailedError(customer.NewErrDeletingCustomerWithActiveSubscriptions(cus.ActiveSubscriptionIDs.OrEmpty())) }`)
**Hooks/validators registered post-construction** — RegisterHooks(...models.ServiceHook[customer.Customer]) and RegisterRequestValidator(customer.RequestValidator) mutate the registries after New. Cross-domain wiring (ledgerresolvers.NewCustomerLedgerHook, subjectcustomer/entitlementvalidator hooks) is attached this way, not via Config. (`env.CustomerService.RegisterHooks(hook)`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `service.go` | Defines Service struct, Config, Validate, New, RegisterHooks. The DI surface of the package. | Adding a dependency means a new Config field + a nil check in Validate; do not reach for slog.Default() or context.Background() fallbacks. |
| `customer.go` | All customer.Service method implementations (List/Create/Delete/Get/GetByUsageAttribution/Update). | Keep each mutation's adapter write, hooks, and publish inside the SAME transaction.Run closure so a hook or publish failure rolls back the write (see TestCustomerService_CreateCustomerRollsBackWhenLedgerProvisioningFails). Reads must stay pass-through. |
| `requestvalidator.go` | Implements customer.RequestValidatorService.RegisterRequestValidator delegating to requestValidatorRegistry.Register. | Validation is invoked per-method via requestValidatorRegistry.ValidateCreate/Update/DeleteCustomer wrapped in models.NewGenericValidationError; do not validate outside that wrapper. |
| `service_test.go` | End-to-end CRUD test using customertestutils.NewTestEnv + DBSchemaMigrate, asserting create/get-by-id/key/idorkey/usage-attribution, update, soft-delete semantics. | Soft delete: GetCustomer by ID still returns the (DeletedAt-set) customer, but Get by key returns *models.GenericNotFoundError; list needs IncludeDeleted:true to see it. |
| `ledger_hook_test.go` | Proves hook + transaction integration: CreateCustomer provisions ledger accounts via a registered hook, and a failing provisioner rolls back the whole create. | Uses ledgertestutils.InitDeps + ledgerresolvers.NewCustomerLedgerHook; the failing-provisioner case asserts the customer is NOT persisted (GenericNotFoundError), validating transactional hook semantics. |

## Anti-Patterns

- Calling s.adapter mutating methods outside transaction.Run/RunWithNoValue, or placing hooks/publish outside that closure so a later failure cannot roll back the write.
- Adding validation, hooks, or event publishing to read methods (Get/List), or skipping requestValidatorRegistry validation on a new mutation.
- Constructing Service via &Service{} literal instead of New(Config), bypassing config.Validate() nil checks.
- Returning bare errors instead of models.NewGenericValidationError / NewGenericPreConditionFailedError / NewErrDeletingCustomerWithActiveSubscriptions for the established failure modes.
- Mutating a soft-deleted customer (cus.IsDeleted()) or deleting one with active subscriptions instead of returning the precondition-failed guard.

## Decisions

- **Hooks and request validators are registered after construction (RegisterHooks/RegisterRequestValidator) rather than injected via Config.** — Cross-domain glue (ledger account provisioning, subject sync, entitlement-deletion guard) is wired at app/common DI time without creating import cycles back into the core customer service, and can be toggled (e.g. credits disabled -> noop ledger hook).
- **Mutation, hooks, and event publishing all run inside one transaction.Run closure.** — Guarantees atomicity: a hook (e.g. ledger provisioning) or publish failure rolls back the customer write, as exercised by the rollback test.
- **Read methods delegate directly to the adapter with no service-layer logic.** — Validation, lifecycle, and eventing only matter for state changes; keeping reads thin avoids redundant overhead on the high-fan-in read path.

## Example: Mutating method: validate, transact, mutate, hook, publish

```
func (s *Service) CreateCustomer(ctx context.Context, input customer.CreateCustomerInput) (*customer.Customer, error) {
	if err := s.requestValidatorRegistry.ValidateCreateCustomer(ctx, input); err != nil {
		return nil, models.NewGenericValidationError(err)
	}
	return transaction.Run(ctx, s.adapter, func(ctx context.Context) (*customer.Customer, error) {
		created, err := s.adapter.CreateCustomer(ctx, input)
		if err != nil {
			return nil, err
		}
		if err = s.hooks.PostCreate(ctx, created); err != nil {
			return nil, err
		}
		if err := s.publisher.Publish(ctx, customer.NewCustomerCreateEvent(ctx, created)); err != nil {
			return nil, fmt.Errorf("failed to publish customer created event: %w", err)
		}
// ...
```

<!-- archie:ai-end -->
