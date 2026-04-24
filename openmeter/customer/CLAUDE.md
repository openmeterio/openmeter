# customer

<!-- archie:ai-start -->

> Customer lifecycle management (CRUD, usage attributions, soft-delete) with two extension registries — RequestValidatorRegistry for pre-mutation cross-domain guards and ServiceHooks[Customer] for post-lifecycle callbacks — that avoid circular imports between billing, ledger, and subscription domains.

## Patterns

**transaction.Run wraps every mutation** — All CreateCustomer, UpdateCustomer, DeleteCustomer calls go through transaction.Run / transaction.RunWithNoValue in service/service.go, ensuring hooks and event publishes are atomic with the DB write. (`service/customer.go: return transaction.Run(ctx, s.adapter, func(ctx context.Context, tx customer.Adapter) (*Customer, error) { ... })`)
**RequestValidatorRegistry pre-mutation fan-out** — Before any DB write, service calls s.requestValidatorRegistry.ValidateXxx(ctx, input) which fans out to all registered validators (billing constraint, entitlement constraint). Errors from multiple validators are joined with errors.Join. (`requestvalidator.go: requestValidatorRegistry.ValidateDeleteCustomer fans out to all registered validators`)
**ServiceHooks[Customer] post-lifecycle fan-out** — models.ServiceHooks[Customer] is embedded in Service; callers register hooks via RegisterHooks(). Hooks must use NewContextWithSkipSubjectCustomer(ctx) to prevent re-entrant invocations from hooks that call back into customer.Service. (`service/service.go: s.hooks.PostCreate(ctx, customer) after successful adapter write`)
**models.Generic* error types for all domain conditions** — All customer-specific errors (KeyConflictError, SubjectKeyConflictError, UpdateAfterDeleteError) embed a models.Generic* error so HTTP encoders map them to the correct status code without special-casing. (`errors.go: KeyConflictError embeds models.NewGenericConflictError(...)`)
**TransactingRepo wrapper on every adapter method** — adapter/customer.go wraps every method body in entutils.TransactingRepo so the adapter automatically rebinds to any in-flight transaction from ctx, or starts its own. (`adapter/customer.go: entutils.TransactingRepo(ctx, a.db, func(ctx context.Context, tx *entdb.Tx) (*Customer, error) { ... })`)
**Soft-delete pattern via DeletedAt** — Customers are never hard-deleted. DeleteCustomer sets DeletedAt; queries default to excluding deleted records (IncludeDeleted=false). Customer.IsDeleted() checks DeletedAt < clock.Now(). (`customer.go: func (c Customer) IsDeleted() bool { return c.DeletedAt != nil && c.DeletedAt.Before(clock.Now()) }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `openmeter/customer/service.go` | Service interface: composes CustomerService + RequestValidatorService + models.ServiceHooks[Customer]. This is the full API surface for the customer domain. | Service.RegisterHooks and Service.RegisterRequestValidator are both needed — one for lifecycle hooks, one for pre-mutation validation. They are separate registries with separate call sites. |
| `openmeter/customer/customer.go` | Customer, CustomerMutate, CustomerID, CustomerUsageAttribution domain types and all input types (CreateCustomerInput, UpdateCustomerInput, ListCustomersInput, GetCustomerInput). | CustomerID is a type alias for models.NamespacedID, not a struct embed. GetCustomerInput supports three lookup modes (by ID, by Key, by IDOrKey) — never accept a raw string; always construct one of the typed lookup structs. |
| `openmeter/customer/requestvalidator.go` | RequestValidator interface, NoopRequestValidator, and requestValidatorRegistry with RWMutex fan-out. Validators enforce cross-domain pre-conditions (billing, entitlements) without importing those packages. | validators.Register() is not idempotent — registering the same validator twice doubles validation calls. Wire providers must guard against double-registration. |
| `openmeter/customer/errors.go` | Typed domain errors: KeyConflictError, SubjectKeyConflictError, UpdateAfterDeleteError, ErrDeletingCustomerWithActiveSubscriptions (as ValidationIssue). | ErrDeletingCustomerWithActiveSubscriptions is a models.ValidationIssue (not a models.GenericError) and carries the list of active subscription IDs in attrs. |
| `openmeter/customer/adapter.go` | Adapter interface: composes CustomerAdapter + entutils.TxCreator. The persistence boundary. | Adapter never returns domain events — that is the service layer's responsibility. Adapter methods return typed domain objects (*Customer) not raw Ent rows. |

## Anti-Patterns

- Calling customer.Service from inside a hook without NewContextWithSkipSubjectCustomer(ctx) — causes infinite re-entrant hook invocations.
- Performing DB writes outside transaction.Run / transaction.RunWithNoValue in the service layer — partial writes are not rolled back on hook or publish failure.
- Returning raw fmt.Errorf for domain conditions (not-found, conflict, validation) — HTTP encoders depend on models.Generic* typed errors.
- Using context.Background() or context.TODO() in service or adapter methods — always propagate caller ctx.
- Importing app/common in test files under customer/testutils — causes import cycles; build test deps from package constructors directly.

## Decisions

- **RequestValidatorRegistry and ServiceHooks are separate registries with distinct call sites.** — Pre-mutation validators run before any DB write and block the operation on failure; post-lifecycle hooks run after success and may have side-effects. Mixing them into one registry would lose this timing guarantee.
- **Soft-delete rather than hard delete for customers and customer_subjects.** — Billing, subscription, and entitlement records reference customer IDs and cannot be orphaned. Soft-delete preserves referential integrity while logically removing the customer.

## Example: Registering a cross-domain pre-mutation validator that blocks customer deletion when billing records exist

```
package billingvalidators

import (
	"context"
	"github.com/openmeterio/openmeter/openmeter/customer"
)

type BillingCustomerValidator struct {
	customer.NoopRequestValidator
	billingAdapter BillingAdapter
}

var _ customer.RequestValidator = (*BillingCustomerValidator)(nil)

func (v *BillingCustomerValidator) ValidateDeleteCustomer(ctx context.Context, input customer.DeleteCustomerInput) error {
// ...
```

<!-- archie:ai-end -->
