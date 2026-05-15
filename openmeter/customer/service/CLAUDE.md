# service

<!-- archie:ai-start -->

> Concrete implementation of customer.Service: orchestrates customer CRUD lifecycle by running RequestValidatorRegistry checks before DB writes, wrapping all mutations in Postgres transactions via transaction.Run, fanning out ServiceHookRegistry callbacks (Pre/Post Create/Update/Delete) inside the transaction, and publishing typed Watermill domain events as the last step. Acts as the single authoritative integration point where hooks and validators attach to the customer domain without importing them directly.

## Patterns

**Config struct + Validate() + New()** — Service is only constructable via New(Config), never by struct literal. Config.Validate() rejects nil fields at construction time, ensuring requestValidatorRegistry is always initialised. (`func New(config Config) (*Service, error) { if err := config.Validate(); err != nil { return nil, err } return &Service{adapter: config.Adapter, requestValidatorRegistry: customer.NewRequestValidatorRegistry(), ...}, nil }`)
**transaction.Run / transaction.RunWithNoValue wraps all mutations** — Every method that writes to the DB wraps its body in transaction.Run or transaction.RunWithNoValue so adapter writes, hook calls, and event publishes share one Postgres transaction and roll back atomically on any error. (`return transaction.Run(ctx, s.adapter, func(ctx context.Context) (*customer.Customer, error) { created, err := s.adapter.CreateCustomer(ctx, input); ...; s.hooks.PostCreate(ctx, created); ...; s.publisher.Publish(ctx, event); return created, nil })`)
**RequestValidatorRegistry called before entering the transaction** — CreateCustomer, UpdateCustomer, and DeleteCustomer call s.requestValidatorRegistry.Validate*Customer before transaction.Run; errors are wrapped with models.NewGenericValidationError. (`if err := s.requestValidatorRegistry.ValidateCreateCustomer(ctx, input); err != nil { return nil, models.NewGenericValidationError(err) }`)
**ServiceHookRegistry fan-out inside the transaction** — s.hooks is a models.ServiceHookRegistry[customer.Customer]; Pre hooks run before the adapter write, Post hooks run after, all inside the transaction so a hook failure rolls back the adapter write. (`if err = s.hooks.PostCreate(ctx, createdCustomer); err != nil { return nil, err }`)
**Watermill event publish as the last step inside the transaction** — After adapter write and hook fan-out, each mutation publishes a typed domain event (customer.NewCustomerCreateEvent etc.) via s.publisher.Publish; publish errors abort the transaction. (`if err := s.publisher.Publish(ctx, customer.NewCustomerCreateEvent(ctx, createdCustomer)); err != nil { return nil, fmt.Errorf("failed to publish customer created event: %w", err) }`)
**models.Generic* error types for all domain conditions** — Not-found, pre-condition-failed, and validation errors are wrapped in models.NewGenericNotFoundError / models.NewGenericPreConditionFailedError / models.NewGenericValidationError so HTTP encoders map them to correct status codes. (`return models.NewGenericPreConditionFailedError(customer.NewErrDeletingCustomerWithActiveSubscriptions(cus.ActiveSubscriptionIDs.OrEmpty()))`)
**Compile-time interface assertion at the top of each file** — Every file that adds methods to Service declares var _ customer.Service = (*Service)(nil) ensuring compile-time interface satisfaction. (`var _ customer.Service = (*Service)(nil)`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `service.go` | Defines Service struct, Config, Config.Validate(), and New(). The only place new service-level dependencies (fields) should be added. | Adding business logic here instead of customer.go; forgetting to validate new Config fields in Validate(); skipping the compile-time assertion. |
| `customer.go` | Implements all customer.Service methods. Pattern per method: validate input → transaction.Run → adapter call → hook fan-out → event publish. | Mutations outside transaction.Run; hooks called after publisher.Publish (hooks must precede publish inside tx); raw fmt.Errorf for domain conditions instead of models.Generic* types. |
| `requestvalidator.go` | Thin delegation: RegisterRequestValidator forwards to s.requestValidatorRegistry.Register and carries the compile-time assertion for customer.RequestValidatorService. | Duplicating validation logic here instead of a registered RequestValidator implementation. |
| `service_test.go` | Full CRUD integration test built from customertestutils.NewTestEnv (no app/common). Tests use t.Context() throughout. | Importing app/common breaks test isolation; using context.Background() instead of t.Context() causes lifecycle/tracing issues. |
| `ledger_hook_test.go` | Integration test for the ledger provisioning hook: registers a hook via env.CustomerService.RegisterHooks, verifies accounts created on CreateCustomer, and verifies transaction rollback when hook fails. | Hook tests must assert rollback: if hook returns an error the customer row must not exist. Use ErrorIs/ErrorAs against models.Generic* types. |

## Anti-Patterns

- Performing DB writes outside transaction.Run / transaction.RunWithNoValue — partial writes are not rolled back on hook or publish failure.
- Calling customer.Service from inside a hook without NewContextWithSkipSubjectCustomer(ctx) — causes infinite re-entrant hook invocations.
- Returning raw fmt.Errorf for domain conditions (not-found, conflict, validation) — HTTP encoders and callers depend on models.Generic* typed errors.
- Importing app/common in test files — breaks isolation and can introduce import cycles; build test deps from customertestutils.NewTestEnv.
- Using context.Background() or context.TODO() in methods or tests — always propagate caller ctx to maintain tracing, cancellation, and t.Context() lifecycle.

## Decisions

- **All mutations wrapped in transaction.Run so hooks and event publishes are atomic with the adapter write.** — A hook failure (e.g. ledger provisioning) must roll back the customer row; publishing a stale event after a partial write would corrupt downstream state.
- **RequestValidatorRegistry and ServiceHookRegistry are separate registries with distinct call sites — validators run before the transaction, hooks run inside it.** — Validators perform cheap pre-condition checks without needing a transaction; hooks need rollback capability. Merging them would force pre-condition checks to acquire a transaction unnecessarily.
- **Service struct is only constructable via New(Config), not by direct struct literal.** — Ensures requestValidatorRegistry is always initialised via NewRequestValidatorRegistry() and required fields are validated at construction time, preventing nil-panic bugs at first method call.

## Example: Adding a new mutating method (e.g. ArchiveCustomer) following the established pattern

```
import (
	"context"
	"fmt"

	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/pkg/framework/transaction"
	"github.com/openmeterio/openmeter/pkg/models"
)

func (s *Service) ArchiveCustomer(ctx context.Context, input customer.ArchiveCustomerInput) (*customer.Customer, error) {
	// 1. Pre-condition check outside transaction
	if err := s.requestValidatorRegistry.ValidateUpdateCustomer(ctx, customer.UpdateCustomerInput{
		CustomerID: input.CustomerID,
	}); err != nil {
		return nil, models.NewGenericValidationError(err)
// ...
```

<!-- archie:ai-end -->
