# service

<!-- archie:ai-start -->

> Concrete implementation of customer.Service: orchestrates customer CRUD lifecycle, runs request validation, fires ServiceHook callbacks (Pre/Post Create/Update/Delete), publishes Watermill domain events, and wraps all mutating operations in Postgres transactions. Acts as the single authoritative place where hooks and validators integrate with the customer domain without importing them directly.

## Patterns

**Compile-time interface assertion** — Every file that adds methods to Service declares `var _ customer.Service = (*Service)(nil)` at the top, ensuring the struct satisfies the interface at compile time. (`var _ customer.Service = (*Service)(nil)`)
**Config struct + Validate() + New()** — Service is constructed via a Config struct with a Validate() method that rejects nil fields; New(config) returns (*Service, error) — never a bare struct literal. (`func New(config Config) (*Service, error) { if err := config.Validate(); err != nil { return nil, err } ... }`)
**transaction.Run / transaction.RunWithNoValue wrapping all mutations** — Every method that writes to the DB (Create, Update, Delete) wraps its body in transaction.Run or transaction.RunWithNoValue so all DB writes, hook calls, and event publishes share one Postgres transaction. (`return transaction.Run(ctx, s.adapter, func(ctx context.Context) (*customer.Customer, error) { ... })`)
**RequestValidatorRegistry called before any DB write** — CreateCustomer, UpdateCustomer, and DeleteCustomer call s.requestValidatorRegistry.Validate*Customer before entering the transaction; validation errors are wrapped with models.NewGenericValidationError. (`if err := s.requestValidatorRegistry.ValidateCreateCustomer(ctx, input); err != nil { return nil, models.NewGenericValidationError(err) }`)
**ServiceHookRegistry fan-out (Pre/Post hooks)** — s.hooks is a models.ServiceHookRegistry[customer.Customer]; all mutating methods call s.hooks.PreDelete/PostCreate/etc. inside the transaction after adapter writes, before event publish. (`if err = s.hooks.PostCreate(ctx, createdCustomer); err != nil { return nil, err }`)
**Watermill event publish as last step inside transaction** — After adapter write and hook fan-out, each mutation publishes a typed domain event (customer.NewCustomerCreateEvent etc.) via s.publisher.Publish; publish errors abort the transaction. (`if err := s.publisher.Publish(ctx, customer.NewCustomerCreateEvent(ctx, createdCustomer)); err != nil { return nil, fmt.Errorf("failed to publish...: %w", err) }`)
**models.Generic* error types for all domain conditions** — Not-found, pre-condition-failed, and validation errors are wrapped in models.NewGenericNotFoundError / models.NewGenericPreConditionFailedError / models.NewGenericValidationError so HTTP encoders can map them to correct status codes. (`return models.NewGenericPreConditionFailedError(customer.NewErrDeletingCustomerWithActiveSubscriptions(...))`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `service.go` | Defines the Service struct, Config, Config.Validate(), and New(). Holds adapter, requestValidatorRegistry, publisher, and hooks fields. The only place new service-level dependencies should be added. | Adding business logic here instead of in customer.go; forgetting to validate new Config fields; skipping the compile-time assertion. |
| `customer.go` | Implements all customer.Service methods. Pattern: validate input → transaction.Run → adapter call → hook fan-out → event publish. | Mutations outside a transaction.Run; hooks called after publisher.Publish (hooks must be inside the tx, publish is last); raw fmt.Errorf for domain conditions instead of models.Generic* types. |
| `requestvalidator.go` | Thin delegation of RegisterRequestValidator to s.requestValidatorRegistry.Register; also carries the compile-time assertion for customer.RequestValidatorService. | Accidentally duplicating validation logic here instead of in a registered RequestValidator. |
| `service_test.go` | Full CRUD integration test built from customertestutils.NewTestEnv (no app/common). Tests use t.Context() throughout. | Importing app/common — breaks test isolation and can cause import cycles; using context.Background() instead of t.Context(). |
| `ledger_hook_test.go` | Integration test for the ledger provisioning hook: registers a hook via env.CustomerService.RegisterHooks, verifies accounts created on CreateCustomer, and verifies transaction rollback when hook fails. | Hook tests must verify rollback: if hook returns an error the customer row must not exist. Use ErrorIs / ErrorAs against models.Generic* types for assertions. |

## Anti-Patterns

- Calling customer.Service from inside a hook without NewContextWithSkipSubjectCustomer(ctx) — causes infinite re-entrant hook invocations.
- Performing DB writes outside transaction.Run / transaction.RunWithNoValue — partial writes are not rolled back on hook or publish failure.
- Returning raw fmt.Errorf for domain conditions (not-found, conflict, validation) — HTTP encoders and callers depend on models.Generic* typed errors.
- Importing app/common in test files — breaks isolation and can introduce import cycles; build test deps from package constructors (customertestutils.NewTestEnv) directly.
- Using context.Background() or context.TODO() in methods or tests — always propagate caller ctx to maintain tracing, cancellation, and t.Context() lifecycle.

## Decisions

- **All mutations wrapped in transaction.Run so hooks and event publishes are atomic with the adapter write.** — A hook failure (e.g. ledger provisioning) must roll back the customer row; publishing a stale event after a partial write would corrupt downstream state.
- **RequestValidatorRegistry and ServiceHookRegistry are separate registries with distinct call sites.** — Validators run before the transaction (cheap pre-condition check); hooks run inside the transaction where rollback is possible. Merging them would force pre-condition checks to acquire a transaction unnecessarily.
- **Service struct is only constructable via New(Config), not by direct struct literal.** — Ensures requestValidatorRegistry is always initialized (NewRequestValidatorRegistry()) and required fields are validated at construction time, preventing nil-panic bugs at first method call.

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
	if err := s.requestValidatorRegistry.ValidateUpdateCustomer(ctx, customer.UpdateCustomerInput{
		CustomerID: input.CustomerID,
	}); err != nil {
		return nil, models.NewGenericValidationError(err)
	}
// ...
```

<!-- archie:ai-end -->
