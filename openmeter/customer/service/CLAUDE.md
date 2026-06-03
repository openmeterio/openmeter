# service

<!-- archie:ai-start -->

> Concrete implementation of customer.Service (package customerservice): orchestrates customer CRUD by running RequestValidatorRegistry guards before DB writes, wrapping every mutation in transaction.Run, fanning out ServiceHookRegistry callbacks inside the transaction, and publishing typed Watermill events as the last step. The single authoritative integration point where cross-domain hooks and validators attach without the customer package importing them; sub-package service/hooks/ holds the entitlement-validator and subject-customer hook implementations.

## Patterns

**Config + Validate() + New()** — Service is only constructable via New(Config); Config.Validate() rejects nil Adapter/Publisher so requestValidatorRegistry is always initialised via NewRequestValidatorRegistry(). (`func New(config Config) (*Service, error) { if err := config.Validate(); err != nil { return nil, err }; return &Service{adapter: config.Adapter, requestValidatorRegistry: customer.NewRequestValidatorRegistry(), ...}, nil }`)
**transaction.Run wraps all mutations** — Every DB-writing method wraps its body in transaction.Run / transaction.RunWithNoValue so adapter writes, hook calls and event publishes share one Postgres transaction and roll back atomically. (`return transaction.Run(ctx, s.adapter, func(ctx context.Context) (*customer.Customer, error) { ... })`)
**Validators before the transaction** — CreateCustomer/UpdateCustomer/DeleteCustomer call s.requestValidatorRegistry.Validate*Customer before transaction.Run, wrapping failures in models.NewGenericValidationError. (`if err := s.requestValidatorRegistry.ValidateCreateCustomer(ctx, input); err != nil { return nil, models.NewGenericValidationError(err) }`)
**Hook fan-out inside the transaction** — s.hooks (models.ServiceHookRegistry[customer.Customer]) runs Pre hooks before the adapter write and Post hooks after, all inside the tx so a hook error rolls back the write. (`if err = s.hooks.PostCreate(ctx, createdCustomer); err != nil { return nil, err }`)
**Event publish as the last in-tx step** — After adapter write and hook fan-out, each mutation publishes a typed event (customer.NewCustomerCreateEvent etc.); publish errors abort the transaction. (`if err := s.publisher.Publish(ctx, customer.NewCustomerCreateEvent(ctx, createdCustomer)); err != nil { return nil, fmt.Errorf("...: %w", err) }`)
**models.Generic* error types for domain conditions** — Not-found, pre-condition-failed, and validation conditions are wrapped in models.NewGenericNotFoundError / NewGenericPreConditionFailedError / NewGenericValidationError so HTTP encoders map them to correct status codes. (`return models.NewGenericPreConditionFailedError(customer.NewErrDeletingCustomerWithActiveSubscriptions(...))`)
**Compile-time interface assertion per file** — Each file adding Service methods declares var _ customer.Service = (*Service)(nil) (or RequestValidatorService) for compile-time satisfaction. (`var _ customer.Service = (*Service)(nil)`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `service.go` | Service struct, Config, Config.Validate(), New(), and RegisterHooks. The only place new service-level dependency fields belong. | Adding business logic here (belongs in customer.go); forgetting to validate new Config fields; skipping the compile-time assertion. |
| `customer.go` | All customer.Service method implementations: validate input → transaction.Run → adapter call → hook fan-out → event publish. | Mutations outside transaction.Run; publishing before hooks (hooks must precede publish inside the tx); raw fmt.Errorf for domain conditions instead of models.Generic*. |
| `requestvalidator.go` | Thin delegation: RegisterRequestValidator forwards to s.requestValidatorRegistry.Register; carries the RequestValidatorService compile-time assertion. | Duplicating validation logic here instead of in a registered RequestValidator implementation. |
| `service_test.go` | Full CRUD integration test built from customertestutils.NewTestEnv (no app/common), using t.Context() throughout. | Importing app/common breaks isolation; context.Background() instead of t.Context() causes lifecycle/tracing issues. |
| `ledger_hook_test.go` | Integration test for the ledger provisioning hook: verifies accounts created on CreateCustomer and rollback when the hook fails. | Hook-failure tests must assert the customer row does not exist; use ErrorIs/ErrorAs against models.Generic* types. |

## Anti-Patterns

- Performing DB writes outside transaction.Run / transaction.RunWithNoValue — partial writes are not rolled back on hook or publish failure.
- Calling customer.Service from inside a hook without NewContextWithSkipSubjectCustomer(ctx) — causes infinite re-entrant hook invocations.
- Returning raw fmt.Errorf for not-found/conflict/validation conditions — HTTP encoders and callers depend on models.Generic* typed errors.
- Importing app/common in test files — breaks isolation and can introduce import cycles; build deps from customertestutils.NewTestEnv.
- Using context.Background()/context.TODO() in methods or tests — always propagate the caller ctx / t.Context().

## Decisions

- **All mutations are wrapped in transaction.Run so hooks and event publishes are atomic with the adapter write.** — A hook failure (e.g. ledger provisioning) must roll back the customer row; publishing a stale event after a partial write would corrupt downstream state.
- **RequestValidatorRegistry and ServiceHookRegistry are separate registries with distinct call sites — validators run before the transaction, hooks inside it.** — Validators perform cheap pre-condition checks without a transaction; hooks need rollback capability. Merging them would force pre-condition checks to acquire a transaction unnecessarily.
- **Service is only constructable via New(Config), never by struct literal.** — Guarantees requestValidatorRegistry is initialised and required fields validated at construction, preventing nil-panic at first method call.

## Example: Add a new mutating method following the established pattern

```
func (s *Service) ArchiveCustomer(ctx context.Context, input customer.ArchiveCustomerInput) (*customer.Customer, error) {
    if err := s.requestValidatorRegistry.ValidateUpdateCustomer(ctx, customer.UpdateCustomerInput{CustomerID: input.CustomerID}); err != nil {
        return nil, models.NewGenericValidationError(err)
    }
    return transaction.Run(ctx, s.adapter, func(ctx context.Context) (*customer.Customer, error) {
        // adapter write -> s.hooks.Pre/PostUpdate -> s.publisher.Publish(customer.NewCustomerUpdateEvent(ctx, cus))
        return cus, nil
    })
}
```

<!-- archie:ai-end -->
